package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	blockchain "ai_block_chain_go/blockchain"
	p2p "ai_block_chain_go/p2p"
)

type ServerConfig struct {
	APIKey      string
	EnableAuth  bool
	RateLimit   int
	RateWindow  time.Duration
	EnableTLS   bool
	TLSCertFile string
	TLSKeyFile  string
	MaxBodyBytes int64
	MetricsPath string
	APIPort     int
	P2PPort     int
	DataDir     string
	Consensus   string
	StrictP2P   bool
	FaucetAmount uint64
}

type rateLimiter struct {
	mu     sync.Mutex
	counts map[string][]time.Time
	limit  int
	window time.Duration
}

type circuitBreaker struct {
	mu            sync.Mutex
	failures     int64
	threshold    int64
	window       time.Duration
	lastFailure  time.Time
	state        string
}

type serverMetrics struct {
	mu            sync.Mutex
	requestCount  int64
	errorCount    int64
	lastRequestAt time.Time
	blocksMined   int64
	peersSeen     int64
	txAccepted    int64
	txRejected    int64
}

func ServerConfigFromEnv() ServerConfig {
	cfg := ServerConfig{
		APIKey:      getEnvOrDefault("TENDER_API_KEY", ""),
		EnableAuth:  getEnvBoolOrDefault("TENDER_ENABLE_AUTH", true),
		RateLimit:   getEnvIntOrDefault("TENDER_RATE_LIMIT", 60),
		RateWindow:  time.Duration(getEnvIntOrDefault("TENDER_RATE_WINDOW_SECONDS", 60)) * time.Second,
		MetricsPath: getEnvOrDefault("TENDER_METRICS_PATH", "/metrics"),
		APIPort:     getEnvIntOrDefault("TENDER_API_PORT", 8080),
		P2PPort:     getEnvIntOrDefault("TENDER_P2P_PORT", 3030),
		DataDir:     getEnvOrDefault("TENDER_DATA_DIR", "./data"),
		Consensus:   strings.ToLower(getEnvOrDefault("TENDER_CONSENSUS", "pos")),
		StrictP2P:   getEnvBoolOrDefault("TENDER_STRICT_P2P", true),
		MaxBodyBytes: getEnvInt64OrDefault("TENDER_MAX_BODY_BYTES", 1<<20),
		FaucetAmount: uint64(getEnvIntOrDefault("TENDER_FAUCET_AMOUNT", 1000)),
	}
	return cfg
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvIntOrDefault(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvInt64OrDefault(key string, fallback int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvBoolOrDefault(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), "request_id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func maxBodyMiddleware(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limit > 0 && r.ContentLength > 0 && r.ContentLength > limit {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			if limit > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func idempotencyMiddleware(next http.Handler) http.Handler {
	seen := make(map[string]struct{})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		if key == "" {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := seen[key]; ok {
			http.Error(w, "duplicate idempotency key", http.StatusConflict)
			return
		}
		seen[key] = struct{}{}
		next.ServeHTTP(w, r)
	})
}

func requireAuth(r *http.Request, cfg ServerConfig) error {
	if !cfg.EnableAuth {
		return nil
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("missing api key")
	}
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return fmt.Errorf("missing authorization")
	}
	prefix := "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return fmt.Errorf("invalid authorization scheme")
	}
	provided := strings.TrimPrefix(authorization, prefix)
	if provided != cfg.APIKey {
		return fmt.Errorf("invalid api key")
	}
	return nil
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit <= 0 {
		limit = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &rateLimiter{counts: make(map[string][]time.Time), limit: limit, window: window}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	entries := rl.counts[key]
	filtered := entries[:0]
	for _, ts := range entries {
		if now.Sub(ts) <= rl.window {
			filtered = append(filtered, ts)
		}
	}
	rl.counts[key] = filtered
	if len(filtered) >= rl.limit {
		return false
	}
	rl.counts[key] = append(filtered, now)
	return true
}

func newCircuitBreaker(threshold int64, window time.Duration) *circuitBreaker {
	return &circuitBreaker{threshold: threshold, window: window, state: "closed"}
}

func (cb *circuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == "open" {
		if time.Since(cb.lastFailure) > cb.window {
			cb.state = "half-open"
			cb.failures = 0
			return true
		}
		return false
	}
	return true
}

