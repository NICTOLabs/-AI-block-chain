package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	blockchain "ai_block_chain_go/blockchain"
)

func TestTransferBinaryAcceptsEncodedTransaction(t *testing.T) {
	cfg := ServerConfigFromEnv()
	cfg.EnableAuth = false
	chain := blockchain.NewBlockchain(blockchain.ProofOfStake, t.TempDir(), "tdr-testnet-1")
	chain.AddAccount("addr-from", 1000, false)
	chain.AddAccount("addr-to", 0, false)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/transfer-binary", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !newRateLimiter(cfg.RateLimit, cfg.RateWindow).allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, _ := io.ReadAll(r.Body)
		tx, err := blockchain.DecodeTransactionBinary(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		chain.EnqueueTransaction(tx)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tx)
	})

	tx := blockchain.Transaction{From: "addr-from", To: "addr-to", Amount: 10, Fee: 50, Nonce: 1, TxType: blockchain.Transfer, ChainID: "tdr-testnet-1"}
	data, err := blockchain.EncodeTransactionBinary(tx)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/transfer-binary", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/octet-stream")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(chain.Pending) != 1 {
		t.Fatalf("expected 1 pending tx, got %d", len(chain.Pending))
	}
}
