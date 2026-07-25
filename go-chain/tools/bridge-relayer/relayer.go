package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type lockRequest struct {
	TxHash    string `json:"tx_hash"`
	From      string `json:"from"`
	Amount    uint64 `json:"amount"`
	Recipient string `json:"recipient"`
}

type mintRequest struct {
	TxHash string `json:"tx_hash"`
	To     string `json:"to"`
	Amount uint64 `json:"amount"`
}

type statusResponse struct {
	ProcessedTxHashes int `json:"processed_tx_hashes"`
}

type healthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type keyFile struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

type relayer struct {
	privKey         ed25519.PrivateKey
	pubKey          ed25519.PublicKey
	pubHex          string
	processedHashes map[string]struct{}
	mu              sync.RWMutex
	tokenAdapter    *TokenAdapter
	nodeAPIBase     string
	nodeAPIKey      string
}

func (r *relayer) handleLock(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var lr lockRequest
	if err := json.Unmarshal(body, &lr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if lr.TxHash == "" || lr.From == "" || lr.Amount == 0 || lr.Recipient == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	r.mu.Lock()
	if _, seen := r.processedHashes[lr.TxHash]; seen {
		r.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "already_processed",
			"tx_hash": lr.TxHash,
		})
		return
	}
	r.processedHashes[lr.TxHash] = struct{}{}
	r.mu.Unlock()

	log.Printf("lock event received tx=%s from=%s amount=%d recipient=%s", lr.TxHash, lr.From, lr.Amount, lr.Recipient)

	payloadHash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d:%s", lr.TxHash, lr.From, lr.Amount, lr.Recipient)))
	signature := hex.EncodeToString(ed25519.Sign(r.privKey, payloadHash[:]))

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "lock_recorded",
		"tx_hash":   lr.TxHash,
		"from":      lr.From,
		"amount":    lr.Amount,
		"recipient": lr.Recipient,
		"signature": signature,
	})
}

func (r *relayer) handleMint(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var mr mintRequest
	if err := json.Unmarshal(body, &mr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if mr.TxHash == "" || mr.To == "" || mr.Amount == 0 {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	r.mu.Lock()
	if _, seen := r.processedHashes[mr.TxHash]; seen {
		r.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "already_processed",
			"tx_hash": mr.TxHash,
		})
		return
	}
	r.processedHashes[mr.TxHash] = struct{}{}
	r.mu.Unlock()

	payloadHash := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", mr.TxHash, mr.To, mr.Amount)))
	signature := hex.EncodeToString(ed25519.Sign(r.privKey, payloadHash[:]))

	mintResult, err := r.tokenAdapter.Mint(mr.To, mr.Amount)
	if err != nil {
		log.Printf("mint failed tx=%s to=%s amount=%d error=%v", mr.TxHash, mr.To, mr.Amount, err)
		http.Error(w, fmt.Sprintf("mint failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("mint executed tx=%s to=%s amount=%d", mr.TxHash, mr.To, mr.Amount)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "mint_executed",
		"tx_hash":   mr.TxHash,
		"to":        mr.To,
		"amount":    mr.Amount,
		"signature": signature,
		"mint":      mintResult,
	})
}

func (r *relayer) handleStatus(w http.ResponseWriter, req *http.Request) {
	r.mu.RLock()
	count := len(r.processedHashes)
	r.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(statusResponse{ProcessedTxHashes: count})
}

func (r *relayer) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
