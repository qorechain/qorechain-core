# QoreChain — Quantum-Safe, AI-Native Layer 1 Blockchain

[![Build](https://github.com/qorechain/qorechain-core/actions/workflows/build.yml/badge.svg)](https://github.com/qorechain/qorechain-core/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-1.1.0-green.svg)](https://github.com/qorechain/qorechain-core/releases/tag/v1.1.0)

QoreChain is the first Layer 1 blockchain with **post-quantum cryptography at genesis**, **AI-native consensus optimization**, a **triple-VM runtime** executing EVM, CosmWasm, and SVM (Solana Virtual Machine) programs on a single chain, and a **complete tokenomics engine** with burn mechanics, governance-boosted staking, and controlled inflation. Built on QoreChain SDK v0.53 with 12 custom modules and 40 registered genesis modules.

## Innovations

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

Nine distinct burn channels (transaction fees, governance penalties, slashing, bridge fees, spam deterrence, epoch excess, manual burns, contract callbacks, cross-VM fees) feed a central burn accounting module. Collected fees are split: 40% to validators, 30% permanently burned, 20% to treasury, 10% to stakers — creating sustained deflationary pressure that increases with network usage.

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
- **RL-Driven Consensus** — On-chain reinforcement learning agent dynamically tunes block time, gas limits, and pool weights with circuit breaker protection
- **Triple-Pool CPoS** — RPoS/DPoS/PoS validator classification with pool-weighted proposer selection
- **QDRW Governance** — Quadratic delegation with reputation weighting and xQORE boost for whale-resistant governance voting
- **Tokenomics Engine** — Burn accounting (9 channels), xQORE governance staking (lock/unlock with PvP rebase), epoch-based inflation decay
- **EVM Runtime** — Full Ethereum compatibility with JSON-RPC on port 8545, EIP-1559 gas, ERC-20 token pairs
- **CosmWasm Runtime** — WebAssembly smart contracts with full lifecycle support
- **SVM Runtime** — BPF program deployment and execution via Rust-backed executor with Solana-compatible RPC
- **Cross-VM Bridge** — EVM ↔ CosmWasm (precompile + events) + SVM (async messaging)
- **Universal Bridge (QCB)** — Cross-chain connectivity to Ethereum, Solana, TON, BSC, Avalanche, Polygon, Arbitrum, Sui + native IBC
- **AI TEE Integration** — Interface specifications for SGX/TDX/SEV-SNP/ARM CCA attestation and secure enclave execution
- **Federated Learning** — On-chain FL coordination interfaces with FedAvg/FedProx/SCAFFOLD aggregation support
- **Progressive Slashing** — Escalating penalties with temporal half-life decay, capped at 33% per infraction
- **Custom Bonding Curve** — Loyalty-aware reward formula with reputation quality factor and protocol phase multiplier
- **Fraud Detection** — Real-time anomaly detection with statistical isolation forest and circuit breaker protection
- **Multilayer Architecture** — Main Chain + Sidechains + Paychains with cross-layer fee bundling

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
┌──────────────────────────────────────────────────────────────────────┐
│                          QoreChain Node                              │
│                                                                      │
│  ┌──────────────────── Virtual Machines ──────────────────────┐     │
│  │  ┌───────┐    ┌──────────┐    ┌───────┐                   │     │
│  │  │  EVM  │    │ CosmWasm │    │  SVM  │                   │     │
│  │  │(Sol.) │◄──►│ (Wasm)   │◄──►│ (BPF) │                   │     │
│  │  └───┬───┘    └────┬─────┘    └───┬───┘                   │     │
│  │      └─────────┬───┘──────────────┘                       │     │
│  │           x/crossvm (bridge)                               │     │
│  └────────────────────────────────────────────────────────────┘     │
│                                                                      │
│  ┌────────────────────── Tokenomics ─────────────────────────┐     │
│  │  ┌──────┐   ┌───────┐   ┌───────────┐                    │     │
│  │  │x/burn│   │x/xqore│   │x/inflation│                    │     │
│  │  │9 chan.│   │lock/  │   │epoch decay│                    │     │
│  │  │40/30/│   │unlock │   │17.5→2%    │                    │     │
│  │  │20/10 │   │PvP    │   │           │                    │     │
│  │  └──────┘   └───────┘   └───────────┘                    │     │
│  └────────────────────────────────────────────────────────────┘     │
│                                                                      │
│  ┌──────────────┐ ┌──────┐ ┌────────────┐ ┌─────┐ ┌──────────┐    │
│  │x/rlconsensus │ │ x/ai │ │x/reputation│ │x/qca│ │ x/bridge │    │
│  │  RL Agent    │ │      │ │            │ │     │ │          │    │
│  └──────┬───────┘ └──┬───┘ └─────┬──────┘ └──┬──┘ └────┬─────┘    │
│   PPO MLP         AI Engine   Scoring    CPoS Pools   Bridge      │
│   Obs/Action      Fraud Det.  Decay      Bonding       PQC-Sign   │
│   Circuit Brk     Fee Opt.    Sigmoid    Slashing      IBC        │
│                   TEE/FL                 QDRW Gov                  │
│  ┌──────┐ ┌──────────┐                                            │
│  │x/pqc │ │ x/multi  │                                            │
│  └──┬───┘ └────┬─────┘                                            │
│  Dilithium    Layer Router                                         │
│  ML-KEM       Sidechains                                           │
│  Hybrid Sig                                                        │
│  SHAKE-256                                                         │
│                                                                      │
│  ┌──────┐ ┌───────┐                                                │
│  │x/svm │ │x/cross│                                                │
│  └──┬───┘ └───┬───┘                                                │
│  BPF Exec   CrossVM Msg                                             │
└────────┬──────┬───────────────────────────────────────┬─────────────┘
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
| **x/burn** | Central burn accounting: 9 burn channels, EndBlocker fee distribution (40% validators / 30% burned / 20% treasury / 10% stakers) |
| **x/xqore** | Governance-boosted staking: lock QOR → mint xQORE (1:1), graduated exit penalties, PvP rebase redistribution |
| **x/inflation** | Epoch-based emission decay: Y1 17.5% → Y2 11% → Y3-4 7% → Y5+ 2%, configurable epoch length |
| **x/rlconsensus** | RL-based dynamic consensus tuning: fixed-point MLP, PPO inference, shadow/conservative/autonomous modes, circuit breaker |
| **x/pqc** | Post-quantum cryptography: Dilithium-5, ML-KEM-1024, hybrid Ed25519 + ML-DSA-87 signatures, SHAKE-256 hashing, algorithm-agile governance |
| **x/ai** | AI engine: transaction routing, anomaly detection, fraud detection, fee optimization, TEE attestation interfaces, federated learning coordination |
| **x/reputation** | Validator reputation scoring: multi-factor formula with temporal decay |
| **x/qca** | QoreChain Consensus Algorithm: triple-pool CPoS, bonding curve, progressive slashing, QDRW governance |
| **x/bridge** | Cross-chain bridge (QCB): hub-and-spoke multi-protocol bridge with PQC-secured attestations |
| **x/multilayer** | Multi-layer architecture: Sidechains + Paychains with cross-layer fee bundling |
| **x/crossvm** | Cross-VM communication: EVM ↔ CosmWasm (precompile) + SVM (async events) |
| **x/svm** | SVM runtime: BPF program deployment/execution, rent collection, Solana-compatible JSON-RPC |

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

## Ante Handler Chain

The transaction processing pipeline for QoreChain SDK transactions:

```
SetUpContext → CircuitBreaker → PQCVerify → PQCHybridVerify → AIAnomaly →
Extension → ValidateBasic → TxTimeout → Memo → MinGasPrice → ConsumeTxSize →
DeductFee → SetPubKey → ValidateSigCount → SigGasConsume → SigVerify →
IncrementSequence
```

Key decorators:
- **PQCVerify**: Validates PQC key algorithm status (enabled/deprecated/revoked)
- **PQCHybridVerify**: Extracts and verifies hybrid Ed25519 + ML-DSA-87 TX extensions
- **AIAnomaly**: Statistical fraud detection and risk scoring

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
- [Multilayer Architecture](docs/MULTILAYER.md)
- [Running a Testnet Node](docs/RUNNING_TESTNET.md)
- [API Reference](docs/API_REFERENCE.md)

## Infrastructure

- 3 separate Go modules: `qorechain-core/`, `sidecar/`, `indexer/`
- 2 Rust crates: `qorepqc` (PQC cryptography), `qoresvm` (BPF executor)
- 40 registered genesis modules, 12 custom modules
- Docker Compose: 6-service deployment stack
- GitHub Actions: 3 CI/CD workflows (build, release, docker)

## License

Apache 2.0 — see [LICENSE](LICENSE)

Core blockchain protocol is open source. PQC cryptographic libraries, BPF execution engine, and AI model weights are distributed as pre-compiled binaries under separate licensing terms.
