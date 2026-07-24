use ed25519_dalek::{Keypair, PublicKey, SecretKey, Signature, Signer, Verifier};
use rand::rngs::OsRng;
use serde::{Deserialize, Serialize};
use sha3::{Digest, Keccak256};
use thiserror::Error;

/// A 32-byte address used for agent wallets.
type Address = [u8; 32];

/// Capabilities that an agent wallet can exercise on-chain.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum Capability {
    SendTransactions,
    PurchaseApiKeys,
    DelegateToAgent,
    RegisterModel,
    OpenPaymentChannel,
}

/// An agent wallet derived deterministically from a model hash.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct AgentWallet {
    pub address: Address,
    pub model_hash: [u8; 32],
    pub pubkey: Vec<u8>,
    pub capabilities: Vec<Capability>,
    pub human_guardian: Option<Address>,
}

/// Errors emitted by the wallet module.
#[derive(Debug, Error)]
pub enum WalletError {
    #[error("human guardian is required for this operation")]
    GuardianRequired,
    #[error("transaction exceeds guardian threshold")]
    ThresholdExceeded,
    #[error("keypair generation failed")]
    KeypairGenerationFailed,
}

impl AgentWallet {
    /// Create a wallet deterministically from a model identifier string.
    pub fn from_model_id(model_id: &str, pubkey: Vec<u8>) -> Self {
        let mut hasher = Keccak256::default();
        hasher.update(model_id.as_bytes());
        let model_hash: [u8; 32] = hasher.finalize().into();
        let mut address = [0u8; 32];
        address.copy_from_slice(&model_hash[0..32]);
        Self {
            address,
            model_hash,
            pubkey: pubkey.clone(),
            capabilities: vec![
                Capability::SendTransactions,
                Capability::PurchaseApiKeys,
                Capability::DelegateToAgent,
                Capability::RegisterModel,
                Capability::OpenPaymentChannel,
            ],
            human_guardian: None,
        }
    }

    /// Create a wallet with an optional human guardian.
    pub fn with_guardian(model_id: &str, pubkey: Vec<u8>, guardian: Option<Address>) -> Self {
        let mut wallet = Self::from_model_id(model_id, pubkey);
        wallet.human_guardian = guardian;
        wallet
    }

    /// Sign a transaction payload with the runtime-provided private key.
    pub fn sign_tx(&self, payload: &[u8]) -> Result<Vec<u8>, WalletError> {
        let secret_bytes = self.pubkey.as_slice();
        if secret_bytes.len() != 32 {
            return Err(WalletError::KeypairGenerationFailed);
        }
        let secret = SecretKey::from_bytes(secret_bytes)
            .map_err(|_| WalletError::KeypairGenerationFailed)?;
        let public = PublicKey::from(&secret);
        let keypair = Keypair { secret, public };
        let signature: Signature = keypair.sign(payload);
        Ok(signature.to_bytes().to_vec())
    }

    /// Validate whether a transaction requires dual-signature approval.
    pub fn requires_guardian_approval(&self, amount: u128, threshold: u128) -> bool {
        self.human_guardian.is_some() && amount >= threshold
    }

    /// Validate a dual-signature approval request.
    pub fn validate_guardian_approval(&self, amount: u128, threshold: u128) -> Result<(), WalletError> {
        if self.requires_guardian_approval(amount, threshold) {
            return Err(WalletError::ThresholdExceeded);
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn wallet_is_deterministic_from_model_id() {
        let first = AgentWallet::from_model_id("gpt-4o-mini", vec![1, 2, 3]);
        let second = AgentWallet::from_model_id("gpt-4o-mini", vec![1, 2, 3]);
        assert_eq!(first.address, second.address);
    }

    #[test]
    fn guardian_approval_is_required_for_large_amounts() {
        let wallet = AgentWallet::with_guardian("gpt-4o-mini", vec![1], Some([9u8; 32]));
        assert!(wallet.requires_guardian_approval(1000, 500));
    }

    #[test]
    fn sign_tx_produces_ed25519_signature() {
        let wallet = AgentWallet::from_model_id("test-model", vec![0xAB; 32]);
        let payload = b"test-payload";
        let sig = wallet.sign_tx(payload);
        assert!(sig.is_ok());
        assert_eq!(sig.unwrap().len(), 64);
    }
}
