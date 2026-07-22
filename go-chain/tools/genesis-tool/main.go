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
	"sort"
	"time"
)

type GenesisAllocation struct {
	Address     string `json:"address"`
	PublicKey   string `json:"public_key"`
	Amount      uint64 `json:"amount"`
	Description string `json:"description"`
	LockupUntil int64  `json:"lockup_until,omitempty"`
	Vesting     string `json:"vesting,omitempty"`
}

type ValidatorMeta struct {
	Address string `json:"address"`
	PublicKey string `json:"public_key"`
	Stake   uint64 `json:"stake"`
	Country string `json:"country"`
	City    string `json:"city"`
	Contact string `json:"contact,omitempty"`
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
	Validators  []ValidatorMeta     `json:"validators"`
	Economics struct {
		BaseGasFee         uint64  `json:"base_gas_fee"`
		BurnRatePercent    uint64  `json:"burn_rate_percent"`
		RewardRatePercent  uint64  `json:"reward_rate_percent"`
		MinStake           uint64  `json:"min_stake"`
		BlockTimeSeconds   int     `json:"block_time_seconds"`
		InflationStartYear int     `json:"inflation_start_year"`
		InflationDecay     float64 `json:"inflation_decay_per_year"`
	} `json:"economics"`
	Multisig struct {
		Threshold int      `json:"threshold"`
		Signers   []string `json:"signers,omitempty"`
	} `json:"multisig"`
}

func main() {
	output := flag.String("output", "genesis_mainnet.json", "Output path for genesis file")
	chainID := flag.String("chain-id", "tdr-mainnet-1", "Chain ID")
	initialSupply := flag.Uint64("initial-supply", 4500000000, "Initial token supply")
	maxSupply := flag.Uint64("max-supply", 10000000000, "Maximum token supply")
	validatorCount := flag.Int("validator-count", 7, "Number of genesis validators")
	teamPct := flag.Float64("team-pct", 0.15, "Team allocation percentage")
	communityPct := flag.Float64("community-pct", 0.35, "Community allocation percentage")
	treasuryPct := flag.Float64("treasury-pct", 0.20, "Treasury allocation percentage")
	stakingPct := flag.Float64("staking-pct", 0.25, "Staking rewards percentage")
	liquidityPct := flag.Float64("liquidity-pct", 0.05, "Liquidity allocation percentage")
	flag.Parse()

	total := float64(*initialSupply)
	pctTotal := teamPct + communityPct + treasuryPct + stakingPct + liquidityPct
	if pctTotal != 1.0 {
		panic(fmt.Sprintf("allocation percentages must sum to 1.0, got %.2f", pctTotal))
	}

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
	genesis.Economics.InflationStartYear = 2026
	genesis.Economics.InflationDecay = 0.15
	genesis.Multisig.Threshold = 2

	allocations := []struct {
		desc string
		pct  float64
		lock bool
	}{
		{"Team and advisors", *teamPct, true},
		{"Community and ecosystem", *communityPct, false},
		{"Treasury and operations", *treasuryPct, false},
		{"Staking rewards pool", *stakingPct, false},
		{"Liquidity and exchange listings", *liquidityPct, false},
	}

	var teamAddr string
	var teamPub string
	for _, alloc := range allocations {
		key, err := generateKeyPair()
		if err != nil {
			panic(err)
		}
		amount := uint64(float64(*initialSupply) * alloc.pct)
		if alloc.desc == "Team and advisors" {
			teamAddr = key.address
			teamPub = key.publicKey
		}
		entry := GenesisAllocation{
			Address:     key.address,
			PublicKey:   key.publicKey,
			Amount:      amount,
			Description: alloc.desc,
			Vesting:     "linear_4_years",
		}
		if alloc.lock {
			entry.LockupUntil = time.Now().AddDate(4, 0, 0).Unix()
		}
		genesis.Allocations = append(genesis.Allocations, entry)
	}

	countries := []string{"KE", "SG", "DE", "NG", "ZA", "AE", "US"}
	cities := map[string]string{
		"KE": "Nairobi",
		"SG": "Singapore",
		"DE": "Frankfurt",
		"NG": "Lagos",
		"ZA": "Cape Town",
		"AE": "Dubai",
		"US": "New York",
	}
	sort.Slice(countries, func(i, j int) bool { return countries[i] < countries[j] })

	for i := 0; i < *validatorCount; i++ {
		key, err := generateKeyPair()
		if err != nil {
			panic(err)
		}
		country := countries[i%len(countries)]
		genesis.Validators = append(genesis.Validators, ValidatorMeta{
			Address:   key.address,
			PublicKey: key.publicKey,
			Stake:     100000,
			Country:   country,
			City:      cities[country],
			Contact:   fmt.Sprintf("validator-%d@tender.network", i+1),
		})
		genesis.Allocations = append(genesis.Allocations, GenesisAllocation{
			Address:     key.address,
			PublicKey:   key.publicKey,
			Amount:      100000,
			Description: fmt.Sprintf("Genesis validator %d stake", i+1),
			Vesting:     "none",
		})
		if i < 2 {
			genesis.Multisig.Signers = append(genesis.Multisig.Signers, key.publicKey)
		}
	}
	if len(genesis.Multisig.Signers) == 0 && len(genesis.Validators) > 0 {
		genesis.Multisig.Signers = append(genesis.Multisig.Signers, genesis.Validators[0].PublicKey)
	}

	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		panic(err)
	}
	_ = os.MkdirAll("genesis", 0o755)
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		panic(err)
	}
	signature := sha256.Sum256(data)
	_ = os.WriteFile(*output+".sha256", []byte(hex.EncodeToString(signature[:])+"  "+filepath.Base(*output)+"\n"), 0o644)

	fmt.Printf("Genesis file written to %s\n", *output)
	fmt.Printf("Genesis SHA256: %s\n", hex.EncodeToString(signature[:]))
	fmt.Printf("Total supply: %d\n", genesis.InitialSupply)
	fmt.Printf("Validators: %d\n", len(genesis.Validators))
	fmt.Printf("Allocations: %d\n", len(genesis.Allocations))
	fmt.Printf("Multisig threshold: %d/%d\n", genesis.Multisig.Threshold, len(genesis.Multisig.Signers))
}

type keyPair struct {
	address    string
	publicKey  string
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
