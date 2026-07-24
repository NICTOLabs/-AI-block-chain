package main

import (
	"flag"
	"fmt"
	"strings"

	blockchain "ai_block_chain_go/blockchain"
	p2p "ai_block_chain_go/p2p"
	api "ai_block_chain_go/api"
)

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
	faucetAmount := flag.Int("faucet-amount", 0, "Default faucet funding amount")
	chainID := flag.String("chain-id", "tdr-mainnet-1", "Chain ID for replay protection")
	flag.Parse()

	envCfg := api.ServerConfigFromEnv()
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
	if *faucetAmount != 0 {
		envCfg.FaucetAmount = uint64(*faucetAmount)
	}

	var chainConsensus blockchain.ConsensusType
	if envCfg.Consensus == "poa" {
		chainConsensus = blockchain.ProofOfAuthority
	} else {
		chainConsensus = blockchain.ProofOfStake
	}

	chain := blockchain.NewBlockchain(chainConsensus, envCfg.DataDir, *chainID)
	peers := []string{}
	if *peer != "" {
		peers = append(peers, *peer)
	}
	if *bootstrapPeers != "" {
		for _, peer := range strings.Split(*bootstrapPeers, ",") {
			peer = strings.TrimSpace(peer)
			if peer == "" {
				continue
			}
			peers = append(peers, peer)
		}
	}
	p2pNode := p2p.NewP2PNode(fmt.Sprintf("0.0.0.0:%d", envCfg.P2PPort), peers, chain, envCfg.StrictP2P)

	go p2pNode.Start()
	go p2pNode.ConnectToPeers()
	go api.StartAPI(chain, envCfg.APIPort, p2pNode, envCfg)

	blockchain.LogJSON("node_start", "node", fmt.Sprintf("currency=%s api_port=%d p2p_port=%d consensus=%s chain_id=%s faucet_amount=%d", blockchain.CurrencySymbol(), envCfg.APIPort, envCfg.P2PPort, envCfg.Consensus, *chainID, envCfg.FaucetAmount))
	<-p2pNode.Shutdown()
}
