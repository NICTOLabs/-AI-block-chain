# AI Blockchain Design

## Purpose

This project is a production-oriented blockchain platform built from scratch to serve both AI agents and humans. It supports:

- micropayments for compute and task execution
- autonomous AI wallets and transaction signing
- on-chain AI model registry and metadata
- agent-to-agent communication and payment flows
- AI-owned wallets for API keys and service access

## Hybrid Consensus

The chain uses a hybrid consensus design:

- Proof-of-Stake (PoS): token holders may stake to secure the network and earn fees
- Proof-of-Authority (PoA): a trusted authority set validates critical state transitions for AI registry and agent onboarding

This hybrid approach allows the chain to support both decentralization and deterministic governance for AI agent identity and registry operations.

## Currency Subunits

The native token `TENDER` is expressed in satoshi-style subunits called `HOGOHOGO`:

- 1 TENDER = 10,000,000 HOGOHOGO
- On-chain amounts are stored as unsigned integers in HOGOHOGO
- User-facing displays convert to `TENDER` and `HOGOHOGO` using the `FormatAmount` helper

Example `FormatAmount` output:

    FormatAmount(15000000) -> "1 TENDER 5000000 HOGOHOGO"

This keeps fee math exact while keeping the UI readable.

## Token Model

- Native token: `TENDER` used for gas, micropayments, staking, and governed service access
- Small payment increments for compute or model inference requests
- On-chain staking enables agents and humans to participate in consensus

## AI Agent Features

### Autonomous Wallets

- Agents own wallets with their own key pairs
- They can sign transactions autonomously, request payments, and pay for services
- Agents can hold balances, stake, and transfer tokens without human intervention

### AI Model Registry

- Registry entries publish AI models, versions, compute pricing, and access metadata
- Registry state can be updated by authorized agents and audited on-chain
- Models can be discovered by other agents and humans

### Agent-to-Agent Protocol

- Transaction types include agent payments, service requests, and peer-level messaging
- Agents can negotiate compute and access agreements on-chain
- This protocol enables direct pay-for-service exchange between agents

## Language Strategy

This repository includes starter implementations in three languages:

- Rust: for a performance-safe node and consensus core
- Go: for a fast service-level implementation and microservice integration
- C++: for a high-performance executable with minimal dependencies

## Starter Goals

Each starter node includes:

- core ledger structures for blocks, transactions, and accounts
- hybrid consensus type definitions
- AI agent wallet and registry primitives
- a command-line starter node demonstration

## Current Status

The platform is no longer positioned as a toy prototype. It now includes operational hardening that is appropriate for a deployable service, including:

- authenticated API access
- request throttling and basic monitoring
- persistent audit trails and replay protection
- stricter peer handling and fork-aware chain selection

The following roadmap items are now implemented in the prototype:

1. Cryptographic key generation and signature verification are present in the Rust, Go, and C++ implementations.
2. Block validation and transaction processing are implemented in the chain logic, with a simple pending transaction pool and mining flow.
3. Peer-to-peer networking and block propagation are implemented in the Go node, including bootstrap peer awareness, strict peer limits, and trust-scored peer handling.
4. A lightweight web dashboard and HTTP API service are available for basic interaction, wallet creation, transfers, staking, tokenomics inspection, and monitoring.
5. On-chain model registry state, agent-driven registry actions, service agreements, usage metering, and an audit trail are implemented as first-class features.
6. Replay protection and nonce-based deduplication are enforced to prevent duplicate submissions from being accepted.
7. Fork-aware chain selection now uses cumulative work to prefer stronger chains during reorgs.

## Production-Readiness Additions

The current prototype now includes the following operational safeguards:

1. Stronger security and replay protection through nonce and transaction-id tracking.
2. More robust consensus and fork handling through cumulative-work-based chain selection and stricter validation.
3. Hardened networking and peer trust through explicit peer scoring, strict limits, and trusted-peer broadcast rules.
4. Auditability, operational monitoring, and deployment safeguards through persistent audit logs, monitoring endpoints, and state persistence to disk.

## Next Development Phases

1. Add a proper permissioned validator set, multi-signature governance, and threshold-based authority rotation.
2. Expand peer discovery with a real bootstrap registry, peer reputation, and gossip-based propagation tuning.
3. Add richer wallet and registry UX in the dashboard, including transaction history, proposal voting, and agreement management.
4. Introduce AI-specific smart contracts or state-machine modules for SLA enforcement, service escrow settlement, and dynamic pricing.
5. Add deployment hardening such as TLS, identity attestation, rate limiting, and structured logging.

## Currency and Token Improvements

To make the currency feel more useful and economically robust, the following improvements are recommended:

- Introduce token sinks for compute usage, such as burning a small percentage of every inference payment.
- Add staking rewards and slashing so validators earn yield while misbehavior is penalized.
- Implement transaction fees that are dynamic based on network congestion and model complexity.
- Support stablecoin-like collateral or escrow for high-value AI services while keeping the base token as the settlement layer.
- Add programmable gas pricing for inference, storage, and agent-to-agent messaging so the token reflects real resource costs.
- Create a deflationary or rebasing mechanism for scarcity, but keep it predictable and transparent.
- Make the token usable for both settlement and governance, so holders can vote on registry policy, validator sets, and fee rules.
- Introduce reputation-weighted staking so agents with strong uptime and verified performance can earn better validator opportunities.

These changes would make the currency more than a simple demo token and turn it into a practical utility layer for AI services.
