package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type WalletFile struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
	Address    string `json:"address"`
	Label      string `json:"label"`
	CreatedAt  string `json:"created_at"`
}

type Transaction struct {
	ID         string `json:"id,omitempty"`
	From       string `json:"from"`
	FromPubKey string `json:"from_pubkey"`
	To         string `json:"to"`
	Amount     uint64 `json:"amount"`
	Fee        uint64 `json:"fee,omitempty"`
	Nonce      uint64 `json:"nonce,omitempty"`
	TxType     string `json:"tx_type"`
	Payload    string `json:"payload,omitempty"`
	Signature  string `json:"signature,omitempty"`
	Timestamp  int64  `json:"timestamp"`
	ChainID    string `json:"chain_id,omitempty"`
}

const defaultAPI = "http://localhost:8080"

func createWallet(label string) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating key: %v\n", err)
		os.Exit(1)
	}
	address := sha256.Sum256(pub)
	addr := hex.EncodeToString(address[:])

	wallet := WalletFile{
		PublicKey:  hex.EncodeToString(pub),
		PrivateKey: hex.EncodeToString(priv),
		Address:    addr,
		Label:      label,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	filename := "tender-wallet-" + addr[:8] + ".json"
	data, _ := json.MarshalIndent(wallet, "", "  ")
	if err := os.WriteFile(filename, data, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "error saving wallet: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== TENDER Wallet Created ===")
	fmt.Printf("  Address:    %s\n", addr)
	fmt.Printf("  Public Key: %s\n", hex.EncodeToString(pub))
	fmt.Printf("  File:       %s\n", filename)
	fmt.Println()
	fmt.Println("IMPORTANT: Back up this file. It contains your private key.")
	fmt.Println("Never share it with anyone.")
}

func showWallet(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading wallet: %v\n", err)
		os.Exit(1)
	}
	var wallet WalletFile
	if err := json.Unmarshal(data, &wallet); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing wallet: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("=== TENDER Wallet ===")
	fmt.Printf("  Address:    %s\n", wallet.Address)
	fmt.Printf("  Public Key: %s\n", wallet.PublicKey)
	fmt.Printf("  Private Key: %s\n", wallet.PrivateKey)
	fmt.Printf("  Label:      %s\n", wallet.Label)
	fmt.Printf("  Created:    %s\n", wallet.CreatedAt)
}

func balanceWallet(apiURL, address string) {
	resp, err := http.Get(apiURL + "/api/accounts")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error connecting to node at %s: %v\n", apiURL, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading response: %v\n", err)
		os.Exit(1)
	}
	var accounts map[string]struct {
		Address string `json:"address"`
		Balance uint64 `json:"balance"`
		Staked  uint64 `json:"staked"`
		IsAgent bool   `json:"is_agent"`
	}
	if err := json.Unmarshal(body, &accounts); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing accounts: %v\n", err)
		os.Exit(1)
	}
	acct, exists := accounts[address]
	if !exists {
		fmt.Printf("Address %s has 0 TDR (not found on chain)\n", address)
		return
	}
	fmt.Printf("Address: %s\n", address)
	fmt.Printf("Balance: %d TDR\n", acct.Balance)
	fmt.Printf("Staked:  %d TDR\n", acct.Staked)
	balanceTdr := acct.Balance / 100_000_000
	balanceHogo := acct.Balance % 100_000_000
	fmt.Printf("Balance: %d.%08d TDR\n", balanceTdr, balanceHogo)
}

