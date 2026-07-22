# Kenya Regulatory Registration Framework for TDR

## 1. Purpose
This document provides a professional compliance framework for engaging with the Central Bank of Kenya (CBK) and the Capital Markets Authority (CMA) regarding the TDR utility token and related network operations.

## 2. Regulatory Sandbox Application Data
### Applicant Information
- Legal entity name:
- Jurisdiction of incorporation:
- Registered address:
- Ultimate beneficial owners (UBOs):
- Contact person and role:

### Product Description
- Token name: TDR
- Token classification: Utility token / payment and access token
- Intended use cases: network gas, staking, validator incentives, and access to AI services
- Geographic scope: Kenya and selected international jurisdictions

### Technical Architecture Summary
- Consensus model: hybrid PoS/PoA
- Wallet and signature scheme: Ed25519
- Transaction propagation: peer-to-peer gossip and mempool
- Auditability: audit trail, monitoring, and cryptographic manifest

## 3. AML/KYC Data Flow Diagram
```text
User onboarding -> KYC screening -> Wallet creation -> Transaction monitoring -> Suspicious activity review -> SAR filing if required
```

### Node Architecture AML/KYC Hooks
- Wallet creation endpoint records customer identity reference and risk score.
- Transaction submission route checks transaction velocity, addresses, and sanctioned list risk.
- Monitoring and audit endpoints retain immutable logs for regulatory review.
- Escalation workflow routes suspicious transactions to compliance operations.

## 4. Risk Mitigation Disclosure Statement
TDR is a utility token intended for network access, staking, and service settlement. It is not designed to function as a profit-sharing instrument, equity security, or deposit substitute. The project implements controls for transaction monitoring, wallet risk scoring, vesting, burn mechanisms, and auditability. Tokens may be subject to restrictions in certain jurisdictions and should not be marketed as an investment product.

## 5. Governance and Controls
- Compliance officer ownership
- Independent annual audit
- Incident response protocol
- Data retention policy
- Suspicious activity reporting process

## 6. Supporting Attachments
- Security Audit Manifest
- Tokenomics configuration
- Validator topology and onboarding records
- Developer SDK documentation
