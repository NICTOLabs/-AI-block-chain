use serde::{Deserialize, Serialize};
use thiserror::Error;

/// Status of an escrow agreement.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum EscrowStatus {
    Open,
    FundsLocked,
    Completed,
    Disputed,
}

/// A simple task escrow contract for AI jobs.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct EscrowContract {
    escrows: std::collections::HashMap<String, EscrowRecord>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct EscrowRecord {
    from: String,
    to: String,
    amount: u128,
    status: EscrowStatus,
}

/// Errors emitted by the escrow contract.
#[derive(Debug, Error)]
pub enum EscrowError {
    #[error("escrow not found")]
    NotFound,
    #[error("escrow already exists")]
    AlreadyExists,
    #[error("escrow is not open")]
    NotOpen,
}

impl EscrowContract {
    /// Create a new escrow contract instance.
    pub fn new() -> Self {
        Self::default()
    }

    /// Open a new escrow.
    pub fn open(&mut self, escrow_id: &str, from: &str, to: &str, amount: u128) -> Result<(), EscrowError> {
        if self.escrows.contains_key(escrow_id) {
            return Err(EscrowError::AlreadyExists);
        }
        self.escrows.insert(
            escrow_id.to_string(),
            EscrowRecord {
                from: from.to_string(),
                to: to.to_string(),
                amount,
                status: EscrowStatus::FundsLocked,
            },
        );
        Ok(())
    }

    /// Complete an escrow.
    pub fn complete(&mut self, escrow_id: &str) -> Result<(), EscrowError> {
        let escrow = self.escrows.get_mut(escrow_id).ok_or(EscrowError::NotFound)?;
        if escrow.status != EscrowStatus::FundsLocked {
            return Err(EscrowError::NotOpen);
        }
        escrow.status = EscrowStatus::Completed;
        Ok(())
    }

    /// Dispute an escrow.
    pub fn dispute(&mut self, escrow_id: &str) -> Result<(), EscrowError> {
        let escrow = self.escrows.get_mut(escrow_id).ok_or(EscrowError::NotFound)?;
        if escrow.status != EscrowStatus::FundsLocked {
            return Err(EscrowError::NotOpen);
        }
        escrow.status = EscrowStatus::Disputed;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn open_and_complete_escrow() {
        let mut contract = EscrowContract::new();
        contract.open("job-1", "alice", "bob", 99).expect("open");
        contract.complete("job-1").expect("complete");
    }
}
