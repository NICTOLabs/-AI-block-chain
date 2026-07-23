# AI Agent Integration Guide

## Purpose
This document explains how AI agent frameworks can discover, evaluate, and transact with TDR programmatically.

## Supported Frameworks
- LangChain
- AutoGen
- Eliza
- Virtuals Protocol
- Bittensor

## Integration Steps
1. Install TDR SDK: `npm install tdr-sdk`
2. Generate agent wallet
3. Fund wallet with TDR for gas/model fees
4. Query balances, gas, and service agreements via SDK
5. Execute transactions programmatically

## Agent Wallet Requirements
- Unique address per agent
- Separate from human-controlled wallets
- Programmable key management
- Spending limits per agent

## Model Registry Discovery
```json
GET /api/registry
Response: {
  "model-id": {
    "owner": "0x...",
    "version": "v1.0",
    "price_per_call": 100,
    "active": true
  }
}
```

## Micro-Payment Requirements
- Fee per call: < $0.001 USD equivalent
- Settlement: per-call or batched per 100 calls
- Refund policy: automatic for failed calls
- Dispute resolution: agent reputation stake

## Agent Reputation
- Performance history on-chain
- Stake-weighted voting
- Slashing for fraudulent behavior
- Blacklist registry for bad actors