func sendTransaction(apiURL, walletFile, toAddress string, amount uint64, fee uint64) {
	data, err := os.ReadFile(walletFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading wallet: %v\n", err)
		os.Exit(1)
	}
	var wallet WalletFile
	if err := json.Unmarshal(data, &wallet); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing wallet: %v\n", err)
		os.Exit(1)
	}

	privBytes, err := hex.DecodeString(wallet.PrivateKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error decoding private key: %v\n", err)
		os.Exit(1)
	}
	priv := ed25519.PrivateKey(privBytes)

	hogohogoAmount := amount * 100_000_000

	tx := Transaction{
		From:    wallet.Address,
		To:      toAddress,
		Amount:  hogohogoAmount,
		Fee:     fee,
		Nonce:   0,
		TxType:  "TRANSFER",
		ChainID: "tdr-mainnet-1",
	}

	tx.FromPubKey = wallet.PublicKey
	tx.Timestamp = time.Now().Unix()
	payload := canonicalBytes(tx)
	hash := sha256.Sum256(payload)
	tx.ID = hex.EncodeToString(hash[:])
	sig := ed25519.Sign(priv, payload)
	tx.Signature = hex.EncodeToString(sig)

	body, _ := json.Marshal(tx)
	resp, err := http.Post(apiURL+"/api/transactions", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error submitting transaction: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "transaction rejected (%d): %s\n", resp.StatusCode, string(respBody))
		os.Exit(1)
	}
	fmt.Println("=== Transaction Sent ===")
	fmt.Printf("  From:  %s\n", wallet.Address)
	fmt.Printf("  To:    %s\n", toAddress)
	fmt.Printf("  Amount: %d TDR\n", amount)
	fmt.Printf("  Tx ID: %s\n", tx.ID)
	fmt.Printf("  Status: accepted\n")
}

func canonicalBytes(tx Transaction) []byte {
	b := strings.Builder{}
	b.WriteString(tx.ChainID)
	b.WriteString("\x00")
	b.WriteString(tx.From)
	b.WriteString("\x00")
	b.WriteString(tx.FromPubKey)
	b.WriteString("\x00")
	b.WriteString(tx.To)
	b.WriteString("\x00")
	b.WriteString(strconv.FormatUint(tx.Amount, 10))
	b.WriteString("\x00")
	b.WriteString(strconv.FormatUint(tx.Fee, 10))
	b.WriteString("\x00")
	b.WriteString(strconv.FormatUint(tx.Nonce, 10))
	b.WriteString("\x00")
	b.WriteString(tx.TxType)
	b.WriteString("\x00")
	b.WriteString(tx.Payload)
	b.WriteString("\x00")
	b.WriteString(strconv.FormatInt(tx.Timestamp, 10))
	return []byte(b.String())
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("TENDER Wallet Tool")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  tender-wallet create [label]             Create a new wallet")
		fmt.Println("  tender-wallet show <file>                Show wallet details")
		fmt.Println("  tender-wallet balance [address] [api]    Check balance of an address")
		fmt.Println("  tender-wallet send <file> <to> <amount> [api] [fee]  Send TDR")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  tender-wallet create my-wallet")
		fmt.Println("  tender-wallet show tender-wallet-abcd1234.json")
		fmt.Println("  tender-wallet balance f7cddfdc...")
		fmt.Println("  tender-wallet send tender-wallet-abcd.json f7cddfdc... 100")
		return
	}

	switch os.Args[1] {
	case "create":
		label := "default"
		if len(os.Args) > 2 {
			label = os.Args[2]
		}
		createWallet(label)
	case "show":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: tender-wallet show <wallet-file>\n")
			os.Exit(1)
		}
		showWallet(os.Args[2])
	case "balance":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: tender-wallet balance <address> [api-url]\n")
			os.Exit(1)
		}
		apiURL := defaultAPI
		if len(os.Args) > 3 {
			apiURL = os.Args[3]
		}
		balanceWallet(apiURL, os.Args[2])
	case "send":
		if len(os.Args) < 5 {
			fmt.Fprintf(os.Stderr, "usage: tender-wallet send <wallet-file> <to-address> <amount> [api-url] [fee]\n")
			os.Exit(1)
		}
		apiURL := defaultAPI
		fee := uint64(1)
		args := os.Args[2:]
		walletFile := args[0]
		toAddress := args[1]
		amount, err := strconv.ParseUint(args[2], 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid amount: %v\n", err)
			os.Exit(1)
		}
		if len(args) > 3 {
			apiURL = args[3]
		}
		if len(args) > 4 {
			fee, err = strconv.ParseUint(args[4], 10, 64)
			if err != nil {
				fmt.Fprintf(os.Stderr, "invalid fee: %v\n", err)
				os.Exit(1)
			}
		}
		sendTransaction(apiURL, walletFile, toAddress, amount, fee)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
