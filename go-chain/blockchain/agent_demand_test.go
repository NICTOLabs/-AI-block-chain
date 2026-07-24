package blockchain

import (
	"testing"
)

func TestAgentTxCountIncrementsOnAgentTransactions(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	owner := NewWallet().Address()
	bc.AddAccount(owner, 1000, false)
	if bc.AgentTxCount != 0 {
		t.Fatalf("expected initial agent tx count 0, got %d", bc.AgentTxCount)
	}
	block := Block{Index: 1, PreviousHash: bc.Chain[len(bc.Chain)-1].BlockHash, Transactions: []Transaction{
		{TxType: RegisterModel, To: owner, From: owner, Payload: "v1", Amount: 10, Fee: BaseFee, Nonce: 1, ChainID: bc.ChainID},
	}}
	bc.applyBlock(block)
	if bc.AgentTxCount != 1 {
		t.Fatalf("expected agent tx count 1 after register model, got %d", bc.AgentTxCount)
	}
	block2 := Block{Index: 2, PreviousHash: block.BlockHash, Transactions: []Transaction{
		{TxType: UpdateModel, To: owner, From: owner, Payload: "v2", Amount: 10, Fee: BaseFee, Nonce: 2, ChainID: bc.ChainID},
	}}
	bc.applyBlock(block2)
	if bc.AgentTxCount != 2 {
		t.Fatalf("expected agent tx count 2 after update model, got %d", bc.AgentTxCount)
	}
}

func TestEstimateFeeRisesWithAgentDemand(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	bc.AgentTxCount = 0
	fee0 := bc.estimateFee(Transaction{TxType: RegisterModel}, 0)
	bc.AgentTxCount = 1000
	feeHigh := bc.estimateFee(Transaction{TxType: RegisterModel}, 0)
	if feeHigh <= fee0 {
		t.Fatalf("expected higher fee with agent demand, got %d vs %d", feeHigh, fee0)
	}
}
