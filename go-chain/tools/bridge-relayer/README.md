# TENDER Bridge Relayer

Minimal HTTP-based bridge relayer for the TENDER project.

## Endpoints

- `POST /relay/lock` — Record a lock event from the source chain
- `POST /relay/mint` — Mint wrapped tokens on the TENDER chain
- `GET  /relay/status` — Number of processed transaction hashes
- `GET  /relayer/health` — Health check

## Running

```bash
go run .
```

Or build:

```bash
go build -o bridge-relayer .
```

## Environment

| Variable | Default | Description |
|---|---|---|
| `RELAYER_PORT` | `9090` | Relayer listen port |
| `TENDER_NODE_URL` | `http://localhost:8080` | TENDER node HTTP API |
| `TENDER_API_KEY` | `""` | API key for the TENDER node |
| `TENDER_DATA_DIR` | `./data` | TENDER node data directory (for burn fallback) |

## Key Management

On first run the relayer generates an ed25519 keypair and saves it to `relayer-key.json`.

## Examples

```bash
curl -X POST http://localhost:9090/relay/lock \
  -H "Content-Type: application/json" \
  -d '{"tx_hash":"0xabc","from":"0xsender","amount":100,"recipient":"0xrecipient"}'

curl -X POST http://localhost:9090/relay/mint \
  -H "Content-Type: application/json" \
  -d '{"tx_hash":"0xabc","to":"0xrecipient","amount":100}'
```
