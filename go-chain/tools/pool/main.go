package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type MiningPool struct {
	mu             sync.Mutex
	address        string
	balance        uint64
	minThreshold   uint64
	workers        map[string]uint64
	totalShares    uint64
	lastPayout     time.Time
	difficulty     uint32
	latestBlockIdx uint64
}

func NewMiningPool(address string, minPayout uint64, difficulty uint32) *MiningPool {
	return &MiningPool{
		address:      address,
		balance:      0,
		minThreshold: minPayout,
		workers:      make(map[string]uint64),
		lastPayout:   time.Now(),
		difficulty:   difficulty,
	}
}

func (p *MiningPool) RegisterWorker(workerID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.workers[workerID]; exists {
		return false
	}
	p.workers[workerID] = 0
	return true
}

func (p *MiningPool) RecordShare(workerID string, shares uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workers[workerID] += shares
	p.totalShares += shares
}

func (p *MiningPool) AddReward(amount uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.balance += amount
}

func (p *MiningPool) Payout() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.balance < p.minThreshold {
		return 0
	}
	amount := p.balance
	p.balance = 0
	p.lastPayout = time.Now()
	return amount
}

func (p *MiningPool) Stats() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]any{
		"address":      p.address,
		"balance":      p.balance,
		"workers":      len(p.workers),
		"total_shares": p.totalShares,
		"min_payout":   p.minThreshold,
		"difficulty":   p.difficulty,
		"last_payout":  p.lastPayout,
	}
}

func (p *MiningPool) ValidateShare(blockHash string, _ uint64, difficulty uint32) bool {
	if len(blockHash) != 64 {
		return false
	}
	for _, c := range blockHash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	prefix := strings.Repeat("0", int((difficulty+3)/4))
	return strings.HasPrefix(blockHash, prefix)
}

type PoolServer struct {
	pool *MiningPool
}

func NewPoolServer(pool *MiningPool) *PoolServer {
	return &PoolServer{pool: pool}
}

func (s *PoolServer) Start(addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/pool/register", s.handleRegister)
	mux.HandleFunc("/pool/work", s.handleWork)
	mux.HandleFunc("/pool/submit", s.handleSubmit)
	mux.HandleFunc("/pool/stats", s.handleStats)
	mux.HandleFunc("/pool/payout", s.handlePayout)
	log.Printf("pool server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("pool server: %v", err)
	}
}

func (s *PoolServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		WorkerID string `json:"worker_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.WorkerID == "" {
		http.Error(w, "worker_id required", http.StatusBadRequest)
		return
	}
	if s.pool.RegisterWorker(req.WorkerID) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "registered", "worker_id": req.WorkerID})
	} else {
		http.Error(w, "worker already registered", http.StatusConflict)
	}
}

func (s *PoolServer) handleWork(w http.ResponseWriter, r *http.Request) {
	workerID := r.URL.Query().Get("worker_id")
	if workerID == "" {
		http.Error(w, "worker_id required", http.StatusBadRequest)
		return
	}
	s.pool.mu.Lock()
	_, registered := s.pool.workers[workerID]
	s.pool.mu.Unlock()
	if !registered {
		http.Error(w, "worker not registered", http.StatusForbidden)
		return
	}
	work := map[string]any{
		"difficulty": s.pool.difficulty,
		"worker_id":  workerID,
		"timestamp":  time.Now().Unix(),
		"block_template": map[string]any{
			"previous_hash": fmt.Sprintf("%064x", s.pool.latestBlockIdx),
			"miner_address": s.pool.address,
			"transactions":  []string{},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(work)
}

func (s *PoolServer) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		WorkerID  string `json:"worker_id"`
		BlockHash string `json:"block_hash"`
		Nonce     uint64 `json:"nonce"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.WorkerID == "" || req.BlockHash == "" {
		http.Error(w, "worker_id and block_hash required", http.StatusBadRequest)
		return
	}
	s.pool.mu.Lock()
	_, registered := s.pool.workers[req.WorkerID]
	s.pool.mu.Unlock()
	if !registered {
		http.Error(w, "worker not registered", http.StatusForbidden)
		return
	}
	if !s.pool.ValidateShare(req.BlockHash, req.Nonce, s.pool.difficulty) {
		http.Error(w, "share does not meet difficulty target", http.StatusBadRequest)
		return
	}
	s.pool.RecordShare(req.WorkerID, 1)
	s.pool.mu.Lock()
	shares := s.pool.workers[req.WorkerID]
	s.pool.mu.Unlock()
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "accepted",
		"worker_id": req.WorkerID,
		"shares":    shares,
	})
}

func (s *PoolServer) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.pool.Stats())
}

func (s *PoolServer) handlePayout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	amount := s.pool.Payout()
	if amount == 0 {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "below_threshold", "balance": s.pool.balance, "min_payout": s.pool.minThreshold})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "payout_queued", "amount": amount, "address": s.pool.address})
}

func main() {
	addr := flag.String("addr", ":8083", "Pool listen address")
	poolAddr := flag.String("pool-address", os.Getenv("POOL_ADDRESS"), "Pool reward address")
	minPayout := flag.Uint64("min-payout", 1000, "Minimum payout threshold in TDR")
	difficulty := flag.Uint("difficulty", 4, "Share difficulty target")
	flag.Parse()

	if *poolAddr == "" {
		fmt.Fprintln(os.Stderr, "error: pool address required (use --pool-address or POOL_ADDRESS env)")
		flag.Usage()
		os.Exit(1)
	}

	pool := NewMiningPool(*poolAddr, *minPayout, uint32(*difficulty))
	server := NewPoolServer(pool)
	server.Start(*addr)
}
