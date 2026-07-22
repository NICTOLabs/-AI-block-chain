package main

import (
	"testing"
)

func TestMempoolReplacesLowerFeeSameSenderNonce(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	bc.AddAccount("alice", 1000, false)

	oldTx := Transaction{ID: "tx-1", From: "alice", To: "bob", Amount: 10, Fee: 5, Nonce: 1, TxType: Transfer}
	newTx := Transaction{ID: "tx-1", From: "alice", To: "bob", Amount: 10, Fee: 25, Nonce: 1, TxType: Transfer}

	bc.EnqueueTransaction(oldTx)
	bc.EnqueueTransaction(newTx)

	if len(bc.Pending) != 1 {
		t.Fatalf("expected one pending transaction after replacement, got %d", len(bc.Pending))
	}
	if bc.Pending[0].Fee != 25 {
		t.Fatalf("expected replacement fee to be 25, got %d", bc.Pending[0].Fee)
	}
}

func TestValidateChainRejectsTamperedBlock(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	block := Block{Index: 1, PreviousHash: bc.Chain[0].BlockHash, Timestamp: 1, Transactions: []Transaction{}, Nonce: 0}
	block.BlockHash = calculateHash(block)
	if bc.validateBlock(block, bc.Chain[0]) != nil {
		t.Fatalf("expected initial block to validate")
	}
	block.Transactions = []Transaction{{ID: "tampered", From: "human1", To: "agentA", Amount: 5, Fee: 5, Nonce: 1, TxType: Transfer}}
	if bc.validateBlock(block, bc.Chain[0]) == nil {
		t.Fatalf("expected tampered block to be rejected")
	}
}

func TestCreateAgreementAndMeterUsage(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	agreement, err := bc.CreateServiceAgreement("agentA", "agentB", "model-1", 100, 2)
	if err != nil {
		t.Fatalf("expected agreement creation, got %v", err)
	}
	if agreement.Status != "active" {
		t.Fatalf("expected active agreement, got %s", agreement.Status)
	}

	meter, err := bc.RecordUsage(agreement.ID, 3)
	if err != nil {
		t.Fatalf("expected usage record, got %v", err)
	}
	if meter.UsageCount != 3 {
		t.Fatalf("expected usage count 3, got %d", meter.UsageCount)
	}
}
