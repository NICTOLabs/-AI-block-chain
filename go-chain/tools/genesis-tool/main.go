package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type GenesisAllocation struct {
	Address     string `json:"address"`
	PublicKey   string `json:"public_key"`
	Amount      uint64 `json:"amount"`
	Description string `json:"description"`
	LockupUntil int64  `json:"lockup_until,omitempty"`
}

type GenesisFile struct {
	ChainID      string              `json:"chain_id"`
	GenesisTime  int64               `json:"genesis_time"`
	Consensus    string              `json:"consensus"`
	InitialSupply uint64             `json:"initial_supply"`
	MaxSupply    uint64              `json:"max_supply"`
	Token        struct {
		Name     string `json:"name"`
		Symbol   string `json:"symbol"`
		Decimals int    `json:"decimals"`
	} `json:"token"`
	Allocations []GenesisAllocation `json:"allocations"`
	Validators  []struct {
		Address string `json:"address"`
		Stake   uint64 `json:"stake"`
		Country string `json:"country"`
	} `json:"validators"`
	Economics struct {
		BaseGasFee         uint64  `json:"base_gas_fee"`
		BurnRatePercent    uint64  `json:"burn_rate_percent"`
		RewardRatePercent  uint64  `json:"reward_rate_percent"`
		MinStake           uint64  `json:"min_stake"`
		BlockTimeSeconds   int     `json:"block_time_seconds"`
	} `json:"economics"`
}

func main() {
	output := flag.String("output", "genesis_mainnet.json", "Output path for genesis file")
	chainID := flag.String("chain-id", "tdr-mainnet-1", "Chain ID")
	initialSupply := flag.Uint64("initial-supply", 4500000000, "Initial token supply")
	maxSupply := flag.Uint64("max-supply", 10000000000, "Maximum token supply")
	teamAmount := flag.Uint64("team-amount", 675000000, "Team allocation")
	communityAmount := flag.Uint64("community-amount", 1575000000, "Community allocation")
	treasuryAmount := flag.Uint64("treasury-amount", 900000000, "Treasury allocation")
	stakingAmount := flag.Uint64("staking-amount", 1125000000, "Staking rewards pool")
	liquidityAmount := flag.Uint64("liquidity-amount", 225000000, "Liquidity allocation")
	validatorCount := flag.Int("validator-count", 7, "Number of genesis validators")
	flag.Parse()

	genesis := GenesisFile{
		ChainID:       *chainID,
		GenesisTime:   time.Now().Unix(),
		Consensus:     "hybrid-pos-poa",
		InitialSupply: *initialSupply,
		MaxSupply:     *maxSupply,
	}
	genesis.Token.Name = "TENDER"
	genesis.Token.Symbol = "TDR"
	genesis.Token.Decimals = 8
	genesis.Economics.BaseGasFee = 5
	genesis.Economics.BurnRatePercent = 1
	genesis.Economics.RewardRatePercent = 4
	genesis.Economics.MinStake = 100
	genesis.Economics.BlockTimeSeconds = 5

	vestingCliff := time.Now().AddDate(4, 0, 0).Unix()

	teamKey, _ := generateKeyPair()
	communityKey, _ := generateKeyPair()
	treasuryKey, _ := generateKeyPair()
	stakingKey, _ := generateKeyPair()
	liquidityKey, _ := generateKeyPair()

	genesis.Allocations = []GenesisAllocation{
		{
			Address:     teamKey.address,
			PublicKey:   teamKey.publicKey,
			Amount:      *teamAmount,
			Description: "Team and advisors - 4-year vesting",
			LockupUntil: vestingCliff,
		},
		{
			Address:     communityKey.address,
			PublicKey:   communityKey.publicKey,
			Amount:      *communityAmount,
			Description: "Community and ecosystem fund",
		},
		{
			Address:     treasuryKey.address,
			PublicKey:   treasuryKey.publicKey,
			Amount:      *treasuryAmount,
			Description: "Treasury and operations",
		},
		{
			Address:     stakingKey.address,
			PublicKey:   stakingKey.publicKey,
			Amount:      *stakingAmount,
			Description: "Staking rewards pool",
		},
		{
			Address:     liquidityKey.address,
			PublicKey:   liquidityKey.publicKey,
			Amount:      *liquidityAmount,
			Description: "Liquidity and exchange listings",
		},
	}

	for i := 0; i < *validatorCount; i++ {
		key, _ := generateKeyPair()
		stake := uint64(100000)
		genesis.Validators = append(genesis.Validators, struct {
			Address string `json:"address"`
			Stake   uint64 `json:"stake"`
			Country string `json:"country"`
		}{
			Address: key.address,
			Stake:   stake,
			Country: "KE",
		})
		genesis.Allocations = append(genesis.Allocations, GenesisAllocation{
			Address:     key.address,
			PublicKey:   key.publicKey,
			Amount:      stake,
			Description: fmt.Sprintf("Genesis validator %d stake", i+1),
		})
	}

	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		panic(err)
	}
	_ = os.MkdirAll("genesis", 0o755)
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("Genesis file written to %s\n", *output)
	fmt.Printf("Total supply: %d\n", genesis.InitialSupply)
	fmt.Printf("Validators: %d\n", len(genesis.Validators))
	fmt.Printf("Allocations: %d\n", len(genesis.Allocations))
}

type keyPair struct {
	address  string
	publicKey string
	privateKey string
}

func generateKeyPair() (keyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return keyPair{}, err
	}
	addressBytes := sha256.Sum256(pub)
	return keyPair{
		address:    hex.EncodeToString(addressBytes[:]),
		publicKey:  hex.EncodeToString(pub),
		privateKey: hex.EncodeToString(priv),
	}, nil
}
