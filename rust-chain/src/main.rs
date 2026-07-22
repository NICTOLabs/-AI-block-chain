use ed25519_dalek::{Keypair, PublicKey, Signature, Signer, Verifier};
use hex::{decode as hex_decode, encode as hex_encode};
use rand::rngs::OsRng;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::HashMap;
use std::time::{SystemTime, UNIX_EPOCH};

#[derive(Debug, Serialize, Deserialize, Clone)]
enum TransactionType {
    Transfer,
    RegisterModel,
    UpdateModel,
    PurchaseApiKey,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
struct Transaction {
    from: String,
    from_pubkey: String,
    to: String,
    amount: u64,
    tx_type: TransactionType,
    payload: Option<String>,
    signature: Option<String>,
    timestamp: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
struct Block {
    index: u64,
    author: String,
    previous_hash: String,
    timestamp: u64,
    transactions: Vec<Transaction>,
    nonce: u64,
    block_hash: String,
}

#[derive(Debug, Clone)]
struct Account {
    address: String,
    balance: u64,
    staked: u64,
    is_agent: bool,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
struct ModelEntry {
    id: String,
    owner: String,
    version: String,
    metadata: String,
    price_per_call: u64,
    active: bool,
}

#[derive(Debug)]
enum ConsensusType {
    ProofOfStake,
    ProofOfAuthority,
}

struct Blockchain {
    chain: Vec<Block>,
    pending: Vec<Transaction>,
    ledger: HashMap<String, Account>,
    registry: HashMap<String, ModelEntry>,
    authorities: Vec<String>,
    consensus: ConsensusType,
}

struct Wallet {
    keypair: Keypair,
}

impl Wallet {
    fn new() -> Self {
        let mut csprng = OsRng {};
        Wallet {
            keypair: Keypair::generate(&mut csprng),
        }
    }

    fn address(&self) -> String {
        let digest = Sha256::digest(self.keypair.public.as_bytes());
        hex_encode(digest)
    }

    fn public_key_hex(&self) -> String {
        hex_encode(self.keypair.public.as_bytes())
    }

    fn sign_transaction(&self, tx: &Transaction) -> String {
        let payload = tx.signing_payload();
        let signature: Signature = self.keypair.sign(payload.as_bytes());
        hex_encode(signature.to_bytes())
    }
}

impl Transaction {
    fn new(from: String, to: String, amount: u64, tx_type: TransactionType, payload: Option<String>) -> Self {
        Transaction {
            from,
            from_pubkey: String::new(),
            to,
            amount,
            tx_type,
            payload,
            signature: None,
            timestamp: now(),
        }
    }

    fn signing_payload(&self) -> String {
        let mut tx = self.clone();
        tx.signature = None;
        serde_json::to_string(&tx).unwrap()
    }
}

impl Blockchain {
    fn new(consensus: ConsensusType) -> Self {
        let mut bc = Self {
            chain: Vec::new(),
            pending: Vec::new(),
            ledger: HashMap::new(),
            registry: HashMap::new(),
            authorities: Vec::new(),
            consensus,
        };
        bc.create_genesis_block();
        bc
    }

    fn create_genesis_block(&mut self) {
        let genesis = Block {
            index: 0,
            author: String::from("genesis"),
            previous_hash: String::from("0"),
            timestamp: now(),
            transactions: vec![],
            nonce: 0,
            block_hash: String::from("genesis"),
        };
        self.chain.push(genesis);
    }

    fn add_account(&mut self, address: &str, balance: u64, is_agent: bool) {
        let account = Account {
            address: address.to_owned(),
            balance,
            staked: 0,
            is_agent,
        };
        self.ledger.insert(address.to_owned(), account);
    }

    fn add_authority(&mut self, address: String) {
        self.authorities.push(address);
    }

    fn submit_transaction(&mut self, tx: Transaction) {
        self.pending.push(tx);
    }

