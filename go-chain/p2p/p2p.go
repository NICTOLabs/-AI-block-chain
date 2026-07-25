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
	Address string   `json:"address"`
	Peers   []string `json:"peers"`
}

type p2pMessage struct {
	Type  string          `json:"type"`
	From  string          `json:"from,omitempty"`
	NodeID string         `json:"node_id,omitempty"`
	Block *blockchain.Block       `json:"block,omitempty"`
	Chain []blockchain.Block      `json:"chain,omitempty"`
	Tx    *blockchain.Transaction `json:"tx,omitempty"`
	Peer  *NodeInfo        `json:"peer,omitempty"`
}

type P2PNode struct {
	addr         string
	peers        []string
	peerScores   map[string]int
	trustedPeers map[string]bool
	chain        *blockchain.Blockchain
	listener     net.Listener
	shutdown     chan struct{}
	maxPeers     int
	strictMode   bool
	nodeSecret   string
	mutedPeers   map[string]time.Time
	peerRateLimits map[string][]time.Time
	peerConns    map[string]net.Conn
	peerWriters  map[string]*bufio.Writer
	peerMu       sync.Mutex
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
	if c, ok := p2p.peerConns[target]; ok {
		w := p2p.peerWriters[target]
		p2p.peerMu.Unlock()
		return c, w, nil
	}
	p2p.peerMu.Unlock()

	conn, err := net.DialTimeout("tcp", target, peerDialTimeout)
	if err != nil {
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

func (p2p *P2PNode) writePeer(target string, payload []byte) error {
	_, w, err := p2p.ensurePeerConn(target)
	if err != nil {
		p2p.peerMu.Lock()
		_ = p2p.peerConns[target].Close()
		delete(p2p.peerConns, target)
		delete(p2p.peerWriters, target)
		p2p.peerMu.Unlock()
		return err
	}
	_, writeErr := w.Write(append(payload, '\n'))
	if writeErr != nil {
		p2p.peerMu.Lock()
		_ = p2p.peerConns[target].Close()
		delete(p2p.peerConns, target)
		delete(p2p.peerWriters, target)
		p2p.peerMu.Unlock()
		return err
	}
	if err := w.Flush(); err != nil {
		p2p.peerMu.Lock()
		_ = p2p.peerConns[target].Close()
		delete(p2p.peerConns, target)
		delete(p2p.peerWriters, target)
		p2p.peerMu.Unlock()
		return err
	}
	return nil
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
				ping := p2pMessage{Type: "ping", From: p2p.addr}
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

func (p2p *P2PNode) Start() {
	go p2p.keepAlivePeers()
	listener, err := net.Listen("tcp", p2p.addr)
	if err != nil {
		blockchain.LogJSON("p2p_listen_failed", p2p.addr, err.Error())
		return
	}
	p2p.listener = listener
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
}

func (p2p *P2PNode) ConnectToPeers() {
	for _, peer := range p2p.peers {
		if peer == "" || peer == p2p.addr {
			continue
		}
		if len(p2p.peers) > p2p.maxPeers {
			break
		}
		go func(target string) {
			_, w, err := p2p.ensurePeerConn(target)
			if err != nil {
				blockchain.LogJSON("connect_peer", target, err.Error())
				return
			}
			p2p.peerScores[target] = 1
			p2p.trustedPeers[target] = true
			_ = p2p.WriteMessage(w, p2pMessage{Type: "hello", From: p2p.addr, Peer: &NodeInfo{Address: p2p.addr, Peers: p2p.peers}})
		}(peer)
	}
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
		var msg p2pMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			blockchain.LogJSON("p2p_decode", remote, err.Error())
			return
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
		}
		if msg.Type == "tx" && msg.Tx != nil {
			p2p.chain.EnqueueTransaction(*msg.Tx)
		}
		if msg.Type == "hello" && msg.Peer != nil {
			if p2p.strictMode && len(p2p.peers) >= p2p.maxPeers {
				continue
			}
			if msg.Peer.Address != "" && msg.Peer.Address != p2p.addr {
				p2p.peers = append(p2p.peers, msg.Peer.Address)
				p2p.peerScores[msg.Peer.Address] = 1
				p2p.trustedPeers[msg.Peer.Address] = true
			}
		}
	}
}

func (p2p *P2PNode) BroadcastBlock(block *blockchain.Block) {
	msg := p2pMessage{Type: "block", Block: block}
	payload, _ := json.Marshal(msg)
	for _, peer := range p2p.peers {
		if peer == "" || peer == p2p.addr || !p2p.trustedPeers[peer] {
			continue
		}
		_ = p2p.writePeer(peer, payload)
	}
}

func (p2p *P2PNode) WriteMessage(w io.Writer, msg p2pMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = w.Write(append(payload, '\n'))
	return err
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
	}
}

func (p2p *P2PNode) Addr() string { return p2p.addr }
func (p2p *P2PNode) Peers() []string { return p2p.peers }
func (p2p *P2PNode) TrustedPeers() map[string]bool { return p2p.trustedPeers }
func (p2p *P2PNode) StrictMode() bool { return p2p.strictMode }

func (p2p *P2PNode) Shutdown() chan struct{} { return p2p.shutdown }
