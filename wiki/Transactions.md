# Transactions

TENDER supports the following transaction types:

- `Transfer`: P2P and agent-to-agent payments
- `RegisterModel`: On-chain model registration with staking lock
- `UpdateModel`: Metadata and pricing updates
- `PurchaseApiKey`: Service access and micropayment settlement
- `RequestReversal`: Request a transfer reversal with mutual consent
- `ConfirmReversal`: Confirm a requested reversal
- `CommitReversal`: Commit an approved reversal

## Reversals

Regular transfers can be reversed through a 3-phase process:
1. RequestReversal: Sender requests reversal
2. ConfirmReversal: Recipient confirms
3. CommitReversal: Either party commits the reversal

Irreversible payments, such as escrow payouts, cannot be reversed.
