package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

type MerkleNode struct {
	Hash  string      `json:"hash"`
	Left  *MerkleNode `json:"left,omitempty"`
	Right *MerkleNode `json:"right,omitempty"`
}

func BuildMerkleTree(items [][]byte) (*MerkleNode, error) {
	if len(items) == 0 {
		return nil, errors.New("empty items")
	}
	var nodes []*MerkleNode
	for _, item := range items {
		sum := sha256.Sum256(item)
		nodes = append(nodes, &MerkleNode{Hash: hex.EncodeToString(sum[:])})
	}
	for len(nodes) > 1 {
		var next []*MerkleNode
		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				left := nodes[i]
				right := nodes[i+1]
				combined := left.Hash + right.Hash
				sum := sha256.Sum256([]byte(combined))
				next = append(next, &MerkleNode{Hash: hex.EncodeToString(sum[:]), Left: left, Right: right})
			} else {
				next = append(next, nodes[i])
			}
		}
		nodes = next
	}
	return nodes[0], nil
}

func GenerateProof(root *MerkleNode, target []byte) ([][]byte, bool) {
	if root == nil {
		return nil, false
	}
	targetHash := sha256.Sum256(target)
	targetHex := hex.EncodeToString(targetHash[:])
	var proof [][]byte
	var findProof func(*MerkleNode) bool
	findProof = func(node *MerkleNode) bool {
		if node == nil {
			return false
		}
		if node.Hash == targetHex {
			return true
		}
		if findProof(node.Left) {
			if node.Right != nil {
				proof = append(proof, append([]byte{0x01}, []byte(node.Right.Hash)...))
			}
			return true
		}
		if findProof(node.Right) {
			if node.Left != nil {
				proof = append(proof, append([]byte{0x00}, []byte(node.Left.Hash)...))
			}
			return true
		}
		return false
	}
	if !findProof(root) {
		return nil, false
	}
	return proof, true
}

func VerifyMerkleProof(rootHash string, target []byte, proof [][]byte) bool {
	current := sha256.Sum256(target)
	currentHex := hex.EncodeToString(current[:])
	if currentHex == rootHash && len(proof) == 0 {
		return true
	}
	for _, siblingWithDir := range proof {
		if len(siblingWithDir) == 0 {
			return false
		}
		isRight := siblingWithDir[0] == 0x01
		siblingHash := string(siblingWithDir[1:])
		var combined string
		if isRight {
			combined = currentHex + siblingHash
		} else {
			combined = siblingHash + currentHex
		}
		sum := sha256.Sum256([]byte(combined))
		currentHex = hex.EncodeToString(sum[:])
	}
	return currentHex == rootHash
}

func CalculateMerkleRootFromHashes(txHashes []string) string {
	if len(txHashes) == 0 {
		return ""
	}
	var items [][]byte
	for _, h := range txHashes {
		decoded, err := hex.DecodeString(h)
		if err != nil {
			return ""
		}
		items = append(items, decoded)
	}
	node, err := BuildMerkleTree(items)
	if err != nil {
		return ""
	}
	return node.Hash
}

func CalculateMerkleRootDeterministic(items []string) string {
	return CalculateMerkleRootFromHashes(items)
}
