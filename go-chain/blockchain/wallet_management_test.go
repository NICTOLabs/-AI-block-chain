package blockchain

import "testing"

func TestCreateManagedWalletPersistsAddress(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	info, err := bc.CreateManagedWallet("human-1", false)
	if err != nil {
		t.Fatalf("expected wallet creation to succeed: %v", err)
	}
	if info.Address == "" {
		t.Fatal("expected wallet address to be populated")
	}
	if bc.Ledger[info.Address] == nil {
		t.Fatal("expected ledger account for managed wallet")
	}
	if len(bc.Wallets) != 1 {
		t.Fatalf("expected one managed wallet, got %d", len(bc.Wallets))
	}
}
