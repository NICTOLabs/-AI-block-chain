use serde::{Deserialize, Serialize};
use thiserror::Error;

use crate::address::{Address, BLSSignature};

/// Minimum stake required to join the PoS validator set.
pub const MIN_STAKE: u128 = 10_000 * 10u128.pow(18);
pub const BLOCK_TIME_MS: u64 = 400;

/// A validator participating in the PoS layer.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Validator {
    pub address: Address,
    pub stake: u128,
    pub pubkey: Vec<u8>,
    pub is_active: bool,
    pub slashed: bool,
}

/// The active validator set for the hybrid consensus model.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct ValidatorSet {
    pub poa_validators: Vec<Address>,
    pub pos_validators: Vec<Validator>,
    pub epoch: u64,
}

/// Errors that can occur while managing a validator set.
#[derive(Debug, Error)]
pub enum ValidatorError {
    #[error("validator is already present")]
    AlreadyPresent,
    #[error("validator does not meet minimum stake")]
    InsufficientStake,
    #[error("validator is not active")]
    InactiveValidator,
    #[error("validator set is empty")]
    EmptyValidatorSet,
    #[error("finality threshold not met")]
    FinalityThresholdNotMet,
}

impl ValidatorSet {
    /// Create a new validator set with a permissioned PoA layer.
    pub fn new(poa_validators: Vec<Address>, epoch: u64) -> Self {
        Self {
            poa_validators,
            pos_validators: Vec::new(),
            epoch,
        }
    }

    /// Register a PoS validator if it satisfies the minimum stake.
    pub fn register_pos_validator(&mut self, address: Address, stake: u128, pubkey: Vec<u8>) -> Result<(), ValidatorError> {
        if stake < MIN_STAKE {
            return Err(ValidatorError::InsufficientStake);
        }
        if self.pos_validators.iter().any(|v| v.address == address) {
            return Err(ValidatorError::AlreadyPresent);
        }
        self.pos_validators.push(Validator {
            address,
            stake,
            pubkey,
            is_active: true,
            slashed: false,
        });
        Ok(())
    }

    /// Slash a validator for misbehavior and deactivate it.
    pub fn slash_validator(&mut self, address: &Address) -> Result<(), ValidatorError> {
        let validator = self
            .pos_validators
            .iter_mut()
            .find(|v| v.address == *address)
            .ok_or(ValidatorError::InactiveValidator)?;
        validator.slashed = true;
        validator.is_active = false;
        Ok(())
    }

    /// Select the next PoA validator in round-robin order.
    pub fn next_poa_validator(&self, index: usize) -> Result<Address, ValidatorError> {
        if self.poa_validators.is_empty() {
            return Err(ValidatorError::EmptyValidatorSet);
        }
        let position = index % self.poa_validators.len();
        Ok(self.poa_validators[position])
    }

    /// Determine the active validator count for the finality threshold.
    pub fn active_validator_count(&self) -> usize {
        let mut count = self.poa_validators.len();
        count += self.pos_validators.iter().filter(|v| v.is_active && !v.slashed).count();
        count
    }

    /// Compute the number of signatures needed to reach 2/3+ of the active set.
    pub fn finality_threshold(&self) -> usize {
        let active = self.active_validator_count();
        if active == 0 {
            return 0;
        }
        ((active as f64 * 2.0 / 3.0).ceil()) as usize
    }

    /// Check whether a set of signatures satisfies finality.
    pub fn check_finality(&self, signatures: &[BLSSignature]) -> Result<(), ValidatorError> {
        let threshold = self.finality_threshold();
        if signatures.len() < threshold {
            return Err(ValidatorError::FinalityThresholdNotMet);
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn registers_validator_once() {
        let mut set = ValidatorSet::new(vec![[1u8; 32]], 1);
        assert!(set.register_pos_validator([2u8; 32], MIN_STAKE, vec![9, 9, 9]).is_ok());
        assert!(matches!(set.register_pos_validator([2u8; 32], MIN_STAKE, vec![9, 9, 9]), Err(ValidatorError::AlreadyPresent)));
    }

    #[test]
    fn slash_deactivates_validator() {
        let mut set = ValidatorSet::new(vec![[1u8; 32]], 1);
        set.register_pos_validator([3u8; 32], MIN_STAKE, vec![1]).expect("register validator");
        set.slash_validator(&[3u8; 32]).expect("slash validator");
        let validator = set.pos_validators.iter().find(|v| v.address == [3u8; 32]).expect("validator exists");
        assert!(validator.slashed);
        assert!(!validator.is_active);
    }

    #[test]
    fn finality_threshold_is_2_3_plus() {
        let set = ValidatorSet::new(vec![[1u8; 32], [2u8; 32]], 1);
        assert_eq!(set.finality_threshold(), 2);
    }
}
