package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Miner struct {
	apiURL    string
	minerAddr string
}

func NewMiner(apiURL, minerAddr string) *Miner {
	return &Miner{apiURL: apiURL, minerAddr: minerAddr}
}

func (m *Miner) Start() {
	fmt.Printf("Miner started: %s\n", m.minerAddr)
	fmt.Printf("API: %s\n", m.apiURL)
	for {
		block, err := m.mineBlock()
		if err != nil {
			fmt.Printf("mine error: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if block != nil {
			idx := fmt.Sprint(block["index"])
			hash := fmt.Sprint(block["block_hash"])
			fmt.Printf("Block mined: index=%s hash=%s\n", idx, hash)
			if err := m.submitBlock(block); err != nil {
				fmt.Printf("submit error: %v\n", err)
			}
		}
	}
}

func (m *Miner) mineBlock() (map[string]interface{}, error) {
	payload, _ := json.Marshal(map[string]string{"miner_address": m.minerAddr})
	resp, err := http.Post(m.apiURL+"/api/mine", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mine endpoint returned %d", resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (m *Miner) submitBlock(block map[string]interface{}) error {
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
	miner := NewMiner(apiURL, minerAddr)
	miner.Start()
}
