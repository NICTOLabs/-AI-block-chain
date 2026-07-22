use serde::{Deserialize, Serialize};
use sha3::{Digest, Keccak256};

use crate::address::{Address, BLSSignature, Hash};
use crate::transaction::Transaction;

/// The header of a block.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct BlockHeader {
    pub height: u64,
    pub timestamp: u64,
    pub prev_hash: Hash,
    pub tx_root: Hash,
    pub state_root: Hash,
    pub validator: Address,
}

/// A full block containing transactions and a validator signature.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Block {
    pub header: BlockHeader,
    pub transactions: Vec<Transaction>,
    pub validator_signature: BLSSignature,
}

impl Block {
    /// Construct a new block from its components.
    pub fn new(header: BlockHeader, transactions: Vec<Transaction>, validator_signature: BLSSignature) -> Self {
        Self {
            header,
            transactions,
            validator_signature,
        }
    }

    /// Compute a deterministic hash for the block payload.
    pub fn block_hash(&self) -> Result<Hash, serde_json::Error> {
        let bytes = serde_json::to_vec(self)?;
        let mut hasher = Keccak256::default();
        hasher.update(bytes);
        Ok(hasher.finalize().into())
    }

    /// Derive a Merkle-like transaction root from the transaction list.
    pub fn tx_root(transactions: &[Transaction]) -> Result<Hash, serde_json::Error> {
        let mut hasher = Keccak256::default();
        for tx in transactions {
            let tx_bytes = serde_json::to_vec(tx)?;
            hasher.update(tx_bytes);
        }
        Ok(hasher.finalize().into())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::transaction::Transaction;

    #[test]
    fn block_hash_is_stable() {
        let header = BlockHeader {
            height: 1,
            timestamp: 1_700_000_000,
            prev_hash: [0u8; 32],
            tx_root: [0u8; 32],
            state_root: [0u8; 32],
            validator: [9u8; 32],
        };
        let block = Block::new(header.clone(), vec![], BLSSignature(vec![1, 2, 3]));
        let first = block.block_hash().expect("hash block");
        let second = block.block_hash().expect("hash block");
        assert_eq!(first, second);
    }

    #[test]
    fn tx_root_is_computed_from_transactions() {
        let tx = Transaction::new_transfer(
            1,
            [1u8; 32],
            [2u8; 32],
            100,
            21000,
            1,
            crate::address::Signature(vec![1]),
        );
        let root = Block::tx_root(&[tx]).expect("compute tx root");
        assert_ne!(root, [0u8; 32]);
    }
}
