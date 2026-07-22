package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RosettaNetworkStatusResponse struct {
	NetworkIdentifier struct {
		Blockchain string `json:"blockchain"`
		Network    string `json:"network"`
	} `json:"network_identifier"`
	GenesisBlockIdentifier struct {
		Index int64  `json:"index"`
		Hash  string `json:"hash"`
	} `json:"genesis_block_identifier"`
	OldestBlockIdentifier struct {
		Index int64  `json:"index"`
		Hash  string `json:"hash"`
	} `json:"oldest_block_identifier"`
	CurrentBlockIdentifier struct {
		Index int64  `json:"index"`
		Hash  string `json:"hash"`
	} `json:"current_block_identifier"`
	CurrentBlockTimestamp int64 `json:"current_block_timestamp"`
}

type RosettaBlockResponse struct {
	Block struct {
		BlockIdentifier struct {
			Index int64  `json:"index"`
			Hash  string `json:"hash"`
		} `json:"block_identifier"`
		ParentBlockIdentifier struct {
			Index int64  `json:"index"`
			Hash  string `json:"hash"`
		} `json:"parent_block_identifier"`
		Timestamp int64 `json:"timestamp"`
		Transactions []map[string]any `json:"transactions"`
	} `json:"block"`
}

type RosettaConstructionSubmitRequest struct {
	Transaction string `json:"transaction"`
}

type RosettaConstructionSubmitResponse struct {
	TransactionIdentifier struct {
		Hash string `json:"hash"`
	} `json:"transaction_identifier"`
}

func RosettaNetworkStatusHandler(bc *Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := RosettaNetworkStatusResponse{}
		resp.NetworkIdentifier.Blockchain = "tdr"
		resp.NetworkIdentifier.Network = "mainnet"
		if len(bc.Chain) > 0 {
			resp.GenesisBlockIdentifier.Index = 0
			resp.GenesisBlockIdentifier.Hash = bc.Chain[0].BlockHash
			resp.OldestBlockIdentifier = resp.GenesisBlockIdentifier
			last := bc.Chain[len(bc.Chain)-1]
			resp.CurrentBlockIdentifier.Index = int64(last.Index)
			resp.CurrentBlockIdentifier.Hash = last.BlockHash
			resp.CurrentBlockTimestamp = last.Timestamp
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func RosettaBlockHandler(bc *Blockchain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		segments := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if len(segments) < 2 || segments[0] != "block" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		targetIndex, err := strconv.ParseInt(segments[1], 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for _, block := range bc.Chain {
			if int64(block.Index) == targetIndex {
				resp := RosettaBlockResponse{}
				resp.Block.BlockIdentifier.Index = int64(block.Index)
				resp.Block.BlockIdentifier.Hash = block.BlockHash
				resp.Block.ParentBlockIdentifier.Index = int64(block.Index) - 1
				resp.Block.ParentBlockIdentifier.Hash = block.PreviousHash
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
		if req.Transaction == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		resp := RosettaConstructionSubmitResponse{}
		resp.TransactionIdentifier.Hash = fmt.Sprintf("tx-%d", time.Now().UnixNano())
		_ = json.NewEncoder(w).Encode(resp)
	}
}
