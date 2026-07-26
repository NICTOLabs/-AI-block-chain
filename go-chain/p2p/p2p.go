package p2p

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	blockchain "ai_block_chain_go/blockchain"
)

const peerKeepAlive = 30 * time.Second
const peerDialTimeout = 3 * time.Second
const peerWriteTimeout = 2 * time.Second

type NodeInfo struct {
	Address         string   `json:"address"`
	Peers           []string `json:"peers"`
	ValidatorPubKey string   `json:"validator_pubkey,omitempty"`
}

type Message struct {
	Type     string          `json:"type"`
	From     string          `json:"from,omitempty"`
	NodeID   string          `json:"node_id,omitempty"`
	Block    *blockchain.Block       `json:"block,omitempty"`
	Chain    []blockchain.Block      `json:"chain,omitempty"`
	Tx       *blockchain.Transaction `json:"tx,omitempty"`
	Peer     *NodeInfo        `json:"peer,omitempty"`
	StreamID string           `json:"stream_id,omitempty"`
}

type P2PNode struct {
	addr           string
	peers          []string
	peerScores     map[string]int
	trustedPeers   map[string]bool
	chain          *blockchain.Blockchain
	listener       net.Listener
	shutdown       chan struct{}
	maxPeers       int
	strictMode     bool
	nodeSecret     string
	mutedPeers     map[string]time.Time
	peerRateLimits map[string][]time.Time
	peerConns      map[string]net.Conn
	peerWriters    map[string]*bufio.Writer
	peerMu         sync.Mutex
	seenMsgIDs     map[string]time.Time
	seenMu         sync.Mutex
	dialBackoff    map[string]time.Time
	backoffMu      sync.Mutex
}

func (p2p *P2PNode) allowPeerRate(remote string) bool {
	now := time.Now()
	entries := p2p.peerRateLimits[remote]
	kept := entries[:0]
	for _, ts := range entries {
		if now.Sub(ts) <= time.Minute {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= 300 {
		return false
	}
	p2p.peerRateLimits[remote] = append(kept, now)
	return true
}

func (p2p *P2PNode) ensurePeerConn(target string) (net.Conn, *bufio.Writer, error) {
	p2p.peerMu.Lock()
	if c, ok := p2p.peerConns[target]; ok && p2p.isConnAlive(c) {
		w := p2p.peerWriters[target]
		p2p.peerMu.Unlock()
		return c, w, nil
	}
	p2p.cleanupPeerLocked(target)
	p2p.peerMu.Unlock()

	if !p2p.allowDial(target) {
		return nil, nil, net.ErrClosed
	}

	conn, err := net.DialTimeout("tcp", target, peerDialTimeout)
	if err != nil {
		p2p.recordDialFailure(target)
		return nil, nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	if tcp, ok := conn.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(peerKeepAlive)
	}

	w := bufio.NewWriter(conn)
	p2p.peerMu.Lock()
	p2p.peerConns[target] = conn
	p2p.peerWriters[target] = w
	p2p.peerMu.Unlock()

	return conn, w, nil
}

func (p2p *P2PNode) isConnAlive(c net.Conn) bool {
	if c == nil {
		return false
	}
	_ = c.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, err := c.Read(make([]byte, 0))
	_ = c.SetReadDeadline(time.Time{})
	return err == nil
}

func (p2p *P2PNode) allowDial(target string) bool {
	p2p.backoffMu.Lock()
	defer p2p.backoffMu.Unlock()
	if p2p.dialBackoff == nil {
		p2p.dialBackoff = make(map[string]time.Time)
	}
	next, ok := p2p.dialBackoff[target]
	if !ok {
		return true
	}
	return time.Now().After(next)
}

func (p2p *P2PNode) recordDialFailure(target string) {
	p2p.backoffMu.Lock()
	defer p2p.backoffMu.Unlock()
	if p2p.dialBackoff == nil {
		p2p.dialBackoff = make(map[string]time.Time)
	}
	backoff := 1 * time.Second
	if prev, ok := p2p.dialBackoff[target]; ok {
		if d := time.Since(prev); d < 0 {
			backoff = time.Duration(-int64(d) * 2)
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		} else if d < 30*time.Second {
			backoff = d * 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
	p2p.dialBackoff[target] = time.Now().Add(backoff)
}

func (p2p *P2PNode) cleanupPeerLocked(target string) {
	if c, ok := p2p.peerConns[target]; ok && c != nil {
		_ = c.Close()
	}
	delete(p2p.peerConns, target)
	delete(p2p.peerWriters, target)
}

func (p2p *P2PNode) writePeer(target string, payload []byte) error {
	_, w, err := p2p.ensurePeerConn(target)
	if err != nil {
		p2p.peerMu.Lock()
		p2p.cleanupPeerLocked(target)
		p2p.peerMu.Unlock()
		return err
	}
	framed := append(payload, '\n')
	_, writeErr := w.Write(framed)
	if writeErr != nil {
		p2p.peerMu.Lock()
		p2p.cleanupPeerLocked(target)
		p2p.peerMu.Unlock()
		return writeErr
	}
	if err := w.Flush(); err != nil {
		p2p.peerMu.Lock()
		p2p.cleanupPeerLocked(target)
		p2p.peerMu.Unlock()
		return err
	}
	return nil
}

func (p2p *P2PNode) WriteMessage(w io.Writer, msg Message) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(append(payload, '\n'))
	return err
}

func (p2p *P2PNode) keepAlivePeers() {
	ticker := time.NewTicker(peerKeepAlive)
	defer ticker.Stop()
	for {
		select {
		case <-p2p.shutdown:
			return
		case <-ticker.C:
			p2p.peerMu.Lock()
			for target, c := range p2p.peerConns {
				if c == nil {
					continue
				}
				ping := Message{Type: "ping", From: p2p.addr}
				payload, _ := json.Marshal(ping)
				w := p2p.peerWriters[target]
				if w != nil {
					_, _ = w.Write(append(payload, '\n'))
					_ = w.Flush()
				}
			}
			p2p.peerMu.Unlock()
		}
	}
}

func (p2p *P2PNode) Start() error {
	go p2p.keepAlivePeers()
	go p2p.startDiscovery()
	listener, err := net.Listen("tcp", p2p.addr)
	if err != nil {
		blockchain.LogJSON("p2p_listen_failed", p2p.addr, err.Error())
	return nil
}
	p2p.listener = listener
	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-p2p.shutdown:
					return
				default:
					blockchain.LogJSON("p2p_accept_error", p2p.addr, err.Error())
				}
				continue
			}
			go p2p.handleConn(conn)
		}
	}()
	return nil
}

