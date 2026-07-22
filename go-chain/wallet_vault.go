package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"
)

type ManagedWallet struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
	Label     string `json:"label"`
	IsAgent   bool   `json:"is_agent"`
}

type WalletVault struct {
	mu      sync.RWMutex
	wallets map[string]*WalletRecord
}

type WalletRecord struct {
	Wallet
	Label      string    `json:"label"`
	IsAgent    bool      `json:"is_agent"`
	Derivation uint32    `json:"derivation_index"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsed   time.Time `json:"last_used"`
	Archived   bool      `json:"archived"`
}

func NewWalletVault() *WalletVault {
	return &WalletVault{wallets: make(map[string]*WalletRecord)}
}

func (v *WalletVault) CreateWallet(label string, isAgent bool) *WalletRecord {
	v.mu.Lock()
	defer v.mu.Unlock()
	wallet := NewWallet()
	record := &WalletRecord{
		Wallet:      *wallet,
		Label:       label,
		IsAgent:     isAgent,
		Derivation:  uint32(len(v.wallets)),
		CreatedAt:   time.Now().UTC(),
		LastUsed:    time.Now().UTC(),
		Archived:    false,
	}
	v.wallets[wallet.Address()] = record
	return record
}

func (v *WalletVault) GetWallet(address string) *WalletRecord {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.wallets[address]
}

func (v *WalletVault) ListWallets() []*WalletRecord {
	v.mu.RLock()
	defer v.mu.RUnlock()
	out := make([]*WalletRecord, 0, len(v.wallets))
	for _, w := range v.wallets {
		if !w.Archived {
			out = append(out, w)
		}
	}
	return out
}

func (v *WalletVault) DerivePath(index uint32) *WalletRecord {
	v.mu.Lock()
	defer v.mu.Unlock()
	seed := make([]byte, 32)
	_, _ = rand.Read(seed)
	pub, priv, err := ed25519.GenerateKey(bytes.NewReader(seed))
	if err != nil {
		return nil
	}
	address := sha256.Sum256(pub)
	record := &WalletRecord{
		Wallet:      Wallet{PublicKey: pub, PrivateKey: priv},
		Label:       fmt.Sprintf("hd-%d", index),
		IsAgent:     false,
		Derivation:  index,
		CreatedAt:   time.Now().UTC(),
		LastUsed:    time.Now().UTC(),
		Archived:    false,
	}
	v.wallets[hex.EncodeToString(address[:])] = record
	return record
}

func (v *WalletVault) Save(path string) error {
	v.mu.RLock()
	defer v.mu.RUnlock()
	data, err := json.MarshalIndent(v.wallets, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (v *WalletVault) Load(path string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &v.wallets)
}

func (v *WalletVault) Archive(address string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if rec, ok := v.wallets[address]; ok {
		rec.Archived = true
		return true
	}
	return false
}

func (v *WalletVault) Count() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.wallets)
}
