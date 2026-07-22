package main

import "testing"

func TestEstimateFeeUsesCongestionAndComplexity(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir())
	transferFee := bc.estimateFee(Transaction{TxType: Transfer}, 0)
	modelFee := bc.estimateFee(Transaction{TxType: RegisterModel, Payload: "model"}, 0)
	if modelFee <= transferFee {
		t.Fatalf("expected model fee to be higher than transfer fee, got %d and %d", transferFee, modelFee)
	}

	congestionFee := bc.estimateFee(Transaction{TxType: Transfer}, 20)
	if congestionFee <= transferFee {
		t.Fatalf("expected congestion to increase the fee, got %d and %d", transferFee, congestionFee)
	}
}

func TestSlashReducesStakeAndBalance(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir())
	bc.AddAccount("alice", 1000, false)
	bc.Ledger["alice"].Staked = 100

	bc.Slash("alice", 40)

	if bc.Ledger["alice"].Staked != 60 {
		t.Fatalf("expected staked amount to drop to 60, got %d", bc.Ledger["alice"].Staked)
	}
	if bc.Ledger["alice"].Balance != 960 {
		t.Fatalf("expected balance to drop to 960, got %d", bc.Ledger["alice"].Balance)
	}
}

func TestCreateEscrowLocksFunds(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir())
	bc.AddAccount("alice", 1000, false)
	bc.AddAccount("bob", 0, false)

	escrow, err := bc.CreateEscrow("alice", "bob", 200, "service-1")
	if err != nil {
		t.Fatalf("expected escrow creation to succeed: %v", err)
	}
	if bc.Ledger["alice"].Balance != 800 {
		t.Fatalf("expected escrow to lock 200 tokens, balance is %d", bc.Ledger["alice"].Balance)
	}
	if escrow.Status != "active" {
		t.Fatalf("expected escrow status to be active, got %s", escrow.Status)
	}
}
