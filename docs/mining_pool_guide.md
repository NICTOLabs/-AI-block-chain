# Mining Pool Operations Guide

## Overview
TDR mining pools coordinate work among multiple miners and distribute rewards proportionally.

## Pool Architecture
- **Work Assignment:** Pool assigns mining work to workers
- **Share Validation:** Workers submit shares; pool validates share difficulty
- **Reward Distribution:** Block rewards distributed per contributed shares
- **Payout Threshold:** Minimum payout enforced to reduce transaction costs

## Running a Pool
```bash
cd go-chain/tools/pool
go run main.go --addr :8083 --min-payout 1000
```

## Pool API
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/pool/stats` | GET | Pool statistics |
| `/pool/register` | POST | Register worker |
| `/pool/work` | GET | Get work assignment |
| `/pool/submit` | POST | Submit mined share/block |

## Economics
- **Pool fee:** 0-5%
- **Payout frequency:** Every 10 blocks or threshold balance
- **Minimum payout:** 1000 TDR
- **Settlement:** TDR on `tdr-mainnet-1`

## Risk & Compliance
- Pool operator must verify miner identities for large payouts
- Anti-sybil protections for worker registration
- Transparent payout history on-chain
