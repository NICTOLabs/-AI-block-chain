# Genesis Signing Ceremony

## Purpose
Formalize the process by which genesis validators sign and finalize `genesis_mainnet.json` for `tdr-mainnet-1`.

## Pre-requisites
- All validators have generated their Ed25519 keypairs via `docs/validator_key_ceremony.md`
- All validator public keys are known and published
- A quorum of `threshold` signers is available simultaneously

## Ceremony Steps

### 1. Generate Unsigned Genesis
```bash
cd go-chain/tools/genesis-tool
go run main.go --action generate --output genesis_mainnet.json --chain-id tdr-mainnet-1 --validator-count 7
```

### 2. Publish Genesis Hash
All participants verify the same `genesis_mainnet.json` hash:
```bash
go run main.go --action verify --genesis genesis_mainnet.json
```

### 3. Collect Signatures
Each signer runs locally, never sharing their private key:
```bash
go run main.go --action sign \
  --genesis genesis_mainnet.json \
  --private-key "${PRIVATE_KEY}" \
  --address "${ADDRESS}" \
  --output genesis_mainnet.signed.json
```

### 4. Merge Signatures
Signatures are collected out-of-band and merged into one file. The tool verifies:
- Signer is in `multisig.signers`
- Signature is valid for the canonical genesis JSON
- No duplicate signatures

### 5. Finalize
Once `len(signatures) >= threshold`, the file is marked `finalized: true` and published.

### 6. Post-Ceremony
- Publish `genesis_mainnet.json` + SHA256 manifest
- Store signing artifacts in HSM-backed vault
- Announce genesis hash publicly

## Signer Designation
- Signer 1: `0x09f046ab4b755d228e06c528d1a8cad540ae92f7`
- Signer 2: Second validator designated by governance

## Security
- Private keys never leave signing machines
- All ceremony steps recorded on video
- Signing machines air-gapped or HSM-backed
- Shamir's Secret Sharing for key backup (3-of-5)
