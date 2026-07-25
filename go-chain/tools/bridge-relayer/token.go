package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TokenAdapter struct {
	nodeAPIBase string
	nodeAPIKey  string
}

type MintResult struct {
	Address   string `json:"address"`
	Amount    uint64 `json:"amount"`
	TxHash    string `json:"tx_hash,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

func NewTokenAdapter(nodeAPIBase, nodeAPIKey string) *TokenAdapter {
	return &TokenAdapter{nodeAPIBase: nodeAPIBase, nodeAPIKey: nodeAPIKey}
}

func (a *TokenAdapter) Mint(to string, amount uint64) (MintResult, error) {
	mintResult := MintResult{
		Address:   to,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}

	adminMintURL := a.nodeAPIBase + "/api/admin/mint"
	reqBody, _ := json.Marshal(map[string]any{
		"address": to,
		"amount":  amount,
	})
	req, err := http.NewRequest(http.MethodPost, adminMintURL, bytes.NewReader(reqBody))
	if err != nil {
		return mintResult, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.nodeAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.nodeAPIKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			_ = json.NewDecoder(resp.Body).Decode(&mintResult)
			return mintResult, nil
		}
		if resp.StatusCode == http.StatusNotFound {
			return a.mintViaFaucet(to, amount)
		}
		body, _ := io.ReadAll(resp.Body)
		return mintResult, fmt.Errorf("admin mint failed: %s", string(body))
	}
	return mintResult, fmt.Errorf("admin mint request failed: %v", err)
}

func (a *TokenAdapter) mintViaFaucet(to string, amount uint64) (MintResult, error) {
	faucetURL := a.nodeAPIBase + "/api/faucet"
	reqBody, _ := json.Marshal(map[string]any{
		"address": to,
		"amount":  amount,
	})
	req, err := http.NewRequest(http.MethodPost, faucetURL, bytes.NewReader(reqBody))
	if err != nil {
		return MintResult{Address: to, Amount: amount}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.nodeAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.nodeAPIKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return MintResult{Address: to, Amount: amount}, fmt.Errorf("faucet mint failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return MintResult{Address: to, Amount: amount}, fmt.Errorf("faucet mint failed: %s", string(body))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return MintResult{Address: to, Amount: amount}, nil
	}

	txHash := ""
	if h, ok := result["tx_id"].(string); ok {
		txHash = h
	} else if h, ok := result["tx_hash"].(string); ok {
		txHash = h
	}

	return MintResult{
		Address:   to,
		Amount:    amount,
		TxHash:    txHash,
		Timestamp: time.Now().Unix(),
	}, nil
}

func (a *TokenAdapter) Burn(from string, amount uint64) error {
	burnURL := a.nodeAPIBase + "/api/transfer"
	reqBody, _ := json.Marshal(map[string]any{
		"from":   from,
		"to":     "0000000000000000000000000000000000000000000000000000000000000000",
		"amount": amount,
	})
	req, err := http.NewRequest(http.MethodPost, burnURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if a.nodeAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.nodeAPIKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("burn via transfer failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("burn via transfer failed: %s", string(body))
	}
	return nil
}


