use serde::{Deserialize, Serialize};
use sha3::{Digest, Keccak256};
use thiserror::Error;

/// A 32-byte address used for micropayment participants.
type Address = [u8; 32];

type Hash = [u8; 32];

/// The state of a payment channel.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum ChannelState {
    Open,
    Closing { initiated_at: u64 },
    Closed,
    Disputed,
}

/// A signed voucher submitted off-chain and settled on-chain.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PaymentVoucher {
    pub channel_id: Hash,
    pub amount: u128,
    pub nonce: u64,
}

/// A payment channel for sub-second AI compute settlement.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct PaymentChannel {
    pub channel_id: Hash,
    pub sender: Address,
    pub receiver: Address,
    pub total_deposit: u128,
    pub spent: u128,
    pub nonce: u64,
    pub expiry_block: u64,
    pub state: ChannelState,
}

/// Errors emitted by the micropayment module.
#[derive(Debug, Error)]
pub enum MicropayError {
    #[error("channel is not open")]
    ChannelNotOpen,
    #[error("voucher is stale")]
    StaleVoucher,
    #[error("channel balance is insufficient")]
    InsufficientBalance,
}

impl PaymentChannel {
    /// Create a new payment channel.
    pub fn new(sender: Address, receiver: Address, total_deposit: u128, expiry_block: u64) -> Self {
        let mut hasher = Keccak256::default();
        hasher.update(sender);
        hasher.update(receiver);
        hasher.update(total_deposit.to_le_bytes());
        hasher.update(expiry_block.to_le_bytes());
        let channel_id: Hash = hasher.finalize().into();
        Self {
            channel_id,
            sender,
            receiver,
            total_deposit,
            spent: 0,
            nonce: 0,
            expiry_block,
            state: ChannelState::Open,
        }
    }

    /// Apply a signed voucher to the channel state.
    pub fn settle(&mut self, voucher: PaymentVoucher) -> Result<(), MicropayError> {
        if self.state != ChannelState::Open {
            return Err(MicropayError::ChannelNotOpen);
        }
        if voucher.nonce < self.nonce {
            return Err(MicropayError::StaleVoucher);
        }
        if self.spent + voucher.amount > self.total_deposit {
            return Err(MicropayError::InsufficientBalance);
        }
        self.spent += voucher.amount;
        self.nonce = voucher.nonce.max(self.nonce + 1);
        Ok(())
    }

    /// Begin closing the channel.
    pub fn close(&mut self, initiated_at: u64) -> Result<(), MicropayError> {
        if self.state != ChannelState::Open {
            return Err(MicropayError::ChannelNotOpen);
        }
        self.state = ChannelState::Closing { initiated_at };
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn settles_voucher_when_balance_is_available() {
        let mut channel = PaymentChannel::new([1u8; 32], [2u8; 32], 100, 10);
        let voucher = PaymentVoucher {
            channel_id: channel.channel_id,
            amount: 40,
            nonce: 1,
        };
        channel.settle(voucher).expect("settle voucher");
        assert_eq!(channel.spent, 40);
    }
}
