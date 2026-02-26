# WebSocket Events

QoreChain provides real-time event streaming through two WebSocket interfaces: the EVM-compatible WebSocket and the QoreChain Consensus Engine RPC WebSocket.

---

## EVM WebSocket

**Endpoint:** `ws://localhost:8546`

The EVM WebSocket supports the standard `eth_subscribe` method for real-time event streaming compatible with Ethereum tooling.

### Subscription Types

| Subscription              | Description                                          |
| ------------------------- | ---------------------------------------------------- |
| `newHeads`                | Emits a header each time a new block is appended     |
| `logs`                    | Emits logs matching an optional filter               |
| `newPendingTransactions`  | Emits transaction hashes entering the mempool        |
| `syncing`                 | Emits sync status updates                            |

### Subscribe to New Blocks

```json
{
  "jsonrpc": "2.0",
  "method": "eth_subscribe",
  "params": ["newHeads"],
  "id": 1
}
```

### Subscribe to Logs with Filter

```json
{
  "jsonrpc": "2.0",
  "method": "eth_subscribe",
  "params": [
    "logs",
    {
      "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD28",
      "topics": ["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"]
    }
  ],
  "id": 2
}
```

### Unsubscribe

```json
{
  "jsonrpc": "2.0",
  "method": "eth_unsubscribe",
  "params": ["0x1a2b3c..."],
  "id": 3
}
```

---

## QoreChain RPC WebSocket

**Endpoint:** `ws://localhost:26657/websocket`

The RPC WebSocket uses the QoreChain Consensus Engine event subscription system. Clients subscribe with a query string that filters events by type and attributes.

### Subscribe to All New Blocks

```json
{
  "jsonrpc": "2.0",
  "method": "subscribe",
  "params": {
    "query": "tm.event='NewBlock'"
  },
  "id": 1
}
```

### Subscribe to All Transactions

```json
{
  "jsonrpc": "2.0",
  "method": "subscribe",
  "params": {
    "query": "tm.event='Tx'"
  },
  "id": 2
}
```

### Subscribe to Module-Specific Events

Filter by event type to receive only events from a specific module:

```json
{
  "jsonrpc": "2.0",
  "method": "subscribe",
  "params": {
    "query": "tm.event='Tx' AND fraud_alert.severity EXISTS"
  },
  "id": 3
}
```

### Unsubscribe

```json
{
  "jsonrpc": "2.0",
  "method": "unsubscribe",
  "params": {
    "query": "tm.event='NewBlock'"
  },
  "id": 4
}
```

---

## Module Events Reference

### PQC Module

| Event Type                | Key Attributes                                                    | Description                                      |
| ------------------------- | ----------------------------------------------------------------- | ------------------------------------------------ |
| `pqc_hybrid_verify`      | `address`, `algorithm`, `result` (pass/fail), `mode`              | Emitted on each hybrid signature verification    |
| `pqc_hybrid_auto_register` | `address`, `algorithm`, `pubkey_hash`                           | Emitted when a PQC key is auto-registered        |

### AI Module

| Event Type                | Key Attributes                                                    | Description                                      |
| ------------------------- | ----------------------------------------------------------------- | ------------------------------------------------ |
| `fraud_alert`             | `severity` (low/medium/high/critical), `address`, `reason`, `score` | Emitted when fraud is detected in a transaction |
| `circuit_breaker`         | `module`, `action` (tripped/reset), `threshold`, `value`          | Emitted when an AI circuit breaker changes state |

### Bridge Module

| Event Type                | Key Attributes                                                    | Description                                      |
| ------------------------- | ----------------------------------------------------------------- | ------------------------------------------------ |
| `deposit_completed`       | `chain_id`, `sender`, `recipient`, `amount`, `asset`, `tx_hash`   | Emitted when an inbound bridge deposit is confirmed |
| `withdrawal_completed`    | `chain_id`, `sender`, `recipient`, `amount`, `asset`, `tx_hash`   | Emitted when an outbound bridge withdrawal is confirmed |

### Cross-VM Module

| Event Type                | Key Attributes                                                    | Description                                      |
| ------------------------- | ----------------------------------------------------------------- | ------------------------------------------------ |
| `crossvm_request`         | `message_id`, `source_vm`, `target_vm`, `sender`, `payload_hash` | Emitted when a cross-VM call is initiated        |
| `crossvm_response`        | `message_id`, `source_vm`, `target_vm`, `success`, `gas_used`    | Emitted when a cross-VM call completes           |
| `crossvm_timeout`         | `message_id`, `source_vm`, `target_vm`, `queued_at_height`       | Emitted when a cross-VM message exceeds queue timeout |

### Multilayer Module

