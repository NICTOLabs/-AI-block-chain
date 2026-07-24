package blockchain

import (
	"compress/gzip"
	"encoding/hex"
	"encoding/json"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	TxMerkleRoot string        `json:"tx_merkle_root,omitempty"`
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
	HogohogoPerTender = 10_000_000
	BaseFee           uint64 = 5 * HogohogoPerTender
	FeeMultiplier     uint64 = 2
	BurnRatePercent   uint64 = 1
	RewardRatePercent uint64 = 4
	MinStake          uint64 = 100 * HogohogoPerTender
	SlashPercent      uint64 = 10
	CurrencyName      string = "TENDER"
	CurrencySubunit   string = "HOGOHOGO"
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

type AuditEntry struct {
	Timestamp int64  `json:"timestamp"`
	Event     string `json:"event"`
	Actor     string `json:"actor"`
	Details   string `json:"details"`
}

type Wallet struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
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

type nodeState struct {
	Chain           []Block                        `json:"chain"`
	Pending         []Transaction                  `json:"pending"`
	Ledger          map[string]*Account            `json:"ledger"`
	Registry        map[string]ModelEntry          `json:"registry"`
	Consensus       string                         `json:"consensus"`
	Authorities     []string                       `json:"authorities"`
	TokenSupply     uint64                         `json:"token_supply"`
	Escrows         map[string]Escrow              `json:"escrows"`
	Proposals       map[string]GovernanceProposal  `json:"proposals"`
	Agreements      map[string]ServiceAgreement    `json:"agreements"`
	UsageMeters     map[string]UsageMeter          `json:"usage_meters"`
	UsedNonces      map[string]map[uint64]struct{} `json:"used_nonces"`
	SeenTxIDs       map[string]struct{}            `json:"seen_tx_ids"`
	AuditTrail      []AuditEntry                   `json:"audit_trail"`
	NextNonce       map[string]uint64              `json:"next_nonce"`
	FinalizedBlocks map[uint64]struct{}            `json:"finalized_blocks"`
	LastFinalized   uint64                         `json:"last_finalized"`
}

type Snapshot struct {
	Header struct {
		ChainID      string    `json:"chain_id"`
		GenesisHash  string    `json:"genesis_hash"`
		Height       uint64    `json:"height"`
		Timestamp    time.Time `json:"timestamp"`
		StateRoot    string    `json:"state_root"`
		TxCount      uint64    `json:"tx_count"`
		ValidatorSet string    `json:"validator_set"`
	} `json:"header"`
	Accounts      map[string]*Account   `json:"accounts"`
	Registry      map[string]ModelEntry `json:"registry"`
	Escrows       map[string]Escrow     `json:"escrows"`
	Proposals     map[string]GovernanceProposal `json:"governance_proposals"`
	Agreements    map[string]ServiceAgreement    `json:"service_agreements"`
	UsageMeters   map[string]UsageMeter `json:"usage_meters"`
	UsedNonces    map[string]map[uint64]struct{} `json:"used_nonces"`
	NextNonce     map[string]uint64      `json:"next_nonce"`
	SeenTxIDs     map[string]struct{}     `json:"seen_tx_ids"`
	Validators    map[string]Validator      `json:"validators"`
	AuditTrail    []AuditEntry         `json:"audit_trail"`
	TokenSupply   uint64               `json:"token_supply"`
	Wallets       map[string]ManagedWallet `json:"managed_wallets"`
}

type StateStore struct {
	mu       sync.Mutex
	dir      string
	latest   *Snapshot
	history  []string
	maxFiles int
}

const maxSnapshotSize = 50 << 20

func NewStateStore(dir string) (*StateStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	store := &StateStore{dir: dir, maxFiles: 64}
	if err := store.loadLatest(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *StateStore) stateRoot(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:8])
}

