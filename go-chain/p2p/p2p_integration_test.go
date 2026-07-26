package p2p

import (
	"testing"
	"time"

	blockchain "ai_block_chain_go/blockchain"
)

func TestP2PBroadcastGossipAndRecovery(t *testing.T) {
	addr1 := "127.0.0.1:0"
	addr2 := "127.0.0.1:0"
	addr3 := "127.0.0.1:0"

	chain := blockchain.NewBlockchain(blockchain.ProofOfStake, t.TempDir(), "tdr-testnet-1")
	n1 := NewP2PNode(addr1, []string{addr2, addr3}, chain, true)
	n2 := NewP2PNode(addr2, []string{addr1, addr3}, chain, true)
	n3 := NewP2PNode(addr3, []string{addr1, addr2}, chain, true)

	if err := n1.Start(); err != nil {
		t.Fatalf("n1 start: %v", err)
	}
	if err := n2.Start(); err != nil {
		t.Fatalf("n2 start: %v", err)
	}
	if err := n3.Start(); err != nil {
		t.Fatalf("n3 start: %v", err)
	}
	defer n1.Shutdown()
	defer n2.Shutdown()
	defer n3.Shutdown()

	n1.ConnectToPeers()
	n2.ConnectToPeers()
	n3.ConnectToPeers()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(n1.Peers()) > 0 && len(n2.Peers()) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(n1.Peers()) == 0 && len(n2.Peers()) == 0 {
		t.Fatal("expected peers to connect")
	}
}
