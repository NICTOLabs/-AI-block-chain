package p2p

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	blockchain "ai_block_chain_go/blockchain"
)

const noiseProto = "/noise/1.0.0"
const yamuxProto = "/yamux/1.0.0"
const libp2pProto = "/tender/1.0.0"
const validatorAuthProto = "/tender/validator-auth/1.0.0"

type LibP2PMessage struct {
	Type      string                 `json:"type,omitempty"`
	From      string                 `json:"from,omitempty"`
	To        string                 `json:"to,omitempty"`
	PubKey    string                 `json:"pub_key,omitempty"`
	Address   string                 `json:"address,omitempty"`
	Signature []byte                 `json:"signature,omitempty"`
	Block     *blockchain.Block      `json:"block,omitempty"`
	Tx        *blockchain.Transaction `json:"tx,omitempty"`
}

type LibP2PNode struct {
	mu            sync.RWMutex
	privKey       ed25519.PrivateKey
	pubKey        ed25519.PublicKey
	peerID        string
	addr          string
	listener      net.Listener
	strictMode    bool
	validatorSet  map[string]struct{}
	validatorAddr string
	peers         map[string]*peerSession
	shutdownCh    chan struct{}
	wg            sync.WaitGroup
}

type peerSession struct {
	remotePeerID string
	conn         net.Conn
	reader       *bufio.Reader
	trusted      bool
	lastSeen     time.Time
	muxers       map[string]bool
}

func NewLibP2PNode(addr string, strictMode bool) (*LibP2PNode, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate identity key: %w", err)
	}
	return &LibP2PNode{
		privKey:       priv,
		pubKey:        pub,
		peerID:       fmt.Sprintf("libp2p:%x", sha256.Sum256(pub)),
		addr:          addr,
		strictMode:    strictMode,
		validatorSet: make(map[string]struct{}),
		peers:         make(map[string]*peerSession),
		shutdownCh:   make(chan struct{}),
	}, nil
}

func (n *LibP2PNode) Start() error {
	ln, err := net.Listen("tcp", n.addr)
	if err != nil {
		return fmt.Errorf("libp2p listen: %w", err)
	}
	n.mu.Lock()
	n.listener = ln
	n.mu.Unlock()
	n.wg.Add(1)
	go n.acceptLoop()
	n.wg.Add(1)
	go n.startMDNS()
	n.wg.Add(1)
	go n.startNATHolePunch()
	return nil
}

func (n *LibP2PNode) Stop() {
	close(n.shutdownCh)
	if n.listener != nil {
		n.listener.Close()
	}
	n.wg.Wait()
}

func (n *LibP2PNode) acceptLoop() {
	defer n.wg.Done()
	for {
		conn, err := n.listener.Accept()
		if err != nil {
			select {
			case <-n.shutdownCh:
				return
			default:
			}
			continue
		}
		n.wg.Add(1)
		go n.handleIncoming(conn)
	}
}

func (n *LibP2PNode) handleIncoming(conn net.Conn) {
	defer conn.Close()
	defer n.wg.Done()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	if !n.noiseHandshakeInbound(conn) {
		log.Printf("libp2p noise handshake failed")
		return
	}
	remotePeerID, ok := n.readVarintString(conn)
	if !ok {
		return
	}
	n.mu.Lock()
	if _, seen := n.peers[string(remotePeerID)]; seen {
		n.mu.Unlock()
		return
	}
	if n.strictMode && !strings.HasPrefix(string(remotePeerID), "validator:") {
		_, required := n.validatorSet[strings.TrimPrefix(string(remotePeerID), "validator:")]
		n.mu.Unlock()
		if !required {
			log.Printf("libp2p rejecting non-validator peer=%s", remotePeerID)
			return
		}
		n.mu.Lock()
	}
	session := &peerSession{
		remotePeerID: string(remotePeerID),
		conn:         conn,
		reader:       bufio.NewReader(conn),
		lastSeen:     time.Now(),
		trusted:      true,
		muxers:       make(map[string]bool),
	}
	n.peers[string(remotePeerID)] = session
	n.mu.Unlock()
	_ = n.sendValidatorAuth(session.conn, LibP2PMessage{
		Type:   "hello",
		From:   n.peerID,
		To:     string(remotePeerID),
		PubKey: fmt.Sprintf("%x", n.pubKey),
	})
	n.readLoop(session)
}

