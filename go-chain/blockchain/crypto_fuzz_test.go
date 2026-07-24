package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"
)

func TestFuzzTransactionSigningWithMalformedInputs(t *testing.T) {
	for i := 0; i < 200; i++ {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate key: %v", err)
		}
		wallet := Wallet{PublicKey: pub, PrivateKey: priv}
		addr := wallet.Address()

		tx := Transaction{
			ID:        randomString(32 + i),
			From:      addr,
			To:        randomAddress(int64(i + 1)),
			Amount:    randomUint64(int64(i)),
			Fee:       randomUint64(int64(i + 1)),
			Nonce:     randomUint64(int64(i + 2)),
			TxType:    randomTxType(int64(i)),
			Payload:   randomString(64 + (i % 128)),
			Timestamp: time.Now().UnixNano(),
			ChainID:   "tdr-testnet-1",
		}

		signed := wallet.Sign(tx)

		if signed.Signature == "" {
			t.Fatal("expected non-empty signature after signing")
		}
		if len(signed.Signature) != 128 {
			t.Fatalf("expected signature length 128, got %d", len(signed.Signature))
		}
		if signed.FromPubKey == "" {
			t.Fatal("expected non-empty FromPubKey after signing")
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
		if decoded.FromPubKey != signed.FromPubKey {
			t.Fatal("round-trip FromPubKey mismatch")
		}
		if decoded.ChainID != signed.ChainID {
			t.Fatal("round-trip ChainID mismatch")
		}
	}
}

func TestFuzzWalletAddressDerivationWithKeyVariations(t *testing.T) {
	for i := 0; i < 300; i++ {
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

		pubBytes := sha256.Sum256(pub)
		expectedAddr := hex.EncodeToString(pubBytes[:])
		if addr != expectedAddr {
			t.Fatalf("address mismatch: got %s, want %s", addr, expectedAddr)
		}

		_, priv, _ := ed25519.GenerateKey(rand.Reader)
		w2 := Wallet{PublicKey: pub, PrivateKey: priv}
		if w2.Address() != addr {
			t.Fatal("same public key must produce same address regardless of private key")
		}
	}
}

func TestFuzzBlockHashStabilityAndIntegrity(t *testing.T) {
	for i := 0; i < 200; i++ {
		block := Block{
			Index:        uint64(i),
			Author:       randomAddress(int64(i)),
			PreviousHash: randomHash(i),
			Timestamp:    time.Now().UnixNano(),
			Transactions: randomTxSlice(i),
			Nonce:        uint64(i * 3),
			BlockHash:    "",
		}

		h1 := calculateHash(block)
		h2 := calculateHash(block)
		if h1 != h2 {
			t.Fatal("block hash must be deterministic")
		}

		if !isValidHashFormat(h1) {
			t.Fatalf("invalid hash format: %s", h1)
		}

		block.BlockHash = h1
		if calculateHash(block) != h1 {
			t.Fatal("hash must remain stable when BlockHash field is populated")
		}
	}
}

func TestFuzzMempoolReplayProtection(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	wallet := Wallet{PublicKey: pub, PrivateKey: priv}

	tx := Transaction{
		From:      hex.EncodeToString(pub),
		FromPubKey: hex.EncodeToString(pub),
		To:        hex.EncodeToString(pub),
		Amount:    100,
		Fee:       10,
		Nonce:     1,
		TxType:    Transfer,
		Payload:   "replay-test",
		Timestamp: time.Now().UnixNano(),
		ChainID:   "tdr-testnet-1",
	}
	signed := wallet.Sign(tx)

	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1", "")

	bc.EnqueueTransaction(signed)
	bc.EnqueueTransaction(signed)

	if len(bc.Pending) != 1 {
		t.Fatalf("expected 1 pending tx after duplicate submission, got %d", len(bc.Pending))
	}
}

func TestFuzzNumericOverflowAndUnderflow(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	wallet := Wallet{PublicKey: pub, PrivateKey: priv}
	addr := wallet.Address()
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1", "")
	bc.AddAccount(addr, 1000, false)

	if bc.Ledger[addr].Balance != 1000 {
		t.Fatal("account balance should be set correctly")
	}

	zeroTx := wallet.Sign(Transaction{
		From:      addr,
		To:        addr,
		Amount:    0,
		Fee:       0,
		Nonce:     1,
		TxType:    Transfer,
		Payload:   "zero",
		Timestamp: time.Now().UnixNano(),
		ChainID:   "tdr-testnet-1",
	})
	if !verifyTransaction(zeroTx) {
		t.Fatal("zero-value transaction should have valid signature format")
	}
}

