# API Reference

The Go node exposes a REST API on port 8080 by default.

## Endpoints

- `GET /health` - Node health check
- `GET /api/chain` - Full chain data
- `GET /api/mempool` - Current mempool
- `GET /api/tokenomics` - Tokenomics metrics
- `GET /api/audit` - Audit trail
- `GET /api/monitoring` - Monitoring metrics
- `POST /api/mine` - Mine a new block
- `POST /api/transfer` - Submit a transfer transaction
- `POST /api/registry/register` - Register a model
- `POST /api/registry/update` - Update a model
- `POST /api/registry/purchase` - Purchase API access
- `POST /api/reversals/request` - Request a reversal
- `POST /api/reversals/confirm` - Confirm a reversal
- `POST /api/reversals/commit` - Commit a reversal
