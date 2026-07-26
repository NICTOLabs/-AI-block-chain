package p2p

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"ai_block_chain_go/blockchain"
)

type PeerID string

type SecureMessage struct {
	Type     string `json:"type"`
	From     PeerID `json:"from"`
	To       PeerID `json:"to,omitempty"`
	Data     []byte `json:"data,omitempty"`
	Nonce    uint64 `json:"nonce"`
	PubKey   string `json:"pub_key,omitempty"`
	Payload  string `json:"payload,omitempty"`
}

type LibP2PNode struct {
	addr       string
	privKey    ed25519.PrivateKey
	pubKey     ed25519.PublicKey
	peerID     PeerID
	peers      map[PeerID]*PeerSession
	mu         sync.RWMutex
	listener   net.Listener
	shutdown   chan struct{}
	maxPeers   int
}

type PeerSession struct {
	PeerID    PeerID
	PubKey    ed25519.PublicKey
	Conn      net.Conn
	LastSeen  time.Time
	Score     int
	Trusted   bool
}

func NewLibP2PNode(addr string) (*LibP2PNode, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	peerID := PeerID(hex.EncodeToString(pub))
	return &LibP2PNode{
		addr:     addr,
		privKey:  priv,
		pubKey:   pub,
		peerID:   peerID,
		peers:    make(map[PeerID]*PeerSession),
		maxPeers: 50,
	}, nil
}

func (n *LibP2PNode) Start() error {
	listener, err := net.Listen("tcp", n.addr)
	if err != nil {
		return err
	}
	n.listener = listener
	log.Printf("libp2p host started on %s peer=%s", n.addr, n.peerID)
	go n.acceptLoop()
	return nil
}

func (n *LibP2PNode) Stop() {
	close(n.shutdown)
	if n.listener != nil {
		n.listener.Close()
	}
	n.mu.Lock()
	for _, peer := range n.peers {
		peer.Conn.Close()
	}
	n.mu.Unlock()
}

func (n *LibP2PNode) acceptLoop() {
	for {
		conn, err := n.listener.Accept()
		if err != nil {
			select {
			case <-n.shutdown:
				return
			default:
				continue
			}
		}
		go n.handleIncoming(conn)
	}
}

func (n *LibP2PNode) handleIncoming(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	var msg SecureMessage
	if err := decoder.Decode(&msg); err != nil {
		return
	}
	if msg.Type != "hello" {
		return
	}
	pubKey, err := hex.DecodeString(msg.PubKey)
	if err != nil || len(pubKey) != ed25519.PublicKeySize {
		return
	}
	remotePeerID := PeerID(hex.EncodeToString(pubKey))
	if !ed25519.Verify(ed25519.PublicKey(pubKey), []byte(n.peerID), []byte(msg.Payload)) {
		return
	}
	n.mu.Lock()
	if len(n.peers) >= n.maxPeers {
		n.mu.Unlock()
		return
	}
	session := &PeerSession{
		PeerID:   remotePeerID,
		PubKey:   ed25519.PublicKey(pubKey),
		Conn:     conn,
		LastSeen: time.Now(),
		Score:    1,
		Trusted:  true,
	}
	n.peers[remotePeerID] = session
	n.mu.Unlock()
	ack := SecureMessage{
		Type:    "ack",
		From:    n.peerID,
		To:      remotePeerID,
		PubKey:  hex.EncodeToString(n.pubKey),
		Payload: string(remotePeerID),
	}
	_ = json.NewEncoder(conn).Encode(ack)
	n.readLoop(remotePeerID, conn)
}

func (n *LibP2PNode) readLoop(peerID PeerID, conn net.Conn) {
	defer func() {
		n.mu.Lock()
		delete(n.peers, peerID)
		n.mu.Unlock()
		conn.Close()
	}()
	decoder := json.NewDecoder(conn)
	for {
		var msg SecureMessage
		if err := decoder.Decode(&msg); err != nil {
			return
		}
		if msg.To != "" && msg.To != n.peerID {
			continue
		}
		log.Printf("libp2p message from=%s type=%s", msg.From, msg.Type)
	}
}

func (n *LibP2PNode) Connect(addr string, remotePub ed25519.PublicKey) error {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return err
	}
	remotePeerID := PeerID(hex.EncodeToString(remotePub))
	hello := SecureMessage{
		Type:    "hello",
		From:    n.peerID,
		To:      remotePeerID,
		PubKey:  hex.EncodeToString(n.pubKey),
		Payload: string(remotePeerID),
	}
	if err := json.NewEncoder(conn).Encode(hello); err != nil {
		conn.Close()
		return err
	}
	n.mu.Lock()
	if len(n.peers) >= n.maxPeers {
		n.mu.Unlock()
		conn.Close()
		return fmt.Errorf("peer limit reached")
	}
	session := &PeerSession{
		PeerID:   remotePeerID,
		PubKey:   remotePub,
		Conn:     conn,
		LastSeen: time.Now(),
		Score:    1,
		Trusted:  true,
	}
	n.peers[remotePeerID] = session
	n.mu.Unlock()
	go n.readLoop(remotePeerID, conn)
	return nil
}

func (n *LibP2PNode) PeerID() PeerID {
	return n.peerID
}

func (n *LibP2PNode) Addr() string {
	return n.addr
}

func (n *LibP2PNode) Peers() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := make([]string, 0, len(n.peers))
	for id := range n.peers {
		out = append(out, string(id))
	}
	return out
}

func (n *LibP2PNode) TrustedPeers() map[string]bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := make(map[string]bool, len(n.peers))
	for id, peer := range n.peers {
		out[string(id)] = peer.Trusted
	}
	return out
}

func (n *LibP2PNode) StrictMode() bool {
	return true
}

func (n *LibP2PNode) Shutdown() chan struct{} {
	if n.shutdown == nil {
		n.shutdown = make(chan struct{})
	}
	return n.shutdown
}

func (n *LibP2PNode) BroadcastTransaction(_ blockchain.Transaction) {}

func (n *LibP2PNode) BroadcastBlock(_ *blockchain.Block) {}
