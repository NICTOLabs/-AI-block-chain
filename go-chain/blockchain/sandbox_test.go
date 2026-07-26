package blockchain

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"testing"
)

func TestSandboxHackingAndReverseEngineering(t *testing.T) {
	dir := t.TempDir()
	bc := NewBlockchain(ProofOfStake, dir, "tdr-sandbox-1")
	_ = bc.SaveToDisk()

	if GlobalSecurity == nil {
		t.Fatalf("security module not initialized")
	}
	if GlobalSecurity.HardeningLevel() < 1 {
		t.Fatalf("hardening level too low")
	}

	GlobalSecurity.AntiTamperAudit()
	if GlobalSecurity.TamperDetected() != 0 {
		t.Fatalf("false tamper detection")
	}

	secret := []byte("super-secret-key")
	GlobalSecurity.ZeroizeSecret(&secret)
	if secret != nil {
		t.Fatalf("secret not zeroized")
	}

	_ = GlobalSecurity.AuditLog()
	_ = GlobalSecurity.IsSealed()
	_ = GlobalSecurity.ValidationFailures()
	GlobalSecurity.SetHardeningLevel(2)
}

func TestSandboxMiningActions(t *testing.T) {
	dir := t.TempDir()
	bc := NewBlockchain(ProofOfStake, dir, "tdr-sandbox-2")
	miner := "sandbox-miner"
	bc.AddAccount(miner, 1000000, false)

	if _, err := bc.MineBlockFor(miner); err != nil {
		t.Fatalf("first mine failed: %v", err)
	}

	stats, ok := bc.MinerStats[miner]
	if !ok || stats == nil {
		t.Fatalf("miner stats missing")
	}
	if stats.BlocksSubmitted == 0 {
		t.Fatalf("miner blocks submitted not recorded")
	}
}

func TestSandboxReward(t *testing.T) {
	dir := t.TempDir()
	bc := NewBlockchain(ProofOfStake, dir, "tdr-sandbox-3")
	miner := "sandbox-reward-miner"
	bc.AddAccount(miner, 1000000, false)

	reward := bc.MineRewardFor(miner)
	if reward != 1 {
		t.Fatalf("first reward should be 1 TDR, got %d", reward)
	}

	second := bc.MineRewardFor(miner)
	if second != 0 {
		t.Fatalf("second reward within 2 weeks should be 0, got %d", second)
	}
}

func TestSandboxAllRound(t *testing.T) {
	dir := t.TempDir()
	bc := NewBlockchain(ProofOfStake, dir, "tdr-sandbox-all")
	_ = bc.LoadGenesis(genesisPathFor(t, "tdr-sandbox-all"))

	if err := bc.RegisterValidator("validator-a", MinStake, "pubkey-a"); err != nil {
		t.Fatalf("register validator failed: %v", err)
	}
	if _, ok := bc.Validators["validator-a"]; !ok {
		t.Fatalf("validator not registered")
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keygen failed: %v", err)
	}
	vote, err := bc.SignFinalityVote(0, priv)
	if err != nil {
		t.Fatalf("sign finality failed: %v", err)
	}
	t.Logf("vote voter=%s blockhash=%s", vote.Voter, vote.BlockHash)
	if !bc.VerifyFinalitySignature(vote) {
		t.Fatalf("finality signature verification failed")
	}

	_, _ = bc.RegisterUniversalWallet("alice", "addr-alice", "Alice", "TDR")
	_, _ = bc.UniversalTransfer("addr-alice", "bob", "TDR", 10)

	if _, ok := bc.UniversalWallets["alice"]; !ok {
		t.Fatalf("universal wallet missing")
	}

	pub := ed25519.PublicKey(priv)
	GlobalSecurity.RecordValidatorSign("validator-a", pub)
}

func genesisPathFor(t *testing.T, chainID string) string {
	t.Helper()
	path := t.TempDir() + "/genesis.json"
	content := `{"chain_id":"` + chainID + `","max_supply":10000000000,"allocations":[{"address":"alloc1","amount":10000000000}],"validators":[{"address":"validator-a","stake":10000000000,"public_key":"pubkey-a"}]}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write genesis: %v", err)
	}
	return path
}
