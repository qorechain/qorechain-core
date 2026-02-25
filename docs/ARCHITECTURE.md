# QoreChain Architecture

## Overview

QoreChain is a Layer 1 blockchain built on QoreChain SDK v0.53 with six key innovations:
1. Post-quantum cryptography at genesis with hybrid Ed25519 + ML-DSA-87 signatures (not retrofitted)
2. AI-native consensus optimization with on-chain reinforcement learning
3. Triple-VM runtime (EVM + CosmWasm + SVM) with cross-VM messaging
4. Deflationary tokenomics engine (burn, governance-boosted staking, controlled inflation)
5. Universal cross-chain bridging with PQC security
6. TEE attestation and federated learning coordination for privacy-preserving AI

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

**Hybrid Signatures (v1.1.0)**: Dual Ed25519 + ML-DSA-87 via TX extensions.
- `HybridSignatureMode`: Disabled / Optional (default) / Required
- `PQCHybridVerifyDecorator` checks TX extensions for PQC signatures alongside classical
- Auto-registration: wallets attach PQC pubkey in extension for first-use onboarding
- SHAKE-256 merkle hash foundation for future post-quantum IAVL tree replacement

The PQC AnteHandler chain: `PQCVerify → PQCHybridVerify` runs before standard signature verification.

### x/ai — AI Engine

The AI module implements multi-layer intelligence:

1. **Transaction Routing**: Optimizes validator selection using `argmin(alpha*Latency + beta*Cost + gamma*Security^-1)`
2. **Anomaly Detection**: Statistical isolation forest scoring with z-score normalization
3. **Fraud Detection**: Multi-layered detection (Sybil, DDoS, flash loan, exploit patterns)
4. **Fee Optimization**: EMA-based congestion prediction with urgency-aware fee estimation
5. **Network Optimization**: Parameter recommendations using reward function analysis

The AI Sidecar extends capabilities via QCAI Backend for deep analysis, contract generation, and auditing.

**TEE Attestation Interfaces (v1.1.0)**: Hardware enclave verification for SGX, TDX, SEV-SNP, and ARM CCA platforms. Defines `TEEVerifier` and `TEEExecutor` interfaces for secure AI model inference inside trusted execution environments.

**Federated Learning Interfaces (v1.1.0)**: On-chain FL coordination via `FederatedCoordinator` interface. Supports FedAvg, FedProx, and SCAFFOLD aggregation methods with gradient submission, round management, and global model hash anchoring for privacy-preserving distributed model training.

### x/reputation — Validator Reputation

Scores validators using: `R_i = alpha*S_i + beta*P_i + gamma*C_i + delta*T_i`

Where:
- S_i = Staking weight
- P_i = Performance (uptime, block production)
- C_i = Community participation
- T_i = Temporal component with decay: `exp(-dt/lambda)`

### x/qca — Consensus Algorithm

Implements reputation-weighted proposer selection that integrates with QoreChain Consensus Engine's PrepareProposal/ProcessProposal ABCI hooks. Includes:
- **Triple-Pool CPoS**: RPoS/DPoS/PoS validator pools with weighted sortition
- **Custom Bonding Curve**: Loyalty-aware rewards factoring stake, duration, reputation, and protocol phase
- **Progressive Slashing**: Escalating penalties with temporal half-life decay (capped at 33%)
- **QDRW Governance**: Quadratic delegation with reputation weighting and xQORE boost

### x/burn — Central Burn Accounting

Nine burn channels feed a unified accounting module. The EndBlocker splits collected fees: 40% validators, 30% burned, 20% treasury, 10% stakers. Tracks total burned, per-source breakdown, and burn history.

### x/xqore — Governance-Boosted Staking

Lock QOR to mint xQORE (1:1 ratio) for doubled QDRW governance weight. Graduated exit penalties (50%/35%/15%/0% based on lock duration) are redistributed to remaining holders via PvP rebase. Satisfies `rlconsensus.TokenomicsKeeper` for real balance lookups.

### x/inflation — Epoch-Based Emission

Controlled inflation with year-over-year decay: Y1 17.5%, Y2 11%, Y3-4 7%, Y5+ 2%. Configurable epoch length (default 100 blocks). Tracks current epoch, year, and cumulative minted supply.

### x/rlconsensus — Reinforcement Learning Consensus

On-chain RL agent with Go-native fixed-point MLP (~73,733 parameters). PPO inference tunes consensus parameters (block time, gas limits, pool weights) every 10 blocks. Shadow/conservative/autonomous/paused agent modes with circuit breaker auto-revert.

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
SetUpContext -> CircuitBreaker -> PQCVerify -> PQCHybridVerify -> AIAnomaly -> Extension -> ValidateBasic -> ... -> SigVerify -> IncrementSequence
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
