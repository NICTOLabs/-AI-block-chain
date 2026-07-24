package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	blockchain "ai_block_chain_go/blockchain"
)

func TestFaucetFundsAccount(t *testing.T) {
	cfg := ServerConfigFromEnv()
	chain := blockchain.NewBlockchain(blockchain.ProofOfStake, t.TempDir(), "tdr-testnet-1")
	address := "0000000000000000000000000000000000000000000000000000000000000001"
	chain.AddAccount(address, 50, false)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/faucet", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Address string `json:"address"`
			Amount  uint64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if payload.Address == "" {
			http.Error(w, "missing address", http.StatusBadRequest)
			return
		}
		amount := payload.Amount
		if amount == 0 {
			amount = cfg.FaucetAmount
		}
		chain.FundAccount(payload.Address, amount)
		chain.RLock()
		balance := chain.Ledger[payload.Address].Balance
		chain.RUnlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"address": payload.Address, "amount": amount, "balance": balance})
	})

	body, _ := json.Marshal(map[string]any{"address": address, "amount": 150})
	req := httptest.NewRequest(http.MethodPost, "/api/faucet", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Address string `json:"address"`
		Amount  uint64 `json:"amount"`
		Balance uint64 `json:"balance"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Address != address {
		t.Fatalf("unexpected address %s", resp.Address)
	}
	if resp.Amount != 150 {
		t.Fatalf("expected amount 150, got %d", resp.Amount)
	}
	if resp.Balance != 200 {
		t.Fatalf("expected balance 200, got %d", resp.Balance)
	}

	chain.RLock()
	defer chain.RUnlock()
	if chain.Ledger[address] == nil || chain.Ledger[address].Balance != 200 {
		t.Fatalf("chain ledger balance mismatch")
	}
}
