#include <chrono>
#include <iomanip>
#include <iostream>
#include <map>
#include <openssl/evp.h>
#include <openssl/sha.h>
#include <openssl/err.h>
#include <sstream>
#include <string>
#include <vector>

enum class TransactionType { Transfer, RegisterModel, UpdateModel, PurchaseApiKey };

struct Transaction {
    std::string from;
    std::string from_public_key;
    std::string to;
    uint64_t amount;
    TransactionType tx_type;
    std::string payload;
    std::string signature;
};

struct Block {
    uint64_t index;
    std::string previous_hash;
    uint64_t timestamp;
    std::vector<Transaction> transactions;
    uint64_t nonce;
    std::string block_hash;
};

struct Account {
    std::string address;
    uint64_t balance;
    uint64_t staked;
    bool is_agent;
};

struct ModelEntry {
    std::string id;
    std::string owner;
    std::string version;
    std::string metadata;
    uint64_t price_per_call;
    bool active;
};

enum class ConsensusType { ProofOfStake, ProofOfAuthority };

struct Blockchain {
    std::vector<Block> chain;
    std::vector<Transaction> pending;
    std::map<std::string, Account> ledger;
    std::map<std::string, ModelEntry> registry;
    std::vector<std::string> authorities;
    ConsensusType consensus;
};

std::string to_hex(const std::vector<unsigned char>& bytes) {
    std::ostringstream ss;
    ss << std::hex << std::setfill('0');
    for (unsigned char byte : bytes) {
        ss << std::setw(2) << static_cast<int>(byte);
    }
    return ss.str();
}

std::vector<unsigned char> from_hex(const std::string& hex) {
    std::vector<unsigned char> bytes;
    bytes.reserve(hex.size() / 2);
    for (size_t i = 0; i < hex.size(); i += 2) {
        unsigned int byte;
        std::stringstream ss;
        ss << std::hex << hex.substr(i, 2);
        ss >> byte;
        bytes.push_back(static_cast<unsigned char>(byte));
    }
    return bytes;
}

std::string sha256_string(const std::string& data) {
    unsigned char hash[SHA256_DIGEST_LENGTH];
    SHA256(reinterpret_cast<const unsigned char*>(data.data()), data.size(), hash);
    return to_hex(std::vector<unsigned char>(hash, hash + SHA256_DIGEST_LENGTH));
}

uint64_t now() {
    return std::chrono::duration_cast<std::chrono::seconds>(
               std::chrono::system_clock::now().time_since_epoch())
        .count();
}

struct Wallet {
    EVP_PKEY* keypair = nullptr;
    std::string public_key_hex;
    std::string address;

    Wallet() {
        EVP_PKEY_CTX* ctx = EVP_PKEY_CTX_new_id(NID_ED25519, nullptr);
        if (!ctx) {
            public_key_hex = "";
            address = "";
            keypair = nullptr;
            return;
        }
        if (EVP_PKEY_keygen_init(ctx) != 1) {
            EVP_PKEY_CTX_free(ctx);
            public_key_hex = "";
            address = "";
            keypair = nullptr;
            return;
        }
        if (EVP_PKEY_keygen(ctx, &keypair) != 1) {
            EVP_PKEY_CTX_free(ctx);
            public_key_hex = "";
            address = "";
            keypair = nullptr;
            return;
        }
        EVP_PKEY_CTX_free(ctx);

        size_t pubkey_len = 32;
        std::vector<unsigned char> pubkey(pubkey_len);
        if (EVP_PKEY_get_raw_public_key(keypair, pubkey.data(), &pubkey_len) != 1) {
            public_key_hex = "";
            address = "";
            return;
        }
        public_key_hex = to_hex(pubkey);
        address = sha256_string(std::string(reinterpret_cast<char*>(pubkey.data()), pubkey.size()));
    }

    ~Wallet() {
        if (keypair) {
            EVP_PKEY_free(keypair);
        }
    }

