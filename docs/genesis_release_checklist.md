# Genesis Verification & Release Checklist

## Pre-Release
- [ ] `go run ./tools/genesis-tool/main.go --action generate` completed
- [ ] All validator public keys collected via `docs/validator_key_ceremony.md`
- [ ] Multisig threshold confirmed (2/3 or 2/2)
- [ ] Designated signers list reviewed:
  - `0x09f046ab4b755d228e06c528d1a8cad540ae92f7`
  - [second signer address]

## Signing Phase
- [ ] Each signer runs `scripts/genesis-cli.sh sign --private-key ... --address ...`
- [ ] Signing machines air-gapped or HSM-backed
- [ ] Video recording of ceremony
- [ ] All signatures collected into `genesis_mainnet.json`
- [ ] `scripts/genesis-cli.sh verify` confirms threshold reached

## Post-Signing
- [ ] `multisig.finalized = true` in genesis file
- [ ] Genesis SHA256 published on website and social media
- [ ] Genesis artifact distributed to validators via out-of-band channel
- [ ] Validators start nodes with `--genesis-file genesis_mainnet.json`

## Release
- [ ] First 100 blocks produced under watched conditions
- [ ] Validator performance metrics published
- [ ] Genesis hash announced publicly

## Emergency
- If a signer key is compromised before finalization:
  1. Remove compromised signer from `multisig.signers`
  2. Regenerate genesis with remaining signers
  3. Restart ceremony
