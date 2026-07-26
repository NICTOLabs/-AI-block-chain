package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ethereumLockEvent struct {
	TxHash    string `json:"tx_hash"`
	Block     uint64 `json:"block_number"`
	LogIndex  uint   `json:"log_index"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	Recipient string `json:"recipient"`
}

type ethereumListener struct {
	rpcURL         string
	relayer        *relayer
	contract       common.Address
	topic          common.Hash
	cancel         context.CancelFunc
	processed      map[common.Hash]struct{}
	mu             sync.RWMutex
}

func newEthereumListener(rpcURL string, relayer *relayer) (*ethereumListener, error) {
	if strings.TrimSpace(rpcURL) == "" {
		return nil, errors.New("missing TENDER_ETHEREUM_RPC_URL")
	}
	return &ethereumListener{rpcURL: rpcURL, relayer: relayer, processed: map[common.Hash]struct{}{}}, nil
}

func (l *ethereumListener) start(ctx context.Context) error {
	client, err := ethclient.DialContext(ctx, l.rpcURL)
	if err != nil {
		return fmt.Errorf("ethclient dial: %w", err)
	}
	logs := make(chan types.Log)
	query := ethereum.FilterQuery{
		Addresses: []common.Address{l.contract},
		Topics:    [][]common.Hash{{l.topic}},
	}
	sub, err := client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("subscribe logs: %w", err)
	}
	cctx, cancel := context.WithCancel(ctx)
	l.cancel = cancel
	log.Printf("ethereum listener started rpc=%s", l.rpcURL)
	for {
		select {
		case <-cctx.Done():
			sub.Unsubscribe()
			return nil
		case err := <-sub.Err():
			log.Printf("ethereum subscription error: %v", err)
			sub.Unsubscribe()
			sub, err = client.SubscribeFilterLogs(cctx, query, logs)
			if err != nil {
				log.Printf("ethereum resubscribe failed: %v", err)
				return err
			}
			continue
		case lg := <-logs:
			l.handleLog(cctx, lg)
		}
	}
}

func (l *ethereumListener) stop() {
	if l.cancel != nil {
		l.cancel()
	}
}

func (l *ethereumListener) handleLog(ctx context.Context, lg types.Log) {
	l.mu.Lock()
	if _, seen := l.processed[lg.TxHash]; seen {
		l.mu.Unlock()
		return
	}
	l.processed[lg.TxHash] = struct{}{}
	l.mu.Unlock()
	event, err := decodeLockEvent(lg)
	if err != nil {
		log.Printf("lock decode failed tx=%s err=%v", lg.TxHash.Hex(), err)
		return
	}
	if event.Recipient == "" {
		event.Recipient = event.From
	}
	l.relayer.handleEthereumLock(ctx, event)
}

func decodeLockEvent(lg types.Log) (ethereumLockEvent, error) {
	var out ethereumLockEvent
	out.TxHash = lg.TxHash.Hex()
	out.Block = lg.BlockNumber
	out.LogIndex = uint(lg.Index)
	if len(lg.Data) >= 64 {
		out.From = common.BytesToAddress(lg.Data[0:32]).Hex()
		out.To = common.BytesToAddress(lg.Data[32:64]).Hex()
	}
	if len(lg.Data) >= 96 {
		out.Recipient = common.BytesToAddress(lg.Data[64:96]).Hex()
		out.Amount = new(big.Int).SetBytes(lg.Data[96:128]).Uint64()
	}
	return out, nil
}
