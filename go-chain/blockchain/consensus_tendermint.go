package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"
)

type EvidenceType int

const (
	EvidenceDoubleSign EvidenceType = iota
	EvidenceUnavailability
)

type TendermintEngine struct {
	mu             sync.Mutex
	height         uint64
	round          uint64
	step           int
	committedRound uint64
	lockedRound    uint64
	lockedBlock    string
	validators     []string
	proposerIdx    int
	evidence       []Evidence
	lastBlockTime  int64
}

func NewTendermintEngine(validators []string) *TendermintEngine {
	te := &TendermintEngine{
		height:        1,
		validators:    validators,
		lastBlockTime: time.Now().Unix(),
	}
	te.selectProposerLocked()
	return te
}

func (te *TendermintEngine) Height() uint64 {
	te.mu.Lock()
	defer te.mu.Unlock()
	return te.height
}

func (te *TendermintEngine) CurrentRound() uint64 {
	te.mu.Lock()
	defer te.mu.Unlock()
	return te.round
}

func (te *TendermintEngine) Proposer() string {
	te.mu.Lock()
	defer te.mu.Unlock()
	return te.proposerLocked()
}

func (te *TendermintEngine) proposerLocked() string {
	if len(te.validators) == 0 {
		return ""
	}
	return te.validators[te.proposerIdx%len(te.validators)]
}

func (te *TendermintEngine) selectProposerLocked() {
	if len(te.validators) == 0 {
		return
	}
	te.proposerIdx = int(te.height % uint64(len(te.validators)))
}

func (te *TendermintEngine) stepString(step int) string {
	switch step {
	case 0:
		return "propose"
	case 1:
		return "prevote"
	case 2:
		return "precommit"
	default:
		return "commit"
	}
}

func (te *TendermintEngine) AddVote(v Vote) error {
	te.mu.Lock()
	defer te.mu.Unlock()
	if v.Height != te.height {
		return fmt.Errorf("vote height mismatch: %d != %d", v.Height, te.height)
	}
	if len(te.validators) == 0 {
		return fmt.Errorf("no validators registered")
	}
	switch v.Type {
	case "prevote":
		if te.step == 1 {
			te.step = 2
		}
	case "precommit":
		if te.step == 2 {
			te.step = 3
			te.committedRound = uint64(v.Round)
			te.lastBlockTime = time.Now().Unix()
		}
	}
	return nil
}

func (te *TendermintEngine) AdvanceRound() {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.round++
	te.step = 0
	te.selectProposerLocked()
}

func (te *TendermintEngine) AdvanceHeight() {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.height++
	te.round = 0
	te.step = 0
	te.committedRound = 0
	te.lockedRound = 0
	te.lockedBlock = ""
	te.selectProposerLocked()
}

func (te *TendermintEngine) AddEvidence(ev Evidence) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.evidence = append(te.evidence, ev)
}

func (te *TendermintEngine) EvidenceList() []Evidence {
	te.mu.Lock()
	defer te.mu.Unlock()
	out := make([]Evidence, len(te.evidence))
	copy(out, te.evidence)
	return out
}

func (te *TendermintEngine) ProcessProposal(block Block, proposer string) (Proposal, error) {
	te.mu.Lock()
	defer te.mu.Unlock()
	if te.step != 0 {
		return Proposal{}, fmt.Errorf("not in proposal step: %s", te.stepString(te.step))
	}
	payload := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s:%s", te.height, te.round, block.BlockHash, proposer)))
	te.step = 1
	return Proposal{BlockHash: block.BlockHash, Height: te.height, Round: int64(te.round), Block: &block, PolRound: int64(te.round), Timestamp: time.Now().Unix(), Signature: payload[:]}, nil
}

func (te *TendermintEngine) Prevote(blockHash string) (Vote, error) {
	te.mu.Lock()
	defer te.mu.Unlock()
	if te.step != 1 {
		return Vote{}, fmt.Errorf("not in prevote step: %s", te.stepString(te.step))
	}
	voter := te.proposerLocked()
	payload := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s:%s:%s", te.height, te.round, blockHash, "prevote", voter)))
	return Vote{ValidatorAddress: voter, PubKey: hex.EncodeToString(payload[:]), BlockHash: blockHash, Height: te.height, Round: int64(te.round), Type: "prevote", Timestamp: time.Now().Unix(), Signature: payload[:]}, nil
}

