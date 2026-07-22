import nacl from 'tweetnacl';
import bs58 from 'bs58';
import { v4 as uuidv4 } from 'uuid';
import { Transaction, TransactionType } from './types';

export interface KeyPair {
  publicKey: string;
  privateKey: string;
}

export class Wallet {
  public address: string;
  public publicKey: string;
  private privateKey: string;

  constructor(keyPair: KeyPair) {
    this.publicKey = keyPair.publicKey;
    this.privateKey = keyPair.privateKey;
    this.address = this._deriveAddress();
  }

  static generate(): Wallet {
    const keyPair = nacl.box.keyPair();
    const publicKey = bs58.encode(keyPair.publicKey);
    const privateKey = bs58.encode(keyPair.secretKey);
    return new Wallet({ publicKey, privateKey });
  }

  static fromPrivateKey(privateKey: string): Wallet | null {
    try {
      const secretKey = bs58.decode(privateKey);
      const publicKey = bs58.encode(nacl.box.keyPair.fromSecretKey(secretKey).publicKey);
      return new Wallet({ publicKey, privateKey });
    } catch {
      return null;
    }
  }

  signTransaction(tx: Omit<Transaction, 'id' | 'signature' | 'timestamp'>, nonce?: number): Transaction {
    const now = Date.now();
    const txToSign: any = { ...tx, timestamp: now };
    if (!txToSign.id) {
      txToSign.id = `tx-${now}-${uuidv4().split('-')[0]}`;
    }
    if (nonce) {
      txToSign.nonce = nonce;
    }

    const payloadString = JSON.stringify({ ...txToSign, signature: '' });
    const messageBytes = new TextEncoder().encode(payloadString);

    const decodedPk = bs58.decode(this.privateKey);
    const signature = nacl.sign.detached(messageBytes, decodedPk);
    const signatureHex = Buffer.from(signature).toString('hex');

    return { ...txToSign, signature: signatureHex };
  }

  getAddress(): string {
    return this.address;
  }

  private _deriveAddress(): string {
    const sha = require('crypto').createHash('sha256');
    const pubBuffer = bs58.decode(this.publicKey);
    const hash = sha.update(pubBuffer).digest('hex');
    return hash;
  }
}
