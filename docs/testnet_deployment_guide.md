# TDR Testnet Deployment Guide

## Prerequisites

- Go 1.22+
- 4 CPU cores, 8GB RAM, 500GB SSD
- Static public IP or DNS
- Open ports: 8080 (API), 3030 (P2P), 9090 (Metrics)

## Step 1: Clone and Build

```bash
git clone https://github.com/NICTOLabs/-AI-block-chain.git
cd -AI-block-chain/go-chain
go build -o tender-node .
```

## Step 2: Generate Validator Identity

```bash
go run validator-onboard/main.go \
  --name validator-kenya-01 \
  --country KE \
  --region Nairobi \
  --stake 10000 \
  --network tdr-testnet \
  --output validator-kenya-01
```

## Step 3: Configure Node

```bash
export TENDER_API_KEY="your-secure-api-key"
export TENDER_CONSENSUS="pos"
export TENDER_DATA_DIR="./data"
export TENDER_ENABLE_AUTH=true
export TENDER_STRICT_P2P=true
```

## Step 4: Start Node

```bash
./tender-node \
  --api-port 8080 \
  --p2p-port 3030 \
  --bootstrap-peers "seed1.tender.network:3030,seed2.tender.network:3030" \
  --chain-id tdr-testnet-1 \
  --consensus pos
```

## Step 5: Verify

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/monitoring
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Peer connection timeout | Check firewall, verify bootstrap peers |
| Chain not syncing | Delete data dir, resync from genesis |
| High memory usage | Reduce max peers, enable state pruning |
