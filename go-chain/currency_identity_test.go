package main

import "testing"

func TestCurrencySymbolUsesTender(t *testing.T) {
	if got := currencySymbol(); got != "TENDER" {
		t.Fatalf("expected currency symbol TENDER, got %q", got)
	}
}
