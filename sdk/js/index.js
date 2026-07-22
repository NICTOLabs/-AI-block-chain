class TenderClient {
  constructor(baseUrl = 'http://127.0.0.1:8080') {
    this.baseUrl = baseUrl.replace(/\/$/, '');
  }

  async health() {
    const response = await fetch(`${this.baseUrl}/health`);
    return response.json();
  }

  async getChain() {
    const response = await fetch(`${this.baseUrl}/api/chain`);
    return response.json();
  }
}

module.exports = { TenderClient };
