# Networking

TENDER uses a libp2p-compatible TCP transport with the following features:

- Ed25519-signed handshakes for peer authentication
- Stream multiplexing for concurrent protocols
- mDNS discovery for local peer finding
- UPnP and NAT-PMP for router traversal
- Connection pooling and exponential backoff
- Message deduplication and relay prevention
- Validator pubkey authentication in strict mode
