# Validator Key Ceremony Procedure

## Overview
This document describes the secure key generation procedure for TDR mainnet validators.

## Prerequisites
- Air-gapped laptop or hardware security module (HSM)
- GPG-encrypted USB drive for key storage
- Two-factor authenticated backup location
- At least two trusted witnesses

## Step 1: Environment Preparation

1. Boot from tailsOS or other trusted live OS
2. Disconnect from all networks (WiFi, Ethernet, Bluetooth)
3. Verify GPG signatures of ceremony software
4. Set up video recording of the entire ceremony

## Step 2: Key Generation

```bash
go run tools/genesis-tool/main.go \
  --output genesis_validator.json \
  --validator \
  --stake 100000 \
  --country KE
```

## Step 3: Key Storage

- Primary key: encrypted USB drive stored in bank vault
- Backup key: encrypted USB drive stored in safety deposit box
- Emergency recovery: Shamir's Secret Sharing (3-of-5)

## Step 4: Public Key Registration

1. Submit public key to `listings@tender.network`
2. Receive confirmation email with validator ID
3. Add to validator set via governance proposal

## Key Management Rules

- Private key never leaves HSM
- Daily key backup verification
- Immediate key rotation if compromise suspected
- Annual key ceremony revalidation
