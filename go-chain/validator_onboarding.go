package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ValidatorIdentity struct {
	Name           string `json:"name"`
	PublicKey      string `json:"public_key"`
	PrivateKeyPath string `json:"private_key_path"`
	Country        string `json:"country"`
	Region         string `json:"region"`
	Weight         uint64 `json:"weight"`
}

type ValidatorTelemetryPayload struct {
	ValidatorID string  `json:"validator_id"`
	PublicKey   string  `json:"public_key"`
	Country     string  `json:"country"`
	Region      string  `json:"region"`
	Weight      uint64  `json:"weight"`
	Seeds       []string `json:"seeds"`
}

func GenerateValidatorIdentity(name, country, region string, weight uint64, seedDir string) (ValidatorIdentity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return ValidatorIdentity{}, err
	}
	if err := os.MkdirAll(seedDir, 0o755); err != nil {
		return ValidatorIdentity{}, err
	}
	keyPath := filepath.Join(seedDir, fmt.Sprintf("%s.key", name))
	if err := os.WriteFile(keyPath, priv, 0o600); err != nil {
		return ValidatorIdentity{}, err
	}
	return ValidatorIdentity{
		Name:           name,
		PublicKey:      hex.EncodeToString(pub),
		PrivateKeyPath: keyPath,
		Country:        country,
		Region:         region,
		Weight:         weight,
	}, nil
}

func (v ValidatorIdentity) TelemetryPayload(seeds []string) ValidatorTelemetryPayload {
	return ValidatorTelemetryPayload{
		ValidatorID: v.Name,
		PublicKey:   v.PublicKey,
		Country:     v.Country,
		Region:      v.Region,
		Weight:      v.Weight,
		Seeds:       seeds,
	}
}

func WriteValidatorPayload(path string, payload ValidatorTelemetryPayload) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