    fn mine_block(&mut self) {
        let previous_hash = self.chain.last().unwrap().block_hash.clone();
        let index = self.chain.len() as u64;
        let author = self.select_block_author();

        let pending = std::mem::take(&mut self.pending);
        let valid_transactions: Vec<Transaction> = pending
            .into_iter()
            .filter(|tx| self.validate_transaction(tx))
            .collect();

        let block = Block {
            index,
            author,
            previous_hash: previous_hash.clone(),
            timestamp: now(),
            transactions: valid_transactions,
            nonce: 0,
            block_hash: String::new(),
        };

        let mut block = self.proof_of_work(block);
        block.block_hash = calculate_hash(&block);
        self.apply_block(&block);
        self.chain.push(block);
    }

    fn select_block_author(&self) -> String {
        match self.consensus {
            ConsensusType::ProofOfAuthority => self
                .authorities
                .first()
                .cloned()
                .unwrap_or_else(|| String::from("authority")),
            ConsensusType::ProofOfStake => String::from("stake-producer"),
        }
    }

    fn proof_of_work(&self, mut block: Block) -> Block {
        loop {
            let hash = calculate_hash(&block);
            if hash.starts_with("0000") {
                block.block_hash = hash;
                break;
            }
            block.nonce += 1;
        }
        block
    }

    fn validate_transaction(&self, tx: &Transaction) -> bool {
        if tx.signature.is_none() {
            return false;
        }

        if !verify_transaction_signature(tx) {
            return false;
        }

        match tx.tx_type {
            TransactionType::Transfer => {
                let sender = self.ledger.get(&tx.from);
                let receiver = self.ledger.get(&tx.to);
                sender.is_some() && receiver.is_some() && sender.unwrap().balance >= tx.amount
            }
            TransactionType::RegisterModel => {
                let sender = self.ledger.get(&tx.from);
                sender.map_or(false, |acc| acc.is_agent) && !self.registry.contains_key(&tx.to)
            }
            TransactionType::UpdateModel => {
                let sender = self.ledger.get(&tx.from);
                self.registry
                    .get(&tx.to)
                    .map_or(false, |entry| entry.owner == tx.from && sender.is_some())
            }
            TransactionType::PurchaseApiKey => {
                let sender = self.ledger.get(&tx.from);
                let model = self.registry.get(&tx.to);
                sender.is_some()
                    && model.is_some()
                    && sender.unwrap().balance >= tx.amount
                    && model.unwrap().active
            }
        }
    }

