# WTDR Bridge Deployment Guide

## Prerequisites
- EVM chain RPC endpoint (Ethereum/BSC/Polygon)
- Bridge operator wallet with gas funds
- Multisig owners for bridge roles
- Audited `go-chain/contracts/WTDR.sol`

## Step 1: Deploy WTDR
```bash
forge init wtdr-bridge
cp WTDR.sol src/
forge build
forge create src/WTDR.sol:WTDR --constructor-args "Wrapped Tender" "WTDR" <bridge-operator> <admin>
```

## Step 2: Configure Roles
```bash
cast send <wtdr-address> "grantRole(bytes32,address)" 0x... <bridge-operator> --rpc-url $RPC
cast send <wtdr-address> "grantRole(bytes32,address)" 0x... <admin> --rpc-url $RPC
```

## Step 3: Bridge Parameters
- Mint cap: 10,000,000,000 * 10**8
- Lock duration: 6 confirmations on source chain
- Release delay: 12 blocks on target chain
- Operator fee: 10 bps

## Step 4: Monitoring
- Alert on bridge balance mismatch > 0.1%
- Alert on failed bridge transactions
- Daily reconciliation report

## Security Checklist
- [ ] Smart contract audit complete
- [ ] Bridge operator HSM key
- [ ] Multisig timelock (24h)
- [ ] Pause mechanism tested
- [ ] Emergency recovery keys
