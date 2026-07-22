import { Transaction, TransactionType, TDRClient } from './types';

export interface TransactionBuildOptions {
  type: TransactionType;
  to: string;
  amount: number;
  fee?: number;
  payload?: string;
}

export class TransactionBuilder {
  private wallet: any;
  private client: TDRClient;
  private nonceCounter: number = 1;

  constructor(wallet: any, client: TDRClient) {
    this.wallet = wallet;
    this.client = client;
  }

  async buildTransaction(options: TransactionBuildOptions): Promise<Transaction> {
    const { type, to, amount, fee = 10, payload = '' } = options;

    const signedTx = this.wallet.signTransaction({
      from: this.wallet.getAddress(),
      from_pubkey: this.wallet.publicKey,
      to,
      amount,
      fee,
      nonce: this.nonceCounter++,
      tx_type: type,
      payload,
      timestamp: Date.now(),
    });

    return signedTx;
  }

  async buildTransfer(to: string, amount: number, fee?: number): Promise<Transaction> {
    return this.buildTransaction({
      type: 'TRANSFER',
      to,
      amount,
      fee,
    });
  }

  async buildRegisterModel(to: string, version: string, pricePerCall: number, fee?: number): Promise<Transaction> {
    const payload = JSON.stringify({
      type: 'model_registration',
      version,
      price_per_call: pricePerCall,
    });
    return this.buildTransaction({
      type: 'REGISTER_MODEL',
      to,
      amount: pricePerCall,
      fee,
      payload,
    });
  }

  async buildUpdateModel(to: string, version: string, amount: number, fee?: number): Promise<Transaction> {
    const payload = JSON.stringify({
      type: 'model_update',
      version,
    });
    return this.buildTransaction({
      type: 'UPDATE_MODEL',
      to,
      amount,
      fee,
      payload,
    });
  }

  async buildPurchaseApiKey(to: string, amount: number, fee?: number): Promise<Transaction> {
    const payload = JSON.stringify({
      type: 'api_key_purchase',
    });
    return this.buildTransaction({
      type: 'PURCHASE_API_KEY',
      to,
      amount,
      fee,
      payload,
    });
  }

  async broadcast(tx: Transaction): Promise<{ id: string; hash: string }> {
    this.client.sendRawTransaction(tx);
    const hash = `${tx.id}-${Date.now()}`;
    return { id: tx.id, hash };
  }
}