    std::string sign(const std::string& message) const {
        EVP_MD_CTX* ctx = EVP_MD_CTX_new();
        if (!ctx) {
            return "";
        }
        if (EVP_DigestSignInit(ctx, nullptr, nullptr, nullptr, keypair) != 1) {
            EVP_MD_CTX_free(ctx);
            return "";
        }

        size_t sig_len = 0;
        if (EVP_DigestSign(ctx, nullptr, &sig_len, reinterpret_cast<const unsigned char*>(message.data()), message.size()) != 1) {
            EVP_MD_CTX_free(ctx);
            return "";
        }

        std::vector<unsigned char> signature(sig_len);
        if (EVP_DigestSign(ctx, signature.data(), &sig_len, reinterpret_cast<const unsigned char*>(message.data()), message.size()) != 1) {
            EVP_MD_CTX_free(ctx);
            return "";
        }
        EVP_MD_CTX_free(ctx);

        signature.resize(sig_len);
        return to_hex(signature);
    }
};

std::string transaction_payload(const Transaction& tx) {
    std::ostringstream ss;
    ss << tx.from << tx.from_public_key << tx.to << tx.amount;
    ss << static_cast<int>(tx.tx_type) << tx.payload;
    return ss.str();
}

bool verify_signature(const std::string& message, const std::string& signature_hex, const std::string& public_key_hex) {
    auto sig_bytes = from_hex(signature_hex);
    auto pubkey_bytes = from_hex(public_key_hex);

    EVP_PKEY* pubkey = EVP_PKEY_new_raw_public_key(NID_ED25519, nullptr, pubkey_bytes.data(), pubkey_bytes.size());
    if (!pubkey) {
        return false;
    }
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    if (!ctx) {
        EVP_PKEY_free(pubkey);
        return false;
    }
    if (EVP_DigestVerifyInit(ctx, nullptr, nullptr, nullptr, pubkey) != 1) {
        EVP_MD_CTX_free(ctx);
        EVP_PKEY_free(pubkey);
        return false;
    }
    int ok = EVP_DigestVerify(ctx, sig_bytes.data(), sig_bytes.size(), reinterpret_cast<const unsigned char*>(message.data()), message.size());
    EVP_MD_CTX_free(ctx);
    EVP_PKEY_free(pubkey);
    return ok == 1;
}

bool verify_transaction(const Transaction& tx) {
    std::string expected_address = sha256_string(tx.from_public_key);
    if (expected_address != tx.from) {
        return false;
    }
    return verify_signature(transaction_payload(tx), tx.signature, tx.from_public_key);
}

std::string calculate_hash(const Block& block) {
    std::ostringstream ss;
    ss << block.index << block.previous_hash << block.timestamp << block.nonce;
    for (const auto& tx : block.transactions) {
        ss << tx.from << tx.from_public_key << tx.to << tx.amount << static_cast<int>(tx.tx_type) << tx.payload << tx.signature;
    }
    return sha256_string(ss.str());
}

void create_genesis_block(Blockchain& bc) {
    Block genesis{0, "0", now(), {}, 0, "genesis"};
    bc.chain.push_back(genesis);
}

Blockchain new_blockchain(ConsensusType consensus) {
    Blockchain bc;
    bc.consensus = consensus;
    create_genesis_block(bc);
    return bc;
}

void add_account(Blockchain& bc, const std::string& address, uint64_t balance, bool is_agent) {
    bc.ledger[address] = Account{address, balance, 0, is_agent};
}

void add_authority(Blockchain& bc, const std::string& address) {
    bc.authorities.push_back(address);
}

bool validate_transaction(const Blockchain& bc, const Transaction& tx) {
    if (!verify_transaction(tx)) {
        return false;
    }
    auto sender_it = bc.ledger.find(tx.from);
    if (sender_it == bc.ledger.end()) {
        return false;
    }
    const Account& sender = sender_it->second;
    switch (tx.tx_type) {
        case TransactionType::Transfer: {
            return bc.ledger.find(tx.to) != bc.ledger.end() && sender.balance >= tx.amount;
        }
        case TransactionType::RegisterModel: {
            return sender.is_agent && bc.registry.find(tx.to) == bc.registry.end();
        }
        case TransactionType::UpdateModel: {
            auto model_it = bc.registry.find(tx.to);
            return model_it != bc.registry.end() && model_it->second.owner == tx.from;
        }
        case TransactionType::PurchaseApiKey: {
            auto model_it = bc.registry.find(tx.to);
            return model_it != bc.registry.end() && sender.balance >= tx.amount && model_it->second.active;
        }
    }
    return false;
}