func (p2p *P2PNode) startDiscovery() {
	_ = p2p.announceMDNS()
}

func (p2p *P2PNode) announceMDNS() error {
	addr, err := net.ResolveUDPAddr("udp", "[224.0.0.251]:5353")
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	announce := "tender-discovery:" + p2p.addr
	_, _ = conn.Write([]byte(announce))
	return nil
}

func (p2p *P2PNode) ConnectToPeers() {
	for _, peer := range p2p.peers {
		if peer == "" || peer == p2p.addr {
			continue
		}
		go func(target string) {
			_, w, err := p2p.ensurePeerConn(target)
			if err != nil {
				blockchain.LogJSON("connect_peer", target, err.Error())
				return
			}
			p2p.peerScores[target] = 1
			p2p.trustedPeers[target] = true
			var pubKey string
			if p2p.chain != nil {
				pubKey = p2p.chain.SelectValidatorPubKey()
			}
			_ = p2p.WriteMessage(w, Message{Type: "hello", From: p2p.addr, Peer: &NodeInfo{Address: p2p.addr, Peers: p2p.peers, ValidatorPubKey: pubKey}})
		}(peer)
	}
}

func (p2p *P2PNode) seenMessageID(id string) bool {
	if id == "" {
		return false
	}
	p2p.seenMu.Lock()
	defer p2p.seenMu.Unlock()
	if p2p.seenMsgIDs == nil {
		p2p.seenMsgIDs = make(map[string]time.Time)
	}
	if _, ok := p2p.seenMsgIDs[id]; ok {
		return true
	}
	p2p.seenMsgIDs[id] = time.Now()
	if len(p2p.seenMsgIDs) > 4096 {
		cut := time.Now().Add(-10 * time.Minute)
		for k, v := range p2p.seenMsgIDs {
			if v.Before(cut) {
				delete(p2p.seenMsgIDs, k)
			}
		}
	}
	return false
}

