use serde::{Deserialize, Serialize};
use sha3::{Digest, Keccak256};

/// A 32-byte address used across the blockchain state.
pub type Address = [u8; 32];

/// A 32-byte generic hash value.
pub type Hash = [u8; 32];

/// A BLS signature payload.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct BLSSignature(pub Vec<u8>);

/// A generic signature wrapper for transaction authentication.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Signature(pub Vec<u8>);

/// Derive a deterministic address from arbitrary bytes using Keccak-256.
pub fn address_from_bytes(bytes: &[u8]) -> Address {
    let mut hasher = Keccak256::default();
    hasher.update(bytes);
    let digest: [u8; 32] = hasher.finalize().into();
    digest
}

/// Derive a deterministic address from a UTF-8 string.
pub fn address_from_string(value: &str) -> Address {
    address_from_bytes(value.as_bytes())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn address_is_deterministic() {
        let first = address_from_string("agent-wallet");
        let second = address_from_string("agent-wallet");
        assert_eq!(first, second);
    }

    #[test]
    fn signature_roundtrip_serializes() {
        let sig = Signature(vec![1, 2, 3, 4]);
        let json = serde_json::to_string(&sig).expect("serialize signature");
        let decoded: Signature = serde_json::from_str(&json).expect("deserialize signature");
        assert_eq!(decoded, sig);
    }
}
