package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	Address  string `json:"address"`
	PublicKey string `json:"public_key"`
	Stake    uint64 `json:"stake"`
	Country  string `json:"country"`
	City     string `json:"city"`
	Contact  string `json:"contact,omitempty"`
}

type GenesisSignature struct {
	SignerAddress string `json:"signer_address"`
	PublicKey     string `json:"public_key"`
	Signature     string `json:"signature"`
	SignedAt      int64  `json:"signed_at"`
}

type GenesisFile struct {
	ChainID       string              `json:"chain_id"`
	GenesisTime   int64               `json:"genesis_time"`
	Consensus     string              `json:"consensus"`
	InitialSupply uint64              `json:"initial_supply"`
	MaxSupply     uint64              `json:"max_supply"`
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
		Threshold int                `json:"threshold"`
		Signers   []string           `json:"signers,omitempty"`
		Signatures []GenesisSignature `json:"signatures,omitempty"`
		Finalized bool               `json:"finalized"`
		FinalizedAt int64            `json:"finalized_at,omitempty"`
	} `json:"multisig"`
}

func mainGenerate() {
	output := flag.String("output", "genesis_mainnet.json", "Output path for genesis file")
	chainID := flag.String("chain-id", "tdr-mainnet-1", "Chain ID")
	initialSupply := flag.Uint64("initial-supply", 4500000000, "Initial token supply")
	maxSupply := flag.Uint64("max-supply", 18446744073709551615, "Maximum token supply")
	validatorCount := flag.Int("validator-count", 7, "Number of genesis validators")
	teamPct := flag.Float64("team-pct", 0.15, "Team allocation percentage")
	communityPct := flag.Float64("community-pct", 0.35, "Community allocation percentage")
	treasuryPct := flag.Float64("treasury-pct", 0.20, "Treasury allocation percentage")
	stakingPct := flag.Float64("staking-pct", 0.25, "Staking rewards percentage")
	liquidityPct := flag.Float64("liquidity-pct", 0.05, "Liquidity allocation percentage")
	flag.Parse()

	pctTotal := *teamPct + *communityPct + *treasuryPct + *stakingPct + *liquidityPct
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

	for _, alloc := range allocations {
		key, err := generateKeyPair()
		if err != nil {
			panic(err)
		}
		amount := uint64(float64(*initialSupply) * alloc.pct)
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

	writeGenesis(*output, genesis)
	fmt.Printf("Genesis phase 1 written to %s\n", *output)
	fmt.Printf("Supply: %d | Validators: %d | Signers: %d/%d\n", genesis.InitialSupply, len(genesis.Validators), len(genesis.Multisig.Signatures), genesis.Multisig.Threshold)
}

func mainSign() {
	genesisPath := flag.String("genesis", "genesis_mainnet.json", "Path to genesis file")
	privKeyHex := flag.String("private-key", "", "Signer private key hex")
	address := flag.String("address", "", "Signer address")
	output := flag.String("output", "", "Output path for signed genesis")
	flag.Parse()

	if *privKeyHex == "" || *address == "" {
		panic("--private-key and --address are required")
	}
	if *output == "" {
		*output = *genesisPath
	}

	genesis := mustLoadGenesis(*genesisPath)
	if genesis.Multisig.Finalized {
		panic("genesis already finalized")
	}

	pub, priv, err := generateKeyPairFromHex(*privKeyHex)
	if err != nil {
		panic(err)
	}
	if !strings.EqualFold(hex.EncodeToString(pub), strings.TrimPrefix(*address, "0x")) {
		panic("private key does not match provided address")
	}

	signerAllowed := false
	for _, s := range genesis.Multisig.Signers {
		if strings.EqualFold(s, hex.EncodeToString(pub)) {
			signerAllowed = true
			break
		}
	}
	if !signerAllowed {
		panic("provided public key is not in the signer set")
	}

	for _, sig := range genesis.Multisig.Signatures {
		if strings.EqualFold(sig.SignerAddress, *address) {
			panic("address already signed")
		}
	}

	canonicalJSON, err := json.Marshal(genesis)
	if err != nil {
		panic(err)
	}
	sig := ed25519.Sign(priv, canonicalJSON)

	genesis.Multisig.Signatures = append(genesis.Multisig.Signatures, GenesisSignature{
		SignerAddress: *address,
		PublicKey:     hex.EncodeToString(pub),
		Signature:     hex.EncodeToString(sig),
		SignedAt:      time.Now().Unix(),
	})

	if len(genesis.Multisig.Signatures) >= genesis.Multisig.Threshold {
		genesis.Multisig.Finalized = true
		genesis.Multisig.FinalizedAt = time.Now().Unix()
	}

	writeGenesis(*output, genesis)
	fmt.Printf("Signed genesis written to %s\n", *output)
	fmt.Printf("Signatures: %d/%d | Finalized: %v\n", len(genesis.Multisig.Signatures), genesis.Multisig.Threshold, genesis.Multisig.Finalized)
}

func mainVerify() {
	genesisPath := flag.String("genesis", "genesis_mainnet.json", "Path to genesis file")
	flag.Parse()

	genesis := mustLoadGenesis(*genesisPath)
	canonicalJSON, err := json.Marshal(genesis)
	if err != nil {
		panic(err)
	}
	hash := sha256.Sum256(canonicalJSON)
	fmt.Printf("Genesis hash: %s\n", hex.EncodeToString(hash[:]))
	fmt.Printf("Finalized: %v\n", genesis.Multisig.Finalized)
	fmt.Printf("Signatures: %d/%d\n", len(genesis.Multisig.Signatures), genesis.Multisig.Threshold)
	for i, sig := range genesis.Multisig.Signatures {
		pub, err := hex.DecodeString(sig.PublicKey)
		if err != nil {
			panic(err)
		}
		sigBytes, err := hex.DecodeString(sig.Signature)
		if err != nil {
			panic(err)
		}
		valid := ed25519.Verify(pub, canonicalJSON, sigBytes)
		fmt.Printf("  %d: %s valid=%v\n", i+1, sig.SignerAddress, valid)
	}
	if !genesis.Multisig.Finalized && len(genesis.Multisig.Signatures) >= genesis.Multisig.Threshold {
		fmt.Println("Threshold reached but genesis not marked finalized")
	} else if genesis.Multisig.Finalized {
		fmt.Println("Genesis finalized")
	}
}

func main() {
	action := flag.String("action", "generate", "Action: generate | sign | verify")
	flag.Parse()
	switch *action {
	case "generate":
		mainGenerate()
	case "sign":
		mainSign()
	case "verify":
		mainVerify()
	default:
		fmt.Printf("unknown action: %s\n", *action)
	}
}

func writeGenesis(path string, genesis GenesisFile) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		panic(err)
	}
}

func mustLoadGenesis(path string) GenesisFile {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var genesis GenesisFile
	if err := json.Unmarshal(data, &genesis); err != nil {
		panic(err)
	}
	return genesis
}

func generateKeyPairFromHex(privHex string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	privBytes, err := hex.DecodeString(strings.TrimPrefix(privHex, "0x"))
	if err != nil {
		return nil, nil, err
	}
	if len(privBytes) != ed25519.PrivateKeySize {
		return nil, nil, errors.New("invalid private key size")
	}
	pub := privBytes.Public()
	return pub.(ed25519.PublicKey), privBytes, nil
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
