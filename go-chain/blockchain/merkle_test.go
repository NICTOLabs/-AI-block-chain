package blockchain

import (
	"testing"

	"crypto/sha256"
	"encoding/hex"
)

func TestBuildMerkleTree(t *testing.T) {
	items := [][]byte{[]byte("a"), []byte("b"), []byte("c")}
	root, err := BuildMerkleTree(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root == nil || root.Hash == "" {
		t.Fatal("expected non-empty root")
	}
}

func TestBuildMerkleTreeSingleItem(t *testing.T) {
	items := [][]byte{[]byte("only")}
	root, err := BuildMerkleTree(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root == nil || root.Hash == "" {
		t.Fatal("expected non-empty root for single item")
	}
}

func TestBuildMerkleTreeEmpty(t *testing.T) {
	_, err := BuildMerkleTree([][]byte{})
	if err == nil {
		t.Fatal("expected error for empty items")
	}
}

func TestGenerateAndVerifyMerkleProof(t *testing.T) {
	items := [][]byte{[]byte("a"), []byte("b"), []byte("c"), []byte("d")}
	root, err := BuildMerkleTree(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, target := range items {
		proof, ok := GenerateProof(root, target)
		if !ok {
			t.Fatalf("expected proof for target")
		}
		if !VerifyMerkleProof(root.Hash, target, proof) {
			t.Fatalf("valid proof should verify")
		}
	}
}

func TestVerifyMerkleProofRejectsWrongTarget(t *testing.T) {
	items := [][]byte{[]byte("a"), []byte("b")}
	root, err := BuildMerkleTree(items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	proof, ok := GenerateProof(root, []byte("a"))
	if !ok {
		t.Fatalf("expected proof")
	}
	if VerifyMerkleProof(root.Hash, []byte("wrong"), proof) {
		t.Fatal("wrong target should not verify")
	}
}

func TestCalculateMerkleRootFromHashes(t *testing.T) {
	items := []string{"abc", "def"}
	var hashes []string
	for _, item := range items {
		hashes = append(hashes, hex.EncodeToString(sha256.New().Sum([]byte(item))))
	}
	root := CalculateMerkleRootFromHashes(hashes)
	if root == "" {
		t.Fatal("expected non-empty merkle root")
	}
}

func TestCalculateMerkleRootFromHashesEmpty(t *testing.T) {
	if CalculateMerkleRootFromHashes([]string{}) != "" {
		t.Fatal("empty hashes should yield empty root")
	}
}

func TestCalculateMerkleRootDeterministic(t *testing.T) {
	items := []string{"tx-1", "tx-2", "tx-3"}
	var hashes []string
	for _, item := range items {
		hashes = append(hashes, hex.EncodeToString(sha256.New().Sum([]byte(item))))
	}
	first := CalculateMerkleRootDeterministic(hashes)
	for i := 0; i < 10; i++ {
		if CalculateMerkleRootDeterministic(hashes) != first {
			t.Fatal("merkle root should be deterministic")
		}
	}
}
