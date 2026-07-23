package main

import "testing"

func TestValidatorEconomicsAndRegistry(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir())
	bc.AddAccount("validatorA", 2000, false)
	bc.Stake("validatorA", 1000)
	if got := bc.selectValidator(); got != "validatorA" {
		t.Fatalf("expected validatorA to be selected, got %s", got)
	}
	bc.Slash("validatorA", 400)
	if account := bc.Ledger["validatorA"]; account.Staked != 600 {
		t.Fatalf("expected staked amount to drop to 600, got %d", account.Staked)
	}
}
