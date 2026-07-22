# Production Blockchain Network Blueprint

This blueprint assumes a production-grade, permissioned PoS/PoA network running the current Go-based prototype as the node engine.

## 1. Production Architecture & Topology

### Recommended topology
- 7 validator nodes in 3 regions (minimum 4 for quorum, 7 recommended for resilience)
- 3 sentry nodes per region to shield validators from direct internet exposure
- 2-3 load-balanced RPC nodes per region for public API traffic
- 1 bastion host for secure administration
- 1 monitoring stack (Prometheus/Grafana/Loki) and 1 backup/restore host

### Text architecture diagram

```text
Users / Wallets / Integrators
        |
        v
   [Load Balancer]
      /      |      \
 [RPC-A]  [RPC-B] [RPC-C]
      |        |       |
      +--------+-------+
               |
         [Sentry Nodes]
               |
         [Validator Nodes]
      (private subnets, no public ingress)
```

### Node sizing
- Validator nodes: 8 vCPU, 16-32 GB RAM, 1 TB NVMe, 1 Gbps NIC
- Sentry nodes: 4 vCPU, 8-16 GB RAM, 500 GB NVMe, 1 Gbps NIC
- RPC nodes: 8 vCPU, 16-32 GB RAM, 1 TB NVMe, 10 Gbps NIC (or 1 Gbps minimum)

### Network requirements
- P2P port: 3030/tcp
- RPC/API port: 8080/tcp
- Metrics: 9100/tcp for node exporter and /metrics for the node itself

## 2. Security & Infrastructure as Code

### AWS Terraform outline
Use the templates in the Terraform folder to provision:
- VPC and private subnets for validators
- Public subnets for bastion/sentries/RPCs
- Security groups with least-privilege ingress rules
- KMS key for encryption and RAM-backed secret handling
- IAM roles for nodes to access KMS securely

### Security model
- Validators run in private subnets only
- No direct public access to validator nodes
- Only sentries can reach validators on P2P and RPC-only ports
- SSH is allowed only from the bastion host
- RPC nodes are the only nodes exposed publicly

### Key management
- Use AWS KMS or CloudHSM for consensus signing keys
- Never store plaintext private keys on disk
- Start the node with an ephemeral key material path and decrypt into memory at boot
- Keep the key access policy tightly scoped to the validator IAM role

## 3. Monitoring, Logging, and High Availability

### Metrics to monitor
- Blocks produced per minute
- Peer count and peer churn
- Missing blocks / stalled block production
- P2P latency and RPC latency
- CPU, memory, disk I/O, and network saturation
- Validator signing errors and slashing risk

### HA and double-signing prevention
- Run validators with one active signer per host
- Use remote signing via KMS/HSM and disable local signing on backup nodes
- Keep a warm standby system that is not active unless failover is triggered
- Ensure backup nodes do not sign blocks unless explicitly promoted

### Backup strategy
- Snapshot the node data directory and chain state every 15 minutes
- Keep 7 daily and 4 weekly snapshots
- Use immutable storage for backups

## 4. Deployment & Upgrade Playbook

### Deployment steps
1. Provision the base network with Terraform.
2. Deploy the monitoring stack.
3. Bootstrap the validator nodes and install the node binary.
4. Configure environment variables and KMS access.
5. Start sentries and RPCs.
6. Join validators and verify peer health and block production.

### Upgrade and emergency runbook
1. Freeze non-essential changes.
2. Put the backup validator into warm standby mode.
3. Upgrade the node binary on one validator at a time.
4. Validate block production after each rollout.
5. If consensus fails or a fork occurs, roll back to the previous known-good release.
6. Restore from the latest verified snapshot if necessary.

## Recommended next steps
- Replace the current local-only startup with a real deployment workflow using Terraform + Ansible + Docker.
- Add TLS for API traffic and consensus peer traffic.
- Introduce role-based access control and secrets rotation.
- Add immutable logging and audit trails.
