package blockchain

import "testing"

func TestCurrencySymbolUsesTender(t *testing.T) {
	if got := CurrencySymbol(); got != "TENDER" {
		t.Fatalf("expected currency symbol TENDER, got %q", got)
	}
}
