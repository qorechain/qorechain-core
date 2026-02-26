# Version History

Public changelog for QoreChain testnet releases.

---

## v1.3.0 -- Rollup Development Kit

**Release focus:** Native rollup infrastructure for sovereign and shared-security rollup deployments.

- **x/rdk module** -- Full Rollup Development Kit with four settlement paradigms: optimistic, zk, pessimistic, and sovereign
- **5 preset profiles** -- Pre-configured rollup templates for DeFi, gaming, NFT, social, and general-purpose use cases
- **Native data availability** -- On-chain DA layer with blob storage, retention management, and pruning lifecycle
- **EndBlocker auto-finalization** -- Automatic batch finalization when the challenge window expires, with no operator intervention required
- **AI-assisted profile selection** -- `suggest-profile` query that recommends an optimal rollup configuration based on the intended use case
- **Multilayer integration** -- Rollups register as layers in the multilayer architecture, inheriting routing, anchoring, and challenge mechanics
- **Bank escrow lifecycle** -- Operator stake is held in escrow during rollup operation and released upon clean shutdown or forfeited on slashing
- **33 unit tests** -- Comprehensive test coverage for rollup creation, batch submission, challenge flow, finalization, DA storage, and lifecycle transitions

---

## v1.2.0 -- IBC & Bridges

**Release focus:** Cross-chain connectivity and advanced account abstractions.

- **25 cross-chain connections** -- 8 IBC channels and 17 QoreChain Bridge (QCB) connections to external networks
- **x/babylon module** -- BTC restaking integration enabling Bitcoin holders to participate in QoreChain staking security
- **x/abstractaccount module** -- Smart account framework with programmable spending rules, session keys, and custom authentication logic
- **x/fairblock module** -- Threshold Identity-Based Encryption (tIBE) stub for MEV-resistant transaction encryption
- **x/gasabstraction module** -- Multi-token gas payment supporting native QOR, IBC-bridged USDC, and IBC-bridged ATOM
- **5-lane TX prioritization** -- Transaction lanes ordered by priority: system, governance, staking, bridge, and general
- **Hermes relayer configs** -- Pre-configured IBC relayer configurations for all supported IBC channels
- **Bridge-to-burn integration** -- Bridge fees are routed through the burn module's 4-way fee distribution (validator, burn, treasury, staker)

---

## v1.1.0 -- PQC Hybrid Signatures

**Release focus:** Post-quantum cryptographic security and algorithm agility.

- **Dual Ed25519 + ML-DSA-87 signatures** -- Every transaction carries both a classical and a post-quantum signature, verified in the AnteHandler chain
- **3 enforcement modes** -- Configurable hybrid signature enforcement: off (mode 0), permissive (mode 1, PQC optional), mandatory (mode 2, PQC required)
- **Auto-registration** -- PQC public keys are automatically registered on the first hybrid transaction, eliminating a separate registration step
- **SHAKE-256 hash foundation** -- All PQC-related hashing operations use SHAKE-256 (SHA-3 family) for quantum-resistant address derivation
- **TEE attestation interfaces** -- Trusted Execution Environment attestation support for proving PQC key generation integrity
- **Federated learning interfaces** -- Interface definitions for privacy-preserving federated model training across validator nodes
- **Algorithm agility framework** -- Pluggable algorithm registry allowing future PQC algorithms to be added via governance without a chain upgrade

---

## v1.0.0 -- Genesis (Tokenomics Engine)

**Release focus:** Chain launch with full tokenomics, multi-VM execution, and AI-assisted operations.

- **x/burn module** -- 10-channel fee burn mechanism with 4-way distribution: 40% validators, 30% burn, 20% treasury, 10% stakers
- **x/xqore module** -- Governance staking derivative with tiered early-unlock penalties and PvP rebase redistribution
- **x/inflation module** -- Epoch-based minting with annual decay from 8% initial rate to a 2% floor, capped at maximum supply
- **RL Consensus module** -- Reinforcement learning agent (PPO) for dynamic chain parameter tuning with circuit breaker safety controls
- **Triple-pool CPoS** -- Classified Proof-of-Stake with Emerald, Sapphire, and Ruby validator pools weighted by reputation scores
- **QDRW governance** -- Dynamic Reward Weighting system allowing governance-approved adjustments to reward distribution across pools
- **EVM + CosmWasm + SVM runtimes** -- Three concurrent execution environments: Ethereum Virtual Machine, CosmWasm smart contracts, and Solana Virtual Machine
- **Cross-VM bridge** -- Seamless message passing and asset transfers between EVM, CosmWasm, and SVM runtimes within a single block
- **PQC Rust FFI** -- High-performance post-quantum cryptography library written in Rust with C FFI bindings for the Go chain binary
- **AI sidecar** -- Off-chain gRPC service providing advanced inference for fraud detection, fee estimation, and network optimization
- **Docker Compose** -- Full containerized deployment with multi-validator testnet, sidecar service, and block indexer
- **Block indexer** -- WebSocket-based block listener with PostgreSQL storage for historical query and analytics
