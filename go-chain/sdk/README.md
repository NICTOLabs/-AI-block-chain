# TDR SDK - Official TypeScript SDK

The official TypeScript/JavaScript SDK for the TDR AI-native blockchain. Connect to TDR nodes, query balances, construct transactions, and broadcast them to the mempool.

## Installation

```bash
npm install tdr-sdk
```

## Quick Start

```typescript
import { TDRClient, Wallet, TransactionBuilder } from 'tdr-sdk';

const client = new TDRClient({
  apiUrl: 'http://localhost:8080',
  apiKey: process.env.TENDER_API_KEY,
  authEnabled: true,
  timeout: 30000,
});

async function main() {
  const status = await client.getNetworkStatus();
  console.log('Network height:', status.height);

  const accounts = await client.getAccounts();
  Object.keys(accounts).forEach(addr => {
    console.log(`Account ${addr}: balance=${accounts[addr].balance}, staked=${accounts[addr].staked}`);
  });

  const wallet = Wallet.generate();
  console.log('New wallet address:', wallet.getAddress());

  const tx = await client.buildTransfer('recipient_address', 100, 5);
  await client.sendRawTransaction(tx);
  console.log('Transaction sent:', tx.id);
}

main().catch(console.error);
```

## Wallet Management

```typescript
import { Wallet } from 'tdr-sdk';

const wallet = Wallet.generate();
console.log('Address:', wallet.getAddress());
console.log('Public Key:', wallet.publicKey);
console.log('Private Key:', wallet.privateKey);

const restored = Wallet.fromPrivateKey(wallet.privateKey);
console.log('Restored address:', restored?.getAddress());
```

## Querying the Blockchain

```typescript
const client = new TDRClient({ apiUrl: 'http://localhost:8080' });

const height = await client.getNetworkStatus();
const chain = await client.getChain();
const mempool = await client.getMempool();
const validators = await client.getValidators();

const balance = await client.getBalance('0x...');
const staked = await client.getStakedBalance('0x...');
```

## Transaction Builder

```typescript
const builder = new TransactionBuilder(wallet, client);

const transferTx = await builder.buildTransfer('recipient_address', 1000, 10);
const modelTx = await builder.buildRegisterModel('model_address', 'v1.0', 50);
const updateTx = await builder.buildUpdateModel('model_address', 'v2.0', 30);
const purchaseTx = await builder.buildPurchaseApiKey('model_address', 100);

await builder.broadcast(transferTx);
```

## Governance & Agreements

```typescript
await client.createProposal('Network upgrade', 'Upgrade to v2.0');
await client.voteProposal('proposal-id-xxx', wallet.getAddress());

await client.createServiceAgreement(
  'provider_address',
  'consumer_address',
  'model_id',
  10,
  1000
);

await client.recordUsage('agreement-id-xxx', 50);
```

## Maintaining This SDK

To add new RPC methods:

1. Add the method type to `src/types.ts`
2. Add the HTTP call to `src/client.ts`
3. Rebuild

```bash
npm run build
npm run test
```

## License

MIT - see LICENSE file for details
