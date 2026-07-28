# Consensus

TENDER uses Tendermint-style Proof-of-Stake consensus.

## Proposer Selection

Proposers are selected by stake with round-robin ordering. The proposer for a given height is deterministic based on the validator set.

## Voting Phases

1. Proposal: The proposer creates a block proposal
2. Prevote: Validators prevote for the proposed block
3. Precommit: Validators precommit after receiving 2/3+ prevotes
4. Commit: Block is committed once 2/3+ precommotes are received

## Finality

Blocks are finalized when they receive a 2/3+ supermajority of precommotes. This provides strong BFT finality guarantees.

## Slashing

Evidence of misbehavior, such as double signing or unavailability, is recorded and results in slashing penalties.
