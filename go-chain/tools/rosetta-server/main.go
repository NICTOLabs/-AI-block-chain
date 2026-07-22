package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type RosettaServer struct {
	chain *Blockchain
	port  string
}

func NewRosettaServer(chain *Blockchain, port string) *RosettaServer {
	return &RosettaServer{chain: chain, port: port}
}

func (s *RosettaServer) Start() {
	r := mux.NewRouter()
	r.HandleFunc("/network/status", s.networkStatus).Methods("GET")
	r.HandleFunc("/block", s.blockByIndex).Methods("GET")
	r.HandleFunc("/construction/submit", s.constructionSubmit).Methods("POST")
	r.HandleFunc("/health", s.health).Methods("GET")

	server := &http.Server{
		Addr:         ":" + s.port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Rosetta server listening on :%s", s.port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("rosetta server error: %v", err)
		}
	}()
}

func (s *RosettaServer) Stop(ctx context.Context) {
	_ = http.NewRequest("POST", "http://localhost"+s.port+"/shutdown", nil)
}

type RosettaNetworkStatusResponse struct {
	CurrentBlockIdentifier struct {
		Index int64  `json:"index"`
		Hash  string `json:"hash"`
	} `json:"current_block_identifier"`
	OldestBlockIdentifier struct {
		Index int64  `json:"index"`
		Hash  string `json:"hash"`
	} `json:"oldest_block_identifier"`
	GenesisBlockIdentifier struct {
		Index int64  `json:"index"`
		Hash  string `json:"hash"`
	} `json:"genesis_block_identifier"`
	CurrentBlockTimestamp int64 `json:"current_block_timestamp"`
	Peers                []string `json:"peers"`
}

func (s *RosettaServer) networkStatus(w http.ResponseWriter, r *http.Request) {
	s.chain.mu.RLock()
	defer s.chain.mu.RUnlock()
	resp := RosettaNetworkStatusResponse{}
	if len(s.chain.Chain) > 0 {
		resp.GenesisBlockIdentifier.Index = 0
		resp.GenesisBlockIdentifier.Hash = s.chain.Chain[0].BlockHash
		resp.OldestBlockIdentifier = resp.GenesisBlockIdentifier
		last := s.chain.Chain[len(s.chain.Chain)-1]
		resp.CurrentBlockIdentifier.Index = int64(last.Index)
		resp.CurrentBlockIdentifier.Hash = last.BlockHash
		resp.CurrentBlockTimestamp = last.Timestamp
	}
	resp.Peers = []string{}
	_ = json.NewEncoder(w).Encode(resp)
}

type RosettaBlockResponse struct {
	Block struct {
		BlockIdentifier       map[string]any `json:"block_identifier"`
		ParentBlockIdentifier map[string]any `json:"parent_block_identifier"`
		Timestamp             int64          `json:"timestamp"`
		Transactions          []map[string]any `json:"transactions"`
	} `json:"block"`
}

func (s *RosettaServer) blockByIndex(w http.ResponseWriter, r *http.Request) {
	indexStr := r.URL.Query().Get("block_index")
	if indexStr == "" {
		http.Error(w, "missing block_index", http.StatusBadRequest)
		return
	}
	var targetIndex uint64
	if _, err := fmt.Sscanf(indexStr, "%d", &targetIndex); err != nil {
		http.Error(w, "invalid block_index", http.StatusBadRequest)
		return
	}
	s.chain.mu.RLock()
	defer s.chain.mu.RUnlock()
	for _, block := range s.chain.Chain {
		if block.Index == targetIndex {
			resp := RosettaBlockResponse{}
			resp.Block.BlockIdentifier = map[string]any{"index": block.Index, "hash": block.BlockHash}
			resp.Block.ParentBlockIdentifier = map[string]any{"index": block.Index - 1, "hash": block.PreviousHash}
			resp.Block.Timestamp = block.Timestamp
			for _, tx := range block.Transactions {
				resp.Block.Transactions = append(resp.Block.Transactions, map[string]any{
					"transaction_identifier": map[string]any{"hash": tx.ID},
					"operations": []map[string]any{{
						"operation_identifier": map[string]any{"index": 0},
						"type": "TRANSFER",
						"status": "SUCCESS",
						"account": map[string]any{"address": tx.From},
						"amount": map[string]any{"value": fmt.Sprintf("-%d", tx.Amount), "currency": map[string]any{"symbol": "TDR", "decimals": 8}},
					}, {
						"operation_identifier": map[string]any{"index": 1},
						"type": "TRANSFER",
						"status": "SUCCESS",
						"account": map[string]any{"address": tx.To},
						"amount": map[string]any{"value": fmt.Sprintf("%d", tx.Amount), "currency": map[string]any{"symbol": "TDR", "decimals": 8}},
					}},
				})
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}
	http.Error(w, "block not found", http.StatusNotFound)
}

type RosettaConstructionSubmitRequest struct {
	Transaction string `json:"transaction"`
	PublicKeys  []struct {
		Bytes     string `json:"bytes"`
		CurveType string `json:"curve_type"`
	} `json:"public_keys"`
}

type RosettaConstructionSubmitResponse struct {
	TransactionIdentifier struct {
		Hash string `json:"hash"`
	} `json:"transaction_identifier"`
	Status map[string]string `json:"status"`
}

func (s *RosettaServer) constructionSubmit(w http.ResponseWriter, r *http.Request) {
	var req RosettaConstructionSubmitRequest
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
	s.chain.mu.Lock()
	s.chain.Pending = append(s.chain.Pending, tx)
	s.chain.mu.Unlock()
	h := sha256.Sum256([]byte(tx.ID + s.chain.ChainID))
	resp := RosettaConstructionSubmitResponse{
		TransactionIdentifier: struct{ Hash string }{Hash: hex.EncodeToString(h[:])},
		Status:                map[string]string{"accepted": "true"},
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *RosettaServer) health(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":    "ok",
		"service":   "rosetta",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
