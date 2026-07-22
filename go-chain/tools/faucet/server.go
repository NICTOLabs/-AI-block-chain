package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type FaucetServer struct {
	faucet *Faucet
}

func NewFaucetServer(faucet *Faucet) *FaucetServer {
	return &FaucetServer{faucet: faucet}
}

func (s *FaucetServer) Start(addr string) {
	http.HandleFunc("/fund", s.handleFund)
	http.HandleFunc("/balance/", s.handleBalance)
	http.HandleFunc("/health", s.handleHealth)
	log.Printf("Faucet listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func (s *FaucetServer) handleFund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	amount := uint64(1000)
	balance, err := s.faucet.Fund(req.Address, amount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"address": req.Address,
		"amount":  amount,
		"balance": balance,
	})
}

func (s *FaucetServer) handleBalance(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Path[len("/balance/"):]
	balance := s.faucet.GetBalance(address)
	_ = json.NewEncoder(w).Encode(map[string]uint64{"balance": balance})
}

func (s *FaucetServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
