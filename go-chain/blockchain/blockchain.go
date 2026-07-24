package blockchain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

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

type Blockchain struct {
	mu              sync.RWMutex
	Chain           []Block
	Pending         []Transaction
	Ledger          map[string]*Account
	Registry        map[string]ModelEntry
	Consensus       ConsensusType
	Authorities     []string
	validatorIdx    int
	DataDir         string
	TokenSupply     uint64
	Escrows         map[string]Escrow
	Proposals       map[string]GovernanceProposal
	Agreements      map[string]ServiceAgreement
	UsageMeters     map[string]UsageMeter
	UsedNonces      map[string]map[uint64]struct{}
	NextNonce       map[string]uint64
	SeenTxIDs       map[string]struct{}
	AuditTrail      []AuditEntry
	Wallets         map[string]ManagedWallet
	Validators      map[string]Validator
	GenesisHash     string
	ChainID         string
	BlockTime       time.Duration
	Difficulty      uint32
	FinalizedBlocks map[uint64]struct{}
	LastFinalized   uint64
	AgentTxCount    uint64
	metrics         *serverMetrics
}

func NewBlockchain(consensus ConsensusType, dataDir string, chainID string) *Blockchain {
	bc := &Blockchain{
		Chain:           []Block{},
		Pending:         []Transaction{},
		Ledger:          make(map[string]*Account),
		Registry:        make(map[string]ModelEntry),
		Consensus:       consensus,
		Authorities:     []string{},
		DataDir:         dataDir,
		TokenSupply:     1_000_000_000,
		Escrows:         make(map[string]Escrow),
		Proposals:       make(map[string]GovernanceProposal),
		Agreements:      make(map[string]ServiceAgreement),
		UsageMeters:     make(map[string]UsageMeter),
		UsedNonces:      make(map[string]map[uint64]struct{}),
		NextNonce:       make(map[string]uint64),
		SeenTxIDs:       make(map[string]struct{}),
		AuditTrail:      []AuditEntry{},
		Wallets:         make(map[string]ManagedWallet),
		Validators:      make(map[string]Validator),
		ChainID:         chainID,
		BlockTime:       time.Second * 5,
		Difficulty:      16,
		metrics:         &serverMetrics{},
		FinalizedBlocks: make(map[uint64]struct{}),
		LastFinalized:   0,
		AgentTxCount:    0,
		metrics:         &serverMetrics{},
	}
	bc.createGenesisBlock()
	_ = os.MkdirAll(dataDir, 0o755)
	if err := bc.LoadFromDisk(); err != nil {
		bc.seedDemoState()
		_ = bc.SaveToDisk()
	}
	return bc
}

