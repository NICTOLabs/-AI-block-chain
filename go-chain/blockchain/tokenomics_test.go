package blockchain

import (
	"testing"
	"time"
)

func TestEstimateFeeUsesCongestionAndComplexity(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	transferFee := bc.estimateFee(Transaction{TxType: Transfer}, 0)
	modelFee := bc.estimateFee(Transaction{TxType: RegisterModel, Payload: "model"}, 0)
	if modelFee <= transferFee {
		t.Fatalf("expected model fee to be higher than transfer fee, got %d and %d", transferFee, modelFee)
	}

	congestionFee := bc.estimateFee(Transaction{TxType: Transfer}, 20)
	if congestionFee <= transferFee {
		t.Fatalf("expected congestion to increase the fee, got %d and %d", transferFee, congestionFee)
	}
}

func TestSlashReducesStakeAndBalance(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	bc.AddAccount("alice", 1000, false)
	bc.Ledger["alice"].Staked = 100

	bc.Slash("alice", 40)

	if bc.Ledger["alice"].Staked != 60 {
		t.Fatalf("expected staked amount to drop to 60, got %d", bc.Ledger["alice"].Staked)
	}
	if bc.Ledger["alice"].Balance != 996 {
		t.Fatalf("expected balance to drop under penalty rule, got %d", bc.Ledger["alice"].Balance)
	}
}

func TestDistributeRewardsAndBurn(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	bc.AddAccount("alice", 1000, false)
	bc.Ledger["alice"].Staked = 100

	bc.DistributeRewards()
	if bc.Ledger["alice"].Balance <= 1000 {
		t.Fatalf("expected staking rewards to increase balance, got %d", bc.Ledger["alice"].Balance)
	}

	initialSupply := bc.TokenSupply
	bc.Burn(5)
	if bc.TokenSupply >= initialSupply {
		t.Fatalf("expected token supply to decrease when burned, got %d", bc.TokenSupply)
	}
}

func TestFeeBurnSplitsPermanentBurnAndCommunityFund(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	bc.AddAccount("miner", 1000, false)
	bc.Ledger[CommunityFundAddress] = &Account{Address: CommunityFundAddress, Balance: 0, Staked: 0, IsAgent: false}

	wallet := NewWallet()
	from := wallet.Address()
	bc.AddAccount(from, 1_000_000_000, false)
	to := "0000000000000000000000000000000000000000000000000000000000000002"
	bc.AddAccount(to, 0, false)

	estFee := bc.estimateFee(Transaction{TxType: Transfer}, 0)
	if estFee < BaseFee+FeeMultiplier+10 {
		estFee = BaseFee + FeeMultiplier + 10
	}
	txFee := estFee + 10000
	totalBurn := txFee * BurnRatePercent / 100
	permanentBurn := totalBurn * 70 / 100
	communityFund := totalBurn - permanentBurn

	if permanentBurn+communityFund != totalBurn {
		t.Fatalf("expected burn split to equal total burn, got %d + %d != %d", permanentBurn, communityFund, totalBurn)
	}
	if permanentBurn == 0 || communityFund == 0 {
		t.Fatal("expected both permanent burn and community fund to be non-zero")
	}

	tx := wallet.Sign(Transaction{
		From:    from,
		To:      to,
		Amount:  10,
		Fee:     txFee,
		Nonce:   1,
		TxType:  Transfer,
		ChainID: bc.ChainID,
		Timestamp: time.Now().Unix(),
	})
	bc.EnqueueTransaction(tx)
	_, _ = bc.MineBlockFor("miner")

	if bc.Ledger[CommunityFundAddress].Balance != communityFund {
		t.Fatalf("expected community fund balance %d, got %d", communityFund, bc.Ledger[CommunityFundAddress].Balance)
	}
}

func TestMintingPauseAndResume(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	bc.AddAccount("miner", 1000, false)
	initialSupply := bc.TokenSupply
	beforeHeight := len(bc.Chain)

	bc.PauseMinting(2 * time.Minute)
	if !bc.IsMintingPaused() {
		t.Fatal("expected minting to be paused")
	}
	if _, err := bc.MineBlockFor("miner"); err == nil {
		t.Fatal("expected mining to fail while minting is paused")
	}
	if bc.TokenSupply != initialSupply {
		t.Fatalf("expected token supply to stay at %d while paused, got %d", initialSupply, bc.TokenSupply)
	}

	bc.ResumeMinting()
	if bc.IsMintingPaused() {
		t.Fatal("expected minting to be resumed")
	}
	if len(bc.Chain) != beforeHeight {
		t.Fatalf("expected chain height to stay at %d while paused, got %d", beforeHeight, len(bc.Chain))
	}
}

func TestCreateEscrowLocksFunds(t *testing.T) {
	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-testnet-1")
	bc.AddAccount("alice", 1000, false)
	bc.AddAccount("bob", 0, false)

	escrow, err := bc.CreateEscrow("alice", "bob", 200, "service-1")
	if err != nil {
		t.Fatalf("expected escrow creation to succeed: %v", err)
	}
	if bc.Ledger["alice"].Balance != 800 {
		t.Fatalf("expected escrow to lock 200 tokens, balance is %d", bc.Ledger["alice"].Balance)
	}
	if escrow.Status != "active" {
		t.Fatalf("expected escrow status to be active, got %s", escrow.Status)
	}
}
