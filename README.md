# -AI-block-chain

A new blockchain and token built from scratch for both AI agents and humans.

## What this repo includes

- `rust-chain/`: full Rust node starter with wallet key generation, transaction signing, and an on-chain AI model registry
- `go-chain/`: Go node starter with signed transactions and model registry primitives
- `cpp-chain/`: C++ node starter with wallet generation, transaction signing, and a minimal registry
- `docs/DESIGN.md`: hybrid PoS/PoA design, agent features, and registry concepts

## Features implemented

- Ed25519 wallet generation and address derivation
- Signed transactions with signature verification
- Hybrid consensus placeholders for PoS and PoA
- On-chain AI model registry and purchase flow
- Agent vs human accounts with autonomous wallet support

## Run the Rust node

```bash
cd rust-chain
source "$HOME/.cargo/env"
cargo run
```

## Run the Go node

```bash
cd go-chain
go run .
```

## Build and run the C++ node

```bash
cd cpp-chain
mkdir -p build && cd build
cmake ..
cmake --build .
./ai_block_chain_cpp
```

## How humans use the chain

1. Run any node implementation to start the network skeleton.
2. Create a human wallet by generating a keypair in Rust/Go/C++.
3. Deposit native `AI` tokens into the human wallet (starter accounts are seeded in the demo).
4. Use signed transfer transactions to pay AI agents for compute and API access.
5. Buy AI model access by sending a `PurchaseApiKey` transaction to the model registry entry.
6. Track balances and purchased API access on-chain.

## How AI agents use the chain

1. Each agent runs an autonomous wallet with its own keypair and derived address.
2. Agents can register models by sending `RegisterModel` transactions with pricing and metadata.
3. Agents can pay each other for compute by sending signed `Transfer` transactions.
4. Agents can update their model metadata and pricing via `UpdateModel` transactions.
5. Agents can purchase API keys or model access directly on-chain, enabling secure service payments.
6. The registry stores model ownership, version, price-per-call, and activation state.

## Example flow

1. `agentA` registers `model-AI-1` with a compute price.
2. A human sends a signed `PurchaseApiKey` transaction to the model entry.
3. `agentA` receives the payment and then service access is granted.
4. `agentA` can also pay `agentB` for compute by issuing a signed `Transfer` transaction.

This repository is a starter platform for building AI-aware blockchain workflows where both humans and autonomous agents hold wallets, sign transactions, register models, and settle micropayments on-chain.
