import axios, { AxiosInstance, AxiosRequestConfig } from 'axios';
import { Transaction, Block, Account, NodeState, TokenomicsInfo, MonitoringInfo, Validator, AuditEntry, ManagedWallet, TDRConfig } from './types';

export class TDRClient {
  private client: AxiosInstance;
  private config: TDRConfig;

  constructor(config: TDRConfig) {
    this.config = config;
    this.client = axios.create({
      baseURL: config.apiUrl,
      timeout: config.timeout || 30000,
      headers: {
        'Content-Type': 'application/json',
        ...(config.apiKey ? { Authorization: `Bearer ${config.apiKey}` } : {}),
      },
    });
  }

  private getHeaders(): Record<string, string> {
    return {
      'Content-Type': 'application/json',
      ...(this.config.apiKey ? { Authorization: `Bearer ${this.config.apiKey}` } : {}),
    };
  }

  async getNetworkStatus(): Promise<MonitoringInfo> {
    const response = await this.client.get('/api/monitoring', { headers: this.getHeaders() });
    return response.data;
  }

  async getChain(): Promise<Block[]> {
    const response = await this.client.get('/api/chain', { headers: this.getHeaders() });
    const state: Partial<NodeState> = response.data;
    return state.chain || [];
  }

  async getAccounts(): Promise<Record<string, Account>> {
    const response = await this.client.get('/api/accounts', { headers: this.getHeaders() });
    return response.data;
  }

  async getAccount(address: string): Promise<Account | undefined> {
    const accounts = await this.getAccounts();
    return accounts[address];
  }

  async getBalance(address: string): Promise<number> {
    const account = await this.getAccount(address);
    return account?.balance || 0;
  }

  async getStakedBalance(address: string): Promise<number> {
    const account = await this.getAccount(address);
    return account?.staked || 0;
  }

  async getMempool(): Promise<Transaction[]> {
    const response = await this.client.get('/api/mempool');
    return response.data || [];
  }

  async getValidators(): Promise<{ consensus: string; authorities: string[]; next_validator: string; validators: Record<string, Validator> }> {
    const response = await this.client.get('/api/validators');
    return response.data;
  }

  async registerValidator(address: string, stake: number): Promise<{ status: string }> {
    const response = await this.client.post('/api/validators/register', { address, stake }, { headers: this.getHeaders() });
    return response.data;
  }

  async getAuditTrail(): Promise<AuditEntry[]> {
    const response = await this.client.get('/api/audit');
    return response.data || [];
  }

  async getTokenomics(): Promise<TokenomicsInfo> {
    const response = await this.client.get('/api/tokenomics');
    return response.data;
  }

  async getRegistry(): Promise<Record<string, ModelEntry>> {
    const response = await this.client.get('/api/registry');
    return response.data;
  }

  async getManagedWallets(): Promise<ManagedWallet[]> {
    const response = await this.client.get('/api/managed-wallets', { headers: this.getHeaders() });
    return response.data.wallets || [];
  }

  async createManagedWallet(label: string, isAgent: boolean): Promise<ManagedWallet> {
    const response = await this.client.post('/api/managed-wallets', { label, is_agent: isAgent }, { headers: this.getHeaders() });
    return response.data;
  }

  async createEscrow(from: string, to: string, amount: number, serviceId: string): Promise<{ id: string; from: string; to: string; amount: number; service_id: string; status: string }> {
    const response = await this.client.post('/api/escrow', { from, to, amount, service_id: serviceId }, { headers: this.getHeaders() });
    return response.data;
  }

  async createProposal(title: string, description: string): Promise<{ id: string; title: string; description: string; votes: Record<string, boolean>; status: string }> {
    const response = await this.client.post('/api/proposals', { title, description }, { headers: this.getHeaders() });
    return response.data;
  }

  async voteProposal(proposalId: string, voter: string): Promise<void> {
    await this.client.post('/api/proposals/vote', { proposal_id: proposalId, voter }, { headers: this.getHeaders() });
  }

  async createServiceAgreement(provider: string, consumer: string, modelId: string, pricePerCall: number, maxCalls: number): Promise<ServiceAgreement> {
    const response = await this.client.post('/api/agreements', { provider, consumer, model_id: modelId, price_per_call: pricePerCall, max_calls: maxCalls }, { headers: this.getHeaders() });
    return response.data;
  }

  async recordUsage(agreementId: string, usageCount: number): Promise<{ agreement_id: string; usage_count: number; total_cost: number }> {
    const response = await this.client.post('/api/usage', { agreement_id: agreementId, usage_count: usageCount }, { headers: this.getHeaders() });
    return response.data;
  }

  async stake(address: string, amount: number): Promise<{ address: string; amount: number }> {
    const response = await this.client.post('/api/stake', { address, amount }, { headers: this.getHeaders() });
    return response.data;
  }

  async getExplorerBlock(index: number): Promise<Block | null> {
    try {
      const response = await this.client.get(`/block/${index}`, { baseURL: this.config.apiUrl.replace(/\/$/, '') });
      return response.data.block;
    } catch {
      return null;
    }
  }

  async transfer(to: string, amount: number, tx: Transaction): Promise<any> {
    const response = await this.client.post('/api/transactions', tx, { headers: this.getHeaders() });
    return response.data;
  }

  async getHealth(): Promise<{ status: string }> {
    const response = await this.client.get('/health');
    return response.data;
  }

  async sendRawTransaction(tx: Transaction): Promise<void> {
    await this.client.post('/api/transactions', tx, { headers: this.getHeaders() });
  }
}