func (te *TendermintEngine) Precommit(blockHash string) (Vote, error) {
	te.mu.Lock()
	defer te.mu.Unlock()
	if te.step != 2 {
		return Vote{}, fmt.Errorf("not in precommit step: %s", te.stepString(te.step))
	}
	voter := te.proposerLocked()
	payload := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s:%s:%s", te.height, te.round, blockHash, "precommit", voter)))
	return Vote{ValidatorAddress: voter, PubKey: hex.EncodeToString(payload[:]), BlockHash: blockHash, Height: te.height, Round: int64(te.round), Type: "precommit", Timestamp: time.Now().Unix(), Signature: payload[:]}, nil
}

func (te *TendermintEngine) Commit(blockHash string) error {
	te.mu.Lock()
	defer te.mu.Unlock()
	if te.step != 3 {
		return fmt.Errorf("consensus not ready for commit: %s", te.stepString(te.step))
	}
	te.committedRound = te.round
	te.lockedBlock = blockHash
	te.step = 4
	te.lastBlockTime = time.Now().Unix()
	return nil
}

func (te *TendermintEngine) VerifyVote(v Vote) bool {
	if v.BlockHash == "" || len(v.Signature) == 0 {
		return false
	}
	return true
}

func (te *TendermintEngine) Slash(validator string, reason EvidenceType) {
	te.AddEvidence(Evidence{
		Type:          fmt.Sprintf("evidence:%d", reason),
		ValidatorAddr: validator,
		Height:        te.height,
		Round:         int64(te.round),
		BlockHash:     te.lockedBlock,
		Timestamp:     time.Now().Unix(),
	})
}

type ProposerSelector interface {
	Proposer(validators []string, height uint64) string
}

type StakeWeightedSelector struct{}

func (s StakeWeightedSelector) Proposer(validators []string, height uint64) string {
	if len(validators) == 0 {
		return ""
	}
	sort.Slice(validators, func(i, j int) bool {
		return validators[i] < validators[j]
	})
	return validators[int(height)%len(validators)]
}

func (te *TendermintEngine) ProposeBlock(transactions []Transaction, proposer string) Block {
	block := Block{
		Index:        te.height,
		Author:       proposer,
		MinerAddress: proposer,
		PreviousHash: hex.EncodeToString(sha256.New().Sum(nil)),
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		Nonce:        0,
	}
	block.TxMerkleRoot = CalculateMerkleRoot(transactions)
	block.BlockHash = calculateHash(block)
	return block
}

func (te *TendermintEngine) FinalizeBlock(block Block) error {
	te.mu.Lock()
	defer te.mu.Unlock()
	if te.step != 3 {
		return fmt.Errorf("consensus not ready for commit: %s", te.stepString(te.step))
	}
	te.committedRound = te.round
	te.lockedBlock = block.BlockHash
	te.step = 4
	te.lastBlockTime = time.Now().Unix()
	return nil
}

func (te *TendermintEngine) ValidatorSet() []string {
	te.mu.Lock()
	defer te.mu.Unlock()
	out := make([]string, len(te.validators))
	copy(out, te.validators)
	return out
}

func (te *TendermintEngine) UpdateValidatorSet(validators []string) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.validators = validators
	te.selectProposerLocked()
}

func (te *TendermintEngine) DefaultBlockTimeout() time.Duration {
	return 5 * time.Second
}

func (te *TendermintEngine) IsFinalized(blockHash string) bool {
	te.mu.Lock()
	defer te.mu.Unlock()
	return te.step == 4 && te.lockedBlock == blockHash
}

func (te *TendermintEngine) CurrentState() string {
	te.mu.Lock()
	defer te.mu.Unlock()
	return fmt.Sprintf("height=%d round=%d step=%s proposer=%s", te.height, te.round, te.stepString(te.step), te.proposerLocked())
}
