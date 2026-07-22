# Exchange Listing Kit - TDR Token

## 1. Rosetta API Integration

### Base URL
```
https://tdr-mainnet-1.tender.network
```

### Required Endpoints
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/network/status` | GET | Current block height, timestamp, peers |
| `/block` | GET | Block details by index |
| `/construction/submit` | POST | Broadcast signed transaction |

### Integration Steps
1. Register API key with `compliance@tender.network`
2. Configure whitelist IP addresses
3. Testnet dry-run with testnet TDR
4. Mainnet production cutover

## 2. ERC-20 Wrapped Token (WTDR)

### Contract Address (Post-Audit)
```
0x[TBD AFTER DEPLOYMENT]
```

### Bridge Operations
| Operation | Description |
|-----------|-------------|
| `bridgeLock` | Lock TDR, mint WTDR on EVM chain |
| `bridgeRelease` | Burn WTDR, release TDR on native chain |
| `bridgeMint` | Mint WTDR for deposit confirmations |

### Audit Requirements
- [ ] Smart contract audit (CertiK or Quantstamp)
- [ ] Bridge security review
- [ ] Multi-sig deployment (3/5)

## 3. CEX Integration Checklist

### Technical
- [ ] Rosetta API tested in sandbox
- [ ] Deposit/withdrawal addresses generated
- [ ] Tag/memo support confirmed (if applicable)
- [ ] Withdrawal processing latency < 5 minutes

### Compliance
- [ ] AML/KYC policy shared
- [ ] Jurisdiction restrictions documented
- [ ] Travel rule implementation confirmed
- [ ] SAR reporting workflow tested

### Operational
- [ ] 24/7 monitoring configured
- [ ] Incident response plan shared
- [ ] Key rotation schedule agreed
- [ ] SLA: 99.9% API uptime

## 4. Contact for Listings
Email: `listings@tender.network`
Subject: `TDR Listing Inquiry - [Exchange Name]`