func (n *LibP2PNode) readLoop(session *peerSession) {
	defer func() {
		n.mu.Lock()
		delete(n.peers, session.remotePeerID)
		n.mu.Unlock()
	}()
	for {
		session.conn.SetDeadline(time.Now().Add(60 * time.Second))
		msg, ok, err := n.readMessage(session.reader)
		if err != nil || !ok {
			return
		}
		n.mu.Lock()
		session.lastSeen = time.Now()
		n.mu.Unlock()
		log.Printf("libp2p message from=%s type=%s", msg.From, msg.Type)
	}
}

func (n *LibP2PNode) Connect(addr string, remotePub ed25519.PublicKey) error {
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return err
	}
	if !n.noiseHandshakeOutbound(conn) {
		conn.Close()
		return fmt.Errorf("noise handshake failed")
	}
	peerID := fmt.Sprintf("libp2p:%x", sha256.Sum256(remotePub))
	if err := n.writeVarintString(conn, []byte(peerID)); err != nil {
		conn.Close()
		return err
	}
	hello := LibP2PMessage{
		Type:   "hello",
		From:   n.peerID,
		To:     peerID,
		PubKey: fmt.Sprintf("%x", n.pubKey),
	}
	if err := n.sendValidatorAuth(conn, hello); err != nil {
		conn.Close()
		return err
	}
	n.mu.Lock()
	if _, seen := n.peers[peerID]; !seen {
		n.peers[peerID] = &peerSession{
			remotePeerID: peerID,
			conn:         conn,
			reader:       bufio.NewReader(conn),
			lastSeen:     time.Now(),
			trusted:      true,
			muxers:       make(map[string]bool),
		}
	}
	n.mu.Unlock()
	go func(s *peerSession) {
		n.readLoop(s)
	}(n.peers[peerID])
	return nil
}

func (n *LibP2PNode) PeerID() string {
	return n.peerID
}

func (n *LibP2PNode) Addr() string {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ip, ok := a.(*net.IPNet); ok && !ip.IP.IsLoopback() && ip.IP.To4() != nil {
			return fmt.Sprintf("%s/%s", n.addr, ip.IP.String())
		}
	}
	return n.addr
}

func (n *LibP2PNode) Peers() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]string, 0, len(n.peers))
	for pid := range n.peers {
		out = append(out, pid)
	}
	return out
}

func (n *LibP2PNode) TrustedPeers() map[string]bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make(map[string]bool, len(n.peers))
	for pid, peer := range n.peers {
		out[pid] = peer.trusted
	}
	return out
}

func (n *LibP2PNode) StrictMode() bool {
	return n.strictMode
}

func (n *LibP2PNode) Shutdown() chan struct{} {
	return n.shutdownCh
}

func (n *LibP2PNode) BroadcastTransaction(tx blockchain.Transaction) {
	n.broadcast(LibP2PMessage{Type: "tx", Tx: &tx})
}

func (n *LibP2PNode) BroadcastBlock(block *blockchain.Block) {
	n.broadcast(LibP2PMessage{Type: "block", Block: block})
}

func (n *LibP2PNode) broadcast(msg LibP2PMessage) {
	n.mu.RLock()
	peers := make([]*peerSession, 0, len(n.peers))
	for _, p := range n.peers {
		peers = append(peers, p)
	}
	n.mu.RUnlock()
	for _, p := range peers {
		_ = n.sendMessage(p, msg)
	}
}

func (n *LibP2PNode) sendMessage(session *peerSession, msg LibP2PMessage) error {
	data, _ := json.Marshal(msg)
	if err := n.writeFrame(session.conn, data); err != nil {
		return err
	}
	return nil
}

func (n *LibP2PNode) sendValidatorAuth(target net.Conn, msg LibP2PMessage) error {
	data, _ := json.Marshal(msg)
	return n.writeFrame(target, data)
}

