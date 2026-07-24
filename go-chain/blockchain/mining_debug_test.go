package blockchain

import (
	"testing"
)

func TestMineBlockIncludesSignedTransfer_Fixture(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	wallet := NewWallet()
	from := wallet.Address()
	bc.AddAccount(from, 1_000_000_000, false)
	to := "0000000000000000000000000000000000000000000000000000000000000001"
	bc.AddAccount(to, 0, false)

	tx := wallet.Sign(Transaction{
		From:    from,
		To:      to,
		Amount:  10,
		Fee:     BaseFee + FeeMultiplier + 10,
		Nonce:   1,
		TxType:  Transfer,
		ChainID: bc.ChainID,
	})
	bc.EnqueueTransaction(tx)

	bc.mu.Lock()
	pending := make([]Transaction, len(bc.Pending))
	copy(pending, bc.Pending)
	bc.mu.Unlock()

	for i, tx := range pending {
		ok, step := ValidateTransactionDebug(bc, tx)
		t.Logf("pending[%d] validate=%v step=%s estimate=%d fee=%d", i, ok, step, bc.estimateFee(tx, len(pending)), tx.Fee)
	}

	block, err := bc.MineBlockFor(from)
	if err != nil {
		t.Fatalf("mine failed: %v", err)
	}
	t.Logf("mined txs=%d pending=%d", len(block.Transactions), len(bc.Pending))
}

func ValidateTransactionDebug(bc *Blockchain, tx Transaction) (bool, string) {
	if tx.ChainID != bc.ChainID {
		return false, "chain_id"
	}
	if !VerifyTransaction(tx) {
		return false, "verify"
	}
	if bc.isReplay(tx) {
		return false, "replay"
	}
	sender, ok := bc.Ledger[tx.From]
	if !ok {
		return false, "sender_missing"
	}
	if tx.From != tx.To && tx.Amount == 0 {
		return false, "zero_amount"
	}
	switch tx.TxType {
	case Transfer:
		_, receiverExists := bc.Ledger[tx.To]
		if !receiverExists {
			return false, "receiver_missing"
		}
		if sender.Balance < tx.Amount {
			return false, "sender_balance"
		}
		return true, "ok"
	case RegisterModel:
		_, exists := bc.Registry[tx.To]
		if !exists {
			return true, "ok"
		}
		if sender.IsAgent {
			return true, "ok"
		}
		return false, "agent_model"
	default:
		return false, "default"
	}
}
