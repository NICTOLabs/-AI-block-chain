package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Snapshot struct {
	Header struct {
		ChainID      string    `json:"chain_id"`
		GenesisHash  string    `json:"genesis_hash"`
		Height       uint64    `json:"height"`
		Timestamp    time.Time `json:"timestamp"`
		StateRoot    string    `json:"state_root"`
		TxCount      uint64    `json:"tx_count"`
		ValidatorSet string    `json:"validator_set"`
	} `json:"header"`
	Accounts      map[string]*Account   `json:"accounts"`
	Registry      map[string]ModelEntry `json:"registry"`
	Escrows       map[string]Escrow     `json:"escrows"`
	Proposals     map[string]GovernanceProposal `json:"governance_proposals"`
	Agreements    map[string]ServiceAgreement    `json:"service_agreements"`
	UsageMeters   map[string]UsageMeter `json:"usage_meters"`
	UsedNonces    map[string]map[uint64]struct{} `json:"used_nonces"`
	NextNonce     map[string]uint64      `json:"next_nonce"`
	SeenTxIDs     map[string]struct{}     `json:"seen_tx_ids"`
	Validators    map[string]ValidatorInfo `json:"validators"`
	AuditTrail    []AuditEntry         `json:"audit_trail"`
	TokenSupply   uint64               `json:"token_supply"`
	Wallets       map[string]ManagedWallet `json:"managed_wallets"`
}

type StateStore struct {
	mu       sync.RWMutex
	dir      string
	latest   *Snapshot
	history  []string
	maxFiles int
}

const maxSnapshotSize = 50 << 20

func NewStateStore(dir string) (*StateStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	store := &StateStore{dir: dir, maxFiles: 64}
	if err := store.loadLatest(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *StateStore) stateRoot(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:8])
}

func (s *StateStore) loadLatest() error {
	files, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}
	var latest string
	var latestMod time.Time
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if len(name) < 12 || name[:8] != "snapshot" || !strings.HasSuffix(name, ".json.gz") {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		if latest == "" || info.ModTime().After(latestMod) {
			latest = name
			latestMod = info.ModTime()
		}
	}
	if latest == "" {
		_ = os.MkdirAll(filepath.Join(s.dir, "data"), 0o755)
		s.latest = &Snapshot{
			Accounts:   make(map[string]*Account),
			Registry:   make(map[string]ModelEntry),
			Escrows:    make(map[string]Escrow),
			Proposals:  make(map[string]GovernanceProposal),
			Agreements: make(map[string]ServiceAgreement),
			UsageMeters: make(map[string]UsageMeter),
			UsedNonces: make(map[string]map[uint64]struct{}),
			NextNonce:  make(map[string]uint64),
			SeenTxIDs:  make(map[string]struct{}),
			Validators: make(map[string]ValidatorInfo),
			Wallets:    make(map[string]ManagedWallet),
		}
		return nil
	}
	return s.readSnapshot(filepath.Join(s.dir, latest))
}

func (s *StateStore) readSnapshot(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	limited := io.LimitReader(gz, maxSnapshotSize+1)
	var snap Snapshot
	if err := json.NewDecoder(limited).Decode(&snap); err != nil {
		return err
	}
	s.latest = &snap
	return nil
}

func (s *StateStore) WriteSnapshot(snap *Snapshot) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	payload, err := json.Marshal(snap)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write(payload)
	_ = gz.Close()
	root := s.stateRoot(payload)
	name := fmt.Sprintf("snapshot_%s.json.gz", root[:12])
	path := filepath.Join(s.dir, name)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", err
	}
	s.history = append(s.history, path)
	s.latest = snap
	if len(s.history) > s.maxFiles {
		old := s.history[0]
		s.history = s.history[1:]
		_ = os.Remove(old)
	}
	return path, nil
}

func (s *StateStore) Snapshot() *Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clone := &Snapshot{
		Accounts:   make(map[string]*Account),
		Registry:   make(map[string]ModelEntry),
		Escrows:    make(map[string]Escrow),
		Proposals:  make(map[string]GovernanceProposal),
		Agreements: make(map[string]ServiceAgreement),
		UsageMeters: make(map[string]UsageMeter),
		UsedNonces: make(map[string]map[uint64]struct{}),
		NextNonce:  make(map[string]uint64),
		SeenTxIDs:  make(map[string]struct{}),
		Validators: make(map[string]ValidatorInfo),
		Wallets:    make(map[string]ManagedWallet),
	}
	for k, v := range s.latest.Accounts {
		a := *v
		clone.Accounts[k] = &a
	}
	for k, v := range s.latest.Registry {
		clone.Registry[k] = v
	}
	for k, v := range s.latest.Escrows {
		clone.Escrows[k] = v
	}
	for k, v := range s.latest.Proposals {
		clone.Proposals[k] = v
	}
	for k, v := range s.latest.Agreements {
		clone.Agreements[k] = v
	}
	for k, v := range s.latest.UsageMeters {
		clone.UsageMeters[k] = v
	}
	for k, v := range s.latest.UsedNonces {
		m := make(map[uint64]struct{}, len(v))
		for nk, nv := range v {
			m[nk] = nv
		}
		clone.UsedNonces[k] = m
	}
	for k, v := range s.latest.NextNonce {
		clone.NextNonce[k] = v
	}
	for k, v := range s.latest.SeenTxIDs {
		clone.SeenTxIDs[k] = v
	}
	for k, v := range s.latest.Validators {
		clone.Validators[k] = v
	}
	clone.AuditTrail = append([]AuditEntry{}, s.latest.AuditTrail...)
	clone.TokenSupply = s.latest.TokenSupply
	clone.Header = s.latest.Header
	return clone
}

func (s *StateStore) Close() error {
	return nil
}

