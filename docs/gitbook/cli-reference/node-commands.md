# Node Commands

Reference for `qorechaind` commands used to initialize, configure, and operate a QoreChain node.

---

## init

Initialize a new node with the given moniker.

```bash
qorechaind init <moniker> --chain-id qorechain-diana
```

| Flag           | Type   | Description                                      |
| -------------- | ------ | ------------------------------------------------ |
| `--chain-id`   | string | Chain identifier (required)                      |
| `--home`       | string | Node home directory (default: `~/.qorechaind`)   |
| `--overwrite`  | bool   | Overwrite existing genesis and config files       |

Creates the directory structure under `--home` with `config/`, `data/`, and an initial `genesis.json`.

---

## start

Start the node and begin syncing or producing blocks.

```bash
qorechaind start [flags]
```

| Flag                    | Type   | Description                                           |
| ----------------------- | ------ | ----------------------------------------------------- |
| `--home`                | string | Node home directory                                   |
| `--minimum-gas-prices`  | string | Minimum gas prices to accept (e.g., `0.001uqor`)     |
| `--pruning`             | string | Pruning strategy: `default`, `nothing`, `everything`  |
| `--halt-height`         | uint   | Stop the node at this block height                    |
| `--halt-time`           | uint   | Stop the node at this Unix timestamp                  |
| `--log_level`           | string | Log verbosity: `info`, `debug`, `warn`, `error`      |
| `--trace`               | bool   | Enable full stack trace on errors                     |

---

## version

Print the `qorechaind` binary version and build information.

```bash
qorechaind version
```

Use `--long` for extended build details including Go version, commit hash, and build tags:

```bash
qorechaind version --long
```

---

## status

Query the running node for its current status, including sync state, latest block height, and consensus information.

```bash
qorechaind status
```

| Flag     | Type   | Description                             |
| -------- | ------ | --------------------------------------- |
| `--node` | string | RPC endpoint (default: `tcp://localhost:26657`) |

Returns JSON with `node_info`, `sync_info`, and `validator_info` sections.

---

## config

Read or write values in the node configuration.

### Set a Configuration Value

```bash
qorechaind config set <key> <value>
```

### Get a Configuration Value

```bash
qorechaind config get <key>
```

Common configuration keys include `chain-id`, `keyring-backend`, `output`, and `node`.

---

## keys

Manage local keyring for signing transactions.

### Add a New Key

```bash
qorechaind keys add <name> [flags]
```

| Flag                | Type   | Description                                        |
| ------------------- | ------ | -------------------------------------------------- |
| `--keyring-backend` | string | Backend: `os`, `file`, `test`                      |
| `--algo`            | string | Key algorithm: `secp256k1` (default), `ed25519`    |
| `--recover`         | bool   | Recover key from mnemonic                           |
| `--multisig`        | string | Comma-separated list of keys for multisig           |
| `--multisig-threshold` | uint | Minimum signatures required                      |

### List All Keys

```bash
qorechaind keys list --keyring-backend <backend>
```

### Show Key Details

```bash
qorechaind keys show <name> [flags]
```

| Flag           | Type | Description                          |
| -------------- | ---- | ------------------------------------ |
| `--bech`       | string | Output format: `acc`, `val`, `cons` |
| `--address`    | bool | Show address only                    |
| `--pubkey`     | bool | Show public key only                 |

### Delete a Key

```bash
qorechaind keys delete <name> --keyring-backend <backend>
```

### Export a Key (Armor-Encrypted)

```bash
qorechaind keys export <name>
```

### Import a Key

```bash
qorechaind keys import <name> <keyfile>
```

---

## genesis

Manage the genesis file.

### Add a Genesis Account

```bash
qorechaind genesis add-genesis-account <address> <coins> [flags]
```

| Flag              | Type   | Description                              |
| ----------------- | ------ | ---------------------------------------- |
| `--vesting-amount`| string | Vesting amount                           |
| `--vesting-end-time` | int | Vesting end time (Unix timestamp)        |

### Create a Genesis Transaction

```bash
qorechaind genesis gentx <key-name> <stake-amount> [flags]
```

| Flag                  | Type   | Description                           |
| --------------------- | ------ | ------------------------------------- |
| `--chain-id`          | string | Chain identifier                      |
| `--moniker`           | string | Validator moniker                     |
| `--commission-rate`   | string | Initial commission rate               |
| `--commission-max-rate` | string | Maximum commission rate             |

### Collect Genesis Transactions

```bash
qorechaind genesis collect-gentxs
```

### Validate the Genesis File

```bash
qorechaind genesis validate-genesis
```

---

## Consensus Engine

These subcommands interact with the QoreChain Consensus Engine layer.

### Show Validator Key

```bash
qorechaind comet show-validator
```

Outputs the consensus public key in JSON format. Used to verify validator identity.

### Show Node ID

```bash
qorechaind comet show-node-id
```

Outputs the P2P node identifier (hex-encoded). Used for persistent peer configuration.

---

## export

Export the current chain state as a JSON genesis file. Useful for chain upgrades or snapshots.

```bash
qorechaind export [flags]
```

| Flag             | Type   | Description                                |
| ---------------- | ------ | ------------------------------------------ |
| `--for-zero-height` | bool | Prepare export for restarting at height 0 |
| `--height`       | int    | Export state at a specific block height     |
| `--home`         | string | Node home directory                         |

---

## rollback

Roll back the chain state by one block. Useful for recovering from a consensus failure.

```bash
qorechaind rollback [flags]
```

| Flag       | Type | Description                                       |
| ---------- | ---- | ------------------------------------------------- |
| `--hard`   | bool | Remove the last block from the block store as well |
| `--home`   | string | Node home directory                             |

This command rolls back both the application state and the consensus state. Use with caution, as it cannot be undone.
