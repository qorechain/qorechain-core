# Transaction Commands

All transaction commands follow the pattern:

```bash
qorechaind tx <module> <command> [args] [flags]
```

**Common flags** apply to every `tx` subcommand:

| Flag                | Type   | Description                                       |
| ------------------- | ------ | ------------------------------------------------- |
| `--from`            | string | Name or address of the signing key                |
| `--chain-id`        | string | Chain identifier (default: from config)           |
| `--fees`            | string | Transaction fees (e.g., `500uqor`)                |
| `--gas`             | string | Gas limit or `auto` for estimation                |
| `--gas-adjustment`  | float  | Gas multiplier when using `auto` (default: 1.0)   |
| `--keyring-backend` | string | Keyring backend: `os`, `file`, `test`             |
| `--node`            | string | RPC endpoint (default: `tcp://localhost:26657`)    |
| `--broadcast-mode`  | string | `sync`, `async`, or `block`                       |
| `-y`                | bool   | Skip confirmation prompt                           |

---

## bank

### send

Transfer tokens from one account to another.

```bash
qorechaind tx bank send <from_address> <to_address> <amount> [flags]
```

---

## staking

### create-validator

Create a new validator on the network.

```bash
qorechaind tx staking create-validator [flags]
```

| Flag                       | Type   | Description                                    |
| -------------------------- | ------ | ---------------------------------------------- |
| `--amount`                 | string | Self-delegation amount (e.g., `1000000uqor`)   |
| `--pubkey`                 | string | Validator consensus public key (JSON)           |
| `--moniker`                | string | Validator display name                          |
| `--commission-rate`        | string | Initial commission rate (e.g., `0.10`)          |
| `--commission-max-rate`    | string | Maximum commission rate                         |
| `--commission-max-change-rate` | string | Maximum daily commission change rate       |
| `--min-self-delegation`    | string | Minimum self-delegation required                |

### edit-validator

Edit an existing validator's description or commission.

```bash
qorechaind tx staking edit-validator [flags]
```

### delegate

Delegate tokens to a validator.

```bash
qorechaind tx staking delegate <validator_address> <amount> [flags]
```

### redelegate

Move delegation from one validator to another.

```bash
qorechaind tx staking redelegate <src_validator> <dst_validator> <amount> [flags]
```

### unbond

Unbond tokens from a validator.

```bash
qorechaind tx staking unbond <validator_address> <amount> [flags]
```

---

## distribution

### withdraw-all-rewards

Withdraw all pending staking rewards.

```bash
qorechaind tx distribution withdraw-all-rewards [flags]
```

### withdraw-rewards

Withdraw rewards from a specific validator.

```bash
qorechaind tx distribution withdraw-rewards <validator_address> [flags]
```

| Flag            | Type | Description                      |
| --------------- | ---- | -------------------------------- |
| `--commission`  | bool | Also withdraw validator commission |

---

## gov

### submit-proposal

Submit a governance proposal.

```bash
qorechaind tx gov submit-proposal <proposal_file.json> [flags]
```

The proposal file is a JSON document specifying the proposal type, title, description, and any messages to execute.

### vote

Vote on an active proposal.

```bash
qorechaind tx gov vote <proposal_id> <option> [flags]
```

Vote options: `yes`, `no`, `abstain`, `no_with_veto`.

### deposit

Add a deposit to a proposal.

```bash
qorechaind tx gov deposit <proposal_id> <amount> [flags]
```

---

## pqc

### register-key

Register a post-quantum public key for an account.

```bash
qorechaind tx pqc register-key <algorithm> <pubkey_hex> [flags]
```

### register-key-v2

Register a PQC key with extended metadata and attestation.

```bash
qorechaind tx pqc register-key-v2 <algorithm> <pubkey_hex> [flags]
```

| Flag              | Type   | Description                        |
| ----------------- | ------ | ---------------------------------- |
| `--attestation`   | string | TEE attestation data (hex)         |
| `--metadata`      | string | Additional key metadata (JSON)     |