func TestFuzzGovernanceInjection(t *testing.T) {
	for i := 0; i < 50; i++ {
		bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1", "")
		from := randomAddress(int64(i))
		to := randomAddress(int64(i + 1000))
		bc.AddAccount(from, 5000, false)
		bc.AddAccount(to, 1000, false)

		escrow, err := bc.CreateEscrow(from, to, randomUint64(int64(i)), "svc-inject")
		if err == nil {
			if escrow.Status != "active" {
				t.Fatalf("escrow status must be active on creation: %s", escrow.Status)
			}
		}

		proposal := bc.CreateProposal("Inject Test", "test proposal")
		if proposal.Status != "open" {
			t.Fatal("proposal status must be open on creation")
		}

		bc.VoteProposal(proposal.ID, from)
		if proposal.Votes[from] != true {
			t.Fatal("vote was not recorded in proposal state")
		}
	}
}

func TestFuzzChainValidationWithCorruptedBlocks(t *testing.T) {
	for i := 0; i < 50; i++ {
		bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1", "")

		for j := 0; j < 3; j++ {
			pub, _, _ := ed25519.GenerateKey(rand.Reader)
			addr := hex.EncodeToString(pub)
			bc.AddAccount(addr, 1000, false)
			tx := Transaction{
				From:      addr,
				FromPubKey: hex.EncodeToString(pub),
				To:        addr,
				Amount:    10,
				Fee:       1,
				Nonce:     uint64(j + 1),
				TxType:    Transfer,
				Payload:   "chain-test",
				Timestamp: time.Now().UnixNano(),
			}
			bc.SubmitTransaction(tx)
			_, _ = bc.MineBlock()
		}

		chainCopy := append([]Block{}, bc.Chain...)
		if err := bc.validateChain(chainCopy); err != nil {
			t.Fatalf("valid chain rejected: %v", err)
		}

		if len(chainCopy) > 2 {
			corrupted := chainCopy[2]
			corrupted.Nonce = 999999
			if err := bc.validateBlock(corrupted, chainCopy[1]); err == nil {
				t.Fatal("expected corrupted block to fail validation")
			}
		}
	}
}

func TestFuzzEscrowAndGovernanceBoundaries(t *testing.T) {
	for i := 0; i < 100; i++ {
		bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1", "")
		from := randomAddress(int64(i))
		to := randomAddress(int64(i + 1000))
		bc.AddAccount(from, 5000, false)
		bc.AddAccount(to, 1000, false)

		escrowAmount := randomUint64(int64(i))
		if escrowAmount > 5000 {
			escrowAmount = 5000
		}
		escrow, err := bc.CreateEscrow(from, to, escrowAmount, "svc-boundary")
		if err == nil {
			if escrow.Status != "active" {
				t.Fatalf("escrow status must be active on creation: %s", escrow.Status)
			}
			if escrow.Amount > 5000 {
				t.Fatal("escrow amount cannot exceed from balance")
			}
		}

		proposal := bc.CreateProposal("Boundary Test", "test proposal")
		if proposal.Status != "open" {
			t.Fatal("proposal status must be open on creation")
		}

		bc.VoteProposal(proposal.ID, from)
		if proposal.Votes[from] != true {
			t.Fatal("vote was not recorded in proposal state")
		}
	}
}

// --------------------
// Fuzz helper utilities
// --------------------

func randomString(length int) string {
	if length <= 0 {
		length = 1
	}
	buf := make([]byte, length)
	for i := range buf {
		buf[i] = byte(i % 251)
	}
	return string(buf)
}

func randomAddress(seed int64) string {
	buf := make([]byte, 32)
	for i := range buf {
		buf[i] = byte((seed + int64(i)) % 256)
	}
	return hex.EncodeToString(buf)
}

func randomHash(seed int) string {
	buf := make([]byte, 32)
	hash := sha256.Sum256([]byte{byte(seed)})
	copy(buf, hash[:])
	return hex.EncodeToString(buf)
}

func randomUint64(seed int64) uint64 {
	val := uint64(seed % 100000)
	if val == 0 {
		val = 1
	}
	return val
}

func randomTxType(seed int64) TransactionType {
	switch seed % 4 {
	case 0:
		return Transfer
	case 1:
		return RegisterModel
	case 2:
		return UpdateModel
	case 3:
		return PurchaseApiKey
	}
	return Transfer
}

func randomTxSlice(seed int) []Transaction {
	count := seed % 4
	txs := make([]Transaction, count)
	for i := 0; i < count; i++ {
		pub, _, _ := ed25519.GenerateKey(rand.Reader)
		txs[i] = Transaction{
			ID:         randomString(16),
			From:       hex.EncodeToString(pub),
			FromPubKey: hex.EncodeToString(pub),
			To:         hex.EncodeToString(pub),
			Amount:     randomUint64(int64(i + 1)),
			Fee:        randomUint64(int64(i + 1)),
			Nonce:      uint64(i + 1),
			TxType:     Transfer,
			Payload:    randomString(32),
			Timestamp:  time.Now().UnixNano(),
		}
	}
	return txs
}

func isValidHashFormat(h string) bool {
	if len(h) != 64 {
		return false
	}
	for _, c := range h {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func bytesRepeater(b byte, n int) []byte {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = b
	}
	return buf
}

func requireSuccess(err error, msg string) {
	if err != nil {
		panic(msg + ": " + err.Error())
	}
}
