package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type Transaction struct {
	ID         string `json:"id,omitempty"`
	From       string `json:"from"`
	FromPubKey string `json:"from_pubkey"`
	To         string `json:"to"`
	Amount     uint64 `json:"amount"`
	Fee        uint64 `json:"fee,omitempty"`
	Nonce      uint64 `json:"nonce,omitempty"`
	TxType     string `json:"tx_type"`
	Payload    string `json:"payload,omitempty"`
	Signature  string `json:"signature,omitempty"`
	Timestamp  int64  `json:"timestamp"`
	ChainID    string `json:"chain_id,omitempty"`
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

type nodeState struct {
	Chain       []Block       `json:"chain"`
	Pending     []Transaction `json:"pending"`
	ChainID     string        `json:"chain_id"`
	TokenSupply uint64        `json:"token_supply"`
}

type RosettaServer struct {
	state *nodeState
	port  string
}

func NewRosettaServer(state *nodeState, port string) *RosettaServer {
	return &RosettaServer{state: state, port: port}
}

func (s *RosettaServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/network/status", s.networkStatus)
	mux.HandleFunc("/block/", s.blockByIndex)
	mux.HandleFunc("/construction/submit", s.constructionSubmit)
	mux.HandleFunc("/health", s.health)

	server := &http.Server{
		Addr:         ":" + s.port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("Rosetta server listening on :%s", s.port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("rosetta server error: %v", err)
	}
}

func (s *RosettaServer) networkStatus(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{
		"current_block_identifier": map[string]any{"index": 0, "hash": "genesis"},
		"oldest_block_identifier":  map[string]any{"index": 0, "hash": "genesis"},
		"genesis_block_identifier": map[string]any{"index": 0, "hash": "genesis"},
		"current_block_timestamp":  time.Now().Unix(),
		"peers":                    []string{},
	}
	if len(s.state.Chain) > 0 {
		last := s.state.Chain[len(s.state.Chain)-1]
		resp["current_block_identifier"] = map[string]any{"index": last.Index, "hash": last.BlockHash}
		resp["current_block_timestamp"] = last.Timestamp
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *RosettaServer) blockByIndex(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/block/"):]
	if path == "" {
		http.Error(w, "missing block index", http.StatusBadRequest)
		return
	}
	var targetIndex uint64
	if _, err := fmt.Sscanf(path, "%d", &targetIndex); err != nil {
		http.Error(w, "invalid block index", http.StatusBadRequest)
		return
	}
	for _, block := range s.state.Chain {
		if block.Index == targetIndex {
			resp := map[string]any{
				"block": map[string]any{
					"block_identifier":       map[string]any{"index": block.Index, "hash": block.BlockHash},
					"parent_block_identifier": map[string]any{"index": block.Index - 1, "hash": block.PreviousHash},
					"timestamp":              block.Timestamp,
					"transactions":           block.Transactions,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}
	http.Error(w, "block not found", http.StatusNotFound)
}

func (s *RosettaServer) constructionSubmit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Transaction string `json:"transaction"`
		Network     struct {
			Blockchain string `json:"blockchain"`
			Network    string `json:"network"`
		} `json:"network_identifier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Transaction == "" {
		http.Error(w, "missing transaction", http.StatusBadRequest)
		return
	}
	var tx Transaction
	if err := json.Unmarshal([]byte(req.Transaction), &tx); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tx.ID = fmt.Sprintf("tx-%d", time.Now().UnixNano())
	s.state.Pending = append(s.state.Pending, tx)
	txHash := sha256.Sum256([]byte(tx.ID + req.Network.Network))
	resp := map[string]any{
		"transaction_identifier": map[string]any{"hash": hex.EncodeToString(txHash[:])},
		"status":                 map[string]any{"accepted": true},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *RosettaServer) health(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "ok",
		"service":   "rosetta",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func verifyTransaction(tx Transaction) bool {
	if tx.ChainID == "" {
		return false
	}
	pubKey, err := hex.DecodeString(tx.FromPubKey)
	if err != nil || len(pubKey) != ed25519.PublicKeySize {
		return false
	}
	addressBytes := sha256.Sum256(pubKey)
	if tx.From != hex.EncodeToString(addressBytes[:]) {
		return false
	}
	sig, err := hex.DecodeString(tx.Signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	payload := tx
	payload.Signature = ""
	payload.ID = ""
	data, _ := json.Marshal(payload)
	return ed25519.Verify(ed25519.PublicKey(pubKey), data, sig)
}

func main() {
	port := "8083"
	if p := os.Getenv("ROSETTA_PORT"); p != "" {
		port = p
	}

	statePath := ""
	if s := os.Getenv("CHAIN_STATE_PATH"); s != "" {
		statePath = s
	}

	state := &nodeState{ChainID: "tdr-mainnet-1"}
	if statePath != "" {
		data, err := os.ReadFile(statePath)
		if err == nil {
			_ = json.Unmarshal(data, state)
		}
	}

	server := NewRosettaServer(state, port)
	server.Start()
}
