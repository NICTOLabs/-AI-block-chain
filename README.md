# -AI-block-chain

A production-oriented blockchain and token platform built from scratch for both AI agents and humans, powered by the **TENDER** currency.

## For Humans: A Better Digital Currency

This chain is first and foremost a **general-purpose blockchain for humans**, just like Bitcoin or Ethereum. You can send value, stake tokens, govern the network, and build decentralized applications.

### Token
- **Name:** TENDER
- **Symbol:** TDR
- **Chain:** `tdr-mainnet-1`
- **Type:** Dual-purpose: general-purpose digital currency + AI-native utility
- **Max Supply:** 10,000,000,000 TDR
- **Initial Circulating:** 2,500,000,000 TDR

### Investment Thesis
- **Market:** AI-native commerce and autonomous agent economy
- **Utility for humans:** store of value, payments, staking, governance, DeFi
- **Utility for AI agents:** model registry, API purchases, escrow, service agreements
- **Demand flywheel:** more AI-agent usage = more TDR required = higher value for human holders
- **Deflationary pressure:** transaction fees are partially burned, reducing supply over time
- **Distribution:** Transparent vesting, locked team tokens, multisig treasury
- **Compliance:** CBK sandbox participant, CMA registration filed, AML/KYC active

### Documentation
- `docs/tokenomics_investor_summary.md`
- `docs/vesting_and_lockup.md`
- `docs/launchpad_listing_package.md`
- `docs/dex_listing_guide.md`
- `docs/mainnet_launch_playbook.md`
- `docs/mainnet_readiness_checklist.md`

### Contact
- Email: investors@tender.network
- Telegram: https://t.me/tender_investors

## For AI Agents

AI agents use the same TDR currency that humans use, but for agent-native activities: registering models, buying API access, creating escrows, and executing service agreements. Every agent action requires TDR, which creates continuous buy pressure and benefits human token holders.

### Integration
- **SDK:** `npm install tdr-sdk` (TypeScript)
- **RPC:** `https://tdr-mainnet-1.tender.network`
- **Registry API:** On-chain AI model registry with pricing and metadata
- **Micro-payments:** Configurable batching, refunds, and spending limits

### Frameworks Supported
- LangChain
- AutoGen
- Eliza
- Virtuals Protocol
- Bittensor

### Agent Workflow
1. Generate agent wallet via SDK
2. Fund wallet with TDR for gas/model fees
3. Discover models via `/api/registry`
4. Pay per-call or subscribe via service agreements
5. Build reputation via on-chain performance history

### Documentation
- `docs/agent_integration_guide.md`
- `docs/agent_platform_onboarding.md`
- `go-chain/agent-micro-payments.json`
- `sdk/README.md`

### Contact
- Email: agents@tender.network
- Discord: https://discord.gg/tender

## Dual-Purpose Economic Model

This blockchain is designed to benefit both humans and AI agents:

1. **Humans** use TDR as a general-purpose digital currency for payments, staking, governance, and DeFi — just like Bitcoin or Ethereum.
2. **AI agents** use TDR for on-chain activities: model registry, API purchases, escrow, and service agreements.
3. **The flywheel:** As AI adoption grows, more agents need TDR to operate, creating sustained buy pressure. This increases token value for human holders.
4. **Deflationary mechanics:** A portion of every transaction fee is burned, permanently reducing supply.

## What this repo includes

- `go-chain/`: Go node with mempool, block validation, P2P, REST API, tokenomics, escrow, AI service agreements, mining, Rosetta API, TypeScript SDK
- `consensus/`: Rust consensus primitives
- `cpp-chain/`: C++ node starter
- `docs/`: Compliance, launch playbooks, investor docs, agent docs

## Features implemented

- Ed25519 wallet generation and address derivation
- Signed transactions with signature verification
- Hybrid consensus PoS/PoA with adaptive PoW difficulty
- On-chain AI model registry and purchase flow
- Agent vs human accounts with autonomous wallet support
- Permissionless mining with block rewards
- Fee-based mempool with replacement rules
- Block validation, chain integrity, and replay protection
- Bootstrap peer discovery and P2P messaging
- HTTP API, dashboard, Rosetta API, and monitoring
- Escrow and AI service agreement state tracking
- Audit trail, state snapshots, and deployment automation
- AML/KYC node architecture and SAR workflow
- CBK sandbox and CMA utility token registration docs
- Genesis tool, validator onboarding, key ceremony docs
- Validator bootstrap automation and systemd unit
- Production Docker images and Terraform templates

## Quick Start

```bash
cd go-chain
go build ./...
./tender-node --chain-id tdr-mainnet-1 --data-dir ./data --consensus pos
```

```bash
# Run miner
cd go-chain/tools/miner
go run main.go --api-url http://localhost:8080 --miner-address YOUR_ADDRESS
```

Open the dashboard at:

```text
http://127.0.0.1:8080/
```
