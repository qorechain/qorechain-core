# Sidecar Operator Guide

## Overview

The QoreChain sidecar system manages chain-specific containers that handle bridge watching and validator operations for supported external networks. Each sidecar runs as an isolated Docker container coordinated by the main `qorechaind` process, providing secure key management, transaction relaying, and event monitoring for cross-chain operations.

Sidecars communicate with the QoreChain node over gRPC and expose a separate admin interface for operator management. A valid on-chain license is required before any sidecar can be activated.

## Prerequisites

- **Docker** installed and running (v20.10+)
- **qorechaind** binary built and configured
- A funded QoreChain account with sufficient QOR for transaction fees
- A valid **license grant** for the target chain (see below)

## Getting a License

Sidecar operation requires an on-chain license issued through the `x/license` module. Licenses can be obtained in two ways:

1. **Governance proposal**: Submit a `LicenseGrantProposal` specifying the operator address, target chain, and requested feature set. The proposal follows standard governance voting.

2. **Admin grant** (testnet / permissioned deployments):

```bash
qorechaind tx license grant \
  --operator <operator-address> \
  --chain <chain-id> \
  --features bridge-watch,relay-tx,validator-ops \
  --duration 8760h \
  --from <admin-key>
```

Verify your license status:

```bash
qorechaind query license list --operator <operator-address>
```

## Enabling a Sidecar

Once a license is active, enable the sidecar for a specific chain:

```bash
qorechaind sidecar enable <chain>
```

This pulls the appropriate container image, configures networking, and starts the sidecar process. The sidecar will begin syncing with the target chain immediately.

Example:

```bash
qorechaind sidecar enable ethereum
```

## Disabling a Sidecar

To stop and remove a running sidecar:

```bash
qorechaind sidecar disable <chain>
```

This gracefully shuts down the container and removes ephemeral state. Persistent data (keys, checkpoints) is retained in the data directory.

## Checking Status

View the current state of all sidecars:

```bash
qorechaind sidecar status
```

Output includes container health, sync progress, last observed block, and license expiry for each active sidecar.

## Key Management

Each sidecar requires chain-specific keys for signing relay transactions. Keys are stored encrypted in the sidecar data directory.

### Import an Existing Key

```bash
qorechaind sidecar keys import <chain> --keyfile <path>
```

The keyfile should contain the private key in the format expected by the target chain. The key is encrypted at rest using the node's keyring passphrase.

### Generate a New Key

```bash
qorechaind sidecar keys generate <chain>
```

Generates a new keypair appropriate for the target chain. The address is printed to stdout. Fund this address on the target chain before enabling bridge operations.

### List Keys

```bash
qorechaind sidecar keys list
```

Displays all imported and generated keys with their chain, address, and creation time.

## Configuration Reference

The `[sidecar]` section in `app.toml` controls global sidecar behavior:

```toml
[sidecar]

# Master switch for the sidecar subsystem.
enabled = true

# gRPC address for sidecar-to-node communication.
grpc_addr = "localhost:9191"

# Admin HTTP address for operator management endpoints.
admin_addr = "localhost:9192"

# Directory for sidecar persistent data (keys, checkpoints, logs).
data_dir = "/home/qorechain/.qorechaind/sidecar-data"
```

| Field        | Type   | Default                          | Description                                |
|--------------|--------|----------------------------------|--------------------------------------------|
| `enabled`    | bool   | `false`                          | Enable the sidecar subsystem               |
| `grpc_addr`  | string | `localhost:9191`                 | gRPC listen address for sidecar comms      |
| `admin_addr` | string | `localhost:9192`                 | Admin HTTP API listen address              |
| `data_dir`   | string | `$HOME/.qorechaind/sidecar-data` | Persistent storage for sidecar state       |

## Supported Chains

| Chain      | Chain ID     | Sidecar Image Tag   | Bridge Type  |
|------------|-------------|----------------------|--------------|
| Ethereum   | `ethereum`  | `sidecar-eth:latest` | EVM Lock/Mint |
| Solana     | `solana`    | `sidecar-sol:latest` | SPL Relay     |
| Polygon    | `polygon`   | `sidecar-poly:latest`| EVM Lock/Mint |
| Arbitrum   | `arbitrum`  | `sidecar-arb:latest` | EVM Lock/Mint |
| BSC        | `bsc`       | `sidecar-bsc:latest` | EVM Lock/Mint |
| Avalanche  | `avalanche` | `sidecar-avax:latest`| EVM Lock/Mint |
| Sui        | `sui`       | `sidecar-sui:latest` | Move Relay    |
| TON        | `ton`       | `sidecar-ton:latest` | FunC Relay    |
| NEAR       | `near`      | `sidecar-near:latest`| NEAR Relay    |
| Aptos      | `aptos`     | `sidecar-apt:latest` | Move Relay    |

## Troubleshooting

### Sidecar fails to start

- Verify Docker is running: `docker info`
- Check that the sidecar image exists: `docker images | grep sidecar`
- Inspect logs: `qorechaind sidecar logs <chain> --tail 100`
- Ensure `[sidecar] enabled = true` in `app.toml`

### License errors

- Confirm your license is active: `qorechaind query license list --operator <addr>`
- Licenses expire automatically. Renew before expiry with a new grant or governance proposal.
- Feature mismatch: ensure the license includes the required feature IDs for your operation.

### Connection refused on gRPC

- Verify `grpc_addr` in `app.toml` matches the address the node is listening on.
- Check firewall rules allow traffic on the configured port.
- If running in Docker, ensure the port is mapped correctly in `docker-compose.yml`.

### Key import failures

- Ensure the keyfile format matches the target chain's expected format.
- Check file permissions: the keyfile must be readable by the `qorechaind` process.
- For encrypted keyfiles, provide the passphrase via `--passphrase` flag or interactive prompt.

### Sidecar stuck syncing

- Check target chain RPC endpoint availability.
- Increase `sync_timeout` in the chain-specific sidecar config if the target chain is slow.
- Review sidecar logs for rate-limiting or authentication errors from the RPC provider.

### High memory usage

- Each sidecar runs in its own container with default memory limits.
- Adjust limits in `docker-compose.override.yml` if needed.
- Prune old checkpoint data: `qorechaind sidecar prune <chain> --keep-last 1000`
