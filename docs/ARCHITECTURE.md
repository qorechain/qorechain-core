# QoreChain Architecture

## Overview

QoreChain is a Layer 1 blockchain built on QoreChain SDK v0.53 with three key innovations:
1. Post-quantum cryptography at genesis (not retrofitted)
2. AI-native consensus optimization
3. Universal cross-chain bridging with PQC security

## Module Architecture

### x/pqc — Post-Quantum Cryptography

The PQC module provides quantum-safe cryptographic operations via a Rust FFI bridge:

- **Dilithium-5**: Digital signatures (NIST FIPS 204)
  - Public key: 2592 bytes
  - Private key: 4896 bytes
  - Signature: 4627 bytes
- **ML-KEM-1024**: Key encapsulation (NIST FIPS 203)
  - Public key: 1568 bytes
  - Ciphertext: 1568 bytes
  - Shared secret: 32 bytes
- **Quantum Random Beacon**: Verifiable random output for consensus

The PQC AnteHandler runs before standard signature verification, providing a quantum-safe security layer.

### x/ai — AI Engine

The AI module implements multi-layer intelligence:

1. **Transaction Routing**: Optimizes validator selection using `argmin(alpha*Latency + beta*Cost + gamma*Security^-1)`
2. **Anomaly Detection**: Statistical isolation forest scoring with z-score normalization
3. **Fraud Detection**: Multi-layered detection (Sybil, DDoS, flash loan, exploit patterns)
4. **Fee Optimization**: EMA-based congestion prediction with urgency-aware fee estimation
5. **Network Optimization**: Parameter recommendations using reward function analysis

The AI Sidecar extends capabilities via QCAI Backend for deep analysis, contract generation, and auditing.

### x/reputation — Validator Reputation

Scores validators using: `R_i = alpha*S_i + beta*P_i + gamma*C_i + delta*T_i`

Where:
- S_i = Staking weight
- P_i = Performance (uptime, block production)
- C_i = Community participation
- T_i = Temporal component with decay: `exp(-dt/lambda)`

### x/qca — Consensus Algorithm

Implements reputation-weighted proposer selection that integrates with QoreChain Consensus Engine's PrepareProposal/ProcessProposal ABCI hooks.

### x/bridge — Cross-Chain Bridge (QCB)

Hub-and-spoke multi-protocol bridge:
- Ethereum (lock-mint model)
- Solana (Wormhole-compatible)
- TON (cross-chain messaging)
- Generic EVM (BSC, Avalanche)
- Native IBC with PQC-secured packets

Security: 7-of-10 PQC multisig, 24h challenge period, circuit breakers.

## AnteHandler Chain

```
SetUpContext -> CircuitBreaker -> PQCVerify -> AIAnomaly -> Extension -> ValidateBasic -> ... -> SigVerify -> IncrementSequence
```

## Deployment Architecture

```
Docker Compose Stack:
  - qorechain-node (QoreChain Consensus Engine + QoreChain SDK)
  - ai-sidecar (gRPC + QCAI Backend)
  - indexer (WebSocket + Postgres)
  - postgres (Block data storage)
  - prometheus (Metrics collection)
  - grafana (Dashboards)
```