func (n *LibP2PNode) noiseHandshakeOutbound(conn net.Conn) bool {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{})
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return false
	}
	payload := sha256.Sum256(append(n.pubKey, nonce...))
	if _, err := conn.Write(payload[:]); err != nil {
		return false
	}
	buf := make([]byte, 32)
	if _, err := conn.Read(buf); err != nil {
		return false
	}
	return bytes.Equal(buf, sha256.New().Sum(nil))
}

func (n *LibP2PNode) noiseHandshakeInbound(conn net.Conn) bool {
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{})
	buf := make([]byte, 32)
	if _, err := conn.Read(buf); err != nil {
		return false
	}
	payload := sha256.Sum256(append(n.pubKey, buf...))
	if _, err := conn.Write(payload[:]); err != nil {
		return false
	}
	return true
}

func (n *LibP2PNode) writeFrame(w interface{ Write([]byte) (int, error) }, data []byte) error {
	_ = w.(interface{ SetWriteDeadline(time.Time) error }).SetWriteDeadline(time.Now().Add(3 * time.Second))
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(data)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	return nil
}

func (n *LibP2PNode) readFrame(r *bufio.Reader) ([]byte, bool) {
	var lenBuf [4]byte
	if _, err := r.Read(lenBuf[:]); err != nil {
		return nil, false
	}
	length := binary.BigEndian.Uint32(lenBuf[:])
	if length > 5*1024*1024 {
		return nil, false
	}
	data := make([]byte, length)
	if _, err := r.Read(data); err != nil {
		return nil, false
	}
	return data, true
}

func (n *LibP2PNode) readMessage(r *bufio.Reader) (LibP2PMessage, bool, error) {
	data, ok := n.readFrame(r)
	if !ok {
		return LibP2PMessage{}, false, nil
	}
	var msg LibP2PMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return LibP2PMessage{}, false, err
	}
	return msg, true, nil
}

func (n *LibP2PNode) writeVarintString(w interface{ Write([]byte) (int, error) }, s []byte) error {
	buf := make([]byte, binary.MaxVarintLen64)
	vn := binary.PutUvarint(buf, uint64(len(s)))
	if _, err := w.Write(buf[:vn]); err != nil {
		return err
	}
	if _, err := w.Write(s); err != nil {
		return err
	}
	return nil
}

func (n *LibP2PNode) readVarintString(r interface{ Read([]byte) (int, error) }) ([]byte, bool) {
	var lenBuf [binary.MaxVarintLen64]byte
	if _, err := r.Read(lenBuf[:]); err != nil {
		return nil, false
	}
	length, vn := binary.Uvarint(lenBuf[:])
	if length == 0 || length > 4096 {
		return nil, false
	}
	buf := make([]byte, length)
	if _, err := r.Read(buf); err != nil {
		return nil, false
	}
	if _, err := r.Read(lenBuf[:vn]); err != nil {
		return nil, false
	}
	return buf, true
}

func (n *LibP2PNode) startMDNS() {
	defer n.wg.Done()
	addr, err := net.ResolveUDPAddr("udp", "[224.0.0.251]:5353")
	if err != nil {
		return
	}
	c, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: addr.Port})
	if err != nil {
		return
	}
	defer c.Close()
	_ = c.SetReadDeadline(time.Now().Add(10 * time.Minute))
	buf := make([]byte, 4096)
	for {
		select {
		case <-n.shutdownCh:
			return
		default:
		}
		n, remote, err := c.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		msg := string(buf[:n])
		if strings.Contains(msg, "tender-discovery") {
			log.Printf("libp2p mDNS discovered peer=%s message=%s", remote, msg)
		}
	}
}

func (n *LibP2PNode) startNATHolePunch() {
	defer n.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-n.shutdownCh:
			return
		case <-ticker.C:
			n.mu.RLock()
			_ = len(n.peers)
			n.mu.RUnlock()
		}
	}
}

func (n *LibP2PNode) ValidatorSet(validators []string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, v := range validators {
		n.validatorSet[v] = struct{}{}
	}
}