| Event Type                | Key Attributes                                                    | Description                                      |
| ------------------------- | ----------------------------------------------------------------- | ------------------------------------------------ |
| `anchor_submitted`        | `layer_id`, `layer_type`, `anchor_hash`, `height`, `submitter`   | Emitted when a layer state anchor is submitted   |
| `layer_status_changed`    | `layer_id`, `previous_status`, `new_status`, `reason`            | Emitted when a layer changes operational status  |

### RDK Module

| Event Type                | Key Attributes                                                    | Description                                      |
| ------------------------- | ----------------------------------------------------------------- | ------------------------------------------------ |
| `rollup_created`          | `rollup_id`, `operator`, `settlement_type`, `profile`            | Emitted when a new rollup is registered          |
| `batch_submitted`         | `rollup_id`, `batch_index`, `state_root`, `tx_count`             | Emitted when a settlement batch is submitted     |
| `batch_finalized`         | `rollup_id`, `batch_index`, `finalized_at_height`                | Emitted when a batch passes its challenge window |
| `da_blob_stored`          | `rollup_id`, `blob_index`, `size_bytes`, `commitment`            | Emitted when a DA blob is stored                 |
| `da_blob_pruned`          | `rollup_id`, `blob_index`, `pruned_at_height`                    | Emitted when a DA blob is pruned after retention |

### Burn Module

| Event Type                | Key Attributes                                                    | Description                                      |
| ------------------------- | ----------------------------------------------------------------- | ------------------------------------------------ |
| `fee_distributed`         | `total_fees`, `validator_amount`, `burn_amount`, `treasury_amount`, `staker_amount` | Emitted when collected fees are distributed |
| `tokens_burned`           | `amount`, `channel`, `block_height`                               | Emitted when tokens are permanently burned       |

### xQORE Module

| Event Type                | Key Attributes                                                    | Description                                      |
| ------------------------- | ----------------------------------------------------------------- | ------------------------------------------------ |
| `xqore_locked`            | `address`, `amount`, `lock_duration`, `tier`                     | Emitted when QOR is locked into xQORE            |
| `xqore_unlocked`          | `address`, `amount`, `penalty_applied`, `penalty_amount`         | Emitted when xQORE is unlocked back to QOR       |
| `pvp_rebase`              | `epoch`, `total_penalty`, `rebase_amount`, `beneficiary_count`   | Emitted during PvP rebase distribution           |

### Inflation Module

| Event Type                | Key Attributes                                                    | Description                                      |
| ------------------------- | ----------------------------------------------------------------- | ------------------------------------------------ |
| `epoch_minted`            | `epoch`, `minted_amount`, `inflation_rate`, `block_height`       | Emitted at the end of each inflation epoch       |

### RL Consensus Module

| Event Type                     | Key Attributes                                                 | Description                                      |
| ------------------------------ | -------------------------------------------------------------- | ------------------------------------------------ |
| `rl_action_applied`            | `action_type`, `param_key`, `old_value`, `new_value`, `reward` | Emitted when the RL agent applies a parameter adjustment |
| `circuit_breaker_triggered`    | `reason`, `param_key`, `attempted_value`, `limit`              | Emitted when the RL circuit breaker blocks an action |

---

## JavaScript Client Example

### EVM WebSocket (ethers.js)

```javascript
import { ethers } from "ethers";

const provider = new ethers.WebSocketProvider("ws://localhost:8546");

// Subscribe to new blocks
provider.on("block", (blockNumber) => {
  console.log("New block:", blockNumber);
});

// Subscribe to contract events
const filter = {
  address: "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD28",
  topics: [ethers.id("Transfer(address,address,uint256)")],
};
provider.on(filter, (log) => {
  console.log("Transfer event:", log);
});
```

### QoreChain RPC WebSocket (Native)

```javascript
const ws = new WebSocket("ws://localhost:26657/websocket");

ws.onopen = () => {
  // Subscribe to fraud alerts
  ws.send(JSON.stringify({
    jsonrpc: "2.0",
    method: "subscribe",
    params: { query: "tm.event='Tx' AND fraud_alert.severity EXISTS" },
    id: 1,
  }));

  // Subscribe to rollup batch submissions
  ws.send(JSON.stringify({
    jsonrpc: "2.0",
    method: "subscribe",
    params: { query: "tm.event='Tx' AND batch_submitted.rollup_id EXISTS" },
    id: 2,
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.result && data.result.events) {
    console.log("Event received:", data.result.events);
  }
};
```

---

## Notes

- **Connection limits**: The default maximum number of WebSocket connections is unlimited (`max-open-connections = 0`). Set a limit in `app.toml` for production deployments.
- **Event buffer**: The RPC WebSocket buffers up to 200 events per subscription. If the client falls behind, older events are dropped.
- **Reconnection**: Clients should implement automatic reconnection with exponential backoff, as WebSocket connections may be interrupted during node restarts or upgrades.
