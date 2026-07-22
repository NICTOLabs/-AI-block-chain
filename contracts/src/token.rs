use serde::{Deserialize, Serialize};
use thiserror::Error;

/// A lightweight native-token contract abstraction.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct TokenContract {
    balances: std::collections::HashMap<String, u128>,
}

/// Errors emitted by the token contract.
#[derive(Debug, Error)]
pub enum TokenError {
    #[error("insufficient balance")]
    InsufficientBalance,
    #[error("token transfer failed")]
    TransferFailed,
}

impl TokenContract {
    /// Create a new token contract instance.
    pub fn new() -> Self {
        Self::default()
    }

    /// Mint tokens into the supplied account.
    pub fn mint(&mut self, account: &str, amount: u128) {
        *self.balances.entry(account.to_string()).or_insert(0) += amount;
    }

    /// Transfer tokens from one account to another.
    pub fn transfer(&mut self, from: &str, to: &str, amount: u128) -> Result<(), TokenError> {
        let from_balance = self.balances.get(from).copied().unwrap_or(0);
        if from_balance < amount {
            return Err(TokenError::InsufficientBalance);
        }
        let to_balance = self.balances.get(to).copied().unwrap_or(0);
        self.balances.insert(from.to_string(), from_balance - amount);
        self.balances.insert(to.to_string(), to_balance + amount);
        Ok(())
    }

    /// Return the balance for an account.
    pub fn balance_of(&self, account: &str) -> u128 {
        self.balances.get(account).copied().unwrap_or(0)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn mint_and_transfer_work() {
        let mut contract = TokenContract::new();
        contract.mint("alice", 100);
        contract.transfer("alice", "bob", 25).expect("transfer");
        assert_eq!(contract.balance_of("alice"), 75);
        assert_eq!(contract.balance_of("bob"), 25);
    }
}
