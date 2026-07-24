package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
)

type TestWallet struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

func NewTestWallet() *TestWallet {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		panic(err)
	}
	return &TestWallet{PublicKey: pub, PrivateKey: priv}
}

func (w *TestWallet) Address() string {
	hash := sha256.Sum256(w.PublicKey)
	return hex.EncodeToString(hash[:])
}

func (w *TestWallet) SignTx(tx Transaction) Transaction {
	tx.FromPubKey = hex.EncodeToString(w.PublicKey)
	tx.Timestamp = 1000000
	clone := tx
	clone.Signature = ""
	data, _ := json.Marshal(clone)
	tx.Signature = hex.EncodeToString(ed25519.Sign(w.PrivateKey, data))
	return tx
}

func TestFullMiningAndTransactions(t *testing.T) {
	fmt.Println("=== TENDER Full Mining + Transaction Integration Test ===")
	fmt.Println()

	bc := NewBlockchain(ProofOfStake, t.TempDir(), "tdr-test-1", "")
	fmt.Printf("Blockchain: chain_id=%s height=%d supply=%d TDR\n\n",
		bc.ChainID, len(bc.Chain), bc.TokenSupply)

	alice := NewTestWallet()
	bob := NewTestWallet()
	carol := NewTestWallet()

	bc.AddAccount(alice.Address(), 0, false)
	bc.AddAccount(bob.Address(), 0, false)
	bc.AddAccount(carol.Address(), 0, false)
	fmt.Printf("Wallets: Alice=%s Bob=%s Carol=%s\n\n",
		alice.Address()[:12], bob.Address()[:12], carol.Address()[:12])

	// Mine 10 blocks to fund wallets (10 TDR each)
	fmt.Println("--- Mining 10 blocks (10 TDR reward each) ---")
	for i := 0; i < 10; i++ {
		var miner *TestWallet
		switch i % 3 {
		case 0:
			miner = alice
		case 1:
			miner = bob
		case 2:
			miner = carol
		}
		block, err := bc.MineBlockFor(miner.Address())
		if err != nil {
			t.Fatalf("mine block %d failed: %v", i, err)
		}
		fmt.Printf("  Block %d: miner=%s txs=%d hash=%s...\n",
			block.Index, block.Author[:8], len(block.Transactions), block.BlockHash[:16])
	}
	fmt.Printf("\nHeight: %d | Supply: %d TDR\n\n", len(bc.Chain), bc.TokenSupply)

	fmt.Println("Balances after mining:")
	bal(t, bc, "Alice", alice.Address())
	bal(t, bc, "Bob", bob.Address())
	bal(t, bc, "Carol", carol.Address())
	fmt.Println()

	// Submit signed transactions (amounts within balances)
	fmt.Println("--- Submitting signed transactions ---")
	tx1 := alice.SignTx(Transaction{
		From: alice.Address(), To: bob.Address(),
		Amount: 3, Fee: 15, TxType: Transfer, ChainID: bc.ChainID,
	})
	tx1.ID = "tx-alice-bob-1"
	bc.EnqueueTransaction(tx1)
	fmt.Printf("  Alice -> Bob:   3 TDR  (fee=15)\n")

	tx2 := bob.SignTx(Transaction{
		From: bob.Address(), To: carol.Address(),
		Amount: 2, Fee: 20, TxType: Transfer, ChainID: bc.ChainID,
	})
	tx2.ID = "tx-bob-carol-1"
	bc.EnqueueTransaction(tx2)
	fmt.Printf("  Bob -> Carol:   2 TDR  (fee=20)\n")

	tx3 := carol.SignTx(Transaction{
		From: carol.Address(), To: alice.Address(),
		Amount: 1, Fee: 25, TxType: Transfer, ChainID: bc.ChainID,
	})
	tx3.ID = "tx-carol-alice-1"
	bc.EnqueueTransaction(tx3)
	fmt.Printf("  Carol -> Alice: 1 TDR  (fee=25)\n")
	fmt.Printf("\nMempool: %d pending\n\n", len(bc.Pending))

	// Mine a block to include transactions
	fmt.Println("--- Mining block with transactions ---")
	block, err := bc.MineBlockFor(carol.Address())
	if err != nil {
		t.Fatalf("mine block failed: %v", err)
	}
	fmt.Printf("  Block %d: included %d transactions\n", block.Index, len(block.Transactions))
	for i, tx := range block.Transactions {
		fmt.Printf("    [%d] %s -> %s : %d TDR (fee=%d)\n",
			i, tx.From[:8], tx.To[:8], tx.Amount, tx.Fee)
	}
	fmt.Println()

	fmt.Println("Final balances:")
	bal(t, bc, "Alice", alice.Address())
	bal(t, bc, "Bob", bob.Address())
	bal(t, bc, "Carol", carol.Address())
	fmt.Println()

	// Verify supply accounting
	fmt.Println("--- Supply Accounting ---")
	fmt.Printf("  Initial supply:    %d TDR\n", InitialSupply)
	fmt.Printf("  Blocks mined:      %d (reward=%d each)\n", len(bc.Chain)-1, BlockRewardBase)
	rewards := uint64(len(bc.Chain)-1) * BlockRewardBase
	fmt.Printf("  Mining rewards:    %d TDR\n", rewards)
	fmt.Printf("  Expected supply:   %d TDR\n", InitialSupply+rewards)
	fmt.Printf("  Actual supply:     %d TDR\n", bc.TokenSupply)
	fmt.Printf("  Supply match:      %v\n", bc.TokenSupply == InitialSupply+rewards)
	fmt.Println()

	fmt.Println("--- PoW Hash Verification ---")
	for i, b := range bc.Chain {
		pow := 0
		for j := 0; j < len(b.BlockHash) && b.BlockHash[j] == '0'; j++ {
			pow++
		}
		fmt.Printf("  Block %d: pow_zeros=%d hash=%s...\n", i, pow, b.BlockHash[:16])
	}

	fmt.Println("\n=== Test Complete ===")
}

func bal(t *testing.T, bc *Blockchain, name, addr string) {
	t.Helper()
	acct, ok := bc.Ledger[addr]
	if ok {
		fmt.Printf("  %s: %d TDR\n", name, acct.Balance)
	} else {
		fmt.Printf("  %s: NOT IN LEDGER\n", name)
	}
}
