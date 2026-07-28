# Architecture

TENDER is implemented across multiple languages:

- Go node: Full-featured with REST API, dashboard, mempool, escrow, and AI service agreements
- Rust node: Lightweight starter with wallet generation, signing, and registry
- C++ node: Minimal high-performance node for embedded and edge deployments
- TypeScript SDK: Browser and Node.js client for dApp integration

## Layered Design

1. Transport Layer: TCP with Noise handshake, stream multiplexing, mDNS discovery, NAT traversal
2. Consensus Layer: Tendermint PoS with proposal, prevote, precommit, commit phases
3. State Layer: Chain, mempool, ledger, registry, escrow, agreements
4. API Layer: REST endpoints for wallets, transfers, staking, registry, monitoring
