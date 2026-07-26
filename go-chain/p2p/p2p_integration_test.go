package p2p

import (
	"testing"
	"time"

	blockchain "ai_block_chain_go/blockchain"
)

func TestP2PPartitionRecoveryWithCatchUp(t *testing.T) {
	addr1 := "127.0.0.1:0"
	addr2 := "127.0.0.1:0"
	addr3 := "127.0.0.1:0"

	chainA := blockchain.NewBlockchain(blockchain.ProofOfStake, t.TempDir(), "tdr-testnet-a")
	chainC := blockchain.NewBlockchain(blockchain.ProofOfStake, t.TempDir(), "tdr-testnet-c")

	n1 := NewP2PNode(addr1, []string{addr2}, chainA, true)
	n2 := NewP2PNode(addr2, []string{addr1}, chainA, true)
	n3 := NewP2PNode(addr3, []string{addr2}, chainC, true)

	chainA.AddAccount("miner-a", 1_000_000_000, false)
	_ = chainA.RegisterValidator("miner-a", 1_000_000_000, "pubkey-a")

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

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(n1.Peers()) > 0 && len(n2.Peers()) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	for i := 0; i < 3; i++ {
		_, _ = chainA.MineBlockFor("miner-a")
	}

	if len(chainA.Chain) < 4 {
		t.Fatalf("expected chainA to grow, got %d", len(chainA.Chain))
	}

	n3.ConnectToPeers()

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(n3.Peers()) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if len(chainC.Chain) == 0 {
		t.Fatal("expected chainC to remain with genesis")
	}
}
