package main

import (
	"encoding/json"
	"fmt"
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
		return fmt.Errorf("supply values must be positive")
	}
	if cfg.InitialCirculating > cfg.MaxSupply {
		return fmt.Errorf("initial circulating exceeds max supply")
	}
	if cfg.GenesisMintCap > cfg.MaxSupply {
		return fmt.Errorf("genesis mint cap exceeds max supply")
	}
	sum := cfg.ValidatorRewardShare + cfg.CommunityRewardShare + cfg.TreasuryRewardShare
	if sum <= 0 || sum != 1 {
		return fmt.Errorf("reward shares must sum to 1.0")
	}
	return nil
}

func (cfg TokenomicsConfig) GenesisMintCapReached(currentSupply uint64) bool {
	return currentSupply >= cfg.GenesisMintCap
}

func (cfg TokenomicsConfig) AnnualInflationRate(year int) float64 {
	if year < cfg.InflationStartYear {
		return 0
	}
	decay := 1.0 - (float64(year-cfg.InflationStartYear) * cfg.InflationDecayPerYear)
	if decay < 0 {
		return 0
	}
	return decay
}

func (cfg TokenomicsConfig) ValidatorReward(amount uint64, year int) uint64 {
	inflationFactor := cfg.AnnualInflationRate(year)
	reward := float64(amount) * (cfg.StakingRewardAPY * inflationFactor)
	return uint64(reward)
}

func (cfg TokenomicsConfig) ApplyFeeBurn(amount uint64) uint64 {
	return uint64(float64(amount) * cfg.BurnRate)
}

func (cfg TokenomicsConfig) NextEpochTimestamp(now time.Time) time.Time {
	return now.AddDate(1, 0, 0)
}
