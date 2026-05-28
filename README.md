# QoreChain — Quantum-Safe, AI-Native Layer 1 Blockchain

[![Build](https://github.com/qorechain/qorechain-core/actions/workflows/build.yml/badge.svg)](https://github.com/qorechain/qorechain-core/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-3.0.2-green.svg)](https://github.com/qorechain/qorechain-core/releases/tag/v3.0.2)

QoreChain is the first Layer 1 blockchain with **post-quantum cryptography at genesis**, **AI-native consensus optimization**, a **triple-VM runtime** executing EVM, CosmWasm, and SVM (Solana Virtual Machine) programs on a single chain, a **native on-chain AMM** with constant-product and stable-swap pricing, a **complete tokenomics engine** with burn mechanics, governance-boosted staking, and controlled inflation, **45 direct cross-chain connections** spanning IBC (with foundation for next-generation IBC v2), EVM, Move, UTXO, Cairo, UNL, SCP, Hashgraph, and account-model ecosystems, a **Rollup Development Kit (RDK)** enabling one-click deployment of application-specific rollups with four settlement paradigms, a **license-gated multi-chain validator bridge** for cross-chain operations across 37 chains, and a **light node network** with stake-weighted reward distribution. Built on Cosmos SDK v0.53 with 21 custom modules and 48 registered genesis modules.

## Innovations

### Native AMM Module (v3.0.0)

The `x/amm` module provides on-chain automated market making with two pricing curves:

- **Constant-product** (`x*y=k`) — Uniswap-V2-style pools for the long tail of token pairs
- **Stable-swap** — Curve-style invariant for low-slippage stable-pair swaps, solved via deterministic Newton iteration

Eight messages cover the complete pool lifecycle: `MsgCreatePool`, `MsgAddLiquidity`, `MsgRemoveLiquidity`, `MsgSwapExactIn`, `MsgSwapExactOut`, `MsgPausePool`, `MsgResumePool`, `MsgSetParams`. Per-pool pause via governance, module-wide kill switch, configurable swap fee with LP-accrual + protocol-fee split, slippage cap, and pool creation fee burned through the standard burn engine. A cross-VM hook lets EVM contracts (and SVM via the existing precompile interface) route swap calls into the AMM, while keeping on-chain routing decisions deterministic. All math uses fixed-point integer arithmetic — zero floating-point in any consensus path.

### IBC Eureka v2 Foundation (v3.0.0)

A `ChainArchitecture` enum on every chain configuration disambiguates classic IBC vs the next-generation IBC v2 stack. New IBC chain onboardings from v3.0.0 forward default to IBC v2 with a configurable client type; existing chains continue on the classic stack and can be migrated by governance proposal. The `x/bridge` module ships public-side packet types and handler-hook interfaces for the next-generation stack and for ICS-27 (Interchain Accounts), ICS-29 (Fee Middleware), and ICS-721 (NFT-IBC) so the validator's cross-network operations can extend cleanly as the upstream modules mature.

### Cross-Network Expansion (v3.0.0)

QoreChain v3.0.0 expands the bridge surface to **37 default chain configurations** across **17 chain architectures** — adding five new architecture families (Cairo VM L2, XRP Ledger UNL, Stellar Consensus Protocol, Hashgraph, Pure Proof-of-Stake) and twenty new chain configurations (zkSync Era, Linea, Scroll, Starknet, Blast, Mantle, Hyperliquid, Berachain, Sonic, Sei, Monad, Plasma, XRPL, Stellar, Hedera, Algorand, Injective, Filecoin FVM, Cronos, Kaia). Each new architecture has a dedicated bridge handler with chain-appropriate confirmation rules; EVM-family chains share the canonical EVM handler with per-chain config injection. The license surface scales accordingly to **74 feature IDs** (1 umbrella + 36 per-chain bridge + 37 per-chain validator) covering all current and previously-onboarded chains.

### Multi-Node Devnet Harness (v3.0.0)

A two-validator local devnet ships with the repo via `docker-compose.devnet.yml` and helper scripts under `scripts/devnet/`. The genesis-creating validator and the joining peer use auto-discovery (genesis fetch + node-ID handshake) so a new operator can boot a working two-node consensus in one command. The included smoke runner exercises block production / finality, liveness slashing, and the bridge sidecar JSON-RPC handshake.

### Light Node Network (v2.0.0)

The `x/lightnode` module enables a decentralized network of light nodes that contribute to chain availability and earn rewards proportional to their uptime and delegated stake. Light node operators register on-chain, send periodic heartbeats to prove liveness, and receive a share of block fees distributed automatically by the EndBlocker.

- **Stake-Weighted Rewards** — Distribution is proportional to `delegated_stake * uptime_factor`, rewarding both commitment and reliability
- **Heartbeat Grace Period** — Configurable tolerance window before marking nodes inactive, accommodating transient network disruptions
- **Auto-Deactivation** — Nodes missing heartbeats beyond the grace period are automatically marked inactive and excluded from rewards until they resume
- **3% Fee Share** — Light node operators receive 3% of block fees, taken from the validator share

### License-Gated Validator Bridge

The `x/license` module provides an on-chain license registry that controls access to extended chain features. Each QoreChain validator can operate as a bridge watcher or full external validator for supported chains — provided they hold the appropriate license.

- **74 Feature IDs** — Granular licenses for bridge watching (`bridge_ethereum`, `bridge_solana`, …) and validator operations (`validator_ethereum`, `validator_solana`, …) across all supported chains, including dedicated licenses for the eight IBC-connected chains
- **License Lifecycle** — Grant → Active → Suspended/Revoked/Expired with governance-controlled administration
- **Auto-Expiry** — EndBlocker automatically expires licenses past their TTL, ensuring stale grants don't persist
- **Sidecar Orchestration** — Licensed validators can run chain-specific sidecar containers managed by a built-in orchestrator that handles container lifecycle, health monitoring, and credential exchange

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

- **Native DA Router** — SHA-256 committed blob storage in the host chain's KVStore with configurable retention periods and automatic pruning via EndBlocker. Celestia IBC integration stubbed for future activation.
- **Bank Escrow Lifecycle** — Rollup creators bond QOR to a module escrow account on creation; bonds are fully returned when the rollup is stopped. A configurable creation burn rate (default 1%) permanently reduces supply with each new rollup.
- **Settlement Engine** — EndBlocker-driven auto-finalization: optimistic batches finalize after the challenge window expires, based batches finalize after 2 host blocks (L1 finality proxy). ZK batches with valid proofs finalize instantly on submission.
- **AI-Assisted Configuration** — The RL consensus module provides advisory `SuggestRollupProfile` and `OptimizeRollupGas` methods, using on-chain reinforcement learning to recommend optimal rollup parameters based on intended use case.
- **Deep Multilayer Integration** — Every rollup is registered as a layer in `x/multilayer` via `RegisterSidechain`, with state anchored via `AnchorState` on each batch settlement.
- **Configurable Sequencing** — Three sequencer modes: Dedicated (single operator), Shared (multi-operator set), and Based (host chain proposers order rollup transactions with configurable priority fee share).

### 45 Direct Cross-Chain Connections

QoreChain connects to **45 blockchain ecosystems** through two complementary protocols:

- **8 IBC channels** — Cosmos Hub, Osmosis, Noble, Celestia, Stride, Akash, Babylon, Injective. Pre-configured Hermes relayer templates with client updates, misbehaviour detection, and packet clearing every 100 blocks.
- **37 QCB bridge endpoints** — Spanning EVM L2/L1, Cairo L2, UNL ledger, SCP ledger, Hashgraph, Pure-PoS, Move, UTXO, and account-model architectures. Each chain has per-type address validation, configurable confirmation depth, circuit breaker volume caps, and PQC-signed validator attestations.
- **17 chain types** — `evm`, `solana`, `ton`, `move`, `sui_move`, `cosmos_ibc`, `aptos_move`, `utxo`, `near`, `cardano`, `polkadot`, `tezos`, `tron`, `starknet`, `xrpl`, `stellar`, `hedera`, `algorand` — covering every major blockchain architecture.

### BTC Restaking via Babylon Protocol (v1.2.0)

The `x/babylon` module integrates with Babylon Protocol to inherit Bitcoin's proof-of-work finality guarantees. Validators can register BTC staking positions (min 100,000 satoshis), and QoreChain epoch state roots are periodically checkpointed to Bitcoin via IBC-relayed Babylon epochs.

### Smart Account Abstraction (v1.2.0)

The `x/abstractaccount` module enables programmable accounts backed by smart contracts — similar to ERC-4337 but at the protocol layer. Three account types (`multisig`, `social_recovery`, `session_based`) support session keys with granular permissions and expiry, per-account daily and per-transaction spending rules, and scoped denom allowlists.

### MEV-Protected Block Space (v1.2.0)

The `x/fairblock` module provides a threshold identity-based encryption (tIBE) framework for encrypted mempools. Transactions are cryptographically opaque to block proposers until after inclusion — eliminating the information asymmetry that enables front-running and sandwich attacks.

### Multi-Token Gas Payment (v1.2.0)

The `x/gasabstraction` module removes the requirement to hold native QOR for transaction fees. Users can pay gas in any accepted IBC-transferred token — currently `ibc/USDC` (1:1 rate) and `ibc/ATOM` (10:1 rate).

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

QoreChain is the only blockchain with **production-ready post-quantum hybrid signatures** — every transaction can carry both a classical Ed25519 signature and an ML-DSA-87 (Dilithium-5) signature simultaneously:

- **No wallet disruption** — Classical wallets (Keplr, MetaMask) continue working unmodified
- **Governance-controlled migration** — Three enforcement modes: Disabled, Optional (default), Required
- **Seamless onboarding** — PQC public keys auto-register via TX extension on first use
- **Three-way verification** — The `PQCHybridVerifyDecorator` handles all combinations of classical and PQC signatures

### On-Chain Reinforcement Learning (PRISM)

A Go-native fixed-point MLP (~73,733 parameters) runs PPO inference directly in the block lifecycle, dynamically tuning consensus parameters (block time, gas limits, pool weights) without any external oracle or sidecar dependency. Deterministic Taylor series math ensures identical results across all validators.

### Triple-Pool Composite Proof-of-Stake

Validators are automatically classified into RPoS (reputation-weighted), DPoS (delegation-weighted), and PoS (standard) pools every 1,000 blocks. Pool-weighted sortition diversifies block production beyond pure stake dominance.

### Quadratic-Reputation Governance (QDRW)

Voting power uses a square-root function dampened by a sigmoid reputation multiplier, preventing whale capture while rewarding long-term honest participation. xQORE holdings double voting weight via the formula `sqrt(staked + 2 * xQORE) * ReputationMultiplier(r)`.

### Deflationary Burn Engine

Ten distinct burn channels feed a central burn accounting module. Collected fees are split: 37% to validators, 30% permanently burned, 20% to treasury, 10% to stakers, and 3% to light node operators — creating sustained deflationary pressure that increases with network usage.

### xQORE Governance-Boosted Staking

Users lock QOR to mint xQORE at a 1:1 ratio, gaining doubled governance weight in QDRW votes. Early exit penalties (50% under 30 days, graduated down to 0% after 180 days) are redistributed to remaining holders via PvP rebase.

### Triple-VM Architecture

The only Layer 1 running three virtual machines (EVM, CosmWasm, SVM) natively within one consensus. Deploy Solidity, Rust/CosmWasm, or BPF programs — all on the same chain, sharing state through cross-VM messaging.

### SVM Runtime with Native Programs & Solana-Compatible RPC

Deploy and execute BPF programs using Solana-compatible tooling. Four native built-in programs (System, SPL Token, ATA, Memo) provide gas-efficient token operations. The JSON-RPC server exposes 20 Solana-compatible methods — existing Solana clients and `@solana/web3.js` work out of the box.

## Key Features

- **PQC-Primary Security** — Dilithium-5 signatures + ML-KEM-1024 key exchange, hybrid Ed25519 + ML-DSA-87 via TX extensions, SHAKE-256 hash foundation, algorithm-agile with governance-controlled migration
- **Hybrid Signature Architecture** — Three enforcement modes (disabled/optional/required), auto-registration onboarding, three-way ante verification
- **Native AMM** — Constant-product + stable-swap pricing curves, cross-VM swap routing, deterministic integer math
- **IBC v2 Foundation** — `ChainArchitecture` enum, ICS-27/29/721 handler-hook interfaces, configurable client type per chain
- **Light Node Network** — Stake-weighted rewards, heartbeat liveness proofs, automatic deactivation, 3% fee share
- **License-Gated Operations** — On-chain license registry with 74 feature IDs, auto-expiry, sidecar container orchestration for multi-chain validator bridge
- **Rollup Development Kit (RDK)** — 4 settlement modes, 3 DA backends, 4 preset profiles, settlement engine with EndBlocker auto-finalization
- **45 Cross-Chain Connections** — 8 IBC channels + 37 QCB bridge endpoints across 17 chain architectures, with PQC-signed attestations
- **BTC Restaking** — Babylon Protocol integration for Bitcoin finality guarantees via IBC-relayed epoch checkpoints
- **Account Abstraction** — Programmable accounts with session keys, spending rules, and social recovery at the protocol layer
- **MEV Protection** — FairBlock tIBE encrypted mempool framework with dedicated block lane
- **Gas Abstraction** — Pay transaction fees in IBC-transferred tokens (USDC, ATOM) without holding native QOR
- **5-Lane Block Prioritization** — PQC, MEV, AI, Default, and Free lanes with static block-space quotas
- **RL-Driven Consensus** — On-chain reinforcement learning agent dynamically tunes block time, gas limits, and pool weights
- **Triple-Pool CPoS** — RPoS/DPoS/PoS validator classification with pool-weighted proposer selection
- **QDRW Governance** — Quadratic delegation with reputation weighting and xQORE boost
- **Tokenomics Engine** — Burn accounting (10 channels), xQORE governance staking, epoch-based inflation decay
- **EVM Runtime** — Full Ethereum compatibility with JSON-RPC on port 8545, EIP-1559 gas, ERC-20 token pairs
- **CosmWasm Runtime** — WebAssembly smart contracts with full lifecycle support
- **SVM Runtime** — BPF program deployment and execution, 4 native programs, 20 Solana-compatible JSON-RPC methods
- **Cross-VM Bridge** — EVM ↔ CosmWasm (precompile + events) + SVM (async messaging)
- **Progressive Slashing** — Escalating penalties with temporal half-life decay, capped at 33% per infraction
- **Multilayer Architecture** — Main Chain + Sidechains + Paychains + Rollups with cross-layer fee bundling and state anchoring
- **Multi-Node Devnet Harness** — Two-validator `docker-compose.devnet.yml` + smoke-test runner shipped in-repo

## Quick Start

### Docker Compose (Recommended)

```bash
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core
docker compose up -d
```

This starts: QoreChain node (with EVM + CosmWasm + SVM runtimes), AI sidecar, block indexer, Postgres, Prometheus, and Grafana.

### Two-Validator Devnet

```bash
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core
docker compose -f docker-compose.devnet.yml up --build -d
./scripts/devnet/smoke.sh
```

See [`scripts/devnet/README.md`](scripts/devnet/README.md) for the operator runbook.

### Build from Source

```bash
# Prerequisites: Go 1.26+, CGO enabled, libqorepqc + libqoresvm (see docs/)
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core

# Build the binary (community build)
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
┌─────────────────────────────────────────────────────────────────────────────┐
│                             QoreChain Node                                  │
│                                                                             │
│  ┌──────────────────── Virtual Machines ──────────────────────┐            │
│  │  ┌───────┐    ┌──────────┐    ┌───────┐                   │            │
│  │  │  EVM  │    │ CosmWasm │    │  SVM  │                   │            │
│  │  │(Sol.) │◄──►│ (Wasm)   │◄──►│ (BPF) │                   │            │
│  │  └───┬───┘    └────┬─────┘    └───┬───┘                   │            │
│  │      └─────────┬───┘──────────────┘                       │            │
│  │           x/crossvm (bridge)  ◄──►  x/amm (swap routing)  │            │
│  └────────────────────────────────────────────────────────────┘            │
│                                                                             │
│  ┌──────────────────── On-Chain Markets ─────────────────────┐            │
│  │  ┌───────┐                                                  │            │
│  │  │ x/amm │  CP + StableSwap pools, cross-VM hook,           │            │
│  │  │       │  protocol-fee share routed via x/burn            │            │
│  │  └───────┘                                                  │            │
│  └────────────────────────────────────────────────────────────┘            │
│                                                                             │
│  ┌────────────────────── Tokenomics ─────────────────────────┐            │
│  │  ┌──────┐   ┌───────┐   ┌───────────┐   ┌───────────┐   │            │
│  │  │x/burn│   │x/xqore│   │x/inflation│   │x/lightnode│   │            │
│  │  │10 ch.│   │lock/  │   │epoch decay│   │heartbeat  │   │            │
│  │  │37/30/│   │unlock │   │17.5→2%    │   │rewards    │   │            │
│  │  │20/10 │   │PvP    │   │           │   │3% share   │   │            │
│  │  └──────┘   └───────┘   └───────────┘   └───────────┘   │            │
│  └────────────────────────────────────────────────────────────┘            │
│                                                                             │
│  ┌──────────── IBC / Bridges / License ─────────────────────┐            │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌─────────┐ │            │
│  │  │x/bridge  │  │x/babylon │  │x/abstract │  │x/gas    │ │            │
│  │  │37 QCB +  │  │BTC re-   │  │ account   │  │abstract.│ │            │
│  │  │8 IBC     │  │staking   │  │session key│  │multi-tok│ │            │
│  │  │+ IBC v2  │  │          │  │           │  │         │ │            │
│  │  └──────────┘  └──────────┘  └───────────┘  └─────────┘ │            │
│  │  ┌──────────┐  ┌──────────┐                               │            │
│  │  │x/fair    │  │x/license │  5-Lane: PQC|MEV|AI|Def|Free │            │
│  │  │ block    │  │74 feature│  tIBE encrypted mempool       │            │
│  │  │tIBE      │  │IDs, auto-│  Sidecar orchestration        │            │
│  │  └──────────┘  │expiry    │                               │            │
│  │                 └──────────┘                               │            │
│  └────────────────────────────────────────────────────────────┘            │
│                                                                             │
│  ┌──── Rollup Development Kit (v1.3.0) ──────────────────────┐            │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌──────────┐ │            │
│  │  │ x/rdk    │  │Settlement│  │ DA Router │  │ Profiles │ │            │
│  │  │ 4 modes: │  │Optimistic│  │ Native    │  │ DeFi     │ │            │
│  │  │ opt/zk/  │  │ZK/Based/ │  │ Celestia* │  │ Gaming   │ │            │
│  │  │ based/   │  │Sovereign │  │ Both      │  │ NFT      │ │            │
│  │  │ sovereign│  │          │  │           │  │ Enterpr. │ │            │
│  │  └──────────┘  └──────────┘  └───────────┘  └──────────┘ │            │
│  └────────────────────────────────────────────────────────────┘            │
│                                                                             │
│  ┌──────────────┐ ┌──────┐ ┌────────────┐ ┌─────┐                        │
│  │x/rlconsensus │ │ x/ai │ │x/reputation│ │x/qca│                        │
│  │  RL Agent    │ │      │ │            │ │     │                        │
│  └──────┬───────┘ └──┬───┘ └─────┬──────┘ └──┬──┘                        │
│   PPO MLP         AI Engine   Scoring    CPoS Pools                       │
│   Obs/Action      Fraud Det.  Decay      Bonding                          │
│   Circuit Brk     Fee Opt.    Sigmoid    Slashing                         │
│   Rollup Adv.     TEE/FL                 QDRW Gov                         │
│                                                                             │
│  ┌──────┐ ┌──────────┐ ┌──────┐ ┌───────┐                                │
│  │x/pqc │ │ x/multi  │ │x/svm │ │x/cross│                                │
│  └──┬───┘ └────┬─────┘ └──┬───┘ └───┬───┘                                │
│  Dilithium    Layer       BPF     CrossVM                                  │
│  ML-KEM       Router      Exec    Messaging                                │
│  Hybrid Sig   Rollups                                                      │
│  SHAKE-256                                                                 │
│                                                                             │
└────────┬──────┬─────────────────────────────────┬───────────────────────────┘
         │      │                                  │
   ┌─────┴─────┐│                         ┌───────┴──────┐
   │libqorepqc ││                         │  Indexer     │
   │(Rust PQC) ││                         │  (Postgres)  │
   └───────────┘│                         └──────────────┘
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
| **x/qca** | Consensus Engine Algorithm: triple-pool CPoS, bonding curve, progressive slashing, QDRW governance |
| **x/burn** | Central burn accounting: 10 burn channels (incl. AMM protocol-fee channel), EndBlocker fee distribution (37% validators / 30% burned / 20% treasury / 10% stakers / 3% light nodes) |
| **x/xqore** | Governance-boosted staking: lock QOR → mint xQORE (1:1), graduated exit penalties, PvP rebase redistribution |
| **x/inflation** | Epoch-based emission decay: Y1 17.5% → Y2 11% → Y3-4 7% → Y5+ 2%, configurable epoch length |
| **x/amm** | Native AMM: constant-product + stable-swap pricing curves, 8 messages (create / add / remove / swap-exact-in / swap-exact-out / pause / resume / set-params), cross-VM swap hook, deterministic integer math |
| **x/bridge** | Cross-chain bridge (QCB): 37 default chain configurations across 17 chain architectures, PQC-signed validator attestations, circuit breaker volume caps, IBC v2 foundation (`ChainArchitecture` enum + ICS-27/29/721 handler hooks) |
| **x/babylon** | BTC restaking adapter: Babylon Protocol IBC integration, epoch checkpoints to Bitcoin, staking position lifecycle |
| **x/abstractaccount** | Smart account abstraction: multisig/social_recovery/session_based accounts, spending rules, session keys with expiry |
| **x/fairblock** | MEV protection: threshold IBE encrypted mempool framework, FairBlockDecorator ante handler |
| **x/gasabstraction** | Multi-token gas payment: accept IBC-transferred tokens (USDC, ATOM) for fees, GasAbstractionDecorator |
| **x/rdk** | Rollup Development Kit: 4 settlement paradigms, 3 DA backends, 4 preset profiles, settlement engine with auto-finalization |
| **x/multilayer** | Multi-layer architecture: Sidechains + Paychains + Rollups with cross-layer fee bundling and state anchoring |
| **x/crossvm** | Cross-VM communication: EVM ↔ CosmWasm (precompile) + SVM (async events), AMM swap routing |
| **x/svm** | SVM runtime: BPF program deployment/execution, native programs (System, SPL Token, ATA, Memo), 20 Solana-compatible JSON-RPC methods |
| **x/lightnode** | Light node network: registration, heartbeat liveness, stake-weighted reward distribution, auto-deactivation |
| **x/license** | On-chain license registry: 74 feature IDs for bridge/validator operations across all supported chains, auto-expiry, lifecycle management |
| **x/vm** | VM routing and lifecycle management |

## Cross-Chain Connectivity

### IBC Channels (8)

| Chain | Prefix | Fee Denom | Notable Integration |
|-------|--------|-----------|---------------------|
| Cosmos Hub | `cosmos` | uatom | Core IBC hub connectivity |
| Osmosis | `osmo` | uosmo | DEX liquidity routing |
| Noble | `noble` | uusdc | Native USDC for gas abstraction |
| Celestia | `celestia` | utia | Data availability layer |
| Stride | `stride` | ustrd | Liquid staking |
| Akash | `akash` | uakt | Decentralized compute |
| Babylon | `bbn` | ubbn | BTC restaking checkpoints |
| Injective | `inj` | inj | DeFi liquidity routing (added v3.0.0) |

### QCB Bridge Endpoints (37)

#### Baseline (17 chains)

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

#### Cross-Network Expansion (20 chains, v3.0.0)

| Chain | Type | Confirmations | Supported Tokens |
|-------|------|---------------|------------------|
| zkSync Era | evm | 12 | ETH, USDC |
| Linea | evm | 12 | ETH, USDC |
| Scroll | evm | 12 | ETH, USDC |
| Starknet | starknet | 12 | ETH, STRK, USDC |
| Blast | evm | 10 | ETH, USDB |
| Mantle | evm | 10 | MNT, USDC |
| Hyperliquid | evm | 10 | USDC |
| Berachain | evm | 10 | BERA, HONEY |
| Sonic | evm | 10 | S, USDC |
| Sei | evm | 10 | SEI, USDC |
| Monad | evm | 30 | MON, USDC |
| Plasma | evm | 10 | XPL, USDT |
| XRP Ledger | xrpl | 4 | XRP, RLUSD |
| Stellar | stellar | 5 | XLM, USDC |
| Hedera | hedera | 4 | HBAR, USDC |
| Algorand | algorand | 4 | ALGO, USDC |
| Filecoin (FVM) | evm | 10 | FIL, USDC |
| Cronos | evm | 12 | CRO, USDC |
| Kaia | evm | 10 | KAIA, USDC |

(Injective is listed in the IBC table above.)

## Token Economics

- **Token**: QOR (display) / uqor (base denomination, 1 QOR = 10^6 uqor)
- **Chain ID**: qorechain-diana (testnet)
- **Bech32 Prefix**: qor (addresses: qor1..., validators: qorvaloper...)

### Emission Schedule

| Year | Inflation Rate | Description |
|------|---------------|-------------|
| 1 | 17.5% | Bootstrap phase — aggressive incentives for early validators |
| 2 | 11.0% | Growth phase — reduced emission as network matures |
| 3-4 | 7.0% | Stabilization — converging toward sustainability |
| 5+ | 2.0% | Long-term — minimal new supply, deflationary via burns |

### Fee Distribution

Every block, collected fees are split:

| Recipient | Share | Purpose |
|-----------|-------|---------|
| Validators | 37% | Block production rewards |
| Burn | 30% | Permanent supply reduction |
| Treasury | 20% | Protocol development fund |
| Stakers | 10% | Passive staking rewards |
| Light Nodes | 3% | Light node operator rewards |

### xQORE Exit Penalties

| Lock Duration | Penalty | Destination |
|--------------|---------|-------------|
| < 30 days | 50% | Redistributed to remaining xQORE holders |
| 30-90 days | 35% | Redistributed to remaining xQORE holders |
| 90-180 days | 15% | Redistributed to remaining xQORE holders |
| > 180 days | 0% | Full withdrawal |

## JSON-RPC Endpoints

| Port | Protocol | Description |
|------|----------|-------------|
| 8545 | HTTP | EVM JSON-RPC (`eth_`, `web3_`, `net_`, `txpool_`, `qor_` namespaces) |
| 8546 | WebSocket | EVM JSON-RPC (WebSocket) |
| 8899 | HTTP | SVM JSON-RPC (Solana-compatible: 20 methods) |
| 1317 | HTTP | REST API |
| 9090 | gRPC | gRPC query endpoints |
| 26657 | HTTP | RPC (blocks, transactions, consensus) |

## Ante Handler Chain

```
SetUpContext → CircuitBreaker → PQCVerify → PQCHybridVerify → AIAnomaly →
FairBlock → Extension → ValidateBasic → TxTimeout → Memo → MinGasPrice →
ConsumeTxSize → GasAbstraction → DeductFee → SetPubKey → ValidateSigCount →
SigGasConsume → SigVerify → IncrementSequence
```

## Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [Sidecar Operator Guide](docs/SIDECAR.md)
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
- [Multilayer Architecture](docs/MULTILAYER.md)
- [Rollup Development Kit (RDK)](docs/RDK.md)
- [Running a Testnet Node](docs/RUNNING_TESTNET.md)
- [Two-Validator Devnet](scripts/devnet/README.md)
- [API Reference](docs/API_REFERENCE.md)

## Infrastructure

- 3 separate Go modules: `qorechain-core/`, `sidecar/`, `indexer/`
- 2 Rust crates: `qorepqc` (PQC cryptography), `qoresvm` (BPF executor + native programs)
- 48 registered genesis modules, 21 custom modules
- Docker Compose: 6-service deployment stack + 2-validator devnet stack
- GitHub Actions: CI/CD workflows (build, release, docker)
- IBC: 8 pre-configured Hermes relayer channel templates

## License

Apache 2.0 — see [LICENSE](LICENSE)

Core blockchain protocol is open source. PQC cryptographic libraries and BPF execution engine are distributed as pre-compiled binaries under separate licensing terms.
