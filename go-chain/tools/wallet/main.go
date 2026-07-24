package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type WalletFile struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
	Address    string `json:"address"`
	Label      string `json:"label"`
	CreatedAt  string `json:"created_at"`
}

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
	fmt.Printf("  Label:      %s\n", wallet.Label)
	fmt.Printf("  Created:    %s\n", wallet.CreatedAt)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("TENDER Wallet Tool")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Println("  tender-wallet create [label]   Create a new wallet")
		fmt.Println("  tender-wallet show <file>      Show wallet details")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  tender-wallet create my-wallet")
		fmt.Println("  tender-wallet show tender-wallet-abcd1234.json")
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
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
