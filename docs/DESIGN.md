# AI Blockchain Design

## Purpose

This project is a new blockchain built from scratch to serve both AI agents and humans. It supports:

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

## Token Model

- Native token: `AI` token used for gas, micropayments, staking, and governed service access
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

The following roadmap items are now implemented in the prototype:

1. Cryptographic key generation and signature verification are present in the Rust, Go, and C++ implementations.
2. Block validation and transaction processing are implemented in the chain logic, with a simple pending transaction pool and mining flow.
3. Basic peer-to-peer networking and block propagation are implemented in the Go node.
4. A lightweight web dashboard and HTTP API service are available for basic interaction.
5. On-chain model registry state and agent-driven registry actions are implemented as first-class transaction types.

## Next Development Phases

1. Add a real mempool with fee bidding, prioritization, and replacement rules.
2. Introduce formal block validation rules, replay protection, and chain reorg handling.
3. Expand peer discovery with bootstrap nodes, handshake messages, and gossip propagation.
4. Add richer API and dashboard workflows for wallet creation, transfers, staking, and model registry management.
5. Introduce AI-specific smart contracts or state-machine modules for service agreements, SLA enforcement, and usage metering.

## Currency and Token Improvements

To make the currency feel more useful and economically robust, the following improvements are recommended:

- Introduce token sinks and sinks for compute usage, such as burning a small percentage of every inference payment.
- Add staking rewards and slashing so validators earn yield while misbehavior is penalized.
- Implement transaction fees that are dynamic based on network congestion and model complexity.
- Support stablecoin-like collateral or escrow for high-value AI services while keeping the base token as the settlement layer.
- Add programmable gas pricing for inference, storage, and agent-to-agent messaging so the token reflects real resource costs.
- Create a deflationary or rebasing mechanism for scarcity, but keep it predictable and transparent.
- Make the token usable for both settlement and governance, so holders can vote on registry policy, validator sets, and fee rules.
- Introduce reputation-weighted staking so agents with strong uptime and verified performance can earn better validator opportunities.

These changes would make the currency more than a simple demo token and turn it into a practical utility layer for AI services.
