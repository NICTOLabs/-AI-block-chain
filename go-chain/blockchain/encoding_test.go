package blockchain

import (
	"testing"
)

func TestTransactionBinaryRoundTrip(t *testing.T) {
	original := Transaction{
		ID:         "tx-1",
		From:       "addr-from",
		FromPubKey: "pubkey-from",
		To:         "addr-to",
		Amount:     100,
		Fee:        50,
		Nonce:      7,
		TxType:     Transfer,
		Payload:    "payload",
		Signature:  "sig",
		Timestamp:  1234567890,
		ChainID:    "tdr-testnet-1",
	}
	data, err := EncodeTransactionBinary(original)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	decoded, err := DecodeTransactionBinary(data)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded.ID != original.ID ||
		decoded.From != original.From ||
		decoded.To != original.To ||
		decoded.Amount != original.Amount ||
		decoded.Fee != original.Fee ||
		decoded.Nonce != original.Nonce ||
		decoded.TxType != original.TxType ||
		decoded.Payload != original.Payload ||
		decoded.Signature != original.Signature ||
		decoded.Timestamp != original.Timestamp ||
		decoded.ChainID != original.ChainID {
		t.Fatalf("transaction roundtrip mismatch: got %+v", decoded)
	}
}

func TestBlockBinaryRoundTrip(t *testing.T) {
	original := Block{
		Index:        1,
		Author:       "miner",
		MinerAddress: "miner",
		PreviousHash: "prev",
		Timestamp:    1234567890,
		BlockHash:    "blockhash",
		TxMerkleRoot: "merkle",
		Nonce:        42,
		Transactions: []Transaction{
			{ID: "tx-1", From: "a", To: "b", Amount: 10, Fee: 5, Nonce: 1, TxType: Transfer, ChainID: "tdr-testnet-1"},
		},
	}
	data, err := EncodeBlockBinary(original)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	decoded, err := DecodeBlockBinary(data)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded.Index != original.Index ||
		decoded.Author != original.Author ||
		decoded.PreviousHash != original.PreviousHash ||
		decoded.BlockHash != original.BlockHash ||
		decoded.TxMerkleRoot != original.TxMerkleRoot ||
		decoded.Nonce != original.Nonce ||
		len(decoded.Transactions) != len(original.Transactions) {
		t.Fatalf("block roundtrip mismatch: got %+v", decoded)
	}
	if len(decoded.Transactions) > 0 {
		if decoded.Transactions[0].ID != original.Transactions[0].ID {
			t.Fatalf("transaction inside block roundtrip mismatch")
		}
	}
}
