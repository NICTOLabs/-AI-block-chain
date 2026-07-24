# Bridge Reconciliation Runbook

## Purpose

This runbook describes operational steps for reconciling a TENDER <-> external-chain bridge during outages, halts, or post-maintenance recovery.

## Prerequisites

- Access to bridge relayer DB and signing service
- Access to both chain RPC endpoints
- Operator PGP key for signing reconciliation proofs

## Normal reconciliation flow

1. Pause bridge relayers if divergent state is detected.
2. Export bridge ledger state from both sides to `bridge/reconciliation/YYYY-MM-DD/`.
3. Compare locked/hashed balances and in-flight transfers.
4. If mismatch < recovery threshold, publish a signed reconciliation proof and resume.
5. If mismatch > recovery threshold, escalate to multisig committee for manual adjudication.

## Emergency runbook

- Detect stuck or replaying transfers by comparing `bridge_locked` vs `chain_burned`.
- When chain halt is declared, freeze mint/burn and record last finality height.
- After restart, replay only transfers with `finalized=true` and `bridge_proof=true`.

## Safety invariants

- `minted == burned + replayed_safe_transfers`
- No duplicate redemption without prior cancellation
- Committee multisig quorum required for manual recovery
