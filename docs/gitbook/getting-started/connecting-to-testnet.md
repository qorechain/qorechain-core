# Connecting to Testnet

Join the live QoreChain Diana testnet by configuring your node with the correct genesis file, peers, and network settings.

---

## Download Genesis

Replace your local genesis file with the official testnet genesis:

```bash
curl -o ~/.qorechaind/config/genesis.json \
  https://raw.githubusercontent.com/qorechain/qorechain-core/main/config/genesis.json
```

This file defines the initial state of the Diana testnet, including the validator set, token allocations, and module parameters.

---

## Configure Peers

Edit your node configuration to connect to existing testnet peers.

Open `~/.qorechaind/config/config.toml` and set the `persistent_peers` field:

```toml
persistent_peers = "node-id@seed1.qorechain.io:26656,node-id@seed2.qorechain.io:26656"
```

Refer to the [QoreChain repository](https://github.com/qorechain/qorechain-core) for the latest peer list.

### Recommended Settings

You may also want to adjust the following in `config.toml`:

```toml
[mempool]
size = 5000

[consensus]
timeout_propose = "3s"
timeout_commit = "5s"
```

These values are tuned for the Diana testnet's block times and throughput.

---

## Start Node

Launch your node to begin syncing with the network:

```bash
./qorechaind start
```

The node connects to peers and begins downloading blocks from genesis. Initial sync time depends on the current chain height and your network speed.

---

## Check Sync Status

Verify that your node is catching up to the latest block:

```bash
curl localhost:26657/status | jq '.result.sync_info.catching_up'
```

- `true` -- The node is still syncing. Wait for it to catch up.
- `false` -- The node is fully synced and processing new blocks.

You can also check the latest block height:

```bash
curl localhost:26657/status | jq '.result.sync_info.latest_block_height'
```

---

## EVM JSON-RPC (MetaMask & ethers/web3 clients)

To use Ethereum tooling against the node, add the network with the correct
**EIP-155 chain ID**:

| Field | Value |
|-------|-------|
| Network name | QoreChain Diana (testnet) |
| RPC URL | `http://<host>:8545` (WS `:8546`) |
| Chain ID | **9800** (mainnet `qorechain-vladi`: 9801) |
| Currency symbol | QOR |

The chain ID is served from `app.toml` `[evm] evm-chain-id`. If EVM transactions
are rejected with `incorrect chain-id; expected 262144`, the node's
`evm-chain-id` is still at the cosmos/evm default — set it to `9800` and restart
(fixed by default in v3.1.69+).

---

## Monitoring

QoreChain exposes several endpoints for monitoring node health and performance.

### Prometheus Metrics

Raw metrics are available at:

```
http://localhost:26660/metrics
```

These metrics can be scraped by any Prometheus-compatible collector.

### Grafana Dashboards

If running via Docker Compose, Grafana is available at:

```
http://localhost:3001
```

Default credentials: `admin` / `admin`. Pre-configured dashboards display block production, transaction throughput, peer connections, and resource usage.

### REST Health Check

The REST API provides a quick status endpoint:

```
http://localhost:1317
```

---

## Ports Reference

| Port | Protocol | Description |
|------|----------|-------------|
| `26657` | TCP | RPC -- query and broadcast transactions |
| `26656` | TCP | P2P -- peer-to-peer network communication |
| `1317` | HTTP | REST API -- query chain state via HTTP |
| `9090` | gRPC | gRPC API -- programmatic chain access |
| `8545` | HTTP | EVM JSON-RPC -- Ethereum-compatible RPC |
| `8546` | WebSocket | EVM WebSocket -- real-time EVM event subscriptions |
| `8899` | HTTP | SVM RPC -- Solana-compatible RPC |
| `26660` | HTTP | Prometheus metrics endpoint |

---

## Next Steps

- [Wallet Setup](wallet-setup.md) -- Configure a wallet for the testnet
- [Your First Transaction](first-transaction.md) -- Send your first QOR transfer
