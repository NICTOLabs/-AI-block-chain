#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if [ -f "$HOME/.cargo/env" ]; then
  source "$HOME/.cargo/env"
fi

echo "Building Rust workspace (consensus, agent-protocol, contracts, sdk/rust)..."
cargo build --workspace

echo "Building rust-chain (standalone)..."
cargo build --manifest-path rust-chain/Cargo.toml

echo "Running Go chain tests..."
cd go-chain
GOFLAGS='' go test ./...

cd "$repo_root"
echo "Building C++ chain..."
cmake -S cpp-chain -B cpp-chain/build >/dev/null
cmake --build cpp-chain/build >/dev/null

echo "Building C++ VM..."
cmake -S vm -B vm/build >/dev/null
cmake --build vm/build >/dev/null

echo "All build targets completed successfully."
