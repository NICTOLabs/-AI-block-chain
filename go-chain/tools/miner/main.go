package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"time"
)

type Miner struct {
	apiURL     string
	minerAddr  string
	threads    int
	hashrate   uint64
}

func NewMiner(apiURL, minerAddr string, threads int) *Miner {
	return &Miner{apiURL: apiURL, minerAddr: minerAddr, threads: threads}
}

func (m *Miner) Start() {
	fmt.Printf("Miner started: %s\n", m.minerAddr)
	fmt.Printf("API: %s\n", m.apiURL)
	fmt.Printf("Threads: %d\n", m.threads)
	for {
		block, err := m.mineBlock()
		if err != nil {
			fmt.Printf("mine error: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if block != nil {
			if err := m.submitBlock(block); err != nil {
				fmt.Printf("submit error: %v\n", err)
			} else {
				fmt.Printf("Block mined: index=%d hash=%s\n", block.Index, block.BlockHash)
			}
		}
	}
}

func (m *Miner) mineBlock() (*Block, error) {
	resp, err := http.Get(m.apiURL + "/api/mine")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mine endpoint returned %d", resp.StatusCode)
	}
	var block Block
	if err := json.NewDecoder(resp.Body).Decode(&block); err != nil {
		return nil, err
	}
	return &block, nil
}

func (m *Miner) submitBlock(block *Block) error {
	payload, _ := json.Marshal(block)
	resp, err := http.Post(m.apiURL+"/api/miner/submit", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("submit endpoint returned %d", resp.StatusCode)
	}
	return nil
}

func CalculateHash(block Block) string {
	clone := block
	clone.BlockHash = ""
	data, _ := json.Marshal(clone)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func HashMatchesTarget(hash string, difficulty uint32) bool {
	target := big.NewInt(1)
	target.Lsh(target, 256-uint(difficulty))
	hashInt := new(big.Int)
	hashInt.SetString(hash, 16)
	return hashInt.Cmp(target) < 0
}

func main() {
	apiURL := os.Getenv("TENDER_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}
	minerAddr := os.Getenv("TENDER_MINER_ADDRESS")
	if minerAddr == "" {
		fmt.Fprintln(os.Stderr, "TENDER_MINER_ADDRESS is required")
		os.Exit(1)
	}
	threads := 1
	if t := os.Getenv("TENDER_MINER_THREADS"); t != "" {
		fmt.Sscanf(t, "%d", &threads)
	}
	miner := NewMiner(apiURL, minerAddr, threads)
	miner.Start()
}
