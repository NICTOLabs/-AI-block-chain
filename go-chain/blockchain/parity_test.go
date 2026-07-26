package blockchain

import (
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
