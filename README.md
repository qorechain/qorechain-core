# QoreChain — Quantum-Safe, AI-Native Layer 1 Blockchain

[![Build](https://github.com/qorechain/qorechain-core/actions/workflows/build.yml/badge.svg)](https://github.com/qorechain/qorechain-core/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

QoreChain is the first Layer 1 blockchain with **post-quantum cryptography at genesis** and **AI-native consensus optimization**. Built on Cosmos SDK v0.53 with custom modules for PQC signatures (Dilithium-5, ML-KEM-1024), AI-driven transaction routing, and universal cross-chain bridging.

## Key Features

- **PQC-Primary Security** — Dilithium-5 signatures + ML-KEM-1024 key exchange (NIST FIPS 203/204 compliant)
- **AI-Native Consensus** — Reputation-weighted validator selection with AI-driven optimization
- **Universal Bridge (QCB)** — Cross-chain connectivity to Ethereum, Solana, TON, BSC, Avalanche + native IBC
- **Fraud Detection** — Real-time anomaly detection with statistical isolation forest and circuit breaker protection
- **Smart Contract AI** — AI-powered contract generation (17 chains) and security auditing via AWS Bedrock

## Quick Start

### Docker Compose (Recommended)

```bash
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core
docker compose up -d
```

This starts: QoreChain node, AI sidecar, block indexer, Postgres, Prometheus, and Grafana.

### Build from Source

```bash
# Prerequisites: Go 1.25+, CGO enabled, libqorepqc (see docs/PQC_INTEGRATION.md)
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core

# Build the binary
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
┌─────────────────────────────────────────────────────────────┐
│                     QoreChain Node                          │
│  ┌──────┐ ┌──────┐ ┌────────────┐ ┌─────┐ ┌────────┐     │
│  │ x/pqc│ │ x/ai │ │x/reputation│ │x/qca│ │x/bridge│     │
│  └──┬───┘ └──┬───┘ └─────┬──────┘ └──┬──┘ └───┬────┘     │
│     │        │            │           │        │            │
│  Dilithium  AI Engine   Scoring    Consensus  Bridge       │
│  ML-KEM     Fraud Det.  Decay      Selection  PQC-Sign    │
│             Fee Opt.                           Circuit Brk │
│             Network Opt                        IBC+PQC     │
└─────┬───────┬───────────────────────────┬─────────────────┘
      │       │                           │
      │  ┌────┴────┐              ┌───────┴──────┐
      │  │AI Sidecar│              │  Indexer     │
      │  │ (gRPC)   │              │  (Postgres)  │
      │  │ Bedrock  │              └──────────────┘
      │  └──────────┘
      │
┌─────┴─────────────┐
│   libqorepqc.so   │
│  (Rust FFI, PQC)  │
└───────────────────┘
```

## Modules

| Module | Description |
|--------|-------------|
| **x/pqc** | Post-quantum cryptography: Dilithium-5 signatures, ML-KEM-1024 key exchange, quantum random beacon |
| **x/ai** | AI engine: transaction routing, anomaly detection, fraud detection, fee optimization, network optimization |
| **x/reputation** | Validator reputation scoring: R_i = alpha*S_i + beta*P_i + gamma*C_i + delta*T_i with temporal decay |
| **x/qca** | QoreChain Consensus Algorithm: reputation-weighted proposer selection |
| **x/bridge** | Cross-chain bridge (QCB): hub-and-spoke multi-protocol bridge with PQC-secured attestations |

## Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [PQC Integration Guide](docs/PQC_INTEGRATION.md)
- [AI Engine Documentation](docs/AI_ENGINE.md)
- [Bridge Documentation](docs/BRIDGE.md)
- [Running a Testnet Node](docs/RUNNING_TESTNET.md)
- [API Reference](docs/API_REFERENCE.md)

## Token Economics

- **Token**: QOR (display) / uqor (base denomination, 1 QOR = 10^6 uqor)
- **Chain ID**: qorechain-diana (testnet)
- **Bech32 Prefix**: qor (addresses: qor1..., validators: qorvaloper...)

## License

Apache 2.0 — see [LICENSE](LICENSE)

Core blockchain protocol is open source. PQC cryptographic libraries and AI model weights are distributed as pre-compiled binaries under separate licensing terms.
