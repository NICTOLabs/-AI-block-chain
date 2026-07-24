package main

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type TransactionType string

const (
	Transfer       TransactionType = "TRANSFER"
	RegisterModel  TransactionType = "REGISTER_MODEL"
	UpdateModel    TransactionType = "UPDATE_MODEL"
	PurchaseApiKey TransactionType = "PURCHASE_API_KEY"
)

type Transaction struct {
	ID         string          `json:"id,omitempty"`
	From       string          `json:"from"`
	FromPubKey string          `json:"from_pubkey"`
	To         string          `json:"to"`
	Amount     uint64          `json:"amount"`
	Fee        uint64          `json:"fee,omitempty"`
	Nonce      uint64          `json:"nonce,omitempty"`
	TxType     TransactionType `json:"tx_type"`
	Payload    string          `json:"payload,omitempty"`
	Signature  string          `json:"signature,omitempty"`
	Timestamp  int64           `json:"timestamp"`
	ChainID    string          `json:"chain_id,omitempty"`
}

type Block struct {
	Index        uint64        `json:"index"`
	Author       string        `json:"author"`
	MinerAddress string        `json:"miner_address"`
	PreviousHash string        `json:"previous_hash"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	Nonce        uint64        `json:"nonce"`
	BlockHash    string        `json:"block_hash"`
}

type Account struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
	Staked  uint64 `json:"staked"`
	IsAgent bool   `json:"is_agent"`
}

type ModelEntry struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	Version      string `json:"version"`
	Metadata     string `json:"metadata"`
	PricePerCall uint64 `json:"price_per_call"`
	Active       bool   `json:"active"`
}

type ConsensusType int

const (
	ProofOfStake ConsensusType = iota
	ProofOfAuthority
)

const (
	BaseFee           uint64 = 5
	FeeMultiplier     uint64 = 2
	BurnRatePercent   uint64 = 1
	RewardRatePercent uint64 = 4
	MinStake          uint64 = 100
	SlashPercent      uint64 = 10
	CurrencyName      string = "TENDER"
	MaxSupply         uint64 = 18_446_744_073_709_551_615
	InitialSupply     uint64 = 4_500_000_000
	BlockRewardBase   uint64 = 10

	AlucardBaseEmission uint64 = 2
	AlucardBonusStart   uint64 = 8
	AlucardDecayFactor  uint64 = 70
	AlucardCycleBlocks  uint64 = 157_788_000
)

type Escrow struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	ServiceID string `json:"service_id"`
	Status    string `json:"status"`
}

type GovernanceProposal struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Votes       map[string]bool `json:"votes"`
	Status      string          `json:"status"`
}

type ServiceAgreement struct {
	ID           string `json:"id"`
	Provider     string `json:"provider"`
	Consumer     string `json:"consumer"`
	ModelID      string `json:"model_id"`
	PricePerCall uint64 `json:"price_per_call"`
	MaxCalls     uint64 `json:"max_calls"`
	Status       string `json:"status"`
}

type UsageMeter struct {
	AgreementID string `json:"agreement_id"`
	UsageCount  uint64 `json:"usage_count"`
	TotalCost   uint64 `json:"total_cost"`
}

type Wallet struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

type AuditEntry struct {
	Timestamp int64  `json:"timestamp"`
	Event     string `json:"event"`
	Actor     string `json:"actor"`
	Details   string `json:"details"`
}

type ManagedWallet struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	PublicKey string `json:"public_key"`
	Label     string `json:"label"`
	IsAgent   bool   `json:"is_agent"`
}

type Validator struct {
	Address     string `json:"address"`
	Stake       uint64 `json:"stake"`
	Active      bool   `json:"active"`
	JoinedAt    int64  `json:"joined_at"`
	Performance uint64 `json:"performance"`
}

type ValidatorInfo = Validator

type Blockchain struct {
	mu           sync.RWMutex
	Chain        []Block
	Pending      []Transaction
	Ledger       map[string]*Account
	Registry     map[string]ModelEntry
	Consensus    ConsensusType
	Authorities  []string
	validatorIdx int
	DataDir      string
	TokenSupply  uint64
	Escrows      map[string]Escrow
	Proposals    map[string]GovernanceProposal
	Agreements   map[string]ServiceAgreement
	UsageMeters  map[string]UsageMeter
	UsedNonces   map[string]map[uint64]struct{}
	NextNonce    map[string]uint64
	SeenTxIDs    map[string]struct{}
	AuditTrail   []AuditEntry
	Wallets      map[string]ManagedWallet
	Validators   map[string]Validator
	GenesisHash  string
	ChainID      string
	BlockTime    time.Duration
	Difficulty   uint32
	metrics      *serverMetrics
}

type P2PNode struct {
	addr         string
	peers        []string
	peerScores   map[string]int
	trustedPeers map[string]bool
	chain        *Blockchain
	listener     net.Listener
	shutdown     chan struct{}
	maxPeers     int
	strictMode   bool
	nodeSecret   string
	mutedPeers   map[string]time.Time
}

type serverConfig struct {
	APIKey      string
	EnableAuth  bool
	RateLimit   int
	RateWindow  time.Duration
	EnableTLS   bool
	MetricsPath string
	APIPort     int
	P2PPort     int
	DataDir     string
	Consensus   string
	StrictP2P   bool
}

type rateLimiter struct {
	mu     sync.Mutex
	counts map[string][]time.Time
	limit  int
	window time.Duration
}

type circuitBreaker struct {
	mu            sync.Mutex
	failures     int64
	threshold    int64
	window       time.Duration
	lastFailure  time.Time
	state        string
}

type serverMetrics struct {
	mu            sync.Mutex
	requestCount  int64
	errorCount    int64
	lastRequestAt time.Time
	blocksMined   int64
	peersSeen     int64
	txAccepted    int64
	txRejected    int64
}

type NodeInfo struct {
	Address string   `json:"address"`
	Peers   []string `json:"peers"`
}

type nodeState struct {
	Chain       []Block                        `json:"chain"`
	Pending     []Transaction                  `json:"pending"`
	Ledger      map[string]*Account            `json:"ledger"`
	Registry    map[string]ModelEntry          `json:"registry"`
	Consensus   string                         `json:"consensus"`
	Authorities []string                       `json:"authorities"`
	TokenSupply uint64                         `json:"token_supply"`
	Escrows     map[string]Escrow              `json:"escrows"`
	Proposals   map[string]GovernanceProposal  `json:"proposals"`
	Agreements  map[string]ServiceAgreement    `json:"agreements"`
	UsageMeters map[string]UsageMeter          `json:"usage_meters"`
	UsedNonces  map[string]map[uint64]struct{} `json:"used_nonces"`
	SeenTxIDs   map[string]struct{}            `json:"seen_tx_ids"`
	AuditTrail  []AuditEntry                   `json:"audit_trail"`
	NextNonce   map[string]uint64              `json:"next_nonce"`
}

type p2pMessage struct {
	Type  string       `json:"type"`
	From  string       `json:"from,omitempty"`
	Block *Block       `json:"block,omitempty"`
	Chain []Block      `json:"chain,omitempty"`
	Tx    *Transaction `json:"tx,omitempty"`
	Peer  *NodeInfo    `json:"peer,omitempty"`
}

func NewWallet() *Wallet {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	return &Wallet{PublicKey: pub, PrivateKey: priv}
}

func (w *Wallet) Address() string {
	hash := sha256.Sum256(w.PublicKey)
	return hex.EncodeToString(hash[:])
}

func (w *Wallet) Sign(tx Transaction) Transaction {
	tx.FromPubKey = hex.EncodeToString(w.PublicKey)
	tx.Timestamp = time.Now().Unix()
	payload := tx.signingPayload()
	tx.Signature = hex.EncodeToString(ed25519.Sign(w.PrivateKey, payload))
	return tx
}

func (tx Transaction) signingPayload() []byte {
	clone := tx
	clone.Signature = ""
	clone.ID = ""
	data, _ := json.Marshal(clone)
	return data
}

func NewBlockchain(consensus ConsensusType, dataDir string, chainID string, genesisPath string) *Blockchain {
	bc := &Blockchain{
		Chain:       []Block{},
		Pending:     []Transaction{},
		Ledger:      make(map[string]*Account),
		Registry:    make(map[string]ModelEntry),
		Consensus:   consensus,
		Authorities: []string{},
		DataDir:     dataDir,
		TokenSupply: InitialSupply,
		Escrows:     make(map[string]Escrow),
		Proposals:   make(map[string]GovernanceProposal),
		Agreements:  make(map[string]ServiceAgreement),
		UsageMeters: make(map[string]UsageMeter),
		UsedNonces:  make(map[string]map[uint64]struct{}),
		NextNonce:   make(map[string]uint64),
		SeenTxIDs:   make(map[string]struct{}),
		AuditTrail:  []AuditEntry{},
		Wallets:     make(map[string]ManagedWallet),
		Validators:  make(map[string]Validator),
		ChainID:     chainID,
		BlockTime:   time.Second * 5,
		Difficulty:  8,
		metrics:     &serverMetrics{},
	}
	bc.createGenesisBlock()
	_ = os.MkdirAll(dataDir, 0o755)
	if err := bc.loadFromDisk(); err != nil {
		if genesisPath != "" {
			if err := bc.loadGenesis(genesisPath); err != nil {
				log.Printf("{\"event\":\"genesis_load_error\",\"error\":\"%v\"}", err)
				bc.seedDemoState()
			}
		} else {
			bc.seedDemoState()
		}
		_ = bc.saveToDisk()
	}
	return bc
}

func (bc *Blockchain) createGenesisBlock() {
	genesisPayload, _ := json.Marshal(map[string]any{
		"chain_id":       bc.ChainID,
		"timestamp":      time.Now().Unix(),
		"initial_supply": bc.TokenSupply,
		"consensus":      consensusName(bc.Consensus),
	})
	genesisHash := sha256.Sum256(genesisPayload)
	genesis := Block{
		Index:        0,
		Author:       "genesis",
		PreviousHash: "0",
		Timestamp:    time.Now().Unix(),
		Transactions: []Transaction{},
		Nonce:        0,
		BlockHash:    hex.EncodeToString(genesisHash[:]),
	}
	bc.Chain = append(bc.Chain, genesis)
	bc.GenesisHash = genesis.BlockHash
}

func (bc *Blockchain) saveToDisk() error {
	state := nodeState{
		Chain:       bc.Chain,
		Pending:     bc.Pending,
		Ledger:      bc.Ledger,
		Registry:    bc.Registry,
		Consensus:   consensusName(bc.Consensus),
		Authorities: bc.Authorities,
		TokenSupply: bc.TokenSupply,
		Escrows:     bc.Escrows,
		Proposals:   bc.Proposals,
		Agreements:  bc.Agreements,
		UsageMeters: bc.UsageMeters,
		UsedNonces:  bc.UsedNonces,
		SeenTxIDs:   bc.SeenTxIDs,
		AuditTrail:  bc.AuditTrail,
		NextNonce:   bc.NextNonce,
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(bc.DataDir, "chain.json")
	return os.WriteFile(path, payload, 0o644)
}

func (bc *Blockchain) loadFromDisk() error {
	path := filepath.Join(bc.DataDir, "chain.json")
	payload, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var state nodeState
	if err := json.Unmarshal(payload, &state); err != nil {
		return err
	}
	bc.Chain = state.Chain
	bc.Pending = state.Pending
	bc.Ledger = state.Ledger
	bc.Registry = state.Registry
	bc.Authorities = state.Authorities
	bc.TokenSupply = state.TokenSupply
	bc.Escrows = state.Escrows
	bc.Proposals = state.Proposals
	bc.Agreements = state.Agreements
	bc.UsageMeters = state.UsageMeters
	bc.UsedNonces = state.UsedNonces
	bc.SeenTxIDs = state.SeenTxIDs
	bc.AuditTrail = state.AuditTrail
	bc.NextNonce = state.NextNonce
	bc.Wallets = make(map[string]ManagedWallet)
	if bc.UsedNonces == nil {
		bc.UsedNonces = make(map[string]map[uint64]struct{})
	}
	if bc.SeenTxIDs == nil {
		bc.SeenTxIDs = make(map[string]struct{})
	}
	if bc.AuditTrail == nil {
		bc.AuditTrail = []AuditEntry{}
	}
	if bc.NextNonce == nil {
		bc.NextNonce = make(map[string]uint64)
	}
	for from, nonceMap := range bc.UsedNonces {
		maxNonce := uint64(0)
		for nonce := range nonceMap {
			if nonce > maxNonce {
				maxNonce = nonce
			}
		}
		if maxNonce > 0 {
			bc.NextNonce[from] = maxNonce + 1
		}
	}
	if len(bc.Chain) > 0 {
		bc.GenesisHash = bc.Chain[0].BlockHash
	}
	if state.Consensus == "poa" {
		bc.Consensus = ProofOfAuthority
	} else {
		bc.Consensus = ProofOfStake
	}
	if len(bc.Chain) == 0 {
		bc.createGenesisBlock()
	}
	return nil
}

func (bc *Blockchain) seedDemoState() {
	bc.AddAccount("human1", 1_000_000, false)
	bc.AddAccount("agentA", 100_000, true)
	bc.AddAccount("agentB", 50_000, true)
	bc.Stake("agentA", 10_000)
	bc.AddAuthority("agentA")
}

type genesisFile struct {
	ChainID       string `json:"chain_id"`
	InitialSupply uint64 `json:"initial_supply"`
	MaxSupply     uint64 `json:"max_supply"`
	Allocations   []struct {
		Address   string `json:"address"`
		PublicKey string `json:"public_key"`
		Amount    uint64 `json:"amount"`
	} `json:"allocations"`
	Validators []struct {
		Address   string `json:"address"`
		PublicKey string `json:"public_key"`
		Stake     uint64 `json:"stake"`
	} `json:"validators"`
}

func (bc *Blockchain) loadGenesis(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var genesis genesisFile
	if err := json.Unmarshal(data, &genesis); err != nil {
		return err
	}
	if genesis.ChainID != "" {
		bc.ChainID = genesis.ChainID
	}
	bc.TokenSupply = 0
	for _, alloc := range genesis.Allocations {
		if alloc.Address != "" {
			bc.addAccountLocked(alloc.Address, alloc.Amount, false)
			bc.TokenSupply += alloc.Amount
		}
	}
	for _, val := range genesis.Validators {
		if val.Address != "" {
			if acct := bc.Ledger[val.Address]; acct != nil {
				acct.IsAgent = false
				acct.Staked = val.Stake
				acct.Balance -= val.Stake
			} else {
				bc.addAccountLocked(val.Address, val.Stake, false)
				bc.Ledger[val.Address].Staked = val.Stake
				bc.TokenSupply += val.Stake
			}
			bc.Validators[val.Address] = Validator{
				Address:     val.Address,
				Stake:       val.Stake,
				Active:      true,
				JoinedAt:    time.Now().Unix(),
				Performance: 100,
			}
			bc.AddAuthority(val.Address)
		}
	}
	bc.appendAuditEntry("genesis_loaded", "system", fmt.Sprintf("chain_id=%s supply=%d allocations=%d validators=%d", bc.ChainID, bc.TokenSupply, len(genesis.Allocations), len(genesis.Validators)))
	return nil
}

func (bc *Blockchain) addAccountLocked(address string, balance uint64, isAgent bool) {
	bc.Ledger[address] = &Account{Address: address, Balance: balance, Staked: 0, IsAgent: isAgent}
	bc.appendAuditEntry("account_created", address, fmt.Sprintf("balance=%d agent=%t", balance, isAgent))
}

func (bc *Blockchain) AddAccount(address string, balance uint64, isAgent bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.addAccountLocked(address, balance, isAgent)
}

func (bc *Blockchain) CreateManagedWallet(label string, isAgent bool) (ManagedWallet, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	wallet := NewWallet()
	address := wallet.Address()
	bc.addAccountLocked(address, 1000, isAgent)
	managed := ManagedWallet{ID: fmt.Sprintf("wallet-%d", time.Now().UnixNano()), Address: address, PublicKey: hex.EncodeToString(wallet.PublicKey), Label: label, IsAgent: isAgent}
	bc.Wallets[managed.ID] = managed
	bc.appendAuditEntry("wallet_created", address, fmt.Sprintf("label=%s agent=%t", label, isAgent))
	return managed, nil
}

func (bc *Blockchain) Stake(address string, amount uint64) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	account := bc.Ledger[address]
	if account == nil || account.Balance < amount {
		return
	}
	account.Balance -= amount
	account.Staked += amount
	bc.appendAuditEntry("stake", address, fmt.Sprintf("amount=%d", amount))
}

func (bc *Blockchain) Slash(address string, amount uint64) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	account := bc.Ledger[address]
	if account == nil || account.Staked < amount {
		return
	}
	account.Staked -= amount
	account.Balance -= amount
	bc.appendAuditEntry("slash", address, fmt.Sprintf("amount=%d", amount))
}

func (bc *Blockchain) estimateFee(tx Transaction, congestion int) uint64 {
	baseComplexity := uint64(1)
	switch tx.TxType {
	case RegisterModel:
		baseComplexity = 3
	case UpdateModel:
		baseComplexity = 2
	case PurchaseApiKey:
		baseComplexity = 2
	}
	congestionFactor := uint64(congestion)
	if congestionFactor > 10 {
		congestionFactor = 10
	}
	baseFee := BaseFee + (baseComplexity * FeeMultiplier) + congestionFactor
	if bc.TokenSupply > 0 {
		if bc.TokenSupply < 1_000_000 {
			baseFee += 2
		} else if bc.TokenSupply > 10_000_000 {
			baseFee -= 1
		}
	}
	return baseFee
}

func (bc *Blockchain) DistributeRewards() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for _, account := range bc.Ledger {
		if account.Staked > 0 {
			reward := account.Staked * RewardRatePercent / 100
			account.Balance += reward
			bc.TokenSupply += reward
		}
	}
}

func (bc *Blockchain) Burn(amount uint64) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if amount > bc.TokenSupply {
		amount = bc.TokenSupply
	}
	bc.TokenSupply -= amount
}

func (bc *Blockchain) GetMaxSupply() uint64 {
	return MaxSupply
}

func BlockReward(blockHeight uint64) uint64 {
	if blockHeight == 0 {
		return AlucardBaseEmission + AlucardBonusStart
	}
	era := blockHeight / AlucardCycleBlocks
	bonus := AlucardBonusStart
	for i := uint64(0); i < era && bonus > 1; i++ {
		bonus = bonus * AlucardDecayFactor / 100
	}
	return AlucardBaseEmission + bonus
}

func (bc *Blockchain) CreateEscrow(from, to string, amount uint64, serviceID string) (Escrow, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	fromAccount := bc.Ledger[from]
	if fromAccount == nil || fromAccount.Balance < amount {
		return Escrow{}, fmt.Errorf("insufficient funds")
	}
	fromAccount.Balance -= amount
	id := fmt.Sprintf("escrow-%d", time.Now().UnixNano())
	escrow := Escrow{ID: id, From: from, To: to, Amount: amount, ServiceID: serviceID, Status: "active"}
	bc.Escrows[id] = escrow
	bc.appendAuditEntry("escrow_created", from, fmt.Sprintf("to=%s amount=%d service_id=%s", to, amount, serviceID))
	return escrow, nil
}

func (bc *Blockchain) CreateProposal(title, description string) GovernanceProposal {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	id := fmt.Sprintf("proposal-%d", time.Now().UnixNano())
	proposal := GovernanceProposal{ID: id, Title: title, Description: description, Votes: make(map[string]bool), Status: "open"}
	bc.Proposals[id] = proposal
	bc.appendAuditEntry("proposal_created", "governance", fmt.Sprintf("id=%s title=%s", id, title))
	return proposal
}

func (bc *Blockchain) VoteProposal(id, voter string) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	proposal := bc.Proposals[id]
	if proposal.ID == "" {
		return
	}
	proposal.Votes[voter] = true
	bc.Proposals[id] = proposal
	bc.appendAuditEntry("proposal_voted", voter, fmt.Sprintf("proposal_id=%s", id))
}

func (bc *Blockchain) AddAuthority(address string) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.Authorities = append(bc.Authorities, address)
}

func (bc *Blockchain) RegisterValidator(address string, stake uint64) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	account := bc.Ledger[address]
	if account == nil || account.Balance < stake || stake < MinStake {
		return fmt.Errorf("insufficient funds or stake below minimum")
	}
	account.Balance -= stake
	account.Staked += stake
	bc.Validators[address] = Validator{Address: address, Stake: stake, Active: true, JoinedAt: time.Now().Unix(), Performance: 100}
	bc.appendAuditEntry("validator_registered", address, fmt.Sprintf("stake=%d", stake))
	return nil
}

func (bc *Blockchain) SubmitMinedBlock(block Block) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if len(bc.Chain) == 0 || block.Index != uint64(len(bc.Chain)) {
		return fmt.Errorf("invalid block index")
	}
	prev := bc.Chain[len(bc.Chain)-1]
	if block.PreviousHash != prev.BlockHash {
		return fmt.Errorf("invalid previous hash")
	}
	if calculateHash(block) != block.BlockHash {
		return fmt.Errorf("invalid block hash")
	}
	if err := bc.validateBlock(block, prev); err != nil {
		return fmt.Errorf("invalid block transactions: %w", err)
	}
	author := block.Author
	if author == "" {
		author = block.MinerAddress
	}
	if author == "" {
		return fmt.Errorf("missing miner address")
	}
	reward := BlockReward(block.Index)
	if bc.TokenSupply+reward <= MaxSupply {
		if account := bc.Ledger[author]; account != nil {
			account.Balance += reward
			bc.TokenSupply += reward
		}
	}
	bc.applyBlock(block)
	bc.Chain = append(bc.Chain, block)
	bc.Pending = []Transaction{}
	bc.adjustDifficulty()
	bc.appendAuditEntry("block_submitted", author, fmt.Sprintf("index=%d txs=%d reward=%d", block.Index, len(block.Transactions), reward))
	atomic.AddInt64(&bc.ensureMetrics().blocksMined, 1)
	return bc.saveToDisk()
}

func (bc *Blockchain) SubmitTransaction(tx Transaction) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	next := bc.NextNonce[tx.From]
	if tx.Nonce < next {
		bc.appendAuditEntry("transaction_rejected", tx.From, fmt.Sprintf("tx_id=%s nonce=%d expected=%d", tx.ID, tx.Nonce, next))
		return
	}
	bc.Pending = append(bc.Pending, tx)
	bc.appendAuditEntry("transaction_submitted", tx.From, fmt.Sprintf("tx_id=%s nonce=%d", tx.ID, tx.Nonce))
}

func (bc *Blockchain) EnqueueTransaction(tx Transaction) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if tx.ID == "" {
		tx.ID = fmt.Sprintf("tx-%d", time.Now().UnixNano())
	}
	if tx.Nonce == 0 {
		tx.Nonce = bc.NextNonce[tx.From]
	}
	if bc.isReplay(tx) {
		bc.appendAuditEntry("transaction_rejected", tx.From, fmt.Sprintf("tx_id=%s nonce=%d", tx.ID, tx.Nonce))
		return
	}
	for i, pending := range bc.Pending {
		if pending.ID == tx.ID && pending.Nonce == tx.Nonce && pending.From == tx.From {
			if tx.Fee > pending.Fee {
				bc.Pending[i] = tx
				bc.appendAuditEntry("transaction_replaced", tx.From, fmt.Sprintf("tx_id=%s", tx.ID))
			}
			return
		}
	}
	if uint64(len(bc.Pending)) >= 5000 {
		lowestFeeIdx := 0
		for i, p := range bc.Pending {
			if p.Fee < bc.Pending[lowestFeeIdx].Fee {
				lowestFeeIdx = i
			}
		}
		if tx.Fee <= bc.Pending[lowestFeeIdx].Fee {
			return
		}
		bc.Pending[lowestFeeIdx] = tx
	} else {
		bc.Pending = append(bc.Pending, tx)
	}
	bc.appendAuditEntry("transaction_queued", tx.From, fmt.Sprintf("tx_id=%s fee=%d", tx.ID, tx.Fee))
}

func (bc *Blockchain) ensureMetrics() *serverMetrics {
	if bc.metrics == nil {
		bc.metrics = &serverMetrics{}
	}
	return bc.metrics
}

func (bc *Blockchain) MineBlock() (*Block, error) {
	return bc.MineBlockFor("")
}

func (bc *Blockchain) MineBlockFor(minerAddress string) (*Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if len(bc.Chain) == 0 {
		bc.createGenesisBlock()
	}
	prevHash := bc.Chain[len(bc.Chain)-1].BlockHash
	author := bc.selectValidator()
	if minerAddress != "" {
		author = minerAddress
	}
	block := Block{
		Index:        uint64(len(bc.Chain)),
		Author:       author,
		MinerAddress: author,
		PreviousHash: prevHash,
		Timestamp:    time.Now().Unix(),
		Transactions: []Transaction{},
		Nonce:        0,
	}
	pending := make([]Transaction, len(bc.Pending))
	copy(pending, bc.Pending)
	sort.Slice(pending, func(i, j int) bool {
		if pending[i].Fee == pending[j].Fee {
			return pending[i].Timestamp < pending[j].Timestamp
		}
		return pending[i].Fee > pending[j].Fee
	})
	for _, tx := range pending {
		if bc.validateTransaction(tx) {
			fee := bc.estimateFee(tx, len(pending))
			if tx.Fee < fee {
				atomic.AddInt64(&bc.metrics.txRejected, 1)
				continue
			}
			if tx.Fee > 0 {
				burnAmount := tx.Fee * BurnRatePercent / 100
				bc.TokenSupply -= burnAmount
				if account := bc.Ledger[author]; account != nil {
					account.Balance += burnAmount
				}
			}
			block.Transactions = append(block.Transactions, tx)
			bc.markTransactionSeen(tx)
			atomic.AddInt64(&bc.ensureMetrics().txAccepted, 1)
		}
	}
	block = bc.proofOfWork(block)
	if err := bc.validateBlock(block, bc.Chain[len(bc.Chain)-1]); err != nil {
		return nil, err
	}
	reward := BlockReward(block.Index)
	if bc.TokenSupply+reward <= MaxSupply {
		if account := bc.Ledger[author]; account != nil {
			account.Balance += reward
			bc.TokenSupply += reward
		}
	}
	bc.applyBlock(block)
	bc.Chain = append(bc.Chain, block)
	bc.Pending = []Transaction{}
	bc.adjustDifficulty()
	bc.appendAuditEntry("block_mined", author, fmt.Sprintf("index=%d txs=%d miner=%s reward=%d", block.Index, len(block.Transactions), author, reward))
	atomic.AddInt64(&bc.ensureMetrics().blocksMined, 1)
	if err := bc.saveToDisk(); err != nil {
		return nil, err
	}
	return &block, nil
}

func (bc *Blockchain) selectValidator() string {
	if bc.Consensus == ProofOfAuthority {
		if len(bc.Authorities) == 0 {
			return "authority"
		}
		bc.validatorIdx = (bc.validatorIdx + 1) % len(bc.Authorities)
		return bc.Authorities[bc.validatorIdx]
	}
	var candidates []string
	for address, validator := range bc.Validators {
		if validator.Active && validator.Stake > 0 {
			candidates = append(candidates, address)
		}
	}
	if len(candidates) == 0 {
		for address, account := range bc.Ledger {
			if account.Staked > 0 {
				candidates = append(candidates, address)
			}
		}
	}
	if len(candidates) == 0 {
		for address := range bc.Ledger {
			candidates = append(candidates, address)
		}
	}
	if len(candidates) == 0 {
		return "validator"
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := bc.Validators[candidates[i]]
		right := bc.Validators[candidates[j]]
		if left.Stake == right.Stake {
			if left.Performance == right.Performance {
				return candidates[i] < candidates[j]
			}
			return left.Performance > right.Performance
		}
		return left.Stake > right.Stake
	})
	bc.validatorIdx = (bc.validatorIdx + 1) % len(candidates)
	return candidates[bc.validatorIdx]
}

func (bc *Blockchain) proofOfWork(block Block) Block {
	target := new(big.Int)
	target.Lsh(big.NewInt(1), 256-uint(bc.Difficulty))
	for {
		hash := calculateHash(block)
		hashVal := new(big.Int)
		hashVal.SetString(hash, 16)
		if hashVal.Cmp(target) < 0 {
			block.BlockHash = hash
			return block
		}
		block.Nonce++
	}
}

func (bc *Blockchain) adjustDifficulty() {
	if len(bc.Chain) < 2 {
		return
	}
	last := bc.Chain[len(bc.Chain)-1]
	prev := bc.Chain[len(bc.Chain)-2]
	actualTime := time.Unix(last.Timestamp, 0).Sub(time.Unix(prev.Timestamp, 0))
	if actualTime < bc.BlockTime && bc.Difficulty < 64 {
		bc.Difficulty++
	} else if actualTime > bc.BlockTime*2 && bc.Difficulty > 0 {
		bc.Difficulty--
	}
}

func calculateHash(block Block) string {
	clone := block
	clone.BlockHash = ""
	data, _ := json.Marshal(clone)
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	quantum := sha512.Sum512(data)
	merged := make([]byte, 32)
	for i := 0; i < 32; i++ {
		merged[i] = second[i] ^ quantum[i]
	}
	final := sha256.Sum256(merged)
	return hex.EncodeToString(final[:])
}

func (bc *Blockchain) validateChain(chain []Block) error {
	if len(chain) == 0 {
		return fmt.Errorf("empty chain")
	}
	if chain[0].Index != 0 {
		return fmt.Errorf("invalid genesis")
	}
	for i := 1; i < len(chain); i++ {
		if err := bc.validateBlock(chain[i], chain[i-1]); err != nil {
			return err
		}
	}
	return nil
}

func (bc *Blockchain) replaceChain(newChain []Block) bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if err := bc.validateChain(newChain); err != nil {
		return false
	}
	if len(newChain) <= len(bc.Chain) {
		return false
	}
	bc.Chain = newChain
	bc.appendAuditEntry("chain_replaced", "network", fmt.Sprintf("height=%d", len(newChain)))
	bc.saveToDisk()
	return true
}

func (bc *Blockchain) validateTransaction(tx Transaction) bool {
	if tx.ChainID != bc.ChainID {
		return false
	}
	if !verifyTransaction(tx) {
		return false
	}
	if bc.isReplay(tx) {
		return false
	}
	sender, ok := bc.Ledger[tx.From]
	if !ok {
		return false
	}
	if tx.From != tx.To && tx.Amount == 0 {
		return false
	}
	switch tx.TxType {
	case Transfer:
		_, receiverExists := bc.Ledger[tx.To]
		return receiverExists && sender.Balance >= tx.Amount
	case RegisterModel:
		_, exists := bc.Registry[tx.To]
		return sender.IsAgent && !exists
	case UpdateModel:
		entry, exists := bc.Registry[tx.To]
		return exists && entry.Owner == tx.From
	case PurchaseApiKey:
		entry, exists := bc.Registry[tx.To]
		return exists && sender.Balance >= tx.Amount && entry.Active
	default:
		return false
	}
}

func (bc *Blockchain) isReplay(tx Transaction) bool {
	if tx.Nonce == 0 {
		return false
	}
	if _, exists := bc.UsedNonces[tx.From]; exists {
		if _, used := bc.UsedNonces[tx.From][tx.Nonce]; used {
			return true
		}
	}
	if bc.NextNonce[tx.From] > tx.Nonce {
		return true
	}
	if tx.ID != "" {
		if _, seen := bc.SeenTxIDs[tx.ID]; seen {
			return true
		}
	}
	return false
}

func (bc *Blockchain) markTransactionSeen(tx Transaction) {
	if tx.ID != "" {
		bc.SeenTxIDs[tx.ID] = struct{}{}
	}
	if tx.Nonce > 0 {
		if bc.UsedNonces[tx.From] == nil {
			bc.UsedNonces[tx.From] = make(map[uint64]struct{})
		}
		bc.UsedNonces[tx.From][tx.Nonce] = struct{}{}
		if bc.NextNonce[tx.From] <= tx.Nonce {
			bc.NextNonce[tx.From] = tx.Nonce + 1
		}
	}
}

func (bc *Blockchain) appendAuditEntry(event, actor, details string) {
	bc.AuditTrail = append(bc.AuditTrail, AuditEntry{Timestamp: time.Now().Unix(), Event: event, Actor: actor, Details: details})
}

func (bc *Blockchain) applyBlock(block Block) {
	for _, tx := range block.Transactions {
		switch tx.TxType {
		case Transfer:
			sender := bc.Ledger[tx.From]
			receiver := bc.Ledger[tx.To]
			if sender != nil && receiver != nil && sender.Balance >= tx.Amount {
				sender.Balance -= tx.Amount
				receiver.Balance += tx.Amount
			}
		case RegisterModel:
			bc.Registry[tx.To] = ModelEntry{
				ID:           tx.To,
				Owner:        tx.From,
				Version:      tx.Payload,
				Metadata:     tx.Payload,
				PricePerCall: tx.Amount,
				Active:       true,
			}
		case UpdateModel:
			entry := bc.Registry[tx.To]
			entry.Version = tx.Payload
			entry.Metadata = tx.Payload
			entry.PricePerCall = tx.Amount
			bc.Registry[tx.To] = entry
		case PurchaseApiKey:
			entry := bc.Registry[tx.To]
			sender := bc.Ledger[tx.From]
			receiver := bc.Ledger[entry.Owner]
			if sender != nil && receiver != nil && sender.Balance >= tx.Amount {
				sender.Balance -= tx.Amount
				receiver.Balance += tx.Amount
			}
		}
	}
}

func (bc *Blockchain) CreateServiceAgreement(provider, consumer, modelID string, pricePerCall, maxCalls uint64) (ServiceAgreement, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if _, exists := bc.Agreements[modelID]; exists {
		return ServiceAgreement{}, fmt.Errorf("agreement already exists")
	}
	agreement := ServiceAgreement{ID: fmt.Sprintf("agreement-%d", time.Now().UnixNano()), Provider: provider, Consumer: consumer, ModelID: modelID, PricePerCall: pricePerCall, MaxCalls: maxCalls, Status: "active"}
	bc.Agreements[agreement.ID] = agreement
	return agreement, nil
}

func (bc *Blockchain) RecordUsage(agreementID string, usageCount uint64) (UsageMeter, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	agreement, exists := bc.Agreements[agreementID]
	if !exists {
		return UsageMeter{}, fmt.Errorf("agreement not found")
	}
	if agreement.MaxCalls > 0 && usageCount > agreement.MaxCalls {
		agreement.Status = "over_limit"
		bc.Agreements[agreementID] = agreement
	}
	meter := bc.UsageMeters[agreementID]
	meter.AgreementID = agreementID
	meter.UsageCount += usageCount
	meter.TotalCost += usageCount * agreement.PricePerCall
	bc.UsageMeters[agreementID] = meter
	return meter, nil
}

func (bc *Blockchain) snapshot() nodeState {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return nodeState{
		Chain:       append([]Block(nil), bc.Chain...),
		Pending:     append([]Transaction(nil), bc.Pending...),
		Ledger:      bc.Ledger,
		Registry:    bc.Registry,
		Consensus:   consensusName(bc.Consensus),
		Authorities: append([]string(nil), bc.Authorities...),
		TokenSupply: bc.TokenSupply,
		Escrows:     bc.Escrows,
		Proposals:   bc.Proposals,
		Agreements:  bc.Agreements,
		UsageMeters: bc.UsageMeters,
		NextNonce:   bc.NextNonce,
	}
}

func verifyTransaction(tx Transaction) bool {
	if tx.ChainID == "" {
		return false
	}
	pubKey, err := hex.DecodeString(tx.FromPubKey)
	if err != nil {
		return false
	}
	if len(pubKey) != ed25519.PublicKeySize {
		return false
	}
	addressBytes := sha256.Sum256(pubKey)
	if tx.From != hex.EncodeToString(addressBytes[:]) {
		return false
	}
	sig, err := hex.DecodeString(tx.Signature)
	if err != nil {
		return false
	}
	if len(sig) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pubKey), tx.signingPayload(), sig)
}

func consensusName(consensus ConsensusType) string {
	if consensus == ProofOfAuthority {
		return "poa"
	}
	return "pos"
}

func currencySymbol() string {
	return CurrencyName
}

func serverConfigFromEnv() serverConfig {
	cfg := serverConfig{
		APIKey:      getEnvOrDefault("TENDER_API_KEY", "change-me-in-production"),
		EnableAuth:  getEnvBoolOrDefault("TENDER_ENABLE_AUTH", true),
		RateLimit:   getEnvIntOrDefault("TENDER_RATE_LIMIT", 60),
		RateWindow:  time.Duration(getEnvIntOrDefault("TENDER_RATE_WINDOW_SECONDS", 60)) * time.Second,
		MetricsPath: getEnvOrDefault("TENDER_METRICS_PATH", "/metrics"),
		APIPort:     getEnvIntOrDefault("TENDER_API_PORT", 8080),
		P2PPort:     getEnvIntOrDefault("TENDER_P2P_PORT", 3030),
		DataDir:     getEnvOrDefault("TENDER_DATA_DIR", "./data"),
		Consensus:   strings.ToLower(getEnvOrDefault("TENDER_CONSENSUS", "pos")),
		StrictP2P:   getEnvBoolOrDefault("TENDER_STRICT_P2P", true),
	}
	return cfg
}

func getEnvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvIntOrDefault(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func getEnvBoolOrDefault(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func main() {
	apiPort := flag.Int("api-port", 0, "HTTP API port")
	p2pPort := flag.Int("p2p-port", 0, "P2P listen port")
	peer := flag.String("peer", "", "Optional peer address")
	bootstrapPeers := flag.String("bootstrap-peers", "", "Comma-separated bootstrap peer addresses")
	dataDir := flag.String("data-dir", "", "Directory to persist blockchain state")
	consensus := flag.String("consensus", "", "Consensus type: pos or poa")
	strictP2P := flag.Bool("strict-p2p", false, "Reject untrusted or duplicate peers")
	apiKey := flag.String("api-key", "", "API key required for protected endpoints")
	enableAuth := flag.Bool("enable-auth", false, "Require API key auth for mutating endpoints")
	rateLimit := flag.Int("rate-limit", 0, "Requests per minute per client")
	chainID := flag.String("chain-id", "tdr-mainnet-1", "Chain ID for replay protection")
	genesisPath := flag.String("genesis", "", "Path to genesis JSON file for initial state")
	flag.Parse()

	envCfg := serverConfigFromEnv()
	if *apiPort != 0 {
		envCfg.APIPort = *apiPort
	}
	if *p2pPort != 0 {
		envCfg.P2PPort = *p2pPort
	}
	if *dataDir != "" {
		envCfg.DataDir = *dataDir
	}
	if *consensus != "" {
		envCfg.Consensus = strings.ToLower(*consensus)
	}
	if *strictP2P {
		envCfg.StrictP2P = true
	}
	if *apiKey != "" {
		envCfg.APIKey = *apiKey
	}
	if *enableAuth {
		envCfg.EnableAuth = true
	}
	if *rateLimit != 0 {
		envCfg.RateLimit = *rateLimit
	}

	var chainConsensus ConsensusType
	if envCfg.Consensus == "poa" {
		chainConsensus = ProofOfAuthority
	} else {
		chainConsensus = ProofOfStake
	}

	chain := NewBlockchain(chainConsensus, envCfg.DataDir, *chainID, *genesisPath)
	p2p := &P2PNode{addr: fmt.Sprintf("0.0.0.0:%d", envCfg.P2PPort), peers: []string{}, peerScores: make(map[string]int), trustedPeers: make(map[string]bool), chain: chain, shutdown: make(chan struct{}), maxPeers: 50, strictMode: envCfg.StrictP2P}
	if *peer != "" {
		p2p.peers = append(p2p.peers, *peer)
	}

	if *bootstrapPeers != "" {
		for _, peer := range strings.Split(*bootstrapPeers, ",") {
			peer = strings.TrimSpace(peer)
			if peer != "" {
				p2p.peers = append(p2p.peers, peer)
			}
		}
	}

	go p2p.start()
	go p2p.connectToPeers()
	go startAPI(chain, envCfg.APIPort, p2p, envCfg)

	log.Printf("{\"event\":\"node_start\",\"currency\":\"%s\",\"api_port\":%d,\"p2p_port\":%d,\"consensus\":\"%s\",\"chain_id\":\"%s\"}", CurrencyName, envCfg.APIPort, envCfg.P2PPort, envCfg.Consensus, *chainID)
	<-p2p.shutdown
}

func startAPI(chain *Blockchain, port int, p2p *P2PNode, cfg serverConfig) {
	metrics := chain.ensureMetrics()
	limiter := newRateLimiter(cfg.RateLimit, cfg.RateWindow)
	cb := newCircuitBreaker(5, 10*time.Second)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	})
	mux.Handle("/", http.FileServer(http.Dir("./web")))
	mux.HandleFunc("/api/chain", func(w http.ResponseWriter, r *http.Request) {
		if !cb.Allow() {
			http.Error(w, "circuit breaker open", http.StatusServiceUnavailable)
			return
		}
		if err := requireAuth(r, cfg); err != nil {
			cb.RecordFailure()
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			cb.RecordFailure()
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		cb.RecordSuccess()
		metrics.recordRequest(true)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(chain.snapshot())
	})
	mux.HandleFunc("/api/audit", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.mu.RLock()
		defer chain.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(chain.AuditTrail)
	})
	mux.HandleFunc("/api/monitoring", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		chain.mu.RLock()
		defer chain.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"height":               len(chain.Chain),
			"pending_transactions": len(chain.Pending),
			"token_supply":         chain.TokenSupply,
			"audit_entries":        len(chain.AuditTrail),
			"peer_count":           len(p2p.peers),
			"trusted_peer_count":   len(p2p.trustedPeers),
			"strict_p2p":           p2p.strictMode,
			"consensus":            consensusName(chain.Consensus),
		})
	})
	mux.HandleFunc("/api/mempool", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.mu.RLock()
		defer chain.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(chain.Pending)
	})
	mux.HandleFunc("/api/transactions", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var tx Transaction
		if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		chain.EnqueueTransaction(tx)
		_ = chain.saveToDisk()
		p2p.broadcastTransaction(tx)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tx)
	})
	mux.HandleFunc("/api/mine", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			MinerAddress string `json:"miner_address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		block, err := chain.MineBlockFor(payload.MinerAddress)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		p2p.broadcastBlock(block)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(block)
	})
	mux.HandleFunc("/api/miner/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var block Block
		if err := json.NewDecoder(r.Body).Decode(&block); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := chain.SubmitMinedBlock(block); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		p2p.broadcastBlock(&block)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted", "hash": block.BlockHash})
	})
	mux.HandleFunc("/api/validators", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.mu.RLock()
		defer chain.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"consensus":      consensusName(chain.Consensus),
			"authorities":    chain.Authorities,
			"next_validator": chain.selectValidator(),
			"validators":     chain.Validators,
		})
	})
	mux.HandleFunc("/api/validators/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Address string `json:"address"`
			Stake   uint64 `json:"stake"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := chain.RegisterValidator(payload.Address, payload.Stake); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.saveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "registered"})
	})
	mux.HandleFunc("/api/bootstrap", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"node":      p2p.addr,
			"peers":     p2p.peers,
			"trusted":   p2p.trustedPeers,
			"validator": chain.selectValidator(),
		})
	})
	mux.HandleFunc("/api/registry", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.mu.RLock()
		defer chain.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(chain.Registry)
	})
	mux.HandleFunc("/api/accounts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.mu.RLock()
		defer chain.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(chain.Ledger)
	})
	mux.HandleFunc("/api/stake", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Address string `json:"address"`
			Amount  uint64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		chain.Stake(payload.Address, payload.Amount)
		_ = chain.saveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"address": payload.Address, "amount": payload.Amount})
	})
	mux.HandleFunc("/api/wallet", func(w http.ResponseWriter, r *http.Request) {
		wallet := NewWallet()
		address := wallet.Address()
		chain.AddAccount(address, 1000, false)
		_ = chain.saveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"address": address, "public_key": hex.EncodeToString(wallet.PublicKey)})
	})
	mux.HandleFunc("/api/managed-wallets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Label   string `json:"label"`
			IsAgent bool   `json:"is_agent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		wallet, err := chain.CreateManagedWallet(payload.Label, payload.IsAgent)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.saveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(wallet)
	})
	mux.HandleFunc("/api/transfer", func(w http.ResponseWriter, r *http.Request) {
		if err := requireAuth(r, cfg); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if !limiter.allow(r.RemoteAddr) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload Transaction
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		chain.EnqueueTransaction(payload)
		_ = chain.saveToDisk()
		p2p.broadcastTransaction(payload)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})
	mux.HandleFunc("/api/tokenomics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.mu.RLock()
		defer chain.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"currency":            CurrencyName,
			"token_supply":        chain.TokenSupply,
			"burn_rate_percent":   BurnRatePercent,
			"reward_rate_percent": RewardRatePercent,
			"base_fee":            BaseFee,
			"escrows":             chain.Escrows,
			"proposals":           chain.Proposals,
		})
	})
	mux.HandleFunc("/api/escrow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			From      string `json:"from"`
			To        string `json:"to"`
			Amount    uint64 `json:"amount"`
			ServiceID string `json:"service_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		escrow, err := chain.CreateEscrow(payload.From, payload.To, payload.Amount, payload.ServiceID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.saveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(escrow)
	})
	mux.HandleFunc("/api/proposals", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		proposal := chain.CreateProposal(payload.Title, payload.Description)
		_ = chain.saveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(proposal)
	})
	mux.HandleFunc("/api/agreements", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			Provider     string `json:"provider"`
			Consumer     string `json:"consumer"`
			ModelID      string `json:"model_id"`
			PricePerCall uint64 `json:"price_per_call"`
			MaxCalls     uint64 `json:"max_calls"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		agreement, err := chain.CreateServiceAgreement(payload.Provider, payload.Consumer, payload.ModelID, payload.PricePerCall, payload.MaxCalls)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.saveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(agreement)
	})
	mux.HandleFunc("/api/usage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			AgreementID string `json:"agreement_id"`
			UsageCount  uint64 `json:"usage_count"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		meter, err := chain.RecordUsage(payload.AgreementID, payload.UsageCount)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = chain.saveToDisk()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meter)
	})
	mux.HandleFunc(cfg.MetricsPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(w, "# HELP tender_http_requests_total Total HTTP requests\n")
		_, _ = fmt.Fprintf(w, "# TYPE tender_http_requests_total counter\n")
		_, _ = fmt.Fprintf(w, "tender_http_requests_total %d\n", atomic.LoadInt64(&metrics.requestCount))
		_, _ = fmt.Fprintf(w, "# HELP tender_http_errors_total Total HTTP errors\n")
		_, _ = fmt.Fprintf(w, "# TYPE tender_http_errors_total counter\n")
		_, _ = fmt.Fprintf(w, "tender_http_errors_total %d\n", atomic.LoadInt64(&metrics.errorCount))
		_, _ = fmt.Fprintf(w, "# HELP tender_blocks_mined_total Total blocks mined\n")
		_, _ = fmt.Fprintf(w, "# TYPE tender_blocks_mined_total counter\n")
		_, _ = fmt.Fprintf(w, "tender_blocks_mined_total %d\n", atomic.LoadInt64(&metrics.blocksMined))
		_, _ = fmt.Fprintf(w, "# HELP tender_peers_seen_total Total peers observed\n")
		_, _ = fmt.Fprintf(w, "# TYPE tender_peers_seen_total counter\n")
		_, _ = fmt.Fprintf(w, "tender_peers_seen_total %d\n", atomic.LoadInt64(&metrics.peersSeen))
		_, _ = fmt.Fprintf(w, "# HELP tender_tx_accepted_total Total accepted transactions\n")
		_, _ = fmt.Fprintf(w, "# TYPE tender_tx_accepted_total counter\n")
		_, _ = fmt.Fprintf(w, "tender_tx_accepted_total %d\n", atomic.LoadInt64(&metrics.txAccepted))
		_, _ = fmt.Fprintf(w, "# HELP tender_tx_rejected_total Total rejected transactions\n")
		_, _ = fmt.Fprintf(w, "# TYPE tender_tx_rejected_total counter\n")
		_, _ = fmt.Fprintf(w, "tender_tx_rejected_total %d\n", atomic.LoadInt64(&metrics.txRejected))
	})
	log.Printf("{\"event\":\"api_listen\",\"port\":%d}", port)
	if err := http.ListenAndServe(":"+strconv.Itoa(port), mux); err != nil {
		log.Fatal(err)
	}
}

func (p2p *P2PNode) start() {
	listener, err := net.Listen("tcp", p2p.addr)
	if err != nil {
		log.Printf("p2p listen: %v", err)
		return
	}
	p2p.listener = listener
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-p2p.shutdown:
				return
			default:
				log.Printf("{\"event\":\"accept_error\",\"error\":\"%v\"}", err)
			}
			continue
		}
		go p2p.handleConn(conn)
	}
}

func (p2p *P2PNode) connectToPeers() {
	for _, peer := range p2p.peers {
		if peer == "" || peer == p2p.addr {
			continue
		}
		if len(p2p.peers) > p2p.maxPeers {
			break
		}
		go func(target string) {
			conn, err := net.Dial("tcp", target)
			if err != nil {
				log.Printf("{\"event\":\"connect_peer\",\"peer\":\"%s\",\"error\":\"%v\"}", target, err)
				return
			}
			defer conn.Close()
			p2p.peerScores[target] = 1
			p2p.trustedPeers[target] = true
			_ = p2p.writeMessage(conn, p2pMessage{Type: "hello", From: p2p.addr, Peer: &NodeInfo{Address: p2p.addr, Peers: p2p.peers}})
			p2p.handleConn(conn)
		}(peer)
	}
}

func (p2p *P2PNode) handleConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	remote := conn.RemoteAddr().String()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("{\"event\":\"p2p_read\",\"error\":\"%v\"}", err)
			}
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 5*1024*1024 {
			log.Printf("{\"event\":\"p2p_oversize\",\"peer\":\"%s\"}", remote)
			return
		}
		var msg p2pMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("{\"event\":\"p2p_decode\",\"error\":\"%v\"}", err)
			return
		}
		if msg.Type == "block" && msg.Block != nil {
			p2p.chain.mu.Lock()
			if len(msg.Chain) > 0 {
				if p2p.chain.replaceChain(msg.Chain) {
					p2p.chain.mu.Unlock()
					continue
				}
			}
			if len(p2p.chain.Chain) < int(msg.Block.Index)+1 || p2p.chain.Chain[len(p2p.chain.Chain)-1].BlockHash != msg.Block.PreviousHash {
				p2p.chain.Chain = append(p2p.chain.Chain, *msg.Block)
				_ = p2p.chain.saveToDisk()
			}
			p2p.chain.mu.Unlock()
			relayPayload, _ := json.Marshal(msg)
			for _, peer := range p2p.peers {
				if peer == "" || peer == p2p.addr || peer == remote || !p2p.trustedPeers[peer] {
					continue
				}
				relayConn, err := net.DialTimeout("tcp", peer, 3*time.Second)
				if err != nil {
					continue
				}
				_, _ = relayConn.Write(append(relayPayload, '\n'))
				relayConn.Close()
			}
		}
		if msg.Type == "tx" && msg.Tx != nil {
			p2p.chain.EnqueueTransaction(*msg.Tx)
			relayPayload, _ := json.Marshal(msg)
			for _, peer := range p2p.peers {
				if peer == "" || peer == p2p.addr || peer == remote || !p2p.trustedPeers[peer] {
					continue
				}
				relayConn, err := net.DialTimeout("tcp", peer, 3*time.Second)
				if err != nil {
					continue
				}
				_, _ = relayConn.Write(append(relayPayload, '\n'))
				relayConn.Close()
			}
		}
		if msg.Type == "hello" && msg.Peer != nil {
			if p2p.strictMode && len(p2p.peers) >= p2p.maxPeers {
				continue
			}
			if msg.Peer.Address != "" && msg.Peer.Address != p2p.addr {
				p2p.peers = append(p2p.peers, msg.Peer.Address)
				p2p.peerScores[msg.Peer.Address] = 1
				p2p.trustedPeers[msg.Peer.Address] = true
			}
		}
	}
}

func (p2p *P2PNode) broadcastTransaction(tx Transaction) {
	msg := p2pMessage{Type: "tx", Tx: &tx}
	payload, _ := json.Marshal(msg)
	for _, peer := range p2p.peers {
		if peer == "" || peer == p2p.addr || !p2p.trustedPeers[peer] {
			continue
		}
		conn, err := net.DialTimeout("tcp", peer, 3*time.Second)
		if err != nil {
			log.Printf("{\"event\":\"broadcast_tx\",\"peer\":\"%s\",\"error\":\"%v\"}", peer, err)
			continue
		}
		_, _ = conn.Write(append(payload, '\n'))
		conn.Close()
	}
}

func (p2p *P2PNode) broadcastBlock(block *Block) {
	msg := p2pMessage{Type: "block", Block: block}
	payload, _ := json.Marshal(msg)
	p2p.peers = append(p2p.peers, p2p.addr)
	for _, peer := range p2p.peers {
		if peer == "" || peer == p2p.addr || !p2p.trustedPeers[peer] {
			continue
		}
		conn, err := net.Dial("tcp", peer)
		if err != nil {
			log.Printf("{\"event\":\"broadcast\",\"peer\":\"%s\",\"error\":\"%v\"}", peer, err)
			continue
		}
		_, _ = conn.Write(append(payload, '\n'))
		conn.Close()
	}
}

func (p2p *P2PNode) writeMessage(conn net.Conn, msg p2pMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = conn.Write(append(payload, '\n'))
	return err
}

func requireAuth(r *http.Request, cfg serverConfig) error {
	if !cfg.EnableAuth {
		return nil
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("missing api key")
	}
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return fmt.Errorf("missing authorization")
	}
	prefix := "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return fmt.Errorf("invalid authorization scheme")
	}
	provided := strings.TrimPrefix(authorization, prefix)
	if provided != cfg.APIKey {
		return fmt.Errorf("invalid api key")
	}
	return nil
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit <= 0 {
		limit = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &rateLimiter{counts: make(map[string][]time.Time), limit: limit, window: window}
}

func newCircuitBreaker(threshold int64, window time.Duration) *circuitBreaker {
	return &circuitBreaker{threshold: threshold, window: window, state: "closed"}
}

func (cb *circuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == "open" {
		if time.Since(cb.lastFailure) > cb.window {
			cb.state = "half-open"
			cb.failures = 0
			return true
		}
		return false
	}
	return true
}

func (cb *circuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.state = "closed"
}

func (cb *circuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailure = time.Now()
	if cb.failures >= cb.threshold {
		cb.state = "open"
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	entries := rl.counts[key]
	filtered := entries[:0]
	for _, ts := range entries {
		if now.Sub(ts) <= rl.window {
			filtered = append(filtered, ts)
		}
	}
	rl.counts[key] = filtered
	if len(filtered) >= rl.limit {
		return false
	}
	rl.counts[key] = append(filtered, now)
	return true
}

func (m *serverMetrics) recordRequest(ok bool) {
	if m == nil {
		return
	}
	atomic.AddInt64(&m.requestCount, 1)
	if !ok {
		atomic.AddInt64(&m.errorCount, 1)
	}
	m.mu.Lock()
	m.lastRequestAt = time.Now()
	m.mu.Unlock()
}

func (bc *Blockchain) validateBlock(block Block, prev Block) error {
	if block.Index != prev.Index+1 {
		return fmt.Errorf("invalid index")
	}
	if block.PreviousHash != prev.BlockHash {
		return fmt.Errorf("invalid previous hash")
	}
	if block.BlockHash == "" {
		return fmt.Errorf("missing block hash")
	}
	if calculateHash(block) != block.BlockHash {
		return fmt.Errorf("invalid block hash")
	}
	seen := make(map[string]struct{})
	for _, tx := range block.Transactions {
		if tx.ID == "" {
			return fmt.Errorf("missing transaction id")
		}
		if _, exists := seen[tx.ID]; exists {
			return fmt.Errorf("duplicate transaction")
		}
		seen[tx.ID] = struct{}{}
		if !bc.validateTransaction(tx) {
			return fmt.Errorf("invalid transaction")
		}
	}
	return nil
}

