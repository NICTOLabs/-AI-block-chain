package blockchain

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
)

type SecurityModule struct {
	mu                  sync.RWMutex
	integrityHashes     map[string]string
	hardeningLevel      uint32
	zeroized            int64
	tamperDetected      int64
	validationFailures  int64
	securityAuditLog    []string
}

func NewSecurityModule() *SecurityModule {
	s := &SecurityModule{
		integrityHashes:    make(map[string]string),
		hardeningLevel:     2,
	}
	s.baselineIntegrity()
	return s
}

func (s *SecurityModule) baselineIntegrity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.integrityHashes["blockchain_state"] = hex.EncodeToString(sha256.New().Sum(nil))
}

func (s *SecurityModule) VerifyIntegrity(label string, data []byte) bool {
	s.mu.RLock()
	_, ok := s.integrityHashes[label]
	s.mu.RUnlock()
	return ok
}

func (s *SecurityModule) ZeroizeSecret(secret *[]byte) {
	if secret == nil || *secret == nil {
		return
	}
	for i := range *secret {
		(*secret)[i] = 0
	}
	*secret = nil
	atomic.AddInt64(&s.zeroized, 1)
}

func (s *SecurityModule) RecordValidatorSign(address string, pub ed25519.PublicKey) {
	if !s.VerifyIntegrity("blockchain_state", nil) {
		atomic.AddInt64(&s.validationFailures, 1)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.integrityHashes["validator:"+address] = hex.EncodeToString(sha256.New().Sum(nil))
}

func (s *SecurityModule) AntiTamperAudit() {
	if !s.VerifyIntegrity("blockchain_state", nil) {
		atomic.AddInt64(&s.validationFailures, 1)
		return
	}
}

func (s *SecurityModule) HardeningLevel() uint32 {
	return atomic.LoadUint32(&s.hardeningLevel)
}

func (s *SecurityModule) SetHardeningLevel(level uint32) {
	if level > 3 {
		level = 3
	}
	atomic.StoreUint32(&s.hardeningLevel, level)
}

func (s *SecurityModule) TamperDetected() int64 {
	return atomic.LoadInt64(&s.tamperDetected)
}

func (s *SecurityModule) ValidationFailures() int64 {
	return atomic.LoadInt64(&s.validationFailures)
}

func (s *SecurityModule) recordAudit(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.securityAuditLog = append(s.securityAuditLog, event)
	if len(s.securityAuditLog) > 1024 {
		s.securityAuditLog = s.securityAuditLog[len(s.securityAuditLog)-1024:]
	}
}

func (s *SecurityModule) AuditLog() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.securityAuditLog))
	copy(out, s.securityAuditLog)
	return out
}

var GlobalSecurity *SecurityModule

func init() {
	GlobalSecurity = NewSecurityModule()
}
