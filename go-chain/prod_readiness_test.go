package main

import (
	"testing"
)

func TestRejectsReplayedNonce(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir())
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

func TestComputeChainWorkPrefersMoreDifficultChain(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir())

	blockA := Block{Index: 1, PreviousHash: "0", Timestamp: 1, Nonce: 1, BlockHash: "abc"}
	blockB := Block{Index: 1, PreviousHash: "0", Timestamp: 2, Nonce: 2, BlockHash: "0000abc"}

	workA := bc.computeChainWork([]Block{blockA})
	workB := bc.computeChainWork([]Block{blockB})

	if workA >= workB {
		t.Fatalf("expected chain with more leading zeroes to have greater work, got %d and %d", workA, workB)
	}
}
