package blockchain

import (
	"testing"
)

func TestRejectsReplayedNonce(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1", "")
	bc.AddAccount("human1", 1000, false)

	tx := Transaction{
		ID:        "tx-replay",
		From:      "human1",
		To:        "agentA",
		Amount:    100,
		Fee:       5,
		Nonce:     1,
		TxType:    Transfer,
		Timestamp: 1,
	}

	bc.UsedNonces["human1"] = map[uint64]struct{}{1: {}}

	if bc.validateTransaction(tx) {
		t.Fatalf("expected replayed nonce to be rejected")
	}
}

func TestRejectsInvalidChainID(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1", "")
	bc.AddAccount("human1", 1000, false)

	tx := Transaction{
		ID:        "tx-bad-chain",
		From:      "human1",
		To:        "agentA",
		Amount:    100,
		Fee:       5,
		Nonce:     1,
		TxType:    Transfer,
		Timestamp: 1,
		ChainID:   "wrong-chain",
	}

	if bc.validateTransaction(tx) {
		t.Fatalf("expected wrong chain ID to be rejected")
	}
}
