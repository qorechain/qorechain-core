# Running a QoreChain Testnet Node

## Docker Compose (Recommended)

```bash
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core
docker compose up -d
```

Services started:
- **qorechain-node**: Port 26657 (RPC), 1317 (REST), 9090 (gRPC)
- **ai-sidecar**: Port 50051 (gRPC)
- **indexer**: Connects to node WebSocket
- **postgres**: Port 5432
- **prometheus**: Port 9091
- **grafana**: Port 3001

## Manual Setup

### Prerequisites
- Go 1.25+
- CGO enabled
- libqorepqc for your platform (see PQC_INTEGRATION.md)

### Build
```bash
CGO_ENABLED=1 go build -o qorechaind ./cmd/qorechaind/
```

### Initialize
```bash
./qorechaind init my-node --chain-id qorechain-diana
```

### Configure
Edit `~/.qorechaind/config/config.toml`:
- Set `persistent_peers` to connect to existing testnet nodes
- Adjust `mempool.size` and `consensus.timeout_commit` as needed

### Start
```bash
./qorechaind start
```

## Monitoring

- **Prometheus**: Metrics at `:26660/metrics`
- **Grafana**: Dashboards at `:3001` (admin/admin)
- **REST API**: `:1317/cosmos/base/tendermint/v1beta1/node_info`
