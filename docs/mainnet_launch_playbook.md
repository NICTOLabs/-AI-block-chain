# TDR Mainnet Launch Playbook

## Phase 0: Pre-Launch (Now - Launch Day)

- [ ] External security audit complete (Trail of Bits / Hacken)
- [ ] Audit report published and critical issues resolved
- [ ] Mainnet genesis file signed by ≥2/3 of validator set
- [ ] Validator onboarding complete: minimum 21 active validators across 5 countries
- [ ] Fund security: multisig treasury, HSM-protected keys, time-locked contracts
- [ ] Monitoring: Prometheus + Grafana dashboards live, PagerDuty on-call rotation active
- [ ] Compliance: CBK sandbox approval obtained, CMA registration in progress
- [ ] Legal: Terms of service, privacy policy, AML/KYC policy published

## Phase 1: Validator Bootstrapping (Launch Day - Week 1)

1. Genesis file distributed to validators via out-of-band channel
2. Validator nodes started with `--chain-id tdr-mainnet-1 --genesis-file genesis_mainnet.json`
3. Peer discovery: bootstrap peers whitelisted, P2P mesh established
4. First 100 blocks mined under PoA authority round-robin
5. Validator performance metrics published daily

## Phase 2: Public Testnet (Week 1 - Week 4)

1. Testnet faucet released with rate limits
2. Developer SDK published to npm and GitHub
3. Bug bounty program launched (minimum $50k pool)
4. CEX sandbox integrations via Rosetta API
5. DEX bridge audit and deployment (WTDR on Ethereum/BSC)

## Phase 3: Mainnet TGE (Week 4+)

1. Token Generation Event: 25% of initial circulating supply unlocked
2. Exchange listings: minimum 1 CEX via Rosetta integration
3. Liquidity mining program launched on DEX
4. Community governance activated (first proposal within 30 days)

## Emergency Procedures

| Incident | Response Time | Owner | Action |
|----------|--------------|-------|--------|
| Validator >33% offline | <5 min | Core team | Activate backup validator set |
| Smart contract bug | <15 min | Security team | Emergency pause via multisig |
| Exchange delisting risk | <1 hour | Legal/Compliance | Activate circuit breaker |
| Regulatory inquiry | <4 hours | Compliance officer | File SAR if required |
