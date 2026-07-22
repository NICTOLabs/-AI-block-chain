# Miner Operations Guide

## Overview
TDR is a mineable Proof-of-Work blockchain. Anyone can mine blocks and earn block rewards.

## Block Reward
- **Base Reward:** 5,000,000 TDR per block
- **Fee Burn:** 1% of transaction fees are burned
- **Dynamic Difficulty:** Adjusts every block to maintain ~5 second block times

## Getting Started

### 1. Get a Mining Address
```bash
# Generate a new wallet
go run ./tools/wallet/main.go --output miner-wallet.json

# Or use existing address
echo "your-miner-address-here"
```

### 2. Fund Your Miner
```bash
# Send TDR to your miner address to pay for transaction fees
curl -X POST http://localhost:8080/api/transfer \
  -H "Content-Type: application/json" \
  -d '{"from":"your-address","to":"miner-address","amount":1000,"fee":5,"nonce":1,"tx_type":"TRANSFER"}'
```

### 3. Start Mining
```bash
# Using the miner tool
cd go-chain/tools/miner
go run main.go

# Or using the API directly
curl -X POST http://localhost:8080/api/mine \
  -H "Content-Type: application/json" \
  -d '{"miner_address":"your-miner-address"}'
```

## Mining Solo

### Option A: Node Mining
```bash
# Start your node with mining enabled
./tender-node --chain-id tdr-mainnet-1 --miner-address YOUR_ADDRESS
```

### Option B: Miner Tool
```bash
# Build and run miner
cd go-chain/tools/miner
go build -o miner .
./miner --api-url http://localhost:8080 --miner-address YOUR_ADDRESS --threads 4
```

## Mining Pool

### Pool Endpoint
```bash
# Get work from pool
GET /pool/work

# Submit mined block
POST /pool/submit
{
  "block": { ... },
  "miner_address": "YOUR_ADDRESS"
}
```

## Rewards Distribution
- **Solo Miner:** 100% of block reward to miner address
- **Pool Miner:** Payout based on contributed shares
- **Minimum Payout:** 100 TDR
- **Payout Frequency:** Every 10 blocks

## Monitoring
```bash
# Check your miner status
curl http://localhost:8080/api/miner/status?address=YOUR_ADDRESS

# View blocks mined
curl http://localhost:8080/api/chain | jq '.chain[] | select(.author == "YOUR_ADDRESS")'
```

## Economics
- **Block Time:** ~5 seconds
- **Daily Blocks:** ~17,280
- **Daily Rewards:** ~86,400,000,000 TDR
- **Halving:** No halving; fixed reward with inflation decay

## Requirements
- CPU: 4+ cores recommended
- RAM: 8GB minimum
- Storage: 100GB+ SSD
- Network: Stable connection to node

## Troubleshooting
| Issue | Solution |
|-------|----------|
| "insufficient funds" | Fund miner address for transaction fees |
| "no work available" | Wait for pending transactions or use `--force-mine` |
| "invalid block hash" | Difficulty too high, increase threads or wait |
| "connection refused" | Check node is running and API port is open |

## Join the Network
- Discord: https://discord.gg/tender
- Telegram: https://t.me/tender_miners
- Email: mining@tender.network