    fn apply_block(&mut self, block: &Block) {
        for tx in block.transactions.iter() {
            match tx.tx_type {
                TransactionType::Transfer => {
                    if tx.from == tx.to {
                        continue;
                    }

                    let sender_balance_ok = self
                        .ledger
                        .get(&tx.from)
                        .map_or(false, |sender| sender.balance >= tx.amount);
                    if !sender_balance_ok || !self.ledger.contains_key(&tx.to) {
                        continue;
                    }

                    {
                        let sender = self.ledger.get_mut(&tx.from).unwrap();
                        sender.balance -= tx.amount;
                    }
                    {
                        let receiver = self.ledger.get_mut(&tx.to).unwrap();
                        receiver.balance += tx.amount;
                    }
                }
                TransactionType::RegisterModel => {
                    let entry = ModelEntry {
                        id: tx.to.clone(),
                        owner: tx.from.clone(),
                        version: tx.payload.clone().unwrap_or_else(|| "v1".to_string()),
                        metadata: tx
                            .payload
                            .clone()
                            .unwrap_or_else(|| "AI model registration".to_string()),
                        price_per_call: tx.amount,
                        active: true,
                    };
                    self.registry.insert(tx.to.clone(), entry);
                }
                TransactionType::UpdateModel => {
                    if let Some(entry) = self.registry.get_mut(&tx.to) {
                        entry.version = tx.payload.clone().unwrap_or_else(|| entry.version.clone());
                        entry.metadata = tx
                            .payload
                            .clone()
                            .unwrap_or_else(|| entry.metadata.clone());
                        entry.price_per_call = tx.amount;
                    }
                }
                TransactionType::PurchaseApiKey => {
                    if let Some(model) = self.registry.get(&tx.to) {
                        let sender_balance_ok = self
                            .ledger
                            .get(&tx.from)
                            .map_or(false, |sender| sender.balance >= tx.amount);
                        if !sender_balance_ok {
                            continue;
                        }

                        {
                            let sender = self.ledger.get_mut(&tx.from).unwrap();
                            sender.balance -= tx.amount;
                        }
                        {
                            let receiver = self.ledger.get_mut(&model.owner).unwrap();
                            receiver.balance += tx.amount;
                        }
                    }
                }
            }
        }
    }
}

fn verify_transaction_signature(tx: &Transaction) -> bool {
    if tx.signature.is_none() {
        return false;
    }

    let pubkey_bytes = match hex_decode(&tx.from_pubkey) {
        Ok(bytes) => bytes,
        Err(_) => return false,
    };

    let signature_bytes = match hex_decode(tx.signature.as_ref().unwrap()) {
        Ok(bytes) => bytes,
        Err(_) => return false,
    };

    let public_key = match PublicKey::from_bytes(&pubkey_bytes) {
        Ok(key) => key,
        Err(_) => return false,
    };

    let derived_address = address_from_public_key(&public_key);
    if derived_address != tx.from {
        return false;
    }

    let signature = match Signature::from_bytes(&signature_bytes) {
        Ok(sig) => sig,
        Err(_) => return false,
    };

    public_key.verify(tx.signing_payload().as_bytes(), &signature).is_ok()
}

fn address_from_public_key(pubkey: &PublicKey) -> String {
    let digest = Sha256::digest(pubkey.as_bytes());
    hex_encode(digest)
}

fn calculate_hash(block: &Block) -> String {
    let block_string = serde_json::to_string(block).unwrap();
    let mut hasher = Sha256::new();
    hasher.update(block_string);
    format!("{:x}", hasher.finalize())
}

fn now() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs()
}

fn main() {
    println!("Starting AI blockchain node (Rust)");
    let mut chain = Blockchain::new(ConsensusType::ProofOfStake);

    let human_wallet = Wallet::new();
    let agent_a_wallet = Wallet::new();
    let agent_b_wallet = Wallet::new();

    let human_address = human_wallet.address();
    let agent_a_address = agent_a_wallet.address();
    let agent_b_address = agent_b_wallet.address();

    chain.add_account(&human_address, 1_000_000, false);
    chain.add_account(&agent_a_address, 100_000, true);
    chain.add_account(&agent_b_address, 50_000, true);
    chain.add_authority(agent_a_address.clone());

    let mut register_model = Transaction::new(
        agent_a_address.clone(),
        String::from("model-AI-1"),
        10,
        TransactionType::RegisterModel,
        Some(String::from("v1: image generator")),
    );
    register_model.from_pubkey = agent_a_wallet.public_key_hex();
    register_model.signature = Some(agent_a_wallet.sign_transaction(&register_model));

    let mut purchase_key = Transaction::new(
        human_address.clone(),
        String::from("model-AI-1"),
        10,
        TransactionType::PurchaseApiKey,
        Some(String::from("purchase api access")),
    );
    purchase_key.from_pubkey = human_wallet.public_key_hex();
    purchase_key.signature = Some(human_wallet.sign_transaction(&purchase_key));

    let mut transfer_payment = Transaction::new(
        agent_a_address.clone(),
        agent_b_address.clone(),
        5_000,
        TransactionType::Transfer,
        Some(String::from("AI compute payment")),
    );
    transfer_payment.from_pubkey = agent_a_wallet.public_key_hex();
    transfer_payment.signature = Some(agent_a_wallet.sign_transaction(&transfer_payment));

    chain.submit_transaction(register_model);
    chain.submit_transaction(purchase_key);
    chain.submit_transaction(transfer_payment);
    chain.mine_block();

    println!("Chain length: {}", chain.chain.len());
    println!("Ledger state: {:#?}", chain.ledger);
    println!("Registry: {:#?}", chain.registry);
}
