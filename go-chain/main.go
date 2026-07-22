package main

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
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
}

type Block struct {
	Index        uint64        `json:"index"`
	Author       string        `json:"author"`
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
	RewardRatePercent uint64 = 1
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
}

type P2PNode struct {
	addr     string
	peers    []string
	chain    *Blockchain
	listener net.Listener
	shutdown chan struct{}
}

type NodeInfo struct {
	Address string   `json:"address"`
	Peers   []string `json:"peers"`
}

type nodeState struct {
	Chain       []Block                       `json:"chain"`
	Pending     []Transaction                 `json:"pending"`
	Ledger      map[string]*Account           `json:"ledger"`
	Registry    map[string]ModelEntry         `json:"registry"`
	Consensus   string                        `json:"consensus"`
	Authorities []string                      `json:"authorities"`
	TokenSupply uint64                        `json:"token_supply"`
	Escrows     map[string]Escrow             `json:"escrows"`
	Proposals   map[string]GovernanceProposal `json:"proposals"`
	Agreements  map[string]ServiceAgreement   `json:"agreements"`
	UsageMeters map[string]UsageMeter         `json:"usage_meters"`
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
	data, _ := json.Marshal(clone)
	return data
}

func NewBlockchain(consensus ConsensusType, dataDir string) *Blockchain {
	bc := &Blockchain{
		Chain:       []Block{},
		Pending:     []Transaction{},
		Ledger:      make(map[string]*Account),
		Registry:    make(map[string]ModelEntry),
		Consensus:   consensus,
		Authorities: []string{},
		DataDir:     dataDir,
		TokenSupply: 1_000_000_000,
		Escrows:     make(map[string]Escrow),
		Proposals:   make(map[string]GovernanceProposal),
		Agreements:  make(map[string]ServiceAgreement),
		UsageMeters: make(map[string]UsageMeter),
	}
	bc.createGenesisBlock()
	_ = os.MkdirAll(dataDir, 0o755)
	if err := bc.loadFromDisk(); err != nil {
		bc.seedDemoState()
		_ = bc.saveToDisk()
	}
	return bc
}

func (bc *Blockchain) createGenesisBlock() {
	genesis := Block{
		Index:        0,
		Author:       "genesis",
		PreviousHash: "0",
		Timestamp:    time.Now().Unix(),
		Transactions: []Transaction{},
		Nonce:        0,
		BlockHash:    "genesis",
	}
	bc.Chain = append(bc.Chain, genesis)
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

func (bc *Blockchain) AddAccount(address string, balance uint64, isAgent bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.Ledger[address] = &Account{Address: address, Balance: balance, Staked: 0, IsAgent: isAgent}
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
	return BaseFee + (baseComplexity * FeeMultiplier) + congestionFactor
}

func (bc *Blockchain) DistributeRewards() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for _, account := range bc.Ledger {
		if account.Staked > 0 {
			reward := account.Staked * RewardRatePercent / 100
			account.Balance += reward
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
	return escrow, nil
}

func (bc *Blockchain) CreateProposal(title, description string) GovernanceProposal {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	id := fmt.Sprintf("proposal-%d", time.Now().UnixNano())
	proposal := GovernanceProposal{ID: id, Title: title, Description: description, Votes: make(map[string]bool), Status: "open"}
	bc.Proposals[id] = proposal
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
}

func (bc *Blockchain) AddAuthority(address string) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.Authorities = append(bc.Authorities, address)
}

func (bc *Blockchain) SubmitTransaction(tx Transaction) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.Pending = append(bc.Pending, tx)
}

func (bc *Blockchain) EnqueueTransaction(tx Transaction) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	for i, pending := range bc.Pending {
		if pending.ID == tx.ID && pending.Nonce == tx.Nonce && pending.From == tx.From {
			if tx.Fee > pending.Fee {
				bc.Pending[i] = tx
				return
			}
			return
		}
	}
	bc.Pending = append(bc.Pending, tx)
}

func (bc *Blockchain) MineBlock() (*Block, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if len(bc.Chain) == 0 {
		bc.createGenesisBlock()
	}
	prevHash := bc.Chain[len(bc.Chain)-1].BlockHash
	author := bc.selectValidator()
	block := Block{
		Index:        uint64(len(bc.Chain)),
		Author:       author,
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
				continue
			}
			if tx.Fee > 0 {
				burnAmount := tx.Fee * BurnRatePercent / 100
				bc.Burn(burnAmount)
			}
			block.Transactions = append(block.Transactions, tx)
		}
	}
	block = bc.proofOfWork(block)
	if err := bc.validateBlock(block, bc.Chain[len(bc.Chain)-1]); err != nil {
		return nil, err
	}
	bc.applyBlock(block)
	bc.Chain = append(bc.Chain, block)
	bc.Pending = []Transaction{}
	bc.DistributeRewards()
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
	for address, account := range bc.Ledger {
		if account.Staked > 0 {
			candidates = append(candidates, address)
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
		left := bc.Ledger[candidates[i]]
		right := bc.Ledger[candidates[j]]
		if left.Staked == right.Staked {
			return candidates[i] < candidates[j]
		}
		return left.Staked > right.Staked
	})
	bc.validatorIdx = (bc.validatorIdx + 1) % len(candidates)
	return candidates[bc.validatorIdx]
}

