# Query Commands

All query commands follow the pattern:

```bash
qorechaind query <module> <command> [args] [flags]
```

**Common flags** apply to every `query` subcommand:

| Flag       | Type   | Description                                       |
| ---------- | ------ | ------------------------------------------------- |
| `--node`   | string | RPC endpoint (default: `tcp://localhost:26657`)    |
| `--output` | string | Output format: `json` or `text`                   |
| `--height` | int    | Query state at a specific block height             |

---

## bank

### balances

Query all balances for an account.

```bash
qorechaind query bank balances <address>
```

### total

Query the total supply of all tokens.

```bash
qorechaind query bank total
```

---

## staking

### validator

Query a single validator by operator address.

```bash
qorechaind query staking validator <validator_address>
```

### validators

List all validators.

```bash
qorechaind query staking validators
```

### delegation

Query a delegation from a delegator to a validator.

```bash
qorechaind query staking delegation <delegator_address> <validator_address>
```

### delegations

Query all delegations for a delegator.

```bash
qorechaind query staking delegations <delegator_address>
```

### unbonding-delegation

Query an unbonding delegation.

```bash
qorechaind query staking unbonding-delegation <delegator_address> <validator_address>
```

---

## distribution

### rewards

Query all delegation rewards for a delegator.

```bash
qorechaind query distribution rewards <delegator_address>
```

### commission

Query validator commission.

```bash
qorechaind query distribution commission <validator_address>
```

---

## gov

### proposal

Query a single proposal by ID.

```bash
qorechaind query gov proposal <proposal_id>
```

### proposals

List all proposals, optionally filtered by status.

```bash
qorechaind query gov proposals [flags]
```

| Flag       | Type   | Description                                                   |
| ---------- | ------ | ------------------------------------------------------------- |
| `--status` | string | Filter by status: `deposit_period`, `voting_period`, `passed`, `rejected` |

### votes

Query votes on a proposal.

```bash
qorechaind query gov votes <proposal_id>
```

---

## pqc

### account

Query PQC key registration status for an account.

```bash
qorechaind query pqc account <address>
```

### algorithms

List all supported PQC algorithms.

```bash
qorechaind query pqc algorithms
```

### algorithm

Query details for a specific PQC algorithm.

```bash
qorechaind query pqc algorithm <algorithm_name>
```

### stats

Query aggregate PQC registration statistics.

```bash
qorechaind query pqc stats
```

### params

Query PQC module parameters.

```bash
qorechaind query pqc params
```

### migration

Query PQC key migration status for an account.

```bash
qorechaind query pqc migration <address>
```

### hybrid-mode

Query the current hybrid signature enforcement mode.

```bash
qorechaind query pqc hybrid-mode
```

---

## xqore

### position

Query xQORE staking position for an address.

```bash
qorechaind query xqore position <address>
```

### params

Query xQORE module parameters.

```bash
qorechaind query xqore params
```

---

## burn

### stats

Query burn statistics across all channels.

```bash
qorechaind query burn stats
```

### params

Query burn module parameters.

```bash
qorechaind query burn params
```

---

## inflation

### rate

Query the current annualized inflation rate.

```bash
qorechaind query inflation rate
```

### epoch

Query the current epoch number and progress.

```bash
qorechaind query inflation epoch
```

### params

Query inflation module parameters.

```bash
qorechaind query inflation params
```

---

## ai

### config

Query AI module configuration.

```bash
qorechaind query ai config
```

### stats

Query aggregated AI processing statistics.

```bash
qorechaind query ai stats
```

### fee-estimate

Get an AI-assisted gas fee estimate.

```bash
qorechaind query ai fee-estimate [flags]
```

| Flag          | Type   | Description                          |
| ------------- | ------ | ------------------------------------ |
| `--tx-type`   | string | Transaction type for estimation      |
| `--urgency`   | string | `low`, `medium`, `high`             |

### investigations

List active fraud investigations.

```bash
qorechaind query ai investigations
```

### recommendations

Get AI-generated network optimization recommendations.

```bash
qorechaind query ai recommendations
```

### circuit-breakers

Query current circuit breaker states.

```bash
qorechaind query ai circuit-breakers
```

---

## reputation

### validators

Query reputation scores for all validators.

```bash
qorechaind query reputation validators
```

### validator

Query reputation score for a specific validator.

```bash
qorechaind query reputation validator <validator_address>
```

---

## bridge

### chains

List all registered bridge chains.

```bash
qorechaind query bridge chains
```

### chain

Query details for a specific bridged chain.

```bash
qorechaind query bridge chain <chain_id>
```

### validators

List active bridge validators.

