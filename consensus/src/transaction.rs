use serde::{Deserialize, Serialize};
use sha3::{Digest, Keccak256};

use crate::address::{Address, Hash, Signature};

/// Transaction kinds supported by the base consensus layer.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum TxType {
    Transfer,
    ContractCall,
    AgentOp(AgentOpType),
}

/// Agent-specific operations that can be embedded into a transaction.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum AgentOpType {
    RegisterModel,
    SendAgentMessage,
    OpenPaymentChannel,
    ClosePaymentChannel,
    PurchaseApiKey,
}

/// A transaction that can be included in a block.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Transaction {
    pub nonce: u64,
    pub from: Address,
    pub to: Address,
    pub value: u128,
    pub data: Vec<u8>,
    pub gas_limit: u64,
    pub gas_price: u64,
    pub tx_type: TxType,
    pub signature: Signature,
}

impl Transaction {
    /// Build a standard transfer transaction.
    pub fn new_transfer(
        nonce: u64,
        from: Address,
        to: Address,
        value: u128,
        gas_limit: u64,
        gas_price: u64,
        signature: Signature,
    ) -> Self {
        Self {
            nonce,
            from,
            to,
            value,
            data: Vec::new(),
            gas_limit,
            gas_price,
            tx_type: TxType::Transfer,
            signature,
        }
    }

    /// Return a deterministic hash of the transaction payload.
    pub fn tx_hash(&self) -> Result<Hash, serde_json::Error> {
        let bytes = serde_json::to_vec(self)?;
        let mut hasher = Keccak256::default();
        hasher.update(bytes);
        Ok(hasher.finalize().into())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn transfer_transaction_hashes_consistently() {
        let tx = Transaction::new_transfer(
            1,
            [1u8; 32],
            [2u8; 32],
            10_000,
            21_000,
            1,
            Signature(vec![9, 8, 7]),
        );
        let first = tx.tx_hash().expect("hash tx");
        let second = tx.tx_hash().expect("hash tx");
        assert_eq!(first, second);
    }
}
