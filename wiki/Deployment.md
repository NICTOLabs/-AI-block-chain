# Deployment

## Running the Go Node

```bash
cd go-chain
go run . --api-port 8080 --p2p-port 3030
```

## Running the Rust Node

```bash
cd rust-chain
source "$HOME/.cargo/env"
cargo run
```

## Running the C++ Node

```bash
cd cpp-chain
mkdir -p build && cd build
cmake ..
cmake --build .
./ai_block_chain_cpp
```

## Dashboard

Open the dashboard at:

```
http://127.0.0.1:8080/
```