func (p2p *P2PNode) handleConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	remote := conn.RemoteAddr().String()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				blockchain.LogJSON("p2p_read", remote, err.Error())
			}
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 5*1024*1024 {
			blockchain.LogJSON("p2p_oversize", remote, "")
			return
		}
		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			blockchain.LogJSON("p2p_decode", remote, err.Error())
			return
		}
		if msg.StreamID != "" && p2p.seenMessageID(msg.StreamID) {
			continue
		}
		if !p2p.allowPeerRate(remote) {
			return
		}
		if msg.NodeID != "" {
			if p2p.trustedPeers[msg.From] && p2p.peerScores[msg.From] > 0 {
				p2p.peerScores[msg.From]++
			} else {
				p2p.peerScores[msg.From] = 1
				p2p.trustedPeers[msg.From] = true
			}
		}
		if msg.Type == "block" && msg.Block != nil {
			p2p.chain.Lock()
			if len(msg.Chain) > 0 {
				if p2p.chain.ReplaceChain(msg.Chain) {
					p2p.chain.Unlock()
					continue
				}
			}
			if len(p2p.chain.Chain) < int(msg.Block.Index)+1 || p2p.chain.Chain[len(p2p.chain.Chain)-1].BlockHash != msg.Block.PreviousHash {
				p2p.chain.Chain = append(p2p.chain.Chain, *msg.Block)
				_ = p2p.chain.SaveToDisk()
			}
			p2p.chain.Unlock()
			relayPayload, _ := json.Marshal(msg)
			for _, peer := range p2p.peers {
				if peer == "" || peer == p2p.addr || peer == remote || !p2p.trustedPeers[peer] {
					continue
				}
				if p2p.seenMessageID(msg.StreamID) {
					continue
				}
				relayConn, err := net.DialTimeout("tcp", peer, 3*time.Second)
				if err != nil {
					continue
				}
				_, _ = relayConn.Write(append(relayPayload, '\n'))
				relayConn.Close()
			}
		}
		if msg.Type == "tx" && msg.Tx != nil {
			p2p.chain.EnqueueTransaction(*msg.Tx)
			relayPayload, _ := json.Marshal(msg)
			for _, peer := range p2p.peers {
				if peer == "" || peer == p2p.addr || peer == remote || !p2p.trustedPeers[peer] {
					continue
				}
				if p2p.seenMessageID(msg.StreamID) {
					continue
				}
				relayConn, err := net.DialTimeout("tcp", peer, 3*time.Second)
				if err != nil {
					continue
				}
				_, _ = relayConn.Write(append(relayPayload, '\n'))
				relayConn.Close()
			}
		}
		if msg.Type == "hello" && msg.Peer != nil {
			if p2p.strictMode && len(p2p.peers) >= p2p.maxPeers {
				continue
			}
			if msg.Peer.Address != "" && msg.Peer.Address != p2p.addr {
				if p2p.strictMode && msg.Peer.ValidatorPubKey != "" {
					if !p2p.chain.IsActiveValidatorPubKey(msg.Peer.ValidatorPubKey) {
						continue
					}
				}
				p2p.peers = append(p2p.peers, msg.Peer.Address)
				p2p.peerScores[msg.Peer.Address] = 1
				p2p.trustedPeers[msg.Peer.Address] = true
			}
		}
	}
}

func (p2p *P2PNode) BroadcastBlock(block *blockchain.Block) {
	msg := Message{Type: "block", Block: block}
	payload, _ := json.Marshal(msg)
	for _, peer := range p2p.peers {
		if peer == "" || peer == p2p.addr || !p2p.trustedPeers[peer] {
			continue
		}
		_ = p2p.writePeer(peer, payload)
	}
}

func (p2p *P2PNode) BroadcastTransaction(tx blockchain.Transaction) {
	msg := Message{Type: "tx", Tx: &tx}
	payload, _ := json.Marshal(msg)
	for _, peer := range p2p.peers {
		if peer == "" || peer == p2p.addr || !p2p.trustedPeers[peer] {
			continue
		}
		_ = p2p.writePeer(peer, payload)
	}
}

func NewP2PNode(addr string, peers []string, chain *blockchain.Blockchain, strictMode bool) *P2PNode {
	return &P2PNode{
		addr:           addr,
		peers:          peers,
		peerScores:     make(map[string]int),
		trustedPeers:   make(map[string]bool),
		chain:          chain,
		shutdown:       make(chan struct{}),
		maxPeers:       50,
		strictMode:     strictMode,
		peerRateLimits: make(map[string][]time.Time),
		peerConns:      make(map[string]net.Conn),
		peerWriters:    make(map[string]*bufio.Writer),
		seenMsgIDs:     make(map[string]time.Time),
		dialBackoff:    make(map[string]time.Time),
	}
}

func (p2p *P2PNode) Addr() string { return p2p.addr }
func (p2p *P2PNode) Peers() []string { return p2p.peers }
func (p2p *P2PNode) TrustedPeers() map[string]bool { return p2p.trustedPeers }
func (p2p *P2PNode) StrictMode() bool { return p2p.strictMode }
func (p2p *P2PNode) AddPeer(peer string) {
	if peer == "" || peer == p2p.addr {
		return
	}
	p2p.peers = append(p2p.peers, peer)
}

func (p2p *P2PNode) Shutdown() chan struct{} { return p2p.shutdown }