Block proof_of_work(Block block) {
    while (true) {
        std::string hash = calculate_hash(block);
        if (hash.rfind("0000", 0) == 0) {
            block.block_hash = hash;
            return block;
        }
        block.nonce++;
    }
}

void apply_block(Blockchain& bc, const Block& block) {
    for (const auto& tx : block.transactions) {
        switch (tx.tx_type) {
            case TransactionType::Transfer: {
                auto sender_it = bc.ledger.find(tx.from);
                auto receiver_it = bc.ledger.find(tx.to);
                if (sender_it != bc.ledger.end() && receiver_it != bc.ledger.end()) {
                    if (sender_it->second.balance >= tx.amount) {
                        sender_it->second.balance -= tx.amount;
                        receiver_it->second.balance += tx.amount;
                    }
                }
                break;
            }
            case TransactionType::RegisterModel: {
                bc.registry[tx.to] = ModelEntry{tx.to, tx.from, tx.payload, tx.payload, tx.amount, true};
                break;
            }
            case TransactionType::UpdateModel: {
                auto it = bc.registry.find(tx.to);
                if (it != bc.registry.end()) {
                    it->second.version = tx.payload;
                    it->second.metadata = tx.payload;
                    it->second.price_per_call = tx.amount;
                }
                break;
            }
            case TransactionType::PurchaseApiKey: {
                auto model_it = bc.registry.find(tx.to);
                if (model_it != bc.registry.end()) {
                    auto sender_it = bc.ledger.find(tx.from);
                    auto receiver_it = bc.ledger.find(model_it->second.owner);
                    if (sender_it != bc.ledger.end() && receiver_it != bc.ledger.end()) {
                        if (sender_it->second.balance >= tx.amount) {
                            sender_it->second.balance -= tx.amount;
                            receiver_it->second.balance += tx.amount;
                        }
                    }
                }
                break;
            }
        }
    }
}

void submit_transaction(Blockchain& bc, const Transaction& tx) {
    bc.pending.push_back(tx);
}

void mine_block(Blockchain& bc) {
    std::string previous_hash = bc.chain.back().block_hash;
    Block block{bc.chain.size(), previous_hash, now(), {}, 0, ""};
    for (const auto& tx : bc.pending) {
        if (validate_transaction(bc, tx)) {
            block.transactions.push_back(tx);
        }
    }
    block = proof_of_work(block);
    apply_block(bc, block);
    bc.chain.push_back(block);
    bc.pending.clear();
}

int main() {
    OpenSSL_add_all_algorithms();
    ERR_load_crypto_strings();

    std::cout << "Starting AI blockchain node (C++)" << std::endl;
    Blockchain bc = new_blockchain(ConsensusType::ProofOfStake);

    Wallet human;
    Wallet agentA;
    Wallet agentB;

    add_account(bc, human.address, 1000000, false);
    add_account(bc, agentA.address, 100000, true);
    add_account(bc, agentB.address, 50000, true);
    add_authority(bc, agentA.address);

    Transaction register_model{agentA.address, agentA.public_key_hex, "model-AI-1", 10, TransactionType::RegisterModel, "v1 inference service", ""};
    register_model.signature = agentA.sign(transaction_payload(register_model));

    Transaction purchase_key{human.address, human.public_key_hex, "model-AI-1", 10, TransactionType::PurchaseApiKey, "buy access", ""};
    purchase_key.signature = human.sign(transaction_payload(purchase_key));

    Transaction payment{agentA.address, agentA.public_key_hex, agentB.address, 5000, TransactionType::Transfer, "AI compute payment", ""};
    payment.signature = agentA.sign(transaction_payload(payment));

    submit_transaction(bc, register_model);
    submit_transaction(bc, purchase_key);
    submit_transaction(bc, payment);

    mine_block(bc);

    std::cout << "Chain length: " << bc.chain.size() << std::endl;
    for (const auto& [addr, account] : bc.ledger) {
        std::cout << "Address: " << addr << ", Balance: " << account.balance << ", Agent: " << account.is_agent << std::endl;
    }
    for (const auto& [id, model] : bc.registry) {
        std::cout << "Model: " << id << ", owner=" << model.owner << ", price=" << model.price_per_call << std::endl;
    }
    std::cout << std::flush;
    return 0;
}
