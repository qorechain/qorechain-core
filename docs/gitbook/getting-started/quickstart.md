# Quickstart

Get a QoreChain node running in minutes. Choose Docker Compose for the fastest setup, or build from source for full control.

---

## Docker Compose (Recommended)

The simplest way to run a full QoreChain environment with all services pre-configured.

```bash
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core
docker compose up -d
```

This starts the following services:

| Service | Ports | Description |
|---------|-------|-------------|
| **qorechain-node** | `26657` (RPC), `1317` (REST), `9090` (gRPC), `8545` (EVM), `8899` (SVM) | Full blockchain node with multi-VM support |
| **ai-sidecar** | `50051` | AI anomaly detection and risk scoring engine |
| **indexer** | -- | Block indexer for historical queries |
| **postgres** | `5432` | Database backend for the indexer |
| **prometheus** | `9091` | Metrics collection |
| **grafana** | `3001` | Monitoring dashboards |

Once all containers are healthy, your node begins syncing with the network.

---

## Build from Source

### Prerequisites

- **Go 1.26+** with CGO enabled
- **Rust toolchain** (for compiling PQC cryptography and SVM runtime libraries)
- **Git**

### Build the Binary

```bash
CGO_ENABLED=1 go build -o qorechaind ./cmd/qorechaind/
```

### Initialize the Node

```bash
./qorechaind init my-node --chain-id qorechain-diana
```

This creates the default configuration and data directories under `~/.qorechaind/`.

### Start the Node

```bash
./qorechaind start
```

The node starts with default settings. See [Connecting to Testnet](connecting-to-testnet.md) for joining the live network with proper genesis and peer configuration.

---

## Verify Installation

Confirm your node is running correctly:

```bash
# Check the binary version
./qorechaind version
```

```bash
# Query the node status via RPC
curl localhost:26657/status
```

A successful response includes the node's `moniker`, `network` (should be `qorechain-diana`), and current block height.

---

## Next Steps

- [Connecting to Testnet](connecting-to-testnet.md) -- Join the live Diana testnet
- [Wallet Setup](wallet-setup.md) -- Configure a wallet to interact with the chain
- [Your First Transaction](first-transaction.md) -- Send your first QOR transfer
