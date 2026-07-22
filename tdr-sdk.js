#!/usr/bin/env node
const axios = require('axios');

class TDRClient {
  constructor({ baseUrl = 'http://127.0.0.1:8080', timeout = 5000 } = {}) {
    this.baseUrl = baseUrl;
    this.client = axios.create({ baseURL: baseUrl, timeout });
  }

  async health() {
    const { data } = await this.client.get('/health');
    return data;
  }

  async getAccount(address) {
    const { data } = await this.client.get(`/api/accounts?address=${encodeURIComponent(address)}`);
    return data;
  }

  async submitTransaction(tx) {
    const { data } = await this.client.post('/api/transactions', tx);
    return data;
  }

  async getMempool() {
    const { data } = await this.client.get('/api/mempool');
    return data;
  }
}

if (typeof module !== 'undefined') {
  module.exports = { TDRClient };
}
