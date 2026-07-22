#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if [ -f "$HOME/.cargo/env" ]; then
  source "$HOME/.cargo/env"
fi

cargo build --manifest-path consensus/Cargo.toml
cargo build --manifest-path agent-protocol/Cargo.toml
cargo build --manifest-path contracts/Cargo.toml

cd go-chain
GOFLAGS='' go test ./...

cd "$repo_root"
cmake -S cpp-chain -B cpp-chain/build >/dev/null
cmake --build cpp-chain/build >/dev/null

echo "All build targets completed successfully."
