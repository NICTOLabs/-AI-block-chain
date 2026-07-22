export type TransactionType = 'TRANSFER' | 'REGISTER_MODEL' | 'UPDATE_MODEL' | 'PURCHASE_API_KEY';

export interface Transaction {
  id?: string;
  from: string;
  from_pubkey: string;
  to: string;
  amount: number;
  fee: number;
  nonce: number;
  tx_type: TransactionType;
  payload?: string;
  signature?: string;
  timestamp: number;
}

export interface Block {
  index: number;
  author: string;
  previous_hash: string;
  timestamp: number;
  transactions: Transaction[];
  nonce: number;
  block_hash: string;
}

export interface Account {
  address: string;
  balance: number;
  staked: number;
  is_agent: boolean;
}

export interface ModelEntry {
  id: string;
  owner: string;
  version: string;
  metadata: string;
  price_per_call: number;
  active: boolean;
}

export interface Escrow {
  id: string;
  from: string;
  to: string;
  amount: number;
  service_id: string;
  status: string;
}

export interface GovernanceProposal {
  id: string;
  title: string;
  description: string;
  votes: Record<string, boolean>;
  status: string;
}

export interface ServiceAgreement {
  id: string;
  provider: string;
  consumer: string;
  model_id: string;
  price_per_call: number;
  max_calls: number;
  status: string;
}

export interface UsageMeter {
  agreement_id: string;
  usage_count: number;
  total_cost: number;
}

export interface ManagedWallet {
  id: string;
  address: string;
  public_key: string;
  label: string;
  is_agent: boolean;
}

export interface Validator {
  address: string;
  stake: number;
  active: boolean;
  joined_at: number;
  performance: number;
}

export interface NodeState {
  chain: Block[];
  pending: Transaction[];
  ledger: Record<string, Account>;
  registry: Record<string, ModelEntry>;
  consensus: string;
  authorities: string[];
  token_supply: number;
  escrows: Record<string, Escrow>;
  proposals: Record<string, GovernanceProposal>;
  agreements: Record<string, ServiceAgreement>;
  usage_meters: Record<string, UsageMeter>;
  used_nonces: Record<string, number[]>;
  seen_tx_ids: string[];
  audit_trail: AuditEntry[];
}

export interface AuditEntry {
  timestamp: number;
  event: string;
  actor: string;
  details: string;
}

export interface TDRConfig {
  apiUrl: string;
  apiKey?: string;
  authEnabled: boolean;
  timeout: number;
}

export interface TokenomicsInfo {
  currency: string;
  token_supply: number;
  burn_rate_percent: number;
  reward_rate_percent: number;
  base_fee: number;
}

export interface MonitoringInfo {
  height: number;
  pending_transactions: number;
  token_supply: number;
  audit_entries: number;
  peer_count: number;
  trusted_peer_count: number;
  strict_p2p: boolean;
  consensus: string;
}
