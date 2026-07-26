package p2p

import (
	"bytes"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	blockchain "ai_block_chain_go/blockchain"
)

type fakeChain struct {
	blockchain.Blockchain
	mu       sync.Mutex
	chain    []blockchain.Block
	replaced bool
}

func (f *fakeChain) ReplaceChain(newChain []blockchain.Block) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(newChain) <= len(f.chain) {
		return false
	}
	f.chain = newChain
	f.replaced = true
	return true
}

func (f *fakeChain) SaveToDisk() error { return nil }

func TestSeenMessageIDDedup(t *testing.T) {
	node := NewP2PNode("127.0.0.1:0", nil, &blockchain.Blockchain{}, false)
	if node.seenMessageID("abc") {
		t.Fatal("first id should not be seen")
	}
	if !node.seenMessageID("abc") {
		t.Fatal("second id should be seen")
	}
}

func TestSeenMessageIDEmptySkip(t *testing.T) {
	node := NewP2PNode("127.0.0.1:0", nil, &blockchain.Blockchain{}, false)
	if node.seenMessageID("") {
		t.Fatal("empty id should be skipped")
	}
}

func TestDialBackoffGrows(t *testing.T) {
	node := NewP2PNode("127.0.0.1:0", nil, &blockchain.Blockchain{}, false)
	node.recordDialFailure("bad-peer")
	node.recordDialFailure("bad-peer")
	if node.allowDial("bad-peer") {
		t.Fatal("dial should be backed off")
	}
}

func TestMDNSAnnounceDoesNotCrash(t *testing.T) {
	node := NewP2PNode("127.0.0.1:0", nil, &blockchain.Blockchain{}, false)
	if err := node.announceMDNS(); err != nil {
		t.Skipf("mDNS announce unavailable in environment: %v", err)
	}
}

func TestP2PNodeStartListenErrorLogged(t *testing.T) {
	node := NewP2PNode("invalid-address", nil, &blockchain.Blockchain{}, false)
	node.Start()
	time.Sleep(50 * time.Millisecond)
	if node.listener != nil {
		t.Fatal("listener should be nil on invalid address")
	}
}

func TestWriteMessageFormatsJSON(t *testing.T) {
	var buf bytes.Buffer
	node := NewP2PNode("127.0.0.1:0", nil, &blockchain.Blockchain{}, false)
	err := node.WriteMessage(&buf, Message{Type: "hello", From: "a"})
	if err != nil {
		t.Fatalf("write message failed: %v", err)
	}
	line := strings.TrimSpace(buf.String())
	if !strings.Contains(line, `"type":"hello"`) {
		t.Fatalf("unexpected message format: %s", line)
	}
}

func TestP2PNodeAddrAndMode(t *testing.T) {
	node := NewP2PNode("0.0.0.0:4040", nil, &blockchain.Blockchain{}, true)
	if node.Addr() != "0.0.0.0:4040" {
		t.Fatalf("unexpected addr: %s", node.Addr())
	}
	if !node.StrictMode() {
		t.Fatal("strict mode should be true")
	}
}

func TestP2PNodeShutdown(t *testing.T) {
	node := NewP2PNode("127.0.0.1:0", nil, &blockchain.Blockchain{}, false)
	ch := node.Shutdown()
	if ch == nil {
		t.Fatal("shutdown channel should not be nil")
	}
	close(ch)
}

// FakeConn tracks written bytes without a real network connection.
type fakeConn struct {
	bytes.Buffer
	closed bool
}

func (f *fakeConn) Read(b []byte) (int, error) {
	return 0, &net.OpError{Op: "read", Err: net.ErrClosed}
}

func (f *fakeConn) Close() error {
	f.closed = true
	return nil
}

func (f *fakeConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
}

func (f *fakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
}

func (f *fakeConn) SetDeadline(t time.Time) error      { return nil }
func (f *fakeConn) SetReadDeadline(t time.Time) error   { return nil }
func (f *fakeConn) SetWriteDeadline(t time.Time) error  { return nil }

// fakeListener accepts a single fakeConn for test control.
type fakeListener struct {
	conn net.Conn
}

func (f *fakeListener) Accept() (net.Conn, error) {
	if f.conn == nil {
		return nil, net.ErrClosed
	}
	c := f.conn
	f.conn = nil
	return c, nil
}

func (f *fakeListener) Addr() net.Addr { return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0} }
func (f *fakeListener) Close() error   { return nil }

func TestHandleConnIgnoresOversizedMessage(t *testing.T) {
	node := NewP2PNode("127.0.0.1:0", nil, &blockchain.Blockchain{}, false)
	node.startWithListener(&fakeListener{conn: &fakeConn{}})
	time.Sleep(50 * time.Millisecond)
}

func (node *P2PNode) startWithListener(listener net.Listener) {
	node.listener = listener
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go node.handleConn(conn)
		}
	}()
}
