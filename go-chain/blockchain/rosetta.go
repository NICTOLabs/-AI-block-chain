package blockchain

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RosettaNetworkIdentifier struct {
	Blockchain string `json:"blockchain"`
	Network    string `json:"network"`
}

type RosettaPartialBlockIdentifier struct {
	Index      int64  `json:"index"`
	Hash       string `json:"hash"`
	SubBlockIdentifier interface{} `json:"sub_block_identifier,omitempty"`
}

type RosettaAccountIdentifier struct {
	Address string `json:"address"`
	SubAccount *RosettaSubAccount `json:"sub_account,omitempty"`
}

type RosettaSubAccount struct {
	Address string `json:"address"`
	MerkleRoot string `json:"merkle_root,omitempty"`
}

type RosettaCurrency struct {
	Symbol   string `json:"symbol"`
	Decimals int64  `json:"decimals"`
}

type RosettaAmount struct {
	Value    string        `json:"value"`
	Currency RosettaCurrency `json:"currency"`
}

type RosettaOperation struct {
	OperationIdentifier *RosettaOperationIdentifier `json:"operation_identifier"`
	Type                 string                      `json:"type"`
	Status               string                      `json:"status"`
	Account              *RosettaAccountIdentifier   `json:"account,omitempty"`
	Amount               *RosettaAmount              `json:"amount,omitempty"`
	CoinChange           *RosettaCoinChange          `json:"coin_change,omitempty"`
}

type RosettaOperationIdentifier struct {
	Index int64 `json:"index"`
}

type RosettaCoinChange struct {
	CoinIdentifier *RosettaCoinIdentifier `json:"coin_identifier,omitempty"`
	CoinAction      string                  `json:"coin_action"`
}

type RosettaCoinIdentifier struct {
	Identifier string `json:"identifier"`
}

type RosettaTransaction struct {
	TransactionIdentifier *RosettaPartialBlockIdentifier `json:"transaction_identifier"`
	Operations            []RosettaOperation              `json:"operations"`
	Metadata              map[string]string               `json:"metadata,omitempty"`
}

type RosettaBlock struct {
	BlockIdentifier       *RosettaPartialBlockIdentifier `json:"block_identifier"`
	ParentBlockIdentifier *RosettaPartialBlockIdentifier `json:"parent_block_identifier"`
	Timestamp             int64                           `json:"timestamp"`
	Transactions          []RosettaTransaction             `json:"transactions"`
	Metadata              map[string]string               `json:"metadata,omitempty"`
}

type RosettaNetworkStatusResponse struct {
	CurrentBlockIdentifier RosettaPartialBlockIdentifier `json:"current_block_identifier"`
	OldestBlockIdentifier  RosettaPartialBlockIdentifier `json:"oldest_block_identifier"`
	GenesisBlockIdentifier RosettaPartialBlockIdentifier `json:"genesis_block_identifier"`
	CurrentBlockTimestamp  int64                           `json:"current_block_timestamp"`
	SyncStatus             map[string]interface{}         `json:"sync_status,omitempty"`
	Peers                  []RosettaPeer                  `json:"peers,omitempty"`
}

