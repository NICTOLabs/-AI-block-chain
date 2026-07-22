# TENDER Go Node

This service provides a lightweight blockchain node for the TENDER currency with:

- HTTP API
- P2P bootstrap scaffolding
- staking and validator logic
- tokenomics and audit trail
- wallet and managed-wallet support

## Run locally

```bash
go run .
```

## Run with Docker

```bash
docker compose up --build
```

## Configuration

The node supports the following environment variables:

- TENDER_API_KEY
- TENDER_ENABLE_AUTH
- TENDER_RATE_LIMIT
- TENDER_RATE_WINDOW_SECONDS
- TENDER_METRICS_PATH
- TENDER_API_PORT
- TENDER_P2P_PORT
- TENDER_DATA_DIR
- TENDER_CONSENSUS
- TENDER_STRICT_P2P
