# QoreChain Architecture

## Overview

QoreChain is a Layer 1 blockchain built on QoreChain SDK v0.53 with seven key innovations:
1. Post-quantum cryptography at genesis with hybrid Ed25519 + ML-DSA-87 signatures (not retrofitted)
2. AI-native consensus optimization with on-chain reinforcement learning
3. Triple-VM runtime (EVM + CosmWasm + SVM) with cross-VM messaging
4. Deflationary tokenomics engine (burn, governance-boosted staking, controlled inflation)
5. Universal cross-chain bridging with PQC security
6. TEE attestation and federated learning coordination for privacy-preserving AI
7. Application-specific rollup deployment with four settlement paradigms (v1.3.0)

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

Ten burn channels feed a unified accounting module: transaction fees, slashing, governance, cross-VM gas, bridge fees, inflation surplus, xQORE penalties, reputation penalties, system cleanup, and rollup creation burns. The EndBlocker splits collected fees: 40% validators, 30% burned, 20% treasury, 10% stakers. Tracks total burned, per-source breakdown, and burn history.

### x/xqore — Governance-Boosted Staking

Lock QOR to mint xQORE (1:1 ratio) for doubled QDRW governance weight. Graduated exit penalties (50%/35%/15%/0% based on lock duration) are redistributed to remaining holders via PvP rebase. Satisfies `rlconsensus.TokenomicsKeeper` for real balance lookups.

### x/inflation — Epoch-Based Emission

Controlled inflation with year-over-year decay: Y1 17.5%, Y2 11%, Y3-4 7%, Y5+ 2%. Configurable epoch length (default 100 blocks). Tracks current epoch, year, and cumulative minted supply.

### x/rlconsensus — Reinforcement Learning Consensus

On-chain RL agent with Go-native fixed-point MLP (~73,733 parameters). PPO inference tunes consensus parameters (block time, gas limits, pool weights) every 10 blocks. Shadow/conservative/autonomous/paused agent modes with circuit breaker auto-revert.

### x/bridge — Cross-Chain Bridge (QCB)

Hub-and-spoke multi-protocol bridge supporting 25 direct cross-chain connections:
- **IBC chains** (8): QoreChain Hub, Osmosis, Noble, Celestia, Stride, Akash, Babylon
- **EVM chains** (6): Ethereum, BSC, Avalanche, Optimism, Base, Tron
- **Non-IBC chains** (5): Solana, TON, Aptos, Bitcoin, NEAR
- **Additional** (2): Cardano, Polkadot, Tezos
- Native IBC with PQC-secured packets

Security: 7-of-10 PQC multisig, 24h challenge period, circuit breakers. Bridge withdrawal fees route to the x/burn module.

### x/rdk — Rollup Development Kit (v1.3.0)

Application-specific rollup deployment framework supporting four settlement paradigms:

- **Optimistic**: Interactive fraud proofs with configurable challenge window (default 7 days). Auto-finalization in EndBlocker after the challenge period expires.
- **ZK (Zero-Knowledge)**: SNARK/STARK validity proofs with instant finality upon proof verification. Recursive proof aggregation supported.
- **Based**: L1-sequenced rollups where host chain proposers order rollup transactions. Includes forced inclusion delay and priority fee sharing with L1 proposers.
- **Sovereign**: Self-sequenced rollups with independent ordering and no proof requirements.

**Preset Profiles**: DeFi (ZK/SNARK, EVM, 500ms blocks, EIP-1559), Gaming (Based, custom VM, 200ms blocks, flat fee), NFT (Optimistic, CosmWasm, 2s blocks), Enterprise (Based, EVM, 1s blocks, subsidized gas), Custom (full configuration control).

**Data Availability**: Native KV-store blob backend (functional) with Celestia IBC backend (stubbed in v1.3.0). Configurable blob retention and automatic pruning via EndBlocker.

**Integration**: Rollups register as `rollup` layer type in x/multilayer for state anchoring via HCS. Rollup creation burns 1% of stake through x/burn. RL consensus module provides AI-assisted profile selection and gas optimization.

### x/babylon — BTC Restaking Adapter (v1.2.0)

BTC restaking coordination module for Babylon protocol integration. Manages BTC staking positions, checkpoint submissions, and epoch snapshots.

### x/abstractaccount — Account Abstraction (v1.2.0)

Smart-contract wallet accounts with spending rules, session keys, daily/per-tx limits, and allowed denomination restrictions.

### x/fairblock — Threshold IBE Mempool (v1.2.0)

FairBlock threshold identity-based encryption stub for encrypted mempool ordering. Passthrough ante decorator in v1.2.0; tIBE decryption reserved for future activation.

### x/gasabstraction — Gas Abstraction (v1.2.0)

Multi-denomination fee payment supporting IBC-received tokens. Static conversion rates for testnet (uqor 1:1, ibc/USDC 1:1, ibc/ATOM 10:1). Ante decorator converts non-native fee denominations to native equivalents.

## AnteHandler Chain

```
SetUpContext → CircuitBreaker → PQCVerify → PQCHybridVerify → AIAnomaly →
FairBlock → SVMComputeBudget → SVMDeductFee → Extension → ValidateBasic →
TxTimeout → Memo → MinGasPrice → ConsumeTxSize → GasAbstraction → DeductFee →
SetPubKey → ValidateSigCount → SigGasConsume → SigVerify → IncrementSequence
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