### migrate-key

Migrate an existing classical key to a hybrid PQC key pair.

```bash
qorechaind tx pqc migrate-key <algorithm> <pqc_pubkey_hex> [flags]
```

---

## xqore

### lock

Lock QOR tokens into an xQORE governance staking position.

```bash
qorechaind tx xqore lock <amount> [flags]
```

| Flag             | Type   | Description                              |
| ---------------- | ------ | ---------------------------------------- |
| `--lock-duration` | string | Lock duration (e.g., `30d`, `90d`, `180d`) |

### unlock

Unlock xQORE back to QOR. Early unlock may incur penalties depending on the penalty tier.

```bash
qorechaind tx xqore unlock <amount> [flags]
```

---

## bridge

### deposit

Initiate a bridge deposit from an external chain.

```bash
qorechaind tx bridge deposit <chain_id> <amount> <asset> [flags]
```

| Flag            | Type   | Description                          |
| --------------- | ------ | ------------------------------------ |
| `--recipient`   | string | Recipient address on QoreChain       |

### withdraw

Initiate a bridge withdrawal to an external chain.

```bash
qorechaind tx bridge withdraw <chain_id> <amount> <asset> <destination_address> [flags]
```

---

## crossvm

### call

Send a cross-VM message between execution environments (EVM, CosmWasm, SVM).

```bash
qorechaind tx crossvm call <target_vm> <contract_address> <payload_hex> [flags]
```

| Flag          | Type   | Description                                |
| ------------- | ------ | ------------------------------------------ |
| `--source-vm` | string | Source VM: `evm`, `cosmwasm`, `svm`        |
| `--gas-limit` | uint   | Gas limit for the cross-VM execution       |

### process-queue

Manually process pending cross-VM messages (operator command).

```bash
qorechaind tx crossvm process-queue [flags]
```

---

## svm

### deploy-program

Deploy a BPF program to the SVM runtime.

```bash
qorechaind tx svm deploy-program <program_binary_path> [flags]
```

| Flag              | Type   | Description                            |
| ----------------- | ------ | -------------------------------------- |
| `--program-id`    | string | Optional program ID (base58)           |

### execute

Execute an instruction on a deployed SVM program.

```bash
qorechaind tx svm execute <program_id> <instruction_data_hex> [flags]
```

| Flag           | Type   | Description                                    |
| -------------- | ------ | ---------------------------------------------- |
| `--accounts`   | string | Comma-separated account pubkeys for the instruction |

### create-account

Create a new SVM account with allocated data space.

```bash
qorechaind tx svm create-account <pubkey> <space> [flags]
```

| Flag        | Type   | Description                                    |
| ----------- | ------ | ---------------------------------------------- |
| `--owner`   | string | Owner program (base58, default: system program) |

---

## multilayer

### register-sidechain

Register a new sidechain layer.

```bash
qorechaind tx multilayer register-sidechain <layer_id> <name> [flags]
```

| Flag               | Type   | Description                              |
| ------------------ | ------ | ---------------------------------------- |
| `--anchor-interval`| uint   | Blocks between mandatory anchor submissions |
| `--stake`          | string | Required stake amount                    |

### register-paychain

Register a new payment channel layer.

```bash
qorechaind tx multilayer register-paychain <layer_id> <name> [flags]
```

### anchor-state

Submit a state anchor for a registered layer.

```bash
qorechaind tx multilayer anchor-state <layer_id> <state_root_hex> <height> [flags]
```

### route-tx

Route a transaction to the optimal layer.

```bash
qorechaind tx multilayer route-tx <tx_data_hex> [flags]
```

| Flag              | Type   | Description                               |
| ----------------- | ------ | ----------------------------------------- |
| `--target-layer`  | string | Force routing to a specific layer          |

### update-layer-status

Update the operational status of a layer (operator only).

