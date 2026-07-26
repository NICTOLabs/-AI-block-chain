package blockchain

import (
	"testing"
)

func BenchmarkMineBlock(b *testing.B) {
	bc := NewBlockchain(ProofOfStake, b.TempDir(), "bench")
	bc.AddAccount("miner", 1000, false)
	for i := 0; i < b.N; i++ {
		_, _ = bc.MineBlockFor("miner")
	}
}

func BenchmarkValidateBlock(b *testing.B) {
	bc := NewBlockchain(ProofOfStake, b.TempDir(), "bench")
	bc.AddAccount("miner", 1000, false)
	block, _ := bc.MineBlockFor("miner")
	if block == nil {
		b.Fatal("expected mined block")
	}
	prev := bc.Chain[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bc.validateBlock(*block, prev)
	}
}

func BenchmarkVerifyTransaction(b *testing.B) {
	wallet := NewWallet()
	tx := wallet.Sign(Transaction{
		From:    wallet.Address(),
		To:      "addr2",
		Amount:  10,
		Fee:     1,
		Nonce:   1,
		TxType:  Transfer,
		ChainID: "bench",
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		VerifyTransaction(tx)
	}
}

func BenchmarkMerkleProof(b *testing.B) {
	items := make([][]byte, 64)
	for i := range items {
		items[i] = []byte{byte(i)}
	}
	root, err := BuildMerkleTree(items)
	if err != nil {
		b.Fatal(err)
	}
	target := items[0]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateProof(root, target)
	}
}
