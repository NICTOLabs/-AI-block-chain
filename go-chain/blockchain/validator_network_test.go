package blockchain

import "testing"

func TestRegisterValidatorUpdatesValidatorSet(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	bc.AddAccount("validator-a", MinStake, false)
	if err := bc.RegisterValidator("validator-a", MinStake); err != nil {
		t.Fatalf("expected validator registration to succeed: %v", err)
	}
	validator, ok := bc.Validators["validator-a"]
	if !ok || !validator.Active {
		t.Fatal("expected validator to be registered and active")
	}
}
