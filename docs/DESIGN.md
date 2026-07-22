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

## Next Development Phases

1. Add cryptographic key generation and signature verification
2. Implement block validation, mempool, and transaction fees
3. Add network discovery and peer-to-peer gossip
4. Build a front-end or API service for human and agent interaction
5. Add AI-specific smart contracts or on-chain state machines for model registry
