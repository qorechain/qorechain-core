# QoreChain — Quantum-Safe, AI-Native Layer 1 Blockchain

[![Build](https://github.com/qorechain/qorechain-core/actions/workflows/build.yml/badge.svg)](https://github.com/qorechain/qorechain-core/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-1.3.0-green.svg)](https://github.com/qorechain/qorechain-core/releases/tag/v1.3.0)

QoreChain is the first Layer 1 blockchain with **post-quantum cryptography at genesis**, **AI-native consensus optimization**, a **triple-VM runtime** executing EVM, CosmWasm, and SVM (Solana Virtual Machine) programs on a single chain, a **complete tokenomics engine** with burn mechanics, governance-boosted staking, and controlled inflation, **25 direct cross-chain connections** spanning IBC, EVM, Move, UTXO, and account-model ecosystems, and a **Rollup Development Kit (RDK)** enabling one-click deployment of application-specific rollups with four settlement paradigms. Built on QoreChain SDK v0.53 with 18 custom modules and 45 registered genesis modules.

## Innovations

### Application-Specific Rollup Deployment (v1.3.0)

The `x/rdk` module is QoreChain's Rollup Development Kit — a protocol-native framework for deploying application-specific rollups directly on the host chain. Unlike generic rollup-as-a-service platforms that require external infrastructure, RDK rollups are first-class citizens of the QoreChain consensus, with settlement, data availability, and lifecycle management handled entirely on-chain.

**Four Settlement Paradigms:**

| Mode | Proof System | Finality | Use Case |
|------|-------------|----------|----------|
| **Optimistic** | Fraud proofs | 7-day challenge window, auto-finalized by EndBlocker | General-purpose rollups with low overhead |
| **ZK (Zero-Knowledge)** | SNARK or STARK | Instant on proof verification | DeFi protocols requiring fast finality |
| **Based** | None (L1-sequenced) | ~2 blocks after submission | Gaming and real-time applications leveraging host chain sequencing |
| **Sovereign** | None | Self-determined | Independent chains using QoreChain only for data availability |

**Four Preset Profiles for One-Click Deployment:**

| Profile | Settlement | Block Time | VM | DA Backend | Gas Model |
|---------|-----------|------------|-----|-----------|-----------|
| **DeFi** | ZK + SNARK | 500ms | EVM | Native | EIP-1559 |
| **Gaming** | Based (L1-sequenced) | 200ms | Custom | Native | Flat fee |
| **NFT** | Optimistic + Fraud | 2,000ms | CosmWasm | Celestia | Standard |
| **Enterprise** | Based | 1,000ms | EVM | Native | Subsidized (zero gas) |

**Key Technical Innovations:**

- **Native DA Router** — SHA-256 committed blob storage in the host chain's KVStore with configurable retention periods and automatic pruning via EndBlocker. Celestia IBC integration stubbed for v1.4.0.
- **Bank Escrow Lifecycle** — Rollup creators bond QOR to a module escrow account on creation; bonds are fully returned when the rollup is stopped. A configurable creation burn rate (default 1%) permanently reduces supply with each new rollup.
- **Settlement Engine** — EndBlocker-driven auto-finalization: optimistic batches finalize after the challenge window expires, based batches finalize after 2 host blocks (L1 finality proxy). ZK batches with valid proofs finalize instantly on submission.
- **AI-Assisted Configuration** — The RL consensus module provides advisory `SuggestRollupProfile` and `OptimizeRollupGas` methods, using on-chain reinforcement learning to recommend optimal rollup parameters based on intended use case.
- **Deep Multilayer Integration** — Every rollup is registered as a layer in `x/multilayer` via `RegisterSidechain`, with state anchored via `AnchorState` on each batch settlement. Layer status transitions (Active/Suspended/Decommissioned) are mirrored automatically.
- **Configurable Sequencing** — Three sequencer modes: Dedicated (single operator), Shared (multi-operator set), and Based (host chain proposers order rollup transactions with configurable priority fee share).

### 25 Direct Cross-Chain Connections (v1.2.0)

QoreChain connects to **25 blockchain ecosystems** through two complementary protocols:

- **8 IBC channels** — Cosmos Hub, Osmosis, Noble, Celestia, Stride, Akash, Babylon, and the QoreChain loopback relay. Pre-configured Hermes relayer templates with client updates, misbehaviour detection, and packet clearing every 100 blocks.
- **17 QCB bridge endpoints** — Ethereum, BSC, Solana, Avalanche, Polygon, Arbitrum, TON, Sui, Optimism, Base, Aptos, Bitcoin, NEAR, Cardano, Polkadot, Tezos, and TRON. Each chain has per-type address validation, configurable confirmation depth, circuit breaker volume caps, and PQC-signed validator attestations.
- **12 chain types** — evm, solana, ton, move, sui_move, cosmos_ibc, aptos_move, utxo, near, cardano, polkadot, tezos, tron — covering every major blockchain architecture.

### BTC Restaking via Babylon Protocol (v1.2.0)

The `x/babylon` module integrates with Babylon Protocol to inherit Bitcoin's proof-of-work finality guarantees. Validators can register BTC staking positions (min 100,000 satoshis), and QoreChain epoch state roots are periodically checkpointed to Bitcoin via IBC-relayed Babylon epochs. This provides a secondary finality layer backed by BTC hashrate without requiring any changes to Bitcoin itself.

### Smart Account Abstraction (v1.2.0)

The `x/abstractaccount` module enables programmable accounts backed by smart contracts — similar to ERC-4337 but at the protocol layer. Three account types (`multisig`, `social_recovery`, `session_based`) support session keys with granular permissions and expiry, per-account daily and per-transaction spending rules, and scoped denom allowlists. This enables wallet UX patterns impossible with standard accounts: dApp session keys for mobile, social recovery as a first-class account type, and programmable spend limits enforced at consensus.

### MEV-Protected Block Space (v1.2.0)

The `x/fairblock` module provides a threshold identity-based encryption (tIBE) framework for encrypted mempools. Transactions are cryptographically opaque to block proposers until after inclusion — eliminating the information asymmetry that enables front-running and sandwich attacks. In v1.2.0 the tIBE threshold decryption is stubbed (passthrough), with activation planned once the validator tIBE key ceremony infrastructure is deployed. The `FairBlockDecorator` ante handler is already wired and ready.

### Multi-Token Gas Payment (v1.2.0)

The `x/gasabstraction` module removes the requirement to hold native QOR for transaction fees. Users can pay gas in any accepted IBC-transferred token — currently `ibc/USDC` (1:1 rate) and `ibc/ATOM` (10:1 rate). The `GasAbstractionDecorator` validates non-native fee denoms before the standard `DeductFee` handler, enabling seamless onboarding from other ecosystems without acquiring QOR first.

### 5-Lane Transaction Prioritization (v1.2.0)

Block space is statically partitioned into five priority lanes so that security-critical transactions can never be crowded out by high-volume standard traffic:

| Lane | Priority | Block Space | Purpose |
|------|----------|-------------|---------|
| PQC | 100 | 15% | Post-quantum hybrid-signature transactions |
| MEV | 90 | 20% | FairBlock tIBE-encrypted transactions |
| AI | 80 | 15% | AI-scored and optimized transactions |
| Default | 50 | 40% | Standard transactions |
| Free | 10 | 10% | Gas-abstracted and sponsored transactions |

### Quantum-Safe Hybrid Signatures (v1.1.0)

QoreChain is the only blockchain with **production-ready post-quantum hybrid signatures** — every transaction can carry both a classical Ed25519 signature and an ML-DSA-87 (Dilithium-5) signature simultaneously. This dual-signature architecture means:

- **No wallet disruption** — Classical wallets (Keplr, MetaMask) continue working unmodified. PQC-enabled wallets attach quantum-resistant signatures as TX extensions alongside classical signatures.
- **Governance-controlled migration** — Three enforcement modes controlled by on-chain governance:
  - **Disabled**: Classical signatures only (PQC extensions ignored)
  - **Optional** (default): PQC signatures verified if present; classical fallback for accounts without PQC keys
  - **Required**: Every transaction must carry both classical and PQC signatures — full quantum resistance
- **Seamless onboarding** — Wallets can attach their PQC public key in the TX extension for automatic first-use registration. No separate registration transaction needed.
- **Three-way verification** — The `PQCHybridVerifyDecorator` ante handler processes three scenarios:
  1. Account has PQC key + extension present → verify both signatures
  2. No PQC key + extension with public key → auto-register + verify (onboarding)
  3. No PQC key + no extension → classical only (or reject if `HybridRequired`)

### SHAKE-256 Post-Quantum Hash Foundation (v1.1.0)

A preparatory SHAKE-256 (SHA-3 family) hash utility layer for future post-quantum Merkle tree replacement. Provides variable-length XOF output, fixed 32-byte hashing, Merkle internal node concatenation, and domain-separated hashing — all pure Go, no FFI dependency.

### AI TEE Attestation & Federated Learning Interfaces (v1.1.0)

Production-grade Go interface specifications for:

- **Trusted Execution Environment (TEE) Attestation** — Enclave verification for SGX, TDX, SEV-SNP, and ARM CCA platforms. Defines attestation data structures, verifier interfaces, and execution result types for secure AI model inference inside hardware enclaves.
- **Federated Learning Coordination** — On-chain FL round management with configurable aggregation methods (FedAvg, FedProx, SCAFFOLD), gradient submission, round status tracking, and global model hash anchoring. Enables privacy-preserving distributed model training with cryptographic guarantees.

### On-Chain Reinforcement Learning

A Go-native fixed-point MLP (~73,733 parameters) runs PPO inference directly in the block lifecycle, dynamically tuning consensus parameters (block time, gas limits, pool weights) without any external oracle or sidecar dependency. Deterministic Taylor series math ensures identical results across all validators.

### Triple-Pool Composite Proof-of-Stake

Validators are automatically classified into RPoS (reputation-weighted), DPoS (delegation-weighted), and PoS (standard) pools every 1,000 blocks. Pool-weighted sortition diversifies block production beyond pure stake dominance.

### Quadratic-Reputation Governance (QDRW)

Voting power uses a square-root function dampened by a sigmoid reputation multiplier, preventing whale capture while rewarding long-term honest participation. A 100x stake advantage yields only ~10x voting power. xQORE holdings double voting weight via the formula `sqrt(staked + 2 * xQORE) * ReputationMultiplier(r)`.

### Deflationary Burn Engine

Nine distinct burn channels (transaction fees, governance penalties, slashing, bridge fees, spam deterrence, epoch excess, manual burns, contract callbacks, cross-VM fees) plus a tenth channel for rollup creation burns feed a central burn accounting module. Collected fees are split: 40% to validators, 30% permanently burned, 20% to treasury, 10% to stakers — creating sustained deflationary pressure that increases with network usage. Bridge withdrawal fees and rollup creation fees are automatically routed to the burn module for permanent supply reduction.

### xQORE Governance-Boosted Staking

Users lock QOR to mint xQORE at a 1:1 ratio, gaining doubled governance weight in QDRW votes. Early exit penalties (50% under 30 days, graduated down to 0% after 180 days) are redistributed to remaining holders via PvP rebase — rewarding conviction and punishing mercenary capital. The longer you hold, the more you earn from others' impatience.

### Controlled Emission Decay

Epoch-based inflation follows a multi-year schedule (17.5% → 11% → 7% → 2%) that front-loads incentives for early validators while converging to a sustainable long-term rate. Combined with the burn engine, QOR reaches a net-deflationary equilibrium as transaction volume grows.

### Triple-VM Architecture

The only Layer 1 running three virtual machines (EVM, CosmWasm, SVM) natively within one consensus. Deploy Solidity, Rust/CosmWasm, or BPF programs — all on the same chain, sharing state through cross-VM messaging.

### Cross-VM Interoperability

EVM contracts call CosmWasm contracts via precompile; CosmWasm contracts call EVM contracts via custom messages; SVM programs participate through async event-based bridging. All three VMs communicate seamlessly.

### SVM Runtime with Solana-Compatible RPC

Deploy and execute BPF programs using Solana-compatible tooling. The JSON-RPC server speaks Solana's `getAccountInfo`, `getBalance`, `getSlot` and more — existing Solana clients work out of the box.

### AI-Native Transaction Processing

Statistical isolation forest fraud detection, multi-dimensional risk scoring, and dynamic fee optimization run in the ante handler chain for every transaction.

### Progressive Slashing with Temporal Decay

Repeat offenders face escalating penalties (up to 33% cap) while old infractions decay with a half-life of 100,000 blocks, balancing accountability with forgiveness.

### Custom Bonding Curve

Validator rewards factor in self-bonded stake, loyalty duration (via deterministic logarithm), reputation quality, and protocol phase, incentivizing long-term commitment over short-term stake farming.

## Key Features

- **PQC-Primary Security** — Dilithium-5 signatures + ML-KEM-1024 key exchange, hybrid Ed25519 + ML-DSA-87 via TX extensions, SHAKE-256 hash foundation, algorithm-agile with governance-controlled migration
- **Hybrid Signature Architecture** — Three enforcement modes (disabled/optional/required), auto-registration onboarding, three-way ante verification, governance-upgradeable
- **Rollup Development Kit (RDK)** — Deploy application-specific rollups with 4 settlement modes (optimistic, ZK, based, sovereign), 3 DA backends, 4 preset profiles, native settlement engine, bank escrow lifecycle, and AI-assisted configuration
- **25 Cross-Chain Connections** — 8 IBC channels + 17 QCB bridge endpoints spanning EVM, Move, UTXO, and account-model ecosystems with PQC-signed attestations
- **BTC Restaking** — Babylon Protocol integration for Bitcoin finality guarantees via IBC-relayed epoch checkpoints
- **Account Abstraction** — Programmable accounts with session keys, spending rules, and social recovery at the protocol layer
- **MEV Protection** — FairBlock tIBE encrypted mempool framework with dedicated block lane and validator threshold decryption
- **Gas Abstraction** — Pay transaction fees in IBC-transferred tokens (USDC, ATOM) without holding native QOR
- **5-Lane Block Prioritization** — PQC, MEV, AI, Default, and Free lanes with static block-space quotas
- **RL-Driven Consensus** — On-chain reinforcement learning agent dynamically tunes block time, gas limits, and pool weights with circuit breaker protection
- **Triple-Pool CPoS** — RPoS/DPoS/PoS validator classification with pool-weighted proposer selection
- **QDRW Governance** — Quadratic delegation with reputation weighting and xQORE boost for whale-resistant governance voting
- **Tokenomics Engine** — Burn accounting (10 channels including rollup creation burns), xQORE governance staking (lock/unlock with PvP rebase), epoch-based inflation decay
- **EVM Runtime** — Full Ethereum compatibility with JSON-RPC on port 8545, EIP-1559 gas, ERC-20 token pairs
- **CosmWasm Runtime** — WebAssembly smart contracts with full lifecycle support
- **SVM Runtime** — BPF program deployment and execution via Rust-backed executor with Solana-compatible RPC
- **Cross-VM Bridge** — EVM ↔ CosmWasm (precompile + events) + SVM (async messaging)
- **AI TEE Integration** — Interface specifications for SGX/TDX/SEV-SNP/ARM CCA attestation and secure enclave execution
- **Federated Learning** — On-chain FL coordination interfaces with FedAvg/FedProx/SCAFFOLD aggregation support
- **Progressive Slashing** — Escalating penalties with temporal half-life decay, capped at 33% per infraction
- **Custom Bonding Curve** — Loyalty-aware reward formula with reputation quality factor and protocol phase multiplier
- **Fraud Detection** — Real-time anomaly detection with statistical isolation forest and circuit breaker protection
- **Multilayer Architecture** — Main Chain + Sidechains + Paychains + Rollups with cross-layer fee bundling and state anchoring

## Quick Start

### Docker Compose (Recommended)

```bash
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core
docker compose up -d
```

This starts: QoreChain node (with EVM + CosmWasm + SVM runtimes), AI sidecar, block indexer, Postgres, Prometheus, and Grafana.

### Build from Source

```bash
# Prerequisites: Go 1.26+, CGO enabled, libqorepqc + libqoresvm (see docs/)
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core

# Build the binary (public community build)
CGO_ENABLED=1 go build -o qorechaind ./cmd/qorechaind/

# Initialize a node
./qorechaind init my-node --chain-id qorechain-diana

# Start the node
./qorechaind start
```

### Connect to Testnet

```bash
# Download genesis and configure peers
curl -o ~/.qorechaind/config/genesis.json https://raw.githubusercontent.com/qorechain/qorechain-core/main/config/genesis.json
# Edit ~/.qorechaind/config/config.toml to add persistent peers
./qorechaind start
```

## Architecture

```
┌────────────────────────────────────────────────────────────────────────────┐
│                            QoreChain Node                                  │
│                                                                            │
│  ┌──────────────────── Virtual Machines ──────────────────────┐           │
│  │  ┌───────┐    ┌──────────┐    ┌───────┐                   │           │
│  │  │  EVM  │    │ CosmWasm │    │  SVM  │                   │           │
│  │  │(Sol.) │◄──►│ (Wasm)   │◄──►│ (BPF) │                   │           │
│  │  └───┬───┘    └────┬─────┘    └───┬───┘                   │           │
│  │      └─────────┬───┘──────────────┘                       │           │
│  │           x/crossvm (bridge)                               │           │
│  └────────────────────────────────────────────────────────────┘           │
│                                                                            │
│  ┌────────────────────── Tokenomics ─────────────────────────┐           │
│  │  ┌──────┐   ┌───────┐   ┌───────────┐                    │           │
│  │  │x/burn│   │x/xqore│   │x/inflation│                    │           │
│  │  │10 ch.│   │lock/  │   │epoch decay│                    │           │
│  │  │40/30/│   │unlock │   │17.5→2%    │                    │           │
│  │  │20/10 │   │PvP    │   │           │                    │           │
│  │  └──────┘   └───────┘   └───────────┘                    │           │
│  └────────────────────────────────────────────────────────────┘           │
│                                                                            │
│  ┌──────────── IBC / Bridges (v1.2.0) ───────────────────────┐           │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌──────────┐ │           │
│  │  │x/bridge  │  │x/babylon │  │x/abstract │  │x/gas     │ │           │
│  │  │17 QCB +  │  │BTC re-   │  │ account   │  │abstract. │ │           │
│  │  │8 IBC     │  │staking   │  │session key│  │multi-tok │ │           │
│  │  └────┬─────┘  └────┬─────┘  └───────────┘  └──────────┘ │           │
│  │  QCB Bridge     Babylon IBC   ERC-4337-like   ibc/USDC    │           │
│  │  PQC-signed     BTC finality  social recov.   ibc/ATOM    │           │
│  │  12 chain types checkpoint    spending rules  fee convert  │           │
│  │  ┌──────────┐                                              │           │
│  │  │x/fair    │  5-Lane Prioritization: PQC|MEV|AI|Def|Free │           │
│  │  │ block    │  tIBE encrypted mempool (stub, v1.2.0)      │           │
│  │  └──────────┘                                              │           │
│  └────────────────────────────────────────────────────────────┘           │
│                                                                            │
│  ┌──── Rollup Development Kit (v1.3.0) ──────────────────────┐           │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌──────────┐ │           │
│  │  │ x/rdk    │  │Settlement│  │ DA Router │  │ Profiles │ │           │
│  │  │ 4 modes: │  │Optimistic│  │ Native    │  │ DeFi     │ │           │
│  │  │ opt/zk/  │  │ZK/Based/ │  │ Celestia* │  │ Gaming   │ │           │
│  │  │ based/   │  │Sovereign │  │ Both      │  │ NFT      │ │           │
│  │  │ sovereign│  │          │  │           │  │ Enterpr. │ │           │
│  │  └────┬─────┘  └────┬─────┘  └───────────┘  └──────────┘ │           │
│  │  Bank escrow    Auto-finalize  SHA-256 commit  AI-assisted │           │
│  │  Burn on create EndBlocker     Blob pruning    RL suggest  │           │
│  │  → x/multilayer (RegisterSidechain + AnchorState)          │           │
│  └────────────────────────────────────────────────────────────┘           │
│                                                                            │
│  ┌──────────────┐ ┌──────┐ ┌────────────┐ ┌─────┐                       │
│  │x/rlconsensus │ │ x/ai │ │x/reputation│ │x/qca│                       │
│  │  RL Agent    │ │      │ │            │ │     │                       │
│  └──────┬───────┘ └──┬───┘ └─────┬──────┘ └──┬──┘                       │
│   PPO MLP         AI Engine   Scoring    CPoS Pools                      │
│   Obs/Action      Fraud Det.  Decay      Bonding                         │
│   Circuit Brk     Fee Opt.    Sigmoid    Slashing                        │
│   Rollup Adv.     TEE/FL                 QDRW Gov                        │
│                                                                            │
│  ┌──────┐ ┌──────────┐                                                   │
│  │x/pqc │ │ x/multi  │                                                   │
│  └──┬───┘ └────┬─────┘                                                   │
│  Dilithium    Layer Router                                                │
│  ML-KEM       Sidechains                                                  │
│  Hybrid Sig   + Rollups                                                   │
│  SHAKE-256                                                                │
│                                                                            │
│  ┌──────┐ ┌───────┐                                                      │
│  │x/svm │ │x/cross│                                                      │
│  └──┬───┘ └───┬───┘                                                      │
│  BPF Exec   CrossVM Msg                                                   │
└────────┬──────┬───────────────────────────────────────┬───────────────────┘
         │      │                                       │
   ┌─────┴─────┐│                              ┌───────┴──────┐
   │libqorepqc ││                              │  Indexer     │
   │(Rust PQC) ││                              │  (Postgres)  │
   └───────────┘│                              └──────────────┘
   ┌───────────┐│  ┌──────────┐
   │libqoresvm ││  │AI Sidecar│
   │(Rust BPF) │└──│ (gRPC)   │
   └───────────┘   └──────────┘
```

## Modules

| Module | Description |
|--------|-------------|
| **x/pqc** | Post-quantum cryptography: Dilithium-5, ML-KEM-1024, hybrid Ed25519 + ML-DSA-87 signatures, SHAKE-256 hashing, algorithm-agile governance |
| **x/ai** | AI engine: transaction routing, anomaly detection, fraud detection, fee optimization, TEE attestation interfaces, federated learning coordination |
| **x/rlconsensus** | RL-based dynamic consensus tuning: fixed-point MLP, PPO inference, shadow/conservative/autonomous modes, circuit breaker, rollup advisory |
| **x/reputation** | Validator reputation scoring: multi-factor formula with temporal decay |
| **x/qca** | QoreChain Consensus Algorithm: triple-pool CPoS, bonding curve, progressive slashing, QDRW governance |
| **x/burn** | Central burn accounting: 10 burn channels (including rollup creation burns), EndBlocker fee distribution (40% validators / 30% burned / 20% treasury / 10% stakers) |
| **x/xqore** | Governance-boosted staking: lock QOR → mint xQORE (1:1), graduated exit penalties, PvP rebase redistribution |
| **x/inflation** | Epoch-based emission decay: Y1 17.5% → Y2 11% → Y3-4 7% → Y5+ 2%, configurable epoch length |
| **x/bridge** | Cross-chain bridge (QCB): 17 non-IBC endpoints across 12 chain types, PQC-signed validator attestations, circuit breaker volume caps, bridge fee burn integration |
| **x/babylon** | BTC restaking adapter: Babylon Protocol IBC integration, epoch checkpoints to Bitcoin, staking position lifecycle management |
| **x/abstractaccount** | Smart account abstraction: multisig/social_recovery/session_based accounts, spending rules, session keys with expiry and granular permissions |
| **x/fairblock** | MEV protection: threshold IBE encrypted mempool framework, FairBlockDecorator ante handler, passthrough stub in v1.2.0 |
| **x/gasabstraction** | Multi-token gas payment: accept IBC-transferred tokens (USDC, ATOM) for fees, GasAbstractionDecorator with static conversion rates |
| **x/rdk** | Rollup Development Kit: 4 settlement paradigms (optimistic/ZK/based/sovereign), 3 DA backends, 4 preset profiles, settlement engine with EndBlocker auto-finalization, bank escrow lifecycle, native DA router with blob pruning, AI-assisted profile selection |
| **x/multilayer** | Multi-layer architecture: Sidechains + Paychains + Rollups with cross-layer fee bundling and state anchoring |
| **x/crossvm** | Cross-VM communication: EVM ↔ CosmWasm (precompile) + SVM (async events) |
| **x/svm** | SVM runtime: BPF program deployment/execution, rent collection, Solana-compatible JSON-RPC |
| **x/vm** | VM routing and lifecycle management |

## Cross-Chain Connectivity

### IBC Channels (8)

Pre-configured Hermes relayer templates with client updates, misbehaviour detection, and automatic packet clearing:

| Chain | Prefix | Fee Denom | Notable Integration |
|-------|--------|-----------|---------------------|
| Cosmos Hub | `cosmos` | uatom | Core IBC hub connectivity |
| Osmosis | `osmo` | uosmo | DEX liquidity routing |
| Noble | `noble` | uusdc | Native USDC for gas abstraction |
| Celestia | `celestia` | utia | Data availability layer |
| Stride | `stride` | ustrd | Liquid staking |
| Akash | `akash` | uakt | Decentralized compute |
| Babylon | `bbn` | ubbn | BTC restaking checkpoints |

### QCB Bridge Endpoints (17)

| Chain | Type | Confirmations | Supported Tokens |
|-------|------|---------------|------------------|
| Ethereum | evm | 12 | ETH, USDC, USDT, WBTC |
| BSC | evm | 15 | BNB, BUSD |
| Solana | solana | 32 | SOL, USDC |
| Avalanche | evm | 12 | AVAX, USDC |
| Polygon | evm | 128 | MATIC, USDC |
| Arbitrum | evm | 12 | ETH, ARB, USDC |
| TON | ton | 10 | TON |
| Sui | sui_move | 3 | SUI |
| Optimism | evm | 10 | ETH, USDC, OP |
| Base | evm | 10 | ETH, USDC |
| Aptos | aptos_move | 6 | APT, USDC |
| Bitcoin | utxo | 6 | BTC |
| NEAR | near | 3 | NEAR |
| Cardano | cardano | 15 | ADA |
| Polkadot | polkadot | 12 | DOT |
| Tezos | tezos | 2 | XTZ |
| TRON | tron | 20 | TRX, USDT |

## Token Economics

- **Token**: QOR (display) / uqor (base denomination, 1 QOR = 10^6 uqor)
- **Chain ID**: qorechain-diana (testnet)
- **Bech32 Prefix**: qor (addresses: qor1..., validators: qorvaloper...)

### Emission Schedule

| Year | Inflation Rate | Description |
|------|---------------|-------------|
| 1 | 17.5% | Bootstrap phase — aggressive incentives for early validators |
| 2 | 11.0% | Growth phase — reduced emission as network matures |
| 3–4 | 7.0% | Stabilization — converging toward sustainability |
| 5+ | 2.0% | Long-term — minimal new supply, deflationary via burns |

### Fee Distribution

Every block, collected fees are split:

| Recipient | Share | Purpose |
|-----------|-------|---------|
| Validators | 40% | Block production rewards |
| Burn | 30% | Permanent supply reduction |
| Treasury | 20% | Protocol development fund |
| Stakers | 10% | Passive staking rewards |

### xQORE Exit Penalties

| Lock Duration | Penalty | Destination |
|--------------|---------|-------------|
| < 30 days | 50% | Redistributed to remaining xQORE holders |
| 30–90 days | 35% | Redistributed to remaining xQORE holders |
| 90–180 days | 15% | Redistributed to remaining xQORE holders |
| > 180 days | 0% | Full withdrawal |

## PQC Hybrid Signature Modes

QoreChain's hybrid signature system supports three governance-controlled enforcement levels:

| Mode | Value | Behavior |
|------|-------|----------|
| **Disabled** | 0 | Classical signatures only; PQC extensions ignored |
| **Optional** | 1 (default) | PQC verified if present; classical fallback for non-PQC accounts |
| **Required** | 2 | Both classical and PQC signatures mandatory on every transaction |

Query the current mode:
```bash
qorechaind query pqc hybrid-mode
# or via JSON-RPC:
curl -X POST http://localhost:8545 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qor_getHybridSignatureMode","params":[]}'
```

## JSON-RPC Endpoints

| Port | Protocol | Description |
|------|----------|-------------|
| 8545 | HTTP | EVM JSON-RPC (`eth_`, `web3_`, `net_`, `txpool_`, `qor_` namespaces) |
| 8546 | WebSocket | EVM JSON-RPC (WebSocket) |
| 8899 | HTTP | SVM JSON-RPC (Solana-compatible: `getAccountInfo`, `getBalance`, `getSlot`, etc.) |
| 1317 | HTTP | REST API |
| 9090 | gRPC | gRPC query endpoints |
| 26657 | HTTP | RPC (blocks, transactions, consensus) |

### Custom `qor_` JSON-RPC Methods

| Method | Description |
|--------|-------------|
| `qor_getPQCKeyStatus` | PQC key registration status for an address |
| `qor_getHybridSignatureMode` | Current hybrid signature mode and description |
| `qor_getAIStats` | AI module statistics and configuration |
| `qor_getCrossVMMessage` | Cross-VM message status by ID |
| `qor_getReputationScore` | Validator reputation score breakdown |
| `qor_getLayerInfo` | Multilayer chain layer information |
| `qor_getBridgeStatus` | Bridge connection status for a remote chain |
| `qor_getRLAgentStatus` | RL agent mode, epoch, and circuit breaker state |
| `qor_getRLObservation` | Latest 25-dimension observation vector |
| `qor_getRLReward` | Latest multi-objective reward signal breakdown |
| `qor_getPoolClassification` | Validator pool assignment (RPoS/DPoS/PoS) |
| `qor_getBurnStats` | Total burned, per-source breakdown, last burn height |
| `qor_getXQOREPosition` | xQORE position for an address (locked, balance, lock time) |
| `qor_getInflationRate` | Current inflation rate, epoch, year, total minted |
| `qor_getTokenomicsOverview` | Combined tokenomics dashboard (burn + xQORE + inflation) |
| `qor_getBTCStakingPosition` | BTC restaking position via Babylon Protocol |
| `qor_getAbstractAccount` | Abstract account details, session keys, and spending rules |
| `qor_getFairBlockStatus` | FairBlock tIBE module status and configuration |
| `qor_getGasAbstractionConfig` | Gas abstraction config and accepted fee tokens |
| `qor_getLaneConfiguration` | Block lane priorities and space allocation |
| `qor_getRollupStatus` | Rollup configuration, status, settlement mode, and DA backend |
| `qor_listRollups` | All registered rollups with profile, settlement mode, and status |
| `qor_getSettlementBatch` | Settlement batch details including proof type, status, and finalization height |
| `qor_suggestRollupProfile` | AI-assisted rollup profile recommendation for a given use case |
| `qor_getDABlobStatus` | DA blob storage status, commitment hash, and pruning state |

## CLI Commands

### PQC Module

```bash
# Query PQC module parameters
qorechaind query pqc params

# Query PQC key status for an account
qorechaind query pqc key-status <address>

# Query current hybrid signature enforcement mode
qorechaind query pqc hybrid-mode

# Query PQC verification statistics
qorechaind query pqc stats
```

### RL Consensus Module

```bash
# Query RL agent status
qorechaind query rlconsensus agent-status

# Query latest observation vector
qorechaind query rlconsensus observation

# Query latest reward signal
qorechaind query rlconsensus reward

# Set agent mode (governance-only)
qorechaind tx rlconsensus set-agent-mode <shadow|conservative|autonomous|paused> --from admin
```

### SVM Module

```bash
# Deploy a BPF program
qorechaind tx svm deploy-program ./my_program.so --from mykey

# Execute a program instruction
qorechaind tx svm execute <program-id-base58> <data-hex> --from mykey

# Create an SVM account
qorechaind tx svm create-account <owner-base58> <space> <lamports> --from mykey

# Query an SVM account
qorechaind query svm account <base58-address>

# Query SVM parameters
qorechaind query svm params
```

### Babylon Module (v1.2.0)

```bash
# Query Babylon restaking configuration
qorechaind query babylon config

# Query BTC staking position for an address
qorechaind query babylon staking-position <address>

# Submit a BTC checkpoint (governance)
qorechaind tx babylon submit-checkpoint <epoch> <state-root-hex> --from admin

# Restake BTC (register staking position)
qorechaind tx babylon btc-restake <btc-tx-hash> <amount-satoshis> --from mykey
```

### Abstract Account Module (v1.2.0)

```bash
# Query abstract account configuration
qorechaind query abstractaccount config

# Query abstract account for an address
qorechaind query abstractaccount account <address>

# Create an abstract account
qorechaind tx abstractaccount create <contract-address> <account-type> --from mykey

# Update spending rules
qorechaind tx abstractaccount update-spending-rules <daily-limit> <per-tx-limit> --from mykey
```

### FairBlock Module (v1.2.0)

```bash
# Query FairBlock configuration
qorechaind query fairblock config

# Query FairBlock module status
qorechaind query fairblock status
```

### Gas Abstraction Module (v1.2.0)

```bash
# Query gas abstraction configuration
qorechaind query gasabstraction config

# Query accepted fee tokens and conversion rates
qorechaind query gasabstraction accepted-tokens
```

### RDK Module (v1.3.0)

```bash
# Query a specific rollup
qorechaind query rdk rollup <rollup-id>

# List all registered rollups (optional --creator filter)
qorechaind query rdk list-rollups [--creator <address>]

# Query a settlement batch
qorechaind query rdk batch <rollup-id> [--index <batch-index>]

# Query RDK module configuration
qorechaind query rdk config

# Get AI-assisted rollup profile suggestion
qorechaind query rdk suggest-profile <use-case>

# Create a new rollup (with preset profile or custom flags)
qorechaind tx rdk create-rollup <rollup-id> \
  --settlement optimistic \
  --sequencer dedicated \
  --da native \
  --vm evm \
  --stake 10000000000 \
  --from mykey

# Pause a rollup
qorechaind tx rdk pause-rollup <rollup-id> --reason "maintenance" --from mykey

# Resume a paused rollup
qorechaind tx rdk resume-rollup <rollup-id> --from mykey

# Permanently stop a rollup (returns bond)
qorechaind tx rdk stop-rollup <rollup-id> --from mykey

# Submit a settlement batch
qorechaind tx rdk submit-batch <rollup-id> <state-root> <data-hash> \
  --proof <proof-hex> --proof-type snark --from mykey

# Challenge a batch (optimistic rollups only)
qorechaind tx rdk challenge-batch <rollup-id> <batch-index> <proof-hex> --from mykey
```

## Ante Handler Chain

The transaction processing pipeline for QoreChain SDK transactions:

```
SetUpContext → CircuitBreaker → PQCVerify → PQCHybridVerify → AIAnomaly →
FairBlock → Extension → ValidateBasic → TxTimeout → Memo → MinGasPrice →
ConsumeTxSize → GasAbstraction → DeductFee → SetPubKey → ValidateSigCount →
SigGasConsume → SigVerify → IncrementSequence
```

Key decorators:
- **PQCVerify**: Validates PQC key algorithm status (enabled/deprecated/revoked)
- **PQCHybridVerify**: Extracts and verifies hybrid Ed25519 + ML-DSA-87 TX extensions
- **AIAnomaly**: Statistical fraud detection and risk scoring
- **FairBlock**: Threshold IBE check for encrypted mempool transactions (passthrough in v1.2.0)
- **GasAbstraction**: Validates and converts non-native fee denominations before fee deduction

## Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [RL Consensus Module](docs/RL_CONSENSUS.md)
- [Consensus Enhancements (CPoS, Bonding, Slashing, QDRW)](docs/CONSENSUS.md)
- [Tokenomics (Burn, xQORE, Inflation)](docs/TOKENOMICS.md)
- [PQC Integration Guide](docs/PQC_INTEGRATION.md)
- [Algorithm Agility](docs/ALGORITHM_AGILITY.md)
- [AI Engine](docs/AI_ENGINE.md)
- [EVM Runtime](docs/EVM.md)
- [EVM Precompiles](docs/EVM_PRECOMPILES.md)
- [Cross-VM Bridge](docs/CROSSVM.md)
- [SVM Runtime](docs/SVM.md)
- [Bridge Documentation](docs/BRIDGE.md)
- [IBC Relay Configuration](config/hermes/)
- [Multilayer Architecture](docs/MULTILAYER.md)
- [Rollup Development Kit (RDK)](docs/RDK.md)
- [Running a Testnet Node](docs/RUNNING_TESTNET.md)
- [API Reference](docs/API_REFERENCE.md)

## Infrastructure

- 3 separate Go modules: `qorechain-core/`, `sidecar/`, `indexer/`
- 2 Rust crates: `qorepqc` (PQC cryptography), `qoresvm` (BPF executor)
- 45 registered genesis modules, 18 custom modules
- Docker Compose: 6-service deployment stack
- GitHub Actions: 3 CI/CD workflows (build, release, docker)
- IBC: 8 pre-configured Hermes relayer channel templates

## License

Apache 2.0 — see [LICENSE](LICENSE)

Core blockchain protocol is open source. PQC cryptographic libraries, BPF execution engine, and AI model weights are distributed as pre-compiled binaries under separate licensing terms.
