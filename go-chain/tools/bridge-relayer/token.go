package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	dataDir := filepath.Join(getTenderDataDir(), "chain.json")
	data, err := os.ReadFile(dataDir)
	if err != nil {
		return fmt.Errorf("read chain data failed: %v", err)
	}

	var state map[string]json.RawMessage
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse chain data failed: %v", err)
	}

	var ledger map[string]struct {
		Address  string `json:"address"`
		Balance  uint64 `json:"balance"`
		Staked   uint64 `json:"staked"`
		IsAgent  bool   `json:"is_agent"`
	}
	if err := json.Unmarshal(state["ledger"], &ledger); err != nil {
		return fmt.Errorf("parse ledger failed: %v", err)
	}

	acct, ok := ledger[from]
	if !ok {
		return fmt.Errorf("account not found: %s", from)
	}
	if acct.Balance < amount {
		return fmt.Errorf("insufficient balance for burn: %d < %d", acct.Balance, amount)
	}

	acct.Balance -= amount
	ledger[from] = acct

	newLedger, _ := json.Marshal(ledger)
	state["ledger"] = newLedger
	newData, _ := json.MarshalIndent(state, "", "  ")
	if err := os.WriteFile(dataDir, newData, 0o644); err != nil {
		return fmt.Errorf("write chain data failed: %v", err)
	}
	return nil
}

func getTenderDataDir() string {
	if d := os.Getenv("TENDER_DATA_DIR"); d != "" {
		return d
	}
	return "./data"
}

func SignMintRequest(privKey []byte, txHash, to string, amount uint64) string {
	payload := []byte(fmt.Sprintf("%s:%s:%d", txHash, to, amount))
	key := sha256.Sum256(privKey)
	mac := hmac.New(sha256.New, key[:])
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
