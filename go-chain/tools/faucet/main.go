package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

type Faucet struct {
	mu         sync.Mutex
	balances  map[string]uint64
	limit     uint64
	cooldown  time.Duration
	lastClaim map[string]time.Time
}

func NewFaucet(limit uint64, cooldown time.Duration) *Faucet {
	return &Faucet{
		balances:  make(map[string]uint64),
		limit:     limit,
		cooldown:  cooldown,
		lastClaim: make(map[string]time.Time),
	}
}

func (f *Faucet) Fund(address string, amount uint64) (uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if amount > f.limit {
		return 0, fmt.Errorf("amount exceeds per-address limit")
	}
	if _, ok := f.balances[address]; !ok {
		f.balances[address] = 0
	}
	if last, ok := f.lastClaim[address]; ok {
		if time.Since(last) < f.cooldown {
			return 0, fmt.Errorf("cooldown period not elapsed")
		}
	}
	f.balances[address] += amount
	f.lastClaim[address] = time.Now()
	return f.balances[address], nil
}

func (f *Faucet) GetBalance(address string) uint64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.balances[address]
}

func (f *Faucet) TopUp(amount uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.limit += amount
}

func GenerateTestToken() string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func GenerateRandomAmount(max uint64) uint64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return uint64(n.Int64())
}