func (cb *circuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = "closed"
}

func (cb *circuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

func (m *serverMetrics) recordRequest(ok bool) {
	if m == nil {
		return
	}
	atomic.AddInt64(&m.requestCount, 1)
	if !ok {
		atomic.AddInt64(&m.errorCount, 1)
	}
	m.mu.Lock()
	m.lastRequestAt = time.Now()
	m.mu.Unlock()
}

func StartAPI(chain *blockchain.Blockchain, port int, p2pNode *p2p.P2PNode, cfg ServerConfig) {
	limiter := newRateLimiter(cfg.RateLimit, cfg.RateWindow)
	cb := newCircuitBreaker(5, 10*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	})
	mux.Handle("/", http.FileServer(http.Dir("./web")))
	mux.HandleFunc("/api/chain", func(w http.ResponseWriter, r *http.Request) {
		if !cb.Allow() {
			http.Error(w, "circuit breaker open", http.StatusServiceUnavailable)
			return
		}
		if err := requireAuth(r, cfg); err != nil {
			cb.RecordFailure()
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			cb.RecordFailure()
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		cb.RecordSuccess()
		chain.RecordRequest(true)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(chain.Snapshot())
	})
	mux.HandleFunc("/api/finality", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodPost {
			var payload struct {
				Index      uint64   `json:"index"`
				Voter      string   `json:"voter"`
				Voters     []string `json:"voters"`
				Signatures []struct {
					VoterHex string `json:"voter"`
					Vote     string `json:"vote"`
					Signature []byte `json:"signature"`
				} `json:"signatures"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if payload.Voter != "" {
				_ = chain.CollectFinalityVote(payload.Index, payload.Voter)
			}
			if len(payload.Voters) > 0 {
				for _, voter := range payload.Voters {
					_ = chain.CollectFinalityVote(payload.Index, voter)
				}
			}
			if len(payload.Signatures) > 0 {
				for _, s := range payload.Signatures {
					vote := blockchain.FinalityVoteSignature{BlockHash: chain.TailHash(), Voter: s.VoterHex, Vote: s.Vote, Signature: s.Signature}
					if chain.VerifyFinalitySignature(vote) {
						chain.FinalitySignatures[payload.Index] = append(chain.FinalitySignatures[payload.Index], vote)
					}
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "voted", "signatures_received": len(payload.Signatures)})
			return
		}
		chain.RLock()
		defer chain.RUnlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"last_finalized": chain.LastFinalized,
			"finalized":      chain.FinalizedBlocks,
			"signatures":     chain.FinalitySignatures,
		})
	})
	mux.HandleFunc("/api/finality/vote", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			BlockIndex uint64 `json:"block_index"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		voter := r.RemoteAddr
		if v, ok := r.Context().Value("validator_address").(string); ok && v != "" {
			voter = v
		}
		_ = chain.CollectFinalityVote(payload.BlockIndex, voter)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "voted", "voter": voter})
	})
	mux.HandleFunc("/api/audit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.RLock()
		defer chain.RUnlock()
		_ = json.NewEncoder(w).Encode(chain.AuditTrail)
	})
	mux.HandleFunc("/api/monitoring", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		chain.RLock()
		defer chain.RUnlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"height":               len(chain.Chain),
			"pending_transactions": len(chain.Pending),
			"token_supply":         chain.TokenSupply,
			"audit_entries":        len(chain.AuditTrail),
			"peer_count":           len(p2pNode.Peers()),
			"trusted_peer_count":   len(p2pNode.TrustedPeers()),
			"strict_p2p":           p2pNode.StrictMode(),
			"consensus":            blockchain.ConsensusName(chain.Consensus),
		})
	})
	mux.HandleFunc("/api/mempool", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.RLock()
		defer chain.RUnlock()
		_ = json.NewEncoder(w).Encode(chain.Pending)
	})
	mux.HandleFunc("/api/transactions", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var tx blockchain.Transaction
		if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		chain.EnqueueTransaction(tx)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tx)
	})
	mux.HandleFunc("/api/mine", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			MinerAddress string `json:"miner_address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		block, err := chain.MineBlockFor(payload.MinerAddress)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(block)
	})
	mux.HandleFunc("/api/miner/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var block blockchain.Block
		if err := json.NewDecoder(r.Body).Decode(&block); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := chain.SubmitMinedBlock(block); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted", "hash": block.BlockHash})
	})
	mux.HandleFunc("/api/validators", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.RLock()
		defer chain.RUnlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"consensus":      blockchain.ConsensusName(chain.Consensus),
			"authorities":    chain.Authorities,
			"next_validator": chain.SelectValidator(),
			"validators":     chain.Validators,
		})
	})
	mux.HandleFunc("/api/validators/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Address  string `json:"address"`
			Stake    uint64 `json:"stake"`
			PublicKey string `json:"public_key"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := chain.RegisterValidator(payload.Address, payload.Stake, payload.PublicKey); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "registered"})
	})
	mux.HandleFunc("/api/bootstrap", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"node":      p2pNode.Addr(),
			"peers":     p2pNode.Peers(),
			"trusted":   p2pNode.TrustedPeers(),
			"validator": chain.SelectValidator(),
		})
	})
	mux.HandleFunc("/api/registry", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.RLock()
		defer chain.RUnlock()
		_ = json.NewEncoder(w).Encode(chain.Registry)
	})
	mux.HandleFunc("/api/registry/register", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			ID           string `json:"id"`
			Owner        string `json:"owner"`
			Version      string `json:"version"`
			Metadata     string `json:"metadata"`
			PricePerCall uint64 `json:"price_per_call"`
			Active       bool   `json:"active"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		chain.RegisterModel(payload.Owner, payload.ID, payload.Version, payload.Metadata, payload.PricePerCall, payload.Active)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "registered", "id": payload.ID})
	})
	mux.HandleFunc("/api/registry/update", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			ID          string `json:"id"`
			Owner       string `json:"owner"`
			Version     string `json:"version"`
			Metadata    string `json:"metadata"`
			PricePerCall uint64 `json:"price_per_call"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		chain.UpdateModel(payload.Owner, payload.ID, payload.Version, payload.Metadata, payload.PricePerCall)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "updated", "id": payload.ID})
	})
	mux.HandleFunc("/api/registry/purchase", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Buyer  string `json:"buyer"`
			ModelID string `json:"model_id"`
			Amount  uint64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		chain.PurchaseApiKey(payload.Buyer, payload.ModelID, payload.Amount)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "purchased", "model_id": payload.ModelID})
	})
	mux.HandleFunc("/api/accounts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.RLock()
		defer chain.RUnlock()
		_ = json.NewEncoder(w).Encode(chain.Ledger)
	})
	mux.HandleFunc("/api/stake", func(w http.ResponseWriter, r *http.Request) {
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
		chain.Stake(payload.Address, payload.Amount)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"address": payload.Address, "amount": payload.Amount})
	})
	mux.HandleFunc("/api/wallet", func(w http.ResponseWriter, r *http.Request) {
		wallet := blockchain.NewWallet()
		address := wallet.Address()
		chain.AddAccount(address, 1000, false)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"address": address, "public_key": hex.EncodeToString(wallet.PublicKey)})
	})
	mux.HandleFunc("/api/universal/register", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Name     string `json:"name"`
			Address  string `json:"address"`
			Label    string `json:"label"`
			Currency string `json:"currency"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		uw, err := chain.RegisterUniversalWallet(payload.Name, payload.Address, payload.Label, payload.Currency)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(uw)
	})
	mux.HandleFunc("/api/universal/lookup", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSpace(r.URL.Query().Get("name"))
		if name == "" {
			http.Error(w, "missing name", http.StatusBadRequest)
			return
		}
		uw, err := chain.LookupWallet(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(uw)
	})
	mux.HandleFunc("/api/universal/transfer", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			From     string `json:"from"`
			ToName   string `json:"to_name"`
			Amount   uint64 `json:"amount"`
			Currency string `json:"currency"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		uw, err := chain.UniversalTransfer(payload.From, payload.ToName, payload.Currency, payload.Amount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "transferred", "recipient": uw})
	})
	mux.HandleFunc("/api/faucet", func(w http.ResponseWriter, r *http.Request) {
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
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
		_ = json.NewEncoder(w).Encode(map[string]any{"address": payload.Address, "amount": amount, "amount_formatted": blockchain.FormatAmount(amount), "balance": balance, "balance_formatted": blockchain.FormatAmount(balance)})
	})
	mux.HandleFunc("/api/admin/mint", func(w http.ResponseWriter, r *http.Request) {
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
		if payload.Address == "" || payload.Amount == 0 {
			http.Error(w, "missing address or amount", http.StatusBadRequest)
			return
		}
		chain.Lock()
		if chain.TokenSupply+payload.Amount > blockchain.MaxSupply {
			chain.Unlock()
			http.Error(w, "mint would exceed max supply", http.StatusBadRequest)
			return
		}
		chain.AddAccount(payload.Address, payload.Amount, false)
		chain.TokenSupply += payload.Amount
		chain.Unlock()
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"address": payload.Address, "amount": payload.Amount, "status": "minted"})
	})
	mux.HandleFunc("/api/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"currency_name":      blockchain.CurrencyName,
			"currency_symbol":    blockchain.CurrencySymbol(),
			"currency_subunit":   blockchain.CurrencySubunit,
			"subunit_factor":     blockchain.HogohogoPerTender,
			"base_fee":           blockchain.BaseFee,
			"base_fee_formatted": blockchain.FormatAmount(blockchain.BaseFee),
			"burn_rate_percent":  blockchain.BurnRatePercent,
			"reward_rate_percent": blockchain.RewardRatePercent,
		})
	})
	mux.HandleFunc("/api/managed-wallets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Label   string `json:"label"`
			IsAgent bool   `json:"is_agent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		wallet, err := chain.CreateManagedWallet(payload.Label, payload.IsAgent)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wallet)
	})
	mux.HandleFunc("/api/transfer", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload blockchain.Transaction
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		chain.EnqueueTransaction(payload)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})
	mux.HandleFunc("/api/transfer-binary", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
	mux.HandleFunc("/api/escrow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			From      string `json:"from"`
			To        string `json:"to"`
			Amount    uint64 `json:"amount"`
			ServiceID string `json:"service_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		escrow, err := chain.CreateEscrow(payload.From, payload.To, payload.Amount, payload.ServiceID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(escrow)
	})
	mux.HandleFunc("/api/reversals/request", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			From    string `json:"from"`
			To      string `json:"to"`
			Amount  uint64 `json:"amount"`
			OriginalTxID string `json:"original_tx_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		tx := blockchain.Transaction{
			From:    payload.From,
			To:      payload.To,
			Amount:  payload.Amount,
			TxType:  blockchain.RequestReversal,
			Payload: payload.OriginalTxID,
		}
		chain.EnqueueTransaction(tx)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "reversal_requested", "original_tx": payload.OriginalTxID})
	})
	mux.HandleFunc("/api/reversals/confirm", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			From        string `json:"from"`
			OriginalTxID string `json:"original_tx_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		tx := blockchain.Transaction{
			From:    payload.From,
			To:      payload.From,
			Amount:  0,
			TxType:  blockchain.ConfirmReversal,
			Payload: payload.OriginalTxID,
		}
		chain.EnqueueTransaction(tx)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "reversal_confirmed", "original_tx": payload.OriginalTxID})
	})
	mux.HandleFunc("/api/reversals/commit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			From        string `json:"from"`
			OriginalTxID string `json:"original_tx_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		tx := blockchain.Transaction{
			From:    payload.From,
			To:      payload.From,
			Amount:  0,
			TxType:  blockchain.CommitReversal,
			Payload: payload.OriginalTxID,
		}
		chain.EnqueueTransaction(tx)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "reversal_committed", "original_tx": payload.OriginalTxID})
	})
	mux.HandleFunc("/api/proposals", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		proposal := chain.CreateProposal(payload.Title, payload.Description)
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(proposal)
	})
	mux.HandleFunc("/api/agreements", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Provider     string `json:"provider"`
			Consumer     string `json:"consumer"`
			ModelID      string `json:"model_id"`
			PricePerCall uint64 `json:"price_per_call"`
			MaxCalls     uint64 `json:"max_calls"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		agreement, err := chain.CreateServiceAgreement(payload.Provider, payload.Consumer, payload.ModelID, payload.PricePerCall, payload.MaxCalls)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(agreement)
	})
	mux.HandleFunc("/api/usage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			AgreementID string `json:"agreement_id"`
			UsageCount  uint64 `json:"usage_count"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		meter, err := chain.RecordUsage(payload.AgreementID, payload.UsageCount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.SaveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meter)
	})
	mux.HandleFunc("/api/tokenomics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.RLock()
		defer chain.RUnlock()
		agentBaseDemand := uint64(1000000)
		agentGrowth := 0.05
		agentDemand := agentBaseDemand + uint64(float64(agentBaseDemand)*agentGrowth*float64(chain.AgentTxCount))
		demandFactor := uint64(0)
		if chain.AgentTxCount > 0 && chain.TokenSupply > 0 {
			demandFactor = uint64(float64(chain.AgentTxCount) / float64(chain.TokenSupply) * 1_000_000_000)
			if demandFactor > 200 {
				demandFactor = 200
			}
		}
		deflPressure := 0.0
		if chain.TokenSupply > 0 {
			if chain.TokenSupply > 5000000000 {
				deflPressure = float64(blockchain.BurnRatePercent) * 1.5
			} else {
				deflPressure = float64(blockchain.BurnRatePercent)
			}
		}
		humanScore := 0.0
		if deflPressure > 0 {
			humanScore = float64(agentDemand) / deflPressure
		} else if agentDemand > 0 {
			humanScore = float64(agentDemand)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"currency":              blockchain.CurrencySymbol(),
			"token_supply":          chain.TokenSupply,
			"burn_rate_percent":     blockchain.BurnRatePercent,
			"reward_rate_percent":   blockchain.RewardRatePercent,
			"base_fee":              blockchain.BaseFee,
			"escrows":               chain.Escrows,
			"proposals":             chain.Proposals,
			"agent_tx_count":        chain.AgentTxCount,
			"agent_demand_pressure": agentDemand,
			"deflationary_pressure": deflPressure,
			"human_value_score":     humanScore,
		})
	})
	mux.HandleFunc("/api/p2p/libp2p/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var msg struct {
			Type       string `json:"type"`
			Block      map[string]any `json:"block,omitempty"`
			Tx         map[string]any `json:"tx,omitempty"`
			FromNodeID string `json:"from_node_id,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		chain.Lock()
		switch msg.Type {
		case "block":
			if msg.Block != nil {
				if idxFloat, ok := msg.Block["index"].(float64); ok {
					idx := uint64(idxFloat)
					if int(idx) < len(chain.Chain) {
						b := chain.Chain[idx]
						if b.BlockHash == "" {
						b.BlockHash, _ = msg.Block["block_hash"].(string)
						b.Author, _ = msg.Block["author"].(string)
						if ts, ok := msg.Block["timestamp"].(float64); ok {
							b.Timestamp = int64(ts)
						}
						b.PreviousHash, _ = msg.Block["previous_hash"].(string)
							if txs, ok := msg.Block["transactions"].([]any); ok {
								for _, txAny := range txs {
									if txMap, ok := txAny.(map[string]any); ok {
										txBytes, _ := json.Marshal(txMap)
										var tx blockchain.Transaction
										if err := json.Unmarshal(txBytes, &tx); err == nil {
											chain.EnqueueTransaction(tx)
										}
									}
								}
							}
							chain.Chain[idx] = b
							_ = chain.SaveToDisk()
						}
					}
				}
			}
		case "tx":
			if msg.Tx != nil {
				txBytes, _ := json.Marshal(msg.Tx)
				var tx blockchain.Transaction
				if err := json.Unmarshal(txBytes, &tx); err == nil {
					chain.EnqueueTransaction(tx)
				}
			}
		}
		chain.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted", "from": msg.FromNodeID})
	})
	blockchain.LogJSON("api_listen", "node", fmt.Sprintf("port=%d", port))
	var handler http.Handler = mux
	handler = maxBodyMiddleware(cfg.MaxBodyBytes)(handler)
	handler = idempotencyMiddleware(handler)
	handler = requestIDMiddleware(handler)
	if cfg.EnableTLS {
		if err := http.ListenAndServeTLS(":"+strconv.Itoa(port), cfg.TLSCertFile, cfg.TLSKeyFile, handler); err != nil {
			fmt.Println(err)
		}
		return
	}
	if err := http.ListenAndServe(":"+strconv.Itoa(port), handler); err != nil {
		fmt.Println(err)
	}
}
