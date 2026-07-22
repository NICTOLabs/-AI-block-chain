# TDR Launch Sequence: From Prototype to Real Currency

## Current Status: Production-Hardened Mainnet-Ready Codebase

All production hardening has been committed and pushed to `main`. The repository now contains:

- Consensus fixes: ChainID-bound transactions, nonce sequencing, adaptive difficulty, deterministic genesis
- Security: P2P oversize checks, state-store snapshots, API circuit breaker, tx expiry
- Tooling: Genesis allocator, testnet faucet, HD wallet vault, validator onboarding
- Operations: Terraform, Docker Compose prod, Nginx TLS, Prometheus/Grafana/Alertmanager
- Compliance: CBK sandbox, CMA utility-token docs, SAR workflow, audit manifest, fuzz suite
- Integration: Rosetta API, ERC-20 WTDR bridge contract, exchange listing kit
- Documentation: Testnet guide, mainnet playbook, validator key ceremony, readiness checklist

## Exact Steps to Become a Real Currency

### 1. External Audit (Critical Path)
- Send `docs/audit_readiness_package.md` to Trail of Bits or Hacken
- Budget: $80k-$150k
- Timeline: 4-6 weeks
- Gate: No critical or high-severity findings unaddressed

### 2. Regulatory Approval
- Submit CBK sandbox application using `docs/kenyan_compliance_framework.md`
- File CMA utility-token registration
- Obtain legal opinion confirming utility-token classification
- Timeline: 8-12 weeks parallel to audit

### 3. Mainnet Genesis
- Run `go run go-chain/tools/genesis-tool/main.go --output genesis_mainnet.json`
- Replace placeholder allocations with real multisig addresses
- Genesis file signed by 2/3+ of validator set
- Genesis SHA256 hash published on website and social media

### 4. Validator Bootstrap
- Minimum 21 validators across 5 countries
- Each validator runs key ceremony per `docs/validator_key_ceremony.md`
- Deploy via `go-chain/deploy/terraform.tf` or manual Docker
- Verify with `docs/testnet_deployment_guide.md`

### 5. Exchange Listings
- Complete Rosetta integration per `docs/exchange_listing_kit.md`
- Deploy and audit WTDR bridge contract
- Submit listing applications to 3-5 CEXs
- Activate DEX liquidity on Uniswap/BSC

### 6. Public Announcement
- Release whitepaper and audit report
- Publish TGE announcement with genesis hash
- Activate bug bounty ($100k+ pool)
- Launch developer portal and SDK

## Risk Mitigation

| Risk | Mitigation | Owner |
|------|------------|-------|
| Audit finds critical flaw | 30-day remediation buffer before launch | Security lead |
| Regulatory delay | Parallel sandbox + legal opinion track | Compliance officer |
| Validator centralization | Geographic distribution requirement | Core team |
| Bridge exploit | Multi-sig + timelock + audit | Smart contract lead |
| Exchange delisting | Transparent utility model, no yield promises | Legal counsel |

## Timeline

- Week 1-6: Security audit
- Week 4-12: Regulatory approval (parallel)
- Week 10-12: Genesis preparation
- Week 13: Validator bootstrap
- Week 14: Mainnet launch
- Week 15+: Exchange listings and DEX bridge

## Success Criteria

- [ ] Audit passes with no critical findings
- [ ] CBK sandbox approval obtained
- [ ] 21+ validators online and producing blocks
- [ ] Rosetta API live and tested by 2+ exchanges
- [ ] WTDR bridge audited and deployed
- [ ] Bug bounty program active
- [ ] SDK published to NPM with 100+ weekly downloads
- [ ] Transaction finality < 10 seconds
- [ ] 99.9% node uptime over 30 days