```bash
qorechaind query bridge validators
```

### operations

List recent bridge operations.

```bash
qorechaind query bridge operations
```

| Flag       | Type   | Description                                    |
| ---------- | ------ | ---------------------------------------------- |
| `--status` | string | Filter: `pending`, `completed`, `failed`       |
| `--chain`  | string | Filter by chain ID                              |

### limits

Query rate limits for a bridged chain.

```bash
qorechaind query bridge limits <chain_id>
```

### estimate

Estimate bridge fee and transfer time.

```bash
qorechaind query bridge estimate <chain_id> <amount> <asset>
```

---

## crossvm

### message

Retrieve a cross-VM message by ID.

```bash
qorechaind query crossvm message <message_id>
```

### pending

List pending cross-VM messages.

```bash
qorechaind query crossvm pending
```

### params

Query Cross-VM module parameters.

```bash
qorechaind query crossvm params
```

---

## svm

### account

Query SVM account information.

```bash
qorechaind query svm account <pubkey>
```

### program

Query deployed SVM program information.

```bash
qorechaind query svm program <program_id>
```

### params

Query SVM module parameters.

```bash
qorechaind query svm params
```

### slot

Query the current SVM slot number.

```bash
qorechaind query svm slot
```

---

## multilayer

### layer

Query details for a specific layer.

```bash
qorechaind query multilayer layer <layer_id>
```

### layers

List all registered layers.

```bash
qorechaind query multilayer layers
```

### anchor

Query a specific anchor record.

```bash
qorechaind query multilayer anchor <anchor_id>
```

### anchors

List recent anchor submissions.

```bash
qorechaind query multilayer anchors [flags]
```

| Flag         | Type   | Description                  |
| ------------ | ------ | ---------------------------- |
| `--layer-id` | string | Filter by layer ID           |
| `--limit`    | uint   | Maximum results to return    |

### routing-stats

Query transaction routing statistics across layers.

```bash
qorechaind query multilayer routing-stats
```

### simulate-route

Simulate transaction routing without execution.

```bash
qorechaind query multilayer simulate-route <tx_data_hex>
```

### params

Query Multilayer module parameters.

```bash
qorechaind query multilayer params
```

---

## rdk

### rollup

Query details for a specific rollup.

```bash
qorechaind query rdk rollup <rollup_id>
```

### rollups

List all registered rollups.

```bash
qorechaind query rdk rollups
```

| Flag       | Type   | Description                                          |
| ---------- | ------ | ---------------------------------------------------- |
| `--status` | string | Filter: `active`, `paused`, `stopped`                |

### batch

Query a specific settlement batch.

```bash
qorechaind query rdk batch <rollup_id> <batch_index>
```

### latest-batch

Query the latest batch for a rollup.

```bash
qorechaind query rdk latest-batch <rollup_id>
```

### suggest-profile

Get an AI-assisted rollup profile recommendation.

```bash
qorechaind query rdk suggest-profile <use_case>
```

### blob

Query a specific DA blob.

```bash
qorechaind query rdk blob <rollup_id> <blob_index>
```

### params

Query RDK module parameters.

```bash
qorechaind query rdk params
```

---

## rlconsensus

### agent-status

Query the current RL agent status and mode.

```bash
qorechaind query rlconsensus agent-status
```

### observation

Query the latest RL observation vector.

```bash
qorechaind query rlconsensus observation
```

### reward

Query cumulative RL reward metrics.

```bash
qorechaind query rlconsensus reward
```

### params

Query RL Consensus module parameters.

```bash
qorechaind query rlconsensus params
```

### policy

Query the active RL policy configuration.

```bash
qorechaind query rlconsensus policy
```

---

## babylon

### staking

Query BTC staking position for an address.

```bash
qorechaind query babylon staking <address>
```

### checkpoint

Query BTC checkpoint data for a given epoch.

```bash
qorechaind query babylon checkpoint <epoch>
```

### params

Query Babylon module parameters.

```bash
qorechaind query babylon params
```

---

## abstractaccount

### account

Query abstract account details.

```bash
qorechaind query abstractaccount account <address>
```

### params

Query Abstract Account module parameters.

```bash
qorechaind query abstractaccount params
```

---

## gasabstraction

### accepted-tokens

List tokens accepted for gas payment.

```bash
qorechaind query gasabstraction accepted-tokens
```

### params

Query Gas Abstraction module parameters.

```bash
qorechaind query gasabstraction params
```

---

## fairblock

### config

Query FairBlock encryption configuration.

```bash
qorechaind query fairblock config
```

### params

Query FairBlock module parameters.

```bash
qorechaind query fairblock params
```