type RosettaPeer struct {
	PeerID string `json:"peer_id"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type RosettaBlockResponse struct {
	Block RosettaBlock `json:"block"`
}

type RosettaConstructionSubmitRequest struct {
	NetworkIdentifier  RosettaNetworkIdentifier                   `json:"network_identifier"`
	Transaction        string                                     `json:"transaction"`
	Signature          string                                     `json:"signature"`
	PublicKeys         []RosettaPublicKey                         `json:"public_keys,omitempty"`
	Metadata           map[string]string                          `json:"metadata,omitempty"`
}

type RosettaPublicKey struct {
	Bytes         string `json:"bytes"`
	CurveType     string `json:"curve_type"`
}

type RosettaConstructionSubmitResponse struct {
	TransactionIdentifier RosettaPartialBlockIdentifier `json:"transaction_identifier"`
	Status                map[string]interface{}         `json:"status,omitempty"`
}

func RosettaNetworkStatusHandler(bc *Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		bc.mu.RLock()
		defer bc.mu.RUnlock()

		resp := RosettaNetworkStatusResponse{}

		if len(bc.Chain) == 0 {
			resp.GenesisBlockIdentifier = RosettaPartialBlockIdentifier{Index: 0, Hash: "genesis"}
			resp.OldestBlockIdentifier = resp.GenesisBlockIdentifier
			resp.CurrentBlockIdentifier = RosettaPartialBlockIdentifier{Index: 0, Hash: "genesis"}
			resp.CurrentBlockTimestamp = time.Now().Unix()
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		resp.GenesisBlockIdentifier = RosettaPartialBlockIdentifier{Index: 0, Hash: bc.Chain[0].BlockHash}
		resp.OldestBlockIdentifier = RosettaPartialBlockIdentifier{Index: 0, Hash: bc.Chain[0].BlockHash}
		last := bc.Chain[len(bc.Chain)-1]
		resp.CurrentBlockIdentifier = RosettaPartialBlockIdentifier{Index: int64(last.Index), Hash: last.BlockHash}
		resp.CurrentBlockTimestamp = last.Timestamp

		resp.Peers = make([]RosettaPeer, 0, len(bc.AuditTrail))
		peerMap := make(map[string]bool)
		for _, entry := range bc.AuditTrail {
			if entry.Event == "block_mined" && entry.Actor != "" {
				if !peerMap[entry.Actor] {
					peerMap[entry.Actor] = true
					resp.Peers = append(resp.Peers, RosettaPeer{
						PeerID: entry.Actor,
						Metadata: map[string]string{"last_event": entry.Event},
					})
				}
			}
		}

		_ = json.NewEncoder(w).Encode(resp)
	}
}

func RosettaBlockHandler(bc *Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		bc.mu.RLock()
		defer bc.mu.RUnlock()

		path := strings.TrimPrefix(r.URL.Path, "/block/")
		if path == "" || path == "/" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		segments := strings.Split(strings.TrimRight(path, "/"), "/")

		targetIndex, err := strconv.ParseInt(segments[0], 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		for _, block := range bc.Chain {
			if int64(block.Index) == targetIndex {
				resp := RosettaBlock{
					BlockIdentifier:       &RosettaPartialBlockIdentifier{Index: int64(block.Index), Hash: block.BlockHash},
					ParentBlockIdentifier: &RosettaPartialBlockIdentifier{Index: int64(block.Index) - 1, Hash: block.PreviousHash},
					Timestamp:             block.Timestamp,
					Transactions:          make([]RosettaTransaction, 0, len(block.Transactions)),
					Metadata:              map[string]string{"author": block.Author},
				}

				for _, tx := range block.Transactions {
					ops := []RosettaOperation{{
						OperationIdentifier: &RosettaOperationIdentifier{Index: 0},
						Type:                 string(tx.TxType),
						Status:               "SUCCESS",
						Account:              &RosettaAccountIdentifier{Address: tx.From},
						Amount:               &RosettaAmount{Value: fmt.Sprintf("-%d", tx.Amount), Currency: RosettaCurrency{Symbol: "TDR", Decimals: 8}},
					}, {
						OperationIdentifier: &RosettaOperationIdentifier{Index: 1},
						Type:                 string(tx.TxType),
						Status:               "SUCCESS",
						Account:              &RosettaAccountIdentifier{Address: tx.To},
						Amount:               &RosettaAmount{Value: fmt.Sprintf("%d", tx.Amount), Currency: RosettaCurrency{Symbol: "TDR", Decimals: 8}},
					}}

					rawBytes, _ := json.Marshal(tx)
					txHash := sha256.Sum256(rawBytes)
					resp.Transactions = append(resp.Transactions, RosettaTransaction{
						TransactionIdentifier: &RosettaPartialBlockIdentifier{Index: int64(block.Index), Hash: hex.EncodeToString(txHash[:])},
						Operations:            ops,
						Metadata:              map[string]string{"tx_id": tx.ID},
					})
				}

				_ = json.NewEncoder(w).Encode(resp)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}
}

func RosettaConstructionSubmitHandler(bc *Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req RosettaConstructionSubmitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(req.Transaction) == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var tx Transaction
		if err := json.Unmarshal([]byte(req.Transaction), &tx); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		bc.mu.Lock()
		tx.ID = fmt.Sprintf("tx-%d", time.Now().UnixNano())
		if tx.Nonce == 0 {
			tx.Nonce = uint64(len(bc.Pending) + 1)
		}
		tx.Timestamp = time.Now().Unix()

		if tx.Signature == "" && len(req.PublicKeys) > 0 {
			pubBytes, err := hex.DecodeString(req.PublicKeys[0].Bytes)
			if err == nil {
				if len(pubBytes) == sha256.Size {
					tx.From = hex.EncodeToString(pubBytes)
				}
			}
		}

		bc.Pending = append(bc.Pending, tx)
		bc.mu.Unlock()

		txID := tx.ID
		txHash := sha256.Sum256([]byte(txID + req.NetworkIdentifier.Network))
		resp := RosettaConstructionSubmitResponse{
			TransactionIdentifier: RosettaPartialBlockIdentifier{Index: int64(bc.Chain[len(bc.Chain)-1].Index), Hash: hex.EncodeToString(txHash[:])},
			Status:                map[string]interface{}{"accepted": "true"},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func hashInt(v int64) string {
	var b bytes.Buffer
	b.WriteString(strconv.FormatInt(v, 10))
	h := sha256.Sum256(b.Bytes())
	return hex.EncodeToString(h[:])
}

func validateRosettaTransaction(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return fmt.Errorf("transaction payload cannot be empty")
	}
	var t map[string]any
	if err := json.Unmarshal([]byte(raw), &t); err != nil {
		return fmt.Errorf("transaction payload must be valid JSON")

	}
	if _, ok := t["from"]; !ok {
		return fmt.Errorf("transaction payload must include sender address")
	}
	if _, ok := t["to"]; !ok {
		return fmt.Errorf("transaction payload must include recipient address")
	}
	return nil
}

func buildRosettaTx(raw string) (Transaction, error) {
	var tx Transaction
	if err := json.Unmarshal([]byte(raw), &tx); err != nil {
		return Transaction{}, fmt.Errorf("invalid transaction JSON: %w", err)
	}
	if tx.From == "" || tx.To == "" {
		return Transaction{}, fmt.Errorf("transaction must include from and to addresses")
	}
	if tx.Amount == 0 {
		return Transaction{}, fmt.Errorf("transaction amount must be greater than 0")
	}
	return tx, nil
}

func formatRosettaAmount(val uint64, decimals int) string {
	s := fmt.Sprintf("%d", val)
	if decimals > 0 {
		if len(s) > decimals {
			return s[:len(s)-decimals] + "." + s[len(s)-decimals:]
		}
		padded := strings.Repeat("0", decimals-len(s)) + s
		return "0." + padded
	}
	return s
}

func parseRosettaAmount(s string) uint64 {
	parts := strings.Split(s, ".")
	if len(parts) == 1 {
		n, _ := strconv.ParseUint(parts[0], 10, 64)
		return n * 100000000
	}
	if len(parts) == 2 {
		intPart, _ := strconv.ParseUint(parts[0], 10, 64)
		decPart := parts[1]
		if len(decPart) > 8 {
			decPart = decPart[:8]
		}
		decPart = decPart + strings.Repeat("0", 8-len(decPart))
		decN, _ := strconv.ParseUint(decPart, 10, 64)
		return intPart*100000000 + decN
	}
	return 0
}

func checkRosettaTxBalance(bc *Blockchain, from string, amount uint64) bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	account, ok := bc.Ledger[from]
	if !ok {
		return false
	}
	return account.Balance >= amount
}

func validateRosettaSignature(tx Transaction, pubHex, sigHex string) bool {
	pubBytes, err := hex.DecodeString(pubHex)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != ed25519.SignatureSize {
		return false
	}
	raw, _ := json.Marshal(tx)
	return ed25519.Verify(ed25519.PublicKey(pubBytes), raw, sigBytes)
}

func rosettaTxToCanonicalJSON(tx Transaction) string {
	txCopy := tx
	txCopy.Signature = ""
	b, _ := json.Marshal(txCopy)
	return string(b)
}