func (bc *Blockchain) proofOfWork(block Block) Block {
	for {
		hash := calculateHash(block)
		if len(hash) >= 4 && hash[:4] == "0000" {
			block.BlockHash = hash
			return block
		}
		block.Nonce++
	}
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
	bc.saveToDisk()
	return true
}

func (bc *Blockchain) validateTransaction(tx Transaction) bool {
	if !verifyTransaction(tx) {
		return false
	}
	sender, ok := bc.Ledger[tx.From]
	if !ok {
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
	}
}

func calculateHash(block Block) string {
	clone := block
	clone.BlockHash = ""
	data, _ := json.Marshal(clone)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func verifyTransaction(tx Transaction) bool {
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

func main() {
	apiPort := flag.Int("api-port", 8080, "HTTP API port")
	p2pPort := flag.Int("p2p-port", 3030, "P2P listen port")
	peer := flag.String("peer", "", "Optional peer address")
	bootstrapPeers := flag.String("bootstrap-peers", "", "Comma-separated bootstrap peer addresses")
	dataDir := flag.String("data-dir", "./data", "Directory to persist blockchain state")
	consensus := flag.String("consensus", "pos", "Consensus type: pos or poa")
	flag.Parse()

	var chainConsensus ConsensusType
	if strings.ToLower(*consensus) == "poa" {
		chainConsensus = ProofOfAuthority
	} else {
		chainConsensus = ProofOfStake
	}

	chain := NewBlockchain(chainConsensus, *dataDir)
	p2p := &P2PNode{addr: fmt.Sprintf("127.0.0.1:%d", *p2pPort), peers: []string{}, chain: chain, shutdown: make(chan struct{})}
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
	go startAPI(chain, *apiPort, p2p)

	fmt.Printf("AI blockchain node running on http://127.0.0.1:%d\n", *apiPort)
	fmt.Printf("P2P listener on %s\n", p2p.addr)
	<-p2p.shutdown
}

func startAPI(chain *Blockchain, port int, p2p *P2PNode) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/api/chain", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(chain.snapshot())
	})
	mux.HandleFunc("/api/mempool", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.mu.RLock()
		defer chain.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(chain.Pending)
	})
	mux.HandleFunc("/api/transactions", func(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tx)
	})
	mux.HandleFunc("/api/mine", func(w http.ResponseWriter, r *http.Request) {
		block, err := chain.MineBlock()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		p2p.broadcastBlock(block)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(block)
	})
	mux.HandleFunc("/api/validators", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.mu.RLock()
		defer chain.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"consensus":      consensusName(chain.Consensus),
			"authorities":    chain.Authorities,
			"next_validator": chain.selectValidator(),
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
	mux.HandleFunc("/api/transfer", func(w http.ResponseWriter, r *http.Request) {
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
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})
	mux.HandleFunc("/api/tokenomics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		chain.mu.RLock()
		defer chain.mu.RUnlock()
		_ = json.NewEncoder(w).Encode(map[string]any{
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
	mux.Handle("/", http.FileServer(http.Dir("./web")))
	log.Printf("API server listening on :%d", port)
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
				log.Printf("accept error: %v", err)
			}
			continue
		}
		go p2p.handleConn(conn)
	}
}

func (p2p *P2PNode) connectToPeers() {
	for _, peer := range p2p.peers {
		if peer == "" {
			continue
		}
		go func(target string) {
			conn, err := net.Dial("tcp", target)
			if err != nil {
				log.Printf("connect to peer %s: %v", target, err)
				return
			}
			defer conn.Close()
			_ = p2p.writeMessage(conn, p2pMessage{Type: "hello", From: p2p.addr, Peer: &NodeInfo{Address: p2p.addr, Peers: p2p.peers}})
			p2p.handleConn(conn)
		}(peer)
	}
}

func (p2p *P2PNode) handleConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("p2p read: %v", err)
			}
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var msg p2pMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("p2p decode: %v", err)
			continue
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
		}
		if msg.Type == "tx" && msg.Tx != nil {
			p2p.chain.EnqueueTransaction(*msg.Tx)
		}
		if msg.Type == "hello" && msg.Peer != nil {
			p2p.peers = append(p2p.peers, msg.Peer.Address)
		}
	}
}

func (p2p *P2PNode) broadcastBlock(block *Block) {
	msg := p2pMessage{Type: "block", Block: block}
	payload, _ := json.Marshal(msg)
	p2p.peers = append(p2p.peers, p2p.addr)
	for _, peer := range p2p.peers {
		if peer == "" || peer == p2p.addr {
			continue
		}
		conn, err := net.Dial("tcp", peer)
		if err != nil {
			log.Printf("broadcast to %s: %v", peer, err)
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
