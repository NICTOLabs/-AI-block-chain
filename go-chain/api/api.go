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
	MetricsPath string
	APIPort     int
	P2PPort     int
	DataDir     string
	Consensus   string
	StrictP2P   bool
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
		APIKey:      getEnvOrDefault("TENDER_API_KEY", "change-me-in-production"),
		EnableAuth:  getEnvBoolOrDefault("TENDER_ENABLE_AUTH", true),
		RateLimit:   getEnvIntOrDefault("TENDER_RATE_LIMIT", 60),
		RateWindow:  time.Duration(getEnvIntOrDefault("TENDER_RATE_WINDOW_SECONDS", 60)) * time.Second,
		MetricsPath: getEnvOrDefault("TENDER_METRICS_PATH", "/metrics"),
		APIPort:     getEnvIntOrDefault("TENDER_API_PORT", 8080),
		P2PPort:     getEnvIntOrDefault("TENDER_P2P_PORT", 3030),
		DataDir:     getEnvOrDefault("TENDER_DATA_DIR", "./data"),
		Consensus:   strings.ToLower(getEnvOrDefault("TENDER_CONSENSUS", "pos")),
		StrictP2P:   getEnvBoolOrDefault("TENDER_STRICT_P2P", true),
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
			Address string `json:"address"`
			Stake   uint64 `json:"stake"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := chain.RegisterValidator(payload.Address, payload.Stake); err != nil {
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
		_ = json.NewEncoder(w).Encode(map[string]any{
			"currency":            blockchain.CurrencySymbol(),
			"token_supply":        chain.TokenSupply,
			"burn_rate_percent":   blockchain.BurnRatePercent,
			"reward_rate_percent": blockchain.RewardRatePercent,
			"base_fee":            blockchain.BaseFee,
			"escrows":             chain.Escrows,
			"proposals":           chain.Proposals,
		})
	})
	blockchain.LogJSON("api_listen", "node", fmt.Sprintf("port=%d", port))
	if err := http.ListenAndServe(":"+strconv.Itoa(port), requestIDMiddleware(mux)); err != nil {
		fmt.Println(err)
	}
}
