package blockchain

import (
	"testing"
)

func TestMineBlockIncludesSignedTransfer(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	wallet := NewWallet()
	from := wallet.Address()
	bc.AddAccount(from, 1_000_000_000, false)
	to := "0000000000000000000000000000000000000000000000000000000000000001"
	bc.AddAccount(to, 0, false)

	fee := bc.estimateFee(Transaction{TxType: Transfer}, 0)
	if fee < BaseFee+FeeMultiplier+10 {
		fee = BaseFee + FeeMultiplier + 10
	}
	tx := wallet.Sign(Transaction{
		From:    from,
		To:      to,
		Amount:  10,
		Fee:     fee + 100,
		Nonce:   1,
		TxType:  Transfer,
		ChainID: bc.ChainID,
	})
	bc.EnqueueTransaction(tx)
	if len(bc.Pending) != 1 {
		t.Fatalf("expected 1 pending tx, got %d", len(bc.Pending))
	}
	pendingID := bc.Pending[0].ID

	block, err := bc.MineBlockFor(from)
	if err != nil {
		t.Fatalf("mine failed: %v", err)
	}
	if block == nil {
		t.Fatal("expected mined block")
	}
	if len(block.Transactions) != 1 {
		t.Fatalf("expected 1 tx in block, got %d", len(block.Transactions))
	}
	if block.Transactions[0].ID != pendingID {
		t.Fatalf("mined tx id %s does not match pending tx id %s", block.Transactions[0].ID, pendingID)
	}
	if bc.Ledger[from].Balance != 1_000_000_000+100-10 {
		t.Fatalf("expected sender balance 1000000090, got %d", bc.Ledger[from].Balance)
	}
	if bc.Ledger[to].Balance != 10 {
		t.Fatalf("expected receiver balance 10, got %d", bc.Ledger[to].Balance)
	}
	if len(bc.Chain) != 2 {
		t.Fatalf("expected chain height 2, got %d", len(bc.Chain))
	}
}
