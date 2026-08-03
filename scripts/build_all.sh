#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if [ -f "$HOME/.cargo/env" ]; then
  source "$HOME/.cargo/env"
fi

echo "Building Rust workspace (consensus, agent-protocol, contracts, sdk/rust)..."
cargo build --workspace

if [ -f "rust-chain/Cargo.toml" ]; then
  echo "Building rust-chain (standalone)..."
  cargo build --manifest-path rust-chain/Cargo.toml
else
  echo "Skipping rust-chain build: rust-chain/Cargo.toml not found"
fi

echo "Running Go chain tests..."
cd go-chain
GOFLAGS='' go test ./...

cd "$repo_root"
echo "Checking optional C++ chain build..."
if [ -f "cpp-chain/CMakeLists.txt" ]; then
  echo "Building C++ chain..."
  cmake -S cpp-chain -B cpp-chain/build >/dev/null
  cmake --build cpp-chain/build >/dev/null
else
  echo "Skipping C++ chain build: cpp-chain/CMakeLists.txt not found"
fi

echo "Checking optional C++ VM build..."
if [ -f "vm/CMakeLists.txt" ]; then
  echo "Building C++ VM..."
  cmake -S vm -B vm/build >/dev/null
  cmake --build vm/build >/dev/null
else
  echo "Skipping C++ VM build: vm/CMakeLists.txt not found"
fi

echo "All build targets completed successfully."
