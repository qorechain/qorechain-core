# JSON-RPC -- qor\_ Namespace

The `qor_` namespace provides QoreChain-specific JSON-RPC methods for querying post-quantum cryptography status, AI analytics, cross-VM messaging, multi-layer state, bridge operations, tokenomics, rollup infrastructure, and reinforcement-learning consensus.

## Connection

| Transport  | Default Address            |
| ---------- | -------------------------- |
| HTTP       | `http://localhost:8545`    |
| WebSocket  | `ws://localhost:8546`      |

The `qor_` namespace is served alongside `eth_`, `web3_`, `net_`, and `txpool_` on the same ports. Enable it in `app.toml`:

```toml
[json-rpc]
api = "eth,web3,net,txpool,qor"
```

---

## Methods

| Method                                            | Parameters                      | Description                                                  |
| ------------------------------------------------- | ------------------------------- | ------------------------------------------------------------ |
| `qor_getPQCKeyStatus`                             | `address` (string)              | Returns PQC key registration status for an account           |
| `qor_getHybridSignatureMode`                      | none                            | Returns current hybrid signature enforcement mode            |
| `qor_getAIStats`                                  | none                            | Returns aggregated AI module processing statistics           |
| `qor_getCrossVMMessage`                           | `messageId` (string)            | Retrieves a cross-VM message by its ID                       |
| `qor_getReputationScore`                          | `validator` (string)            | Returns reputation score for a validator address             |
| `qor_getLayerInfo`                                | `layerId` (string)              | Returns metadata and status for a registered layer           |
| `qor_getBridgeStatus`                             | `chainId` (string)              | Returns bridge status and locked totals for a chain          |
| `qor_getRLAgentStatus`                            | none                            | Returns current RL agent mode and operational status         |
| `qor_getRLObservation`                            | none                            | Returns the latest RL observation vector                     |
| `qor_getRLReward`                                 | none                            | Returns cumulative RL reward metrics                         |
| `qor_getPoolClassification`                       | `validator` (string)            | Returns CPoS pool classification for a validator             |
| `qor_getBurnStats`                                | none                            | Returns burn statistics across all channels                  |
| `qor_getXQOREPosition`                            | `address` (string)              | Returns xQORE staking position for an address                |
| `qor_getInflationRate`                             | none                            | Returns current annualized inflation rate                    |
| `qor_getTokenomicsOverview`                       | none                            | Returns combined burn, inflation, and supply overview        |
| `qor_getRollupStatus`                             | `rollupId` (string)             | Returns status and configuration for a specific rollup       |
| `qor_listRollups`                                 | none                            | Returns a list of all registered rollups                     |
| `qor_getSettlementBatch`                          | `rollupId` (string), `batchIndex` (int) | Returns a specific settlement batch for a rollup     |
| `qor_suggestRollupProfile`                        | `useCase` (string)              | AI-assisted rollup profile recommendation for a use case     |
| `qor_getDABlobStatus`                             | `rollupId` (string), `blobIndex` (int)  | Returns status of a specific DA blob                 |
| `qor_getBTCStakingPosition`                       | `address` (string)              | Returns BTC staking position via the Babylon module          |
| `qor_getAbstractAccount`                          | `address` (string)              | Returns abstract account details and spending rules          |
| `qor_getFairBlockStatus`                          | none                            | Returns FairBlock encryption status and configuration        |
| `qor_getGasAbstractionConfig`                     | none                            | Returns accepted tokens and gas abstraction parameters       |
| `qor_getLaneConfiguration`                        | none                            | Returns 5-lane TX prioritization configuration               |

---

## Examples

### qor\_getBurnStats

**Request:**

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "qor_getBurnStats",
    "params": [],
    "id": 1
  }'
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "total_burned": "1250000000",
    "total_distributed": "4750000000",
    "channels": {
      "validator_share": "0.40",
      "burn_share": "0.30",
      "treasury_share": "0.20",
      "staker_share": "0.10"
    },
    "last_block_burned": "342891"
  }
}
```

### qor\_getRLAgentStatus

**Request:**

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "qor_getRLAgentStatus",
    "params": [],
    "id": 2
  }'
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "agent_mode": "observe",
    "observation_interval": 10,
    "circuit_breaker": {
      "active": false,
      "max_param_change": "0.10",
      "cooldown_blocks": 100
    },
    "last_observation_block": "342885",
    "cumulative_reward": "18.742"
  }
}
```

### qor\_getRollupStatus

**Request:**

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "qor_getRollupStatus",
    "params": ["rollup-001"],
    "id": 3
  }'
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "rollup_id": "rollup-001",
    "status": "active",
    "settlement_type": "optimistic",
    "operator": "qor1abc123...",
    "total_batches": 147,
    "last_finalized_batch": 145,
    "pending_batches": 2,
    "stake": "10000000000uqor",
    "created_at_height": 100230
  }
}
```

---

## Error Codes

| Code   | Message                    | Description                          |
| ------ | -------------------------- | ------------------------------------ |
| -32600 | Invalid Request            | Malformed JSON-RPC request           |
| -32601 | Method not found           | The method does not exist            |
| -32602 | Invalid params             | Missing or invalid parameters        |
| -32603 | Internal error             | Server-side processing error         |
| -32000 | Module disabled            | The queried module is not enabled    |
| -32001 | Entity not found           | The requested resource does not exist|