func (bc *Blockchain) createGenesisBlock() {
	genesisPayload, _ := json.Marshal(map[string]any{
		"chain_id":       bc.ChainID,
		"timestamp":      time.Now().Unix(),
		"initial_supply": bc.TokenSupply,
		"consensus":      ConsensusName(bc.Consensus),
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

func (bc *Blockchain) SaveToDisk() error {
	state := nodeState{
		Chain:           bc.Chain,
		Pending:         bc.Pending,
		Ledger:          bc.Ledger,
		Registry:        bc.Registry,
		Consensus:       ConsensusName(bc.Consensus),
		Authorities:     bc.Authorities,
		TokenSupply:     bc.TokenSupply,
		Escrows:         bc.Escrows,
		Proposals:       bc.Proposals,
		Agreements:      bc.Agreements,
		UsageMeters:     bc.UsageMeters,
		UsedNonces:      bc.UsedNonces,
		SeenTxIDs:       bc.SeenTxIDs,
		AuditTrail:      bc.AuditTrail,
		NextNonce:       bc.NextNonce,
		FinalizedBlocks: bc.FinalizedBlocks,
		LastFinalized:   bc.LastFinalized,
	}
	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(bc.DataDir, "chain.json")
	return os.WriteFile(path, payload, 0o644)
}

func (bc *Blockchain) LoadFromDisk() error {
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
	bc.FinalizedBlocks = state.FinalizedBlocks
	bc.LastFinalized = state.LastFinalized
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

func (bc *Blockchain) EnsureMetrics() *serverMetrics {
	if bc.metrics == nil {
		bc.metrics = &serverMetrics{}
	}
	return bc.metrics
}

func (bc *Blockchain) RecordRequest(ok bool) {
	if bc == nil || bc.metrics == nil {
		return
	}
	atomic.AddInt64(&bc.metrics.requestCount, 1)
	if !ok {
		atomic.AddInt64(&bc.metrics.errorCount, 1)
	}
	bc.metrics.mu.Lock()
	bc.metrics.lastRequestAt = time.Now()
	bc.metrics.mu.Unlock()
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

func (bc *Blockchain) FundAccount(address string, amount uint64) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if bc.Ledger[address] == nil {
		bc.Ledger[address] = &Account{Address: address, Balance: amount, Staked: 0, IsAgent: false}
		bc.appendAuditEntry("account_created", address, fmt.Sprintf("balance=%d funded=true", amount))
	} else {
		bc.Ledger[address].Balance += amount
		bc.appendAuditEntry("account_funded", address, fmt.Sprintf("amount=%d", amount))
	}
	bc.TokenSupply += amount
	_ = bc.SaveToDisk()
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
	penalty := amount * SlashPercent / 100
	if penalty == 0 {
		penalty = 1
	}
	account.Staked -= amount
	account.Balance -= penalty
	bc.appendAuditEntry("slash", address, fmt.Sprintf("amount=%d penalty=%d", amount, penalty))
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
	blockReward := uint64(100)
	if account := bc.Ledger[author]; account != nil {
		account.Balance += blockReward
		bc.TokenSupply += blockReward
	}
	bc.applyBlock(block)
	bc.Chain = append(bc.Chain, block)
	bc.Pending = []Transaction{}
	bc.adjustDifficulty()
	bc.appendAuditEntry("block_submitted", author, fmt.Sprintf("index=%d txs=%d", block.Index, len(block.Transactions)))
	atomic.AddInt64(&bc.EnsureMetrics().blocksMined, 1)
	return bc.SaveToDisk()
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
	for i, pending := range bc.Pending {
		if pending.ID == tx.ID && pending.Nonce == tx.Nonce && pending.From == tx.From {
			if tx.Fee > pending.Fee {
				bc.Pending[i] = tx
				bc.appendAuditEntry("transaction_replaced", tx.From, fmt.Sprintf("tx_id=%s", tx.ID))
			}
			return
		}
	}
	if bc.isReplay(tx) {
		bc.appendAuditEntry("transaction_rejected", tx.From, fmt.Sprintf("tx_id=%s nonce=%d", tx.ID, tx.Nonce))
		return
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
	bc.markTransactionSeen(tx)
	bc.appendAuditEntry("transaction_queued", tx.From, fmt.Sprintf("tx_id=%s fee=%d", tx.ID, tx.Fee))
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
	author := bc.SelectValidator()
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
			block.Transactions = append(block.Transactions, tx)
			bc.markTransactionSeen(tx)
			atomic.AddInt64(&bc.EnsureMetrics().txAccepted, 1)
		}
	}
	block = bc.proofOfWork(block)
	if err := bc.validateBlock(block, bc.Chain[len(bc.Chain)-1]); err != nil {
		return nil, err
	}
	blockReward := uint64(100)
	if account := bc.Ledger[author]; account != nil {
		account.Balance += blockReward
		bc.TokenSupply += blockReward
	}
	bc.applyBlock(block)
	bc.Chain = append(bc.Chain, block)
	bc.Pending = []Transaction{}
	bc.adjustDifficulty()
	bc.appendAuditEntry("block_mined", author, fmt.Sprintf("index=%d txs=%d miner=%s", block.Index, len(block.Transactions), author))
	atomic.AddInt64(&bc.EnsureMetrics().blocksMined, 1)
	if err := bc.SaveToDisk(); err != nil {
		return nil, err
	}
	return &block, nil
}

func (bc *Blockchain) SelectValidator() string {
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
	block.TxMerkleRoot = CalculateMerkleRoot(block.Transactions)
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

func (bc *Blockchain) ReplaceChain(newChain []Block) bool {
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
	_ = bc.SaveToDisk()
	return true
}

func (bc *Blockchain) validateTransaction(tx Transaction) bool {
	if tx.ChainID != bc.ChainID {
		return false
	}
	if !VerifyTransaction(tx) {
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
	return false
}

func (bc *Blockchain) markTransactionSeen(tx Transaction) {
	if tx.ID != "" {
		bc.SeenTxIDs[tx.ID] = struct{}{}
	}
}

func (bc *Blockchain) markTransactionUsed(tx Transaction) {
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
		bc.markTransactionSeen(tx)
		bc.markTransactionUsed(tx)
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

func (bc *Blockchain) Snapshot() nodeState {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return nodeState{
		Chain:           append([]Block(nil), bc.Chain...),
		Pending:         append([]Transaction(nil), bc.Pending...),
		Ledger:          bc.Ledger,
		Registry:        bc.Registry,
		Consensus:       ConsensusName(bc.Consensus),
		Authorities:     append([]string(nil), bc.Authorities...),
		TokenSupply:     bc.TokenSupply,
		Escrows:         bc.Escrows,
		Proposals:       bc.Proposals,
		Agreements:      bc.Agreements,
		UsageMeters:     bc.UsageMeters,
		NextNonce:       bc.NextNonce,
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
	if block.TxMerkleRoot != "" {
		if CalculateMerkleRoot(block.Transactions) != block.TxMerkleRoot {
			return fmt.Errorf("invalid merkle root")
		}
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
		if len(CanonicalSigningBytes(tx)) == 0 {
			return fmt.Errorf("missing canonical signing bytes")
		}
		if !bc.validateTransaction(tx) {
			return fmt.Errorf("invalid transaction")
		}
	}
	return nil
}

func (bc *Blockchain) FinalizeBlocksAt(index uint64) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	if index >= uint64(len(bc.Chain)) {
		return fmt.Errorf("invalid block index")
	}
	bc.LastFinalized = index
	bc.FinalizedBlocks[index] = struct{}{}
	return nil
}

func (bc *Blockchain) IsFinalized(index uint64) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	_, ok := bc.FinalizedBlocks[index]
	return ok
}

func (bc *Blockchain) RLock()   { bc.mu.RLock() }
func (bc *Blockchain) RUnlock() { bc.mu.RUnlock() }

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

func (bc *Blockchain) Lock()   { bc.mu.Lock() }
func (bc *Blockchain) Unlock() { bc.mu.Unlock() }
