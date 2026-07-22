package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type MiningPool struct {
	mu             sync.Mutex
	address        string
	balance        uint64
	minThreshold   uint64
	workers        map[string]bool
	totalShares    uint64
	lastPayout     time.Time
}

func NewMiningPool(address string, minPayout uint64) *MiningPool {
	return &MiningPool{
		address:      address,
		balance:      0,
		minThreshold: minPayout,
		workers:      make(map[string]bool),
		lastPayout:   time.Now(),
	}
}

func (p *MiningPool) RegisterWorker(workerID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.workers[workerID] = true
	return true
}

func (p *MiningPool) RecordShare(workerID string, shares uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalShares += shares
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

func (p *MiningPool) AddReward(amount uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.balance += amount
}

func (p *MiningPool) Stats() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]any{
		"address":       p.address,
		"balance":       p.balance,
		"workers":       len(p.workers),
		"total_shares":  p.totalShares,
		"min_payout":    p.minThreshold,
		"last_payout":   p.lastPayout,
	}
}

type PoolServer struct {
	pool *MiningPool
}

func NewPoolServer(pool *MiningPool) *PoolServer {
	return &PoolServer{pool: pool}
}

func (s *PoolServer) Start(addr string) {
	http.HandleFunc("/pool/register", s.handleRegister)
	http.HandleFunc("/pool/work", s.handleWork)
	http.HandleFunc("/pool/submit", s.handleSubmit)
	http.HandleFunc("/pool/stats", s.handleStats)
	http.ListenAndServe(addr, nil)
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
	if s.pool.RegisterWorker(req.WorkerID) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "registered"})
	} else {
		http.Error(w, "already registered", http.StatusConflict)
	}
}

func (s *PoolServer) handleWork(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "no_work",
		"note":   "pool mining not yet integrated with node",
	})
}

func (s *PoolServer) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Block interface{} `json:"block"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (s *PoolServer) handleStats(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(s.pool.Stats())
}
