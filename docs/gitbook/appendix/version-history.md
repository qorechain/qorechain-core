# Version History

Public changelog for QoreChain testnet releases. Current release: **v3.1.70**.
Releases from v3.1.x onward are published as lockstep git tags on
`qorechain-core` (and the private extensions repo); the v1.x/v2.x/v3.0.x
entries below are grouped by feature milestone.

---

## v3.1.x -- Hardening, proof systems, and native light-clients

**Release focus:** production hardening of the v3.x feature set, real proof
verification, and a sweep of operational fixes found during multi-validator
verification.

- **Custom-module transaction + query pipeline** -- all 14 proto-bound custom
  modules (pqc, ai, amm, bridge, crossvm, license, lightnode, multilayer, qca,
  rdk, reputation, rlconsensus, svm, abstractaccount) now have real
  proto-generated `Msg` and `Query` services with working CLI subcommands.
  (Earlier builds printed "proto-bound query handlers are not yet generated".)
- **STARK verifier** -- a from-scratch transparent STARK verifier in
  `x/rdk/stark` (prime field, Fiat-Shamir transcript, Merkle commitments, FRI
  low-degree test, AIR + DEEP composition), wired into RDK settlement behind an
  opt-in `QSTK` verification-key gate.
- **Native bridge light-clients** -- deposit verification that checks the source
  chain's own consensus/state instead of trusting the bridge validator quorum:
  Ethereum (BLS12-381 sync committee + MPT inclusion), L2 state-anchor
  verification for the Ethereum rollups, Wormhole VAA, BLS/ed25519 quorums,
  Starknet Pedersen, and Bitcoin SPV. PQC-attestation is retained as a fallback.
- **rlconsensus out of shadow mode** -- the RL agent runs in `conservative` mode
  with live observation/reward vectors and an armed circuit breaker.
- **License authority persistence (v3.1.70)** -- the `x/license` grant authority
  is now stored on-chain and survives node restarts; previously it lived only in
  the keeper struct (set in `InitGenesis`) and reverted to the gov module account
  after any restart, blocking all new grants.
- **EVM chain-id config fix (v3.1.69)** -- a fresh node's `app.toml` now defaults
  `[evm] evm-chain-id` to the resolved network chain ID (testnet `9800`) instead
  of the cosmos/evm default `262144`; without this the JSON-RPC backend rejected
  every `eth_sendRawTransaction` with "incorrect chain-id".
- **Tokenomics fee split** -- fee distribution is the 5-way 37/30/20/10/3
  (validators / burn / treasury / stakers / light nodes); the light-node 3% is
  taken from the validator share.
- **Docker + node-only deployment** -- a `docker-compose.node.yml` for exchanges
  and integrators that need only to sync/query/submit (no AI, bridge, or other
  licensed components), plus libwasmvm/runtime fixes for the full stack.

## v3.0.0 -- Native AMM and cross-network expansion

**Release focus:** on-chain trading and a major bridge surface expansion.

- **x/amm module** -- native constant-product and stable-swap AMM with
  governance-pausable pools, LP-accrual + protocol-fee split, slippage caps, and
  a cross-VM hook so EVM/SVM contracts can route swaps into the AMM.
- **Cross-network expansion** -- the bridge surface grows to **37 default chain
  configurations across 17 chain architectures**, adding five architecture
  families (Cairo VM L2, XRP Ledger UNL, Stellar Consensus Protocol, Hashgraph,
  Pure Proof-of-Stake) and twenty new chains (zkSync Era, Linea, Scroll,
  Starknet, Blast, Mantle, Hyperliquid, Berachain, Sonic, Sei, Monad, Plasma,
  XRPL, Stellar, Hedera, Algorand, Injective, Filecoin, Cronos, Kaia). The
  license surface scales to the bridge/validator per-chain feature IDs.
- **IBC Eureka (v2) foundation** -- a `ChainArchitecture` enum disambiguates
  classic IBC vs IBC v2, with public packet types and handler hooks for ICS-27
  (Interchain Accounts), ICS-29 (Fee Middleware), and ICS-721 (NFT-IBC).
- **Total: 45 cross-chain connections** (8 IBC + 37 QCB), **21 custom modules**,
  **48 registered genesis modules**.

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

- **x/burn module** -- 10-channel fee burn mechanism with 5-way distribution: 37% validators, 30% burn, 20% treasury, 10% stakers, 3% light nodes
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
