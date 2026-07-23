package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"time"
)

type TokenomicsConfig struct {
	Network                string  `json:"network"`
	NativeToken            string  `json:"native_token"`
	MaxSupply              uint64  `json:"max_supply"`
	InitialCirculating     uint64  `json:"initial_circulating"`
	GenesisMintCap         uint64  `json:"genesis_mint_cap"`
	StakingRewardAPY       float64 `json:"staking_reward_apy"`
	BaseGasFee             uint64  `json:"base_gas_fee"`
	BurnRate               float64 `json:"burn_rate"`
	InflationStartYear     int     `json:"inflation_start_year"`
	InflationDecayPerYear  float64 `json:"inflation_decay_per_year"`
	ValidatorRewardShare   float64 `json:"validator_reward_share"`
	CommunityRewardShare   float64 `json:"community_reward_share"`
	TreasuryRewardShare    float64 `json:"treasury_reward_share"`
	MinInflationRate       float64 `json:"min_inflation_rate"`
	ValidatorPerformanceDecay float64 `json:"validator_performance_decay"`
	BlockRewardBase        uint64  `json:"block_reward_base"`
}

func LoadTokenomicsConfig(path string) (TokenomicsConfig, error) {
	var cfg TokenomicsConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (cfg TokenomicsConfig) Validate() error {
	if cfg.MaxSupply == 0 || cfg.InitialCirculating == 0 {
		return errors.New("supply values must be positive")
	}
	if cfg.InitialCirculating > cfg.MaxSupply {
		return errors.New("initial circulating exceeds max supply")
	}
	if cfg.GenesisMintCap > cfg.MaxSupply {
		return errors.New("genesis mint cap exceeds max supply")
	}
	if cfg.InitialCirculating > cfg.GenesisMintCap {
		return errors.New("initial circulating cannot exceed genesis mint cap")
	}
	sum := cfg.ValidatorRewardShare + cfg.CommunityRewardShare + cfg.TreasuryRewardShare
	if math.Abs(sum-1.0) > 1e-9 {
		return fmt.Errorf("reward shares must sum to 1.0, got %.6f", sum)
	}
	if cfg.StakingRewardAPY < 0 || cfg.StakingRewardAPY > 1 {
		return errors.New("staking reward APY must be between 0 and 1")
	}
	if cfg.BurnRate < 0 || cfg.BurnRate > 1 {
		return errors.New("burn rate must be between 0 and 1")
	}
	if cfg.MinInflationRate < 0 {
		return errors.New("min inflation rate cannot be negative")
	}
	if cfg.InflationDecayPerYear < 0 || cfg.InflationDecayPerYear > 1 {
		return errors.New("inflation decay per year must be between 0 and 1")
	}
	return nil
}

func (cfg TokenomicsConfig) GenesisMintCapReached(currentSupply uint64) bool {
	return currentSupply >= cfg.GenesisMintCap
}

func (cfg TokenomicsConfig) CanMint(amount uint64, currentSupply uint64) bool {
	if cfg.GenesisMintCapReached(currentSupply) {
		return false
	}
	return currentSupply+amount <= cfg.GenesisMintCap
}

func (cfg TokenomicsConfig) AnnualInflationRate(year int) float64 {
	if year < cfg.InflationStartYear {
		return 0
	}
	decay := 1.0 - (float64(year-cfg.InflationStartYear) * cfg.InflationDecayPerYear)
	if decay < 0 {
		return 0
	}
	return math.Max(decay, cfg.MinInflationRate)
}

func (cfg TokenomicsConfig) CalculateInflationDecaySchedule(years int) map[int]float64 {
	schedule := make(map[int]float64)
	for y := 0; y <= years; y++ {
		schedule[y] = cfg.AnnualInflationRate(cfg.InflationStartYear + y)
	}
	return schedule
}

func (cfg TokenomicsConfig) CalculateTokenSupplyAtYear(year int) uint64 {
	if year < cfg.InflationStartYear {
		return cfg.GenesisMintCap
	}
	elapsed := year - cfg.InflationStartYear
	decay := 1.0 - float64(elapsed)*cfg.InflationDecayPerYear
	if decay < 0 {
		decay = 0
	}
	inflationFactor := math.Max(decay, cfg.MinInflationRate)
	supply := float64(cfg.GenesisMintCap) * (1.0 + inflationFactor)
	return uint64(supply)
}

func (cfg TokenomicsConfig) CalculateDisinflationProjection(years int) []TokenSupplyProjection {
	projections := make([]TokenSupplyProjection, 0, years+1)
	for y := 0; y <= years; y++ {
		supply := cfg.CalculateTokenSupplyAtYear(cfg.InflationStartYear + y)
		projections = append(projections, TokenSupplyProjection{
			Year:            cfg.InflationStartYear + y,
			ProjectedSupply: supply,
			InflationRate:   cfg.AnnualInflationRate(cfg.InflationStartYear + y),
			MaxSupply:       cfg.MaxSupply,
		})
	}
	return projections
}

func (cfg TokenomicsConfig) ValidatorBlockReward(blockHeight uint64, validatorCount int, validatorPerformance float64) uint64 {
	year := int(blockHeight / 100000)
	inflationFactor := cfg.AnnualInflationRate(year)

	reward := float64(cfg.BlockRewardBase) * inflationFactor

	if validatorPerformance < 0.5 {
		reward *= cfg.ValidatorPerformanceDecay
	}

	reward = reward / float64(validatorCount)

	return uint64(math.Floor(reward))
}

func (cfg TokenomicsConfig) DistributeBlockReward(totalReward uint64) BlockRewardDistribution {
	validatorShare := uint64(float64(totalReward) * cfg.ValidatorRewardShare)
	communityShare := uint64(float64(totalReward) * cfg.CommunityRewardShare)
	treasuryShare := totalReward - validatorShare - communityShare

	return BlockRewardDistribution{
		TotalReward:          totalReward,
		ValidatorShare:       validatorShare,
		CommunityShare:       communityShare,
		TreasuryShare:        treasuryShare,
		ValidatorPercentage:  cfg.ValidatorRewardShare,
		CommunityPercentage:  cfg.CommunityRewardShare,
		TreasuryPercentage:   cfg.TreasuryRewardShare,
	}
}

func (cfg TokenomicsConfig) ApplyStakingRewards(block *Blockchain, blockHeight uint64) {
	block.mu.Lock()
	defer block.mu.Unlock()

	validators := make([]Validator, 0)
	for _, v := range block.Validators {
		if v.Active {
			validators = append(validators, v)
		}
	}
	totalStaked := uint64(0)
	for _, v := range validators {
		totalStaked += v.Stake
	}

	if totalStaked == 0 {
		return
	}

	rewardPool := uint64(float64(cfg.BlockRewardBase) * float64(cfg.ValidatorRewardShare) * cfg.AnnualInflationRate(int(blockHeight/100000)))

	for _, v := range validators {
		if !v.Active {
			continue
		}
		percent := float64(v.Stake) / float64(totalStaked)
		reward := uint64(float64(rewardPool) * percent)
		account := block.Ledger[v.Address]
		if account != nil {
			account.Balance += reward
			block.TokenSupply += reward
		}
	}
}

func (cfg TokenomicsConfig) ApplyTransactionFeeBurn(amount uint64) uint64 {
	return uint64(float64(amount) * cfg.BurnRate)
}

func FormatTokenomics(amount uint64) string {
	return FormatAmount(amount)
}

func (cfg TokenomicsConfig) CalculateEffectiveBurnRate(currentSupply uint64) float64 {
	if currentSupply > cfg.MaxSupply/2 {
		return cfg.BurnRate * 1.5
	}
	return cfg.BurnRate
}

func (cfg TokenomicsConfig) CalculateTreasuryAllocation(revenue uint64, year int) uint64 {
	inflationFactor := cfg.AnnualInflationRate(year)
	baseAllocation := uint64(float64(revenue) * cfg.TreasuryRewardShare)
	inflationBoost := uint64(float64(baseAllocation) * inflationFactor * 0.5)
	return baseAllocation + inflationBoost
}

func (cfg TokenomicsConfig) NextEpochTimestamp(now time.Time) time.Time {
	return now.AddDate(1, 0, 0)
}

func (cfg TokenomicsConfig) CalculateAPYForStake(amount uint64, years int) float64 {
	baseAPY := cfg.StakingRewardAPY
	rate := baseAPY
	for y := 0; y < years; y++ {
		rate += cfg.StakingRewardAPY * cfg.AnnualInflationRate(cfg.InflationStartYear+y) * 0.1
	}
	return math.Min(rate, cfg.StakingRewardAPY*3)
}

func (cfg TokenomicsConfig) ToJSON() ([]byte, error) {
	return json.MarshalIndent(cfg, "", "  ")
}

type TokenSupplyProjection struct {
	Year            int     `json:"year"`
	ProjectedSupply uint64  `json:"projected_supply"`
	InflationRate   float64 `json:"inflation_rate"`
	MaxSupply       uint64  `json:"max_supply"`
}

type BlockRewardDistribution struct {
	TotalReward          uint64  `json:"total_reward"`
	ValidatorShare       uint64  `json:"validator_share"`
	CommunityShare       uint64  `json:"community_share"`
	TreasuryShare        uint64  `json:"treasury_share"`
	ValidatorPercentage  float64 `json:"validator_percentage"`
	CommunityPercentage  float64 `json:"community_percentage"`
	TreasuryPercentage   float64 `json:"treasury_percentage"`
}
