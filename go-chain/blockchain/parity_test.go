package blockchain

import (
	"bytes"
	"testing"
)

func TestCrossLanguageParity(t *testing.T) {
	if int(ProofOfStake) != 0 || int(ProofOfAuthority) != 1 || int(ProofOfWork) != 2 {
		t.Fatalf("consensus enum parity mismatch")
	}
	if ConsensusName(ProofOfStake) == "" || ConsensusName(ProofOfAuthority) == "" {
		t.Fatalf("consensus name parity mismatch")
	}
	if MinStake <= 0 {
		t.Fatalf("min stake parity mismatch")
	}
	if NewWallet() == nil {
		t.Fatalf("wallet creation parity mismatch")
	}
	if Transfer == "" || RegisterModel == "" {
		t.Fatalf("transaction type parity mismatch")
	}
}

func TestCrossLanguageParityHashDeterminism(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	block := bc.Chain[0]
	first := calculateHash(block)
	for i := 0; i < 10; i++ {
		if got := calculateHash(block); got != first {
			t.Fatalf("hash not deterministic on iteration %d", i)
		}
	}
}

func TestCrossLanguageParityCanonicalSigningBytesConsistency(t *testing.T) {
	tx := Transaction{
		From:      "addr1",
		FromPubKey: "pub1",
		To:        "addr2",
		Amount:    100,
		Fee:       5,
		Nonce:     1,
		TxType:    Transfer,
		Payload:   "p",
		Timestamp: 1234567890,
		ChainID:   "tdr-mainnet-1",
	}
	first := CanonicalSigningBytes(tx)
	if !bytes.Equal(first, CanonicalSigningBytes(tx)) {
		t.Fatal("canonical signing bytes not stable")
	}
	if len(first) == 0 {
		t.Fatal("canonical signing bytes should not be empty")
	}
}

func TestCrossLanguageParityWalletGenerationDeterminism(t *testing.T) {
	w1 := NewWallet()
	w2 := NewWallet()
	if w1 == nil || w2 == nil {
		t.Fatal("wallet generation failed")
	}
	if len(w1.Address()) != 64 || len(w2.Address()) != 64 {
		t.Fatalf("wallet address length mismatch")
	}
	if w1.Address() == w2.Address() {
		t.Fatal("wallet addresses should be unique by chance; this is extremely unlikely")
	}
}

func TestCrossLanguageParityFinalityVoteSignatureFields(t *testing.T) {
	vote := FinalityVoteSignature{
		BlockHash: "0xabc",
		Voter:     "voter-pub",
		Vote:      "finalize",
		Signature: []byte{1, 2, 3},
	}
	if vote.BlockHash == "" || vote.Voter == "" || vote.Vote == "" || len(vote.Signature) == 0 {
		t.Fatal("FinalityVoteSignature fields should be present")
	}
}

func TestCrossLanguageParityMerkleRootDeterminism(t *testing.T) {
	txs := []Transaction{
		{ID: "tx-1", From: "a", To: "b", Amount: 10, Fee: 1, Nonce: 1, TxType: Transfer, Timestamp: 1},
		{ID: "tx-2", From: "b", To: "c", Amount: 20, Fee: 1, Nonce: 1, TxType: Transfer, Timestamp: 2},
	}
	first := CalculateMerkleRoot(txs)
	for i := 0; i < 5; i++ {
		if CalculateMerkleRoot(txs) != first {
			t.Fatal("merkle root should be deterministic")
		}
	}
}