func (s *StateStore) loadLatest() error {
	files, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}
	var latest string
	var latestHeight uint64
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if len(name) < 12 || name[:8] != "snapshot" {
			continue
		}
		var h uint64
		if _, err := fmt.Sscanf(name, "snapshot_%d.json.gz", &h); err != nil {
			continue
		}
		if h > latestHeight {
			latestHeight = h
			latest = name
		}
	}
	if latest == "" {
		_ = os.MkdirAll(filepath.Join(s.dir, "data"), 0o755)
	}
	return nil
}


func (s *StateStore) readSnapshot(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	limited := io.LimitReader(gz, maxSnapshotSize+1)
	var snap Snapshot
	if err := json.NewDecoder(limited).Decode(&snap); err != nil {
		return err
	}
	s.latest = &snap
	return nil
}

func FormatAmount(amount uint64) string {
	tender := amount / HogohogoPerTender
	hogohogo := amount % HogohogoPerTender
	return fmt.Sprintf("%d TENDER %06d HOGOHOGO", tender, hogohogo)
}

func ParseAmount(text string) (uint64, error) {
	text = strings.TrimSpace(text)
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return 0, fmt.Errorf("empty amount")
	}
	tender, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	var hogohogo uint64
	if len(parts) >= 2 {
		v, e := strconv.ParseUint(parts[1], 10, 64)
		if e == nil && v < HogohogoPerTender {
			hogohogo = v
		}
	}
	return tender*HogohogoPerTender + hogohogo, nil
}

func ConsensusName(consensus ConsensusType) string {
	if consensus == ProofOfAuthority {
		return "poa"
	}
	return "pos"
}

func CurrencySymbol() string {
	return CurrencyName
}

func VerifyTransaction(tx Transaction) bool {
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
	return ed25519.Verify(ed25519.PublicKey(pubKey), CanonicalSigningBytes(tx), sig)
}

func LogJSON(event, actor, details string) {
	fmt.Printf("{\"ts\":%d,\"event\":\"%s\",\"actor\":\"%s\",\"details\":\"%s\"}\n", time.Now().Unix(), event, actor, details)
}

func NewRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
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
	payload := CanonicalSigningBytes(tx)
	hash := sha256.Sum256(payload)
tx.ID = hex.EncodeToString(hash[:])
	tx.Signature = hex.EncodeToString(ed25519.Sign(w.PrivateKey, payload))
	return tx
}

func CanonicalSigningBytes(tx Transaction) []byte {
	b := strings.Builder{}
	b.WriteString(tx.ChainID)
	b.WriteString("\x00")
	b.WriteString(tx.From)
	b.WriteString("\x00")
	b.WriteString(tx.FromPubKey)
	b.WriteString("\x00")
	b.WriteString(tx.To)
	b.WriteString("\x00")
	b.WriteString(strconv.FormatUint(tx.Amount, 10))
	b.WriteString("\x00")
	b.WriteString(strconv.FormatUint(tx.Fee, 10))
	b.WriteString("\x00")
	b.WriteString(strconv.FormatUint(tx.Nonce, 10))
	b.WriteString("\x00")
	b.WriteString(string(tx.TxType))
	b.WriteString("\x00")
	b.WriteString(tx.Payload)
	b.WriteString("\x00")
	b.WriteString(strconv.FormatInt(tx.Timestamp, 10))
	return []byte(b.String())
}
func CalculateMerkleRoot(transactions []Transaction) string {
	if len(transactions) == 0 {
		return ""
	}
	var hashes [][]byte
for _, tx := range transactions {
		sum := sha256.Sum256([]byte(tx.ID))
		hashes = append(hashes, sum[:])
	}
	for len(hashes) > 1 {
		var next [][]byte
		for i := 0; i < len(hashes); i += 2 {
			if i+1 < len(hashes) {
				c := append(append([]byte{}, hashes[i]...), hashes[i+1]...)
				out := sha256.Sum256(c)
				next = append(next, out[:])
			} else {
				next = append(next, hashes[i])
			}
		}
		hashes = next
	}
	return hex.EncodeToString(hashes[0])
}

func calculateHash(block Block) string {
	clone := block
	clone.BlockHash = ""
	data, _ := json.Marshal(clone)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
