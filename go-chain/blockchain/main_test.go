package blockchain

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"
)

func TestFuzzTransactionSigningAndSerialization(t *testing.T) {
	for i := 0; i < 200; i++ {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}

		tx := Transaction{
			ID:        string(bytes.Repeat([]byte{byte(i)}, 32)),
			From:      hex.EncodeToString(pub),
			To:        hex.EncodeToString(pub),
			Amount:    uint64(i*7) + 1,
			Fee:       uint64(i*3) + 1,
			Nonce:     uint64(i),
			TxType:    Transfer,
			Payload:   string(bytes.Repeat([]byte{byte(i % 251)}, 64)),
			Timestamp: time.Now().UnixNano(),
		}

		wallet := Wallet{PublicKey: pub, PrivateKey: priv}
		signed := wallet.Sign(tx)
		if signed.Signature == "" {
			t.Fatal("expected non-empty signature")
		}

		payload, err := json.Marshal(signed)
		if err != nil {
			t.Fatalf("marshal signed tx: %v", err)
		}
		var decoded Transaction
		if err := json.Unmarshal(payload, &decoded); err != nil {
			t.Fatalf("unmarshal signed tx: %v", err)
		}
		if decoded.Signature != signed.Signature {
			t.Fatal("round-trip signature mismatch")
		}
	}
}

func TestFuzzWalletAddressDerivationAgainstEdgeCases(t *testing.T) {
	for i := 0; i < 200; i++ {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		wallet := Wallet{PublicKey: pub}
		addr := wallet.Address()
		if len(addr) == 0 {
			t.Fatal("expected non-empty address")
		}
		if len(addr) != 64 {
			t.Fatalf("expected 64-hex address, got %d chars", len(addr))
		}
	}
}

func TestMerkleRootValidation(t *testing.T) {
	txs := []Transaction{
		{ID: "tx-1", From: "a", To: "b", Amount: 10, Fee: 1, Nonce: 1, TxType: Transfer, Timestamp: 1},
		{ID: "tx-2", From: "b", To: "c", Amount: 20, Fee: 1, Nonce: 1, TxType: Transfer, Timestamp: 2},
	}
	root := CalculateMerkleRoot(txs)
	if root == "" {
		t.Fatal("expected non-empty merkle root")
	}
	badTxs := []Transaction{
		{ID: "tx-1", From: "a", To: "b", Amount: 10, Fee: 1, Nonce: 1, TxType: Transfer, Timestamp: 1},
		{ID: "tx-3", From: "b", To: "c", Amount: 20, Fee: 1, Nonce: 1, TxType: Transfer, Timestamp: 2},
	}
	if root == CalculateMerkleRoot(badTxs) {
		t.Fatal("different transactions should produce a different merkle root")
	}
	if CalculateMerkleRoot(nil) != "" {
		t.Fatal("empty transactions should yield empty merkle root")
	}
}

func TestCanonicalSigningBytesStable(t *testing.T) {
	tx := Transaction{
		ID:        "tx-1",
		From:      "addr1",
		FromPubKey: "pub1",
		To:        "addr2",
		Amount:    100,
		Fee:       5,
		Nonce:     1,
		TxType:    Transfer,
		Payload:   "p",
		Timestamp: 1234567890,
		ChainID:   "tdr-mainnet-1",
	}
	first := CanonicalSigningBytes(tx)
	for i := 0; i < 100; i++ {
		if got := CanonicalSigningBytes(tx); !bytes.Equal(got, first) {
			t.Fatal("canonical signing bytes are not stable")
		}
	}
}