```bash
qorechaind tx multilayer update-layer-status <layer_id> <status> [flags]
```

Status values: `active`, `paused`, `draining`.

### challenge-anchor

Submit a fraud challenge against a state anchor.

```bash
qorechaind tx multilayer challenge-anchor <layer_id> <anchor_hash> <proof_hex> [flags]
```

---

## rdk

### create-rollup

Register a new rollup with the Rollup Development Kit.

```bash
qorechaind tx rdk create-rollup <rollup_id> [flags]
```

| Flag                | Type   | Description                                          |
| ------------------- | ------ | ---------------------------------------------------- |
| `--settlement-type` | string | `optimistic`, `zk`, `pessimistic`, `sovereign`       |
| `--profile`         | string | Preset: `defi`, `gaming`, `nft`, `social`, `general` |
| `--stake`           | string | Operator stake amount                                 |
| `--da-enabled`      | bool   | Enable native data availability                       |

### submit-batch

Submit a settlement batch for a rollup.

```bash
qorechaind tx rdk submit-batch <rollup_id> <state_root_hex> <batch_data_path> [flags]
```

### challenge-batch

Submit a fraud challenge against a settlement batch (optimistic rollups).

```bash
qorechaind tx rdk challenge-batch <rollup_id> <batch_index> <proof_hex> [flags]
```

### finalize-batch

Manually finalize a batch that has passed the challenge window.

```bash
qorechaind tx rdk finalize-batch <rollup_id> <batch_index> [flags]
```

### pause-rollup

Pause a rollup (operator only).

```bash
qorechaind tx rdk pause-rollup <rollup_id> [flags]
```

### resume-rollup

Resume a paused rollup (operator only).

```bash
qorechaind tx rdk resume-rollup <rollup_id> [flags]
```

### stop-rollup

Permanently stop a rollup and release its stake (operator only).

```bash
qorechaind tx rdk stop-rollup <rollup_id> [flags]
```

---

## babylon

### submit-btc-checkpoint

Submit a BTC checkpoint for an epoch.

```bash
qorechaind tx babylon submit-btc-checkpoint <epoch> <checkpoint_hex> [flags]
```

### btc-restake

Restake BTC via the Babylon integration.

```bash
qorechaind tx babylon btc-restake <amount> [flags]
```

| Flag              | Type   | Description                          |
| ----------------- | ------ | ------------------------------------ |
| `--btc-tx-hash`   | string | Bitcoin transaction hash as proof    |

---

## abstractaccount

### create

Create an abstract account with programmable spending rules.

```bash
qorechaind tx abstractaccount create [flags]
```

| Flag               | Type   | Description                               |
| ------------------ | ------ | ----------------------------------------- |
| `--spending-rules` | string | JSON file defining spending rules          |

### update-spending-rules

Update the spending rules for an existing abstract account.

```bash
qorechaind tx abstractaccount update-spending-rules <rules_file.json> [flags]
```

---

## rlconsensus

### set-agent-mode

Set the RL agent operational mode (governance only).

```bash
qorechaind tx rlconsensus set-agent-mode <mode> [flags]
```

Mode values: `0` (off), `1` (observe), `2` (suggest), `3` (auto).

### resume-agent

Resume the RL agent after a circuit breaker trip.

```bash
qorechaind tx rlconsensus resume-agent [flags]
```

### update-policy

Update the RL agent policy configuration (governance only).

```bash
qorechaind tx rlconsensus update-policy <policy_file.json> [flags]
```

### update-reward-weights

Update the reward weight configuration for the RL agent.

```bash
qorechaind tx rlconsensus update-reward-weights [flags]
```

| Flag                  | Type   | Description                       |
| --------------------- | ------ | --------------------------------- |
| `--throughput-weight` | string | Weight for throughput reward       |
| `--latency-weight`    | string | Weight for latency reward          |
| `--security-weight`   | string | Weight for security reward         |
