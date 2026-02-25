# QoreChain — Quantum-Safe, AI-Native Layer 1 Blockchain

[![Build](https://github.com/qorechain/qorechain-core/actions/workflows/build.yml/badge.svg)](https://github.com/qorechain/qorechain-core/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.9.0-green.svg)](https://github.com/qorechain/qorechain-core/releases/tag/v0.9.0)

QoreChain is the first Layer 1 blockchain with **post-quantum cryptography at genesis**, **AI-native consensus optimization**, and a **triple-VM runtime** executing EVM, CosmWasm, and SVM (Solana Virtual Machine) programs on a single chain. Built on QoreChain SDK v0.53 with 9 custom modules and 37 registered genesis modules.

## Innovations

- **On-Chain Reinforcement Learning** — A Go-native fixed-point MLP (~73,733 parameters) runs PPO inference directly in the block lifecycle, dynamically tuning consensus parameters (block time, gas limits, pool weights) without any external oracle or sidecar dependency. Deterministic Taylor series math ensures identical results across all validators.
- **Triple-Pool Composite Proof-of-Stake** — Validators are automatically classified into RPoS (reputation-weighted), DPoS (delegation-weighted), and PoS (standard) pools every 1,000 blocks. Pool-weighted sortition diversifies block production beyond pure stake dominance.
- **Quadratic-Reputation Governance (QDRW)** — Voting power uses a square-root function dampened by a sigmoid reputation multiplier, preventing whale capture while rewarding long-term honest participation. A 100x stake advantage yields only ~10x voting power.
- **Triple-VM Architecture** — The only Layer 1 running three virtual machines (EVM, CosmWasm, SVM) natively within one consensus. Deploy Solidity, Rust/CosmWasm, or BPF programs — all on the same chain, sharing state through cross-VM messaging.
- **Quantum-Safe from Genesis** — Dilithium-5 and ML-KEM-1024 (NIST FIPS 203/204) are first-class citizens, not bolted-on afterthoughts. Algorithm-agile design allows governance-controlled migration to future PQC standards.
- **Cross-VM Interoperability** — EVM contracts call CosmWasm contracts via precompile; CosmWasm contracts call EVM contracts via custom messages; SVM programs participate through async event-based bridging. All three VMs communicate seamlessly.
- **SVM Runtime with Solana-Compatible RPC** — Deploy and execute BPF programs using Solana-compatible tooling. The JSON-RPC server speaks Solana's `getAccountInfo`, `getBalance`, `getSlot` and more — existing Solana clients work out of the box.
- **AI-Native Transaction Processing** — Statistical isolation forest fraud detection, multi-dimensional risk scoring, and dynamic fee optimization run in the ante handler chain for every transaction.
- **Progressive Slashing with Temporal Decay** — Repeat offenders face escalating penalties (up to 33% cap) while old infractions decay with a half-life of 100,000 blocks, balancing accountability with forgiveness.
- **Custom Bonding Curve** — Validator rewards factor in self-bonded stake, loyalty duration (via deterministic logarithm), reputation quality, and protocol phase, incentivizing long-term commitment over short-term stake farming.

## Key Features

- **PQC-Primary Security** — Dilithium-5 signatures + ML-KEM-1024 key exchange, algorithm-agile with governance-controlled migration
- **RL-Driven Consensus** — On-chain reinforcement learning agent dynamically tunes block time, gas limits, and pool weights with circuit breaker protection
- **Triple-Pool CPoS** — RPoS/DPoS/PoS validator classification with pool-weighted proposer selection
- **QDRW Governance** — Quadratic delegation with reputation weighting for whale-resistant governance voting
- **EVM Runtime** — Full Ethereum compatibility with JSON-RPC on port 8545, EIP-1559 gas, ERC-20 token pairs
- **CosmWasm Runtime** — WebAssembly smart contracts with full lifecycle support
- **SVM Runtime** — BPF program deployment and execution via Rust-backed executor with Solana-compatible RPC
- **Cross-VM Bridge** — EVM ↔ CosmWasm (precompile + events) + SVM (async messaging)
- **Universal Bridge (QCB)** — Cross-chain connectivity to Ethereum, Solana, TON, BSC, Avalanche, Polygon, Arbitrum, Sui + native IBC
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
# Prerequisites: Go 1.25+, CGO enabled, libqorepqc + libqoresvm (see docs/)
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
│  ┌──────────────┐ ┌──────┐ ┌────────────┐ ┌─────┐ ┌──────────┐    │
│  │x/rlconsensus │ │ x/ai │ │x/reputation│ │x/qca│ │ x/bridge │    │
│  │  RL Agent    │ │      │ │            │ │     │ │          │    │
│  └──────┬───────┘ └──┬───┘ └─────┬──────┘ └──┬──┘ └────┬─────┘    │
│   PPO MLP         AI Engine   Scoring    CPoS Pools   Bridge      │
│   Obs/Action      Fraud Det.  Decay      Bonding       PQC-Sign   │
│   Circuit Brk     Fee Opt.    Sigmoid    Slashing      IBC        │
│                                          QDRW Gov                  │
│  ┌──────┐ ┌──────────┐                                            │
│  │x/pqc │ │ x/multi  │                                            │
│  └──┬───┘ └────┬─────┘                                            │
│  Dilithium    Layer Router                                         │
│  ML-KEM       Sidechains                                           │
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
| **x/rlconsensus** | RL-based dynamic consensus tuning: fixed-point MLP, PPO inference, shadow/conservative/autonomous modes, circuit breaker |
| **x/pqc** | Post-quantum cryptography: Dilithium-5, ML-KEM-1024, algorithm-agile governance, dual-signature key migration |
| **x/ai** | AI engine: transaction routing, anomaly detection, fraud detection, fee optimization, network optimization |
| **x/reputation** | Validator reputation scoring: multi-factor formula with temporal decay |
| **x/qca** | QoreChain Consensus Algorithm: triple-pool CPoS, bonding curve, progressive slashing, QDRW governance |
| **x/bridge** | Cross-chain bridge (QCB): hub-and-spoke multi-protocol bridge with PQC-secured attestations |
| **x/multilayer** | Multi-layer architecture: Sidechains + Paychains with cross-layer fee bundling |
| **x/crossvm** | Cross-VM communication: EVM ↔ CosmWasm (precompile) + SVM (async events) |
| **x/svm** | SVM runtime: BPF program deployment/execution, rent collection, Solana-compatible JSON-RPC |

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
| `qor_getAIStats` | AI module statistics and configuration |
| `qor_getCrossVMMessage` | Cross-VM message status by ID |
| `qor_getReputationScore` | Validator reputation score breakdown |
| `qor_getLayerInfo` | Multilayer chain layer information |
| `qor_getBridgeStatus` | Bridge connection status for a remote chain |
| `qor_getRLAgentStatus` | RL agent mode, epoch, and circuit breaker state |
| `qor_getRLObservation` | Latest 25-dimension observation vector |
| `qor_getRLReward` | Latest multi-objective reward signal breakdown |
| `qor_getPoolClassification` | Validator pool assignment (RPoS/DPoS/PoS) |

## CLI Commands

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

## Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [RL Consensus Module](docs/RL_CONSENSUS.md)
- [Consensus Enhancements (CPoS, Bonding, Slashing, QDRW)](docs/CONSENSUS.md)
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

## Token Economics

- **Token**: QOR (display) / uqor (base denomination, 1 QOR = 10^6 uqor)
- **Chain ID**: qorechain-diana (testnet)
- **Bech32 Prefix**: qor (addresses: qor1..., validators: qorvaloper...)

## Infrastructure

- 3 separate Go modules: `qorechain-core/`, `sidecar/`, `indexer/`
- 2 Rust crates: `qorepqc` (PQC cryptography), `qoresvm` (BPF executor)
- 37 registered genesis modules, 9 custom modules
- Docker Compose: 6-service deployment stack
- GitHub Actions: 3 CI/CD workflows (build, release, docker)

## License

Apache 2.0 — see [LICENSE](LICENSE)

Core blockchain protocol is open source. PQC cryptographic libraries, BPF execution engine, and AI model weights are distributed as pre-compiled binaries under separate licensing terms.
