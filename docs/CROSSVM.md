# QoreChain Cross-VM Communication (x/crossvm)

## Overview

The `x/crossvm` module enables communication between the EVM and CosmWasm virtual machines running on QoreChain. It supports both synchronous calls (via an EVM precompile) and asynchronous message passing (via an event-based queue).

## Architecture

```
  Solidity Contract                           CosmWasm Contract
       │                                           │
       │  call(0x...0901, abi.encode(...))         │  emit crossvm_request event
       │                                           │
       ▼                                           ▼
  ┌──────────────────────────────────────────────────────┐
  │                    x/crossvm                         │
  │                                                      │
  │  Sync Path (Precompile)     Async Path (Queue)       │
  │  ┌─────────────────┐       ┌─────────────────────┐   │
  │  │ EVM → CosmWasm  │       │ CosmWasm → EVM      │   │
  │  │ Immediate exec  │       │ Queued, next block   │   │
  │  │ via wasm.Execute│       │ via ProcessQueue     │   │
  │  └─────────────────┘       └─────────────────────┘   │
  └──────────────────────────────────────────────────────┘
```

## Communication Paths

### Synchronous: EVM → CosmWasm (Precompile)

Solidity contracts call a precompile at address `0x0000000000000000000000000000000000000901` to invoke CosmWasm contracts synchronously within the same transaction:

```solidity
// Solidity example
address constant CROSSVM = 0x0000000000000000000000000000000000000901;

(bool success, bytes memory result) = CROSSVM.call(
    abi.encode(
        cosmwasmContractAddr,  // bech32 address of CosmWasm contract
        executeMsg,            // JSON-encoded CosmWasm execute message
        funds                  // coins to send
    )
);
```

The precompile:
1. Decodes the ABI-encoded call
2. Calls the CosmWasm contract via `wasm.PermissionedKeeper.Execute()`
3. Returns the result ABI-encoded back to the EVM caller

### Asynchronous: CosmWasm → EVM (Event Queue)

CosmWasm contracts can trigger EVM calls by submitting cross-VM messages to the queue:

1. A `MsgCrossVMCall` transaction is submitted with the target EVM contract and payload
2. The message is stored with `pending` status and added to the processing queue
3. The module's `EndBlocker` processes the queue:
   - Timed-out messages (exceeded `QueueTimeoutBlocks`) are marked `timed_out`
   - Pending messages are executed against the target VM
   - Results are emitted as `crossvm_response` events

## Message Types

### MsgCrossVMCall

Triggers a cross-VM contract call.

| Field | Type | Description |
|-------|------|-------------|
| `sender` | string | Bech32 address of the sender |
| `source_vm` | string | Source VM type (`evm` or `cosmwasm`) |
| `target_vm` | string | Target VM type (must differ from source) |
| `target_contract` | string | Target contract address |
| `payload` | bytes | Encoded call data |
| `funds` | Coins | Coins to transfer with the call |

### MsgProcessQueue

Triggers manual processing of the pending message queue. Normally the EndBlocker handles this automatically.

| Field | Type | Description |
|-------|------|-------------|
| `authority` | string | Module authority address |

## Message Lifecycle

```
  Submitted ──► Pending ──┬──► Executed (success)
                          ├──► Failed (execution error)
                          └──► Timed Out (exceeded timeout blocks)
```

Each message transitions through these states:
- **pending**: Stored in queue, awaiting processing
- **executed**: Successfully executed on the target VM
- **failed**: Execution failed (error stored in message)
- **timed_out**: Exceeded `QueueTimeoutBlocks` without execution

## Events

| Event Type | Attributes | Description |
|-----------|-----------|-------------|
| `crossvm_request` | message_id, source_vm, target_vm, target_contract, sender | New cross-VM message submitted |
| `crossvm_response` | message_id, status | Message execution completed |
| `crossvm_timeout` | message_id | Message timed out |

## Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `max_message_size` | uint64 | 65,536 | Maximum payload size in bytes |
| `max_queue_size` | uint64 | 1,000 | Maximum pending messages in queue |
| `queue_timeout_blocks` | int64 | 100 | Blocks before a message times out |
| `enabled` | bool | true | Module enable/disable switch |

## CLI Commands

### Transactions

```bash
# Submit a cross-VM call
qorechaind tx crossvm call \
  --source-vm cosmwasm \
  --target-vm evm \
  --target-contract 0xdeadbeef \
  --payload '{"method":"transfer","args":[...]}' \
  --from mykey

# Process the queue manually (authority only)
qorechaind tx crossvm process-queue --from authority
```

### Queries

```bash
# Query a cross-VM message by ID
qorechaind query crossvm message <message-id>

# List pending messages
qorechaind query crossvm pending

# Query module params
qorechaind query crossvm params
```

## JSON-RPC Integration

The `qor_` JSON-RPC namespace includes a cross-VM message query:

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "qor_getCrossVMMessage",
    "params": ["<message-id>"],
    "id": 1
  }'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "message_id": "abc123...",
    "source_vm": "cosmwasm",
    "target_vm": "evm",
    "target_contract": "0xdeadbeef",
    "status": "executed",
    "created_height": 1000,
    "executed_height": 1001
  }
}
```

## Store Keys

| Key | Format | Description |
|-----|--------|-------------|
| `crossvm/msg/{id}` | JSON-encoded CrossVMMessage | Message storage |
| `crossvm/queue/{id}` | Message ID bytes | Pending queue |
| `crossvm/params` | JSON-encoded Params | Module parameters |

## Open-Core Architecture

The cross-VM module follows QoreChain's open-core pattern:

- **Public build** (`!proprietary`): Stub keeper that returns safe defaults. The module registers in genesis but cross-VM calls return "not available in public build" errors.
- **Proprietary build** (`proprietary`): Full keeper with precompile registration, CosmWasm execution via `PermissionedKeeper`, event queue processing, and PQC-signed cross-VM messages.

## Security

- Cross-VM messages are validated before queue insertion (valid VMs, non-empty payload, size limits)
- The precompile path runs within the EVM gas context — gas metering applies to CosmWasm execution
- Queue timeout prevents message accumulation from stalled VMs
- The module can be disabled via the `enabled` parameter without chain restart
- PQC signing is available for cross-VM attestations via the `x/pqc` module integration
