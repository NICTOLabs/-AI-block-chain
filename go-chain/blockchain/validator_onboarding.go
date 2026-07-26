package blockchain

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type ValidatorOnboarding struct {
	mu sync.RWMutex
	pending map[string]ed25519.PrivateKey
}

func NewValidatorOnboarding() *ValidatorOnboarding {
	return &ValidatorOnboarding{pending: make(map[string]ed25519.PrivateKey)}
}

func (o *ValidatorOnboarding) GenerateKeyFor(address string) (string, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	hexPub := hex.EncodeToString(pub)
	o.mu.Lock()
	o.pending[hexPub] = priv
	o.mu.Unlock()
	return hexPub, nil
}

func (o *ValidatorOnboarding) ClaimPrivateKey(pubHex string) (ed25519.PrivateKey, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	priv, ok := o.pending[pubHex]
	if ok {
		delete(o.pending, pubHex)
	}
	return priv, ok
}

func (o *ValidatorOnboarding) RegisterValidatorWithKey(bc *Blockchain, address string, stake uint64, pubHex string) error {
	priv, ok := o.ClaimPrivateKey(pubHex)
	if !ok {
		return fmt.Errorf("unknown validator public key")
	}
	if err := bc.RegisterValidator(address, stake, pubHex); err != nil {
		return err
	}
	bc.Validators[address] = Validator{
		Address:     address,
		Stake:       stake,
		Active:      true,
		JoinedAt:    time.Now().Unix(),
		Performance: 100,
		PublicKey:   pubHex,
	}
	_ = bc.SaveToDisk()
	secret := []byte(priv)
	GlobalSecurity.ZeroizeSecret(&secret)
	return nil
}
