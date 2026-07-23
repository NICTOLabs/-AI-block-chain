# DEX Liquidity & Listing Guide

## Initial DEX Offering (IDO) Pair
- **Pair:** TDR / ETH or TDR / USDT
- **Protocol:** Uniswap V3 or PancakeSwap
- **Initial liquidity:** 250,000 TDR + equivalent ETH/USDT
- **Price range:** ±20% around IDO price
- **Lock duration:** 12 months minimum

## Liquidity Provider (LP) Workflow
1. Approve DEX router for TDR
2. Add liquidity at chosen price range
3. Receive LP NFT
4. Lock LP NFT via `Timer` or `Unlock` protocol
5. Publish liquidity lock proof

## Lock Proof Format
- Lock address
- Lock transaction hash
- Unlock timestamp
- LP provider address
- LP balance at lock time

## Audit Requirements
- DEX router audit
- LP token audit
- Lock contract audit
- Bridge audit (if applicable)

## Risk Mitigation
- Do not unlock liquidity before 12 months
- Avoid single-asset exposure
- Use time-locked governance for liquidity changes
- Publish weekly liquidity reports

## Bridge Compatibility
- WTDR wrapped token on EVM chains
- Bridge operator multisig (2/3)
- Bridge pause mechanism
- Daily reconciliation reports
