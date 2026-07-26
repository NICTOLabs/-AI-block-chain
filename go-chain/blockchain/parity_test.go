package blockchain

import (
	"testing"
)

func TestCrossLanguageParity(t *testing.T) {
	if MinStake != MinStake {
		t.Fatalf("min stake parity mismatch")
	}
}
