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

Solidity contracts call the precompile at `0x0000000000000000000000000000000000000901` to execute a CosmWasm contract within the same transaction. The canonical ABI is published as [`contracts/abi/IQoreCrossVM.json`](../contracts/abi/IQoreCrossVM.json) with the Solidity interface [`contracts/interfaces/IQoreCrossVM.sol`](../contracts/interfaces/IQoreCrossVM.sol):

```solidity
function executeCrossVMCall(uint8 targetVM, string targetContract, bytes payload)
    external returns (bytes result);   // selector 0x3c11078c
```

```solidity
import {IQoreCrossVM} from "./interfaces/IQoreCrossVM.sol";

IQoreCrossVM constant CROSSVM = IQoreCrossVM(0x0000000000000000000000000000000000000901);

// targetVM 1 = CosmWasm; payload is the contract's JSON execute message.
bytes memory result = CROSSVM.executeCrossVMCall(
    1,
    "qor1...",                                   // allowlisted CosmWasm contract
    bytes('{"transfer":{"recipient":"qor1...","amount":"1"}}')
);
```

The precompile (from the v3.2.5 `crossvm-bridge-abi` switch):
1. Accepts only the selector above, a plain `CALL` with no value, and `targetVM = 1`; `STATICCALL`, `DELEGATECALL` and `CALLCODE` revert, and so do `targetVM` 0 (EVM) and 2 (SVM).
2. Refuses any target not listed in the `bridge_allowed_targets` parameter, and every call while `MsgCrossVMCall` is tripped on the circuit breaker.
3. Executes the contract with the precompile's caller (the contract or account that called `0x0901`, as its `qor1` address) as the CosmWasm sender, on the gas the calling frame has left.
4. Returns the contract's response `data`, ABI-encoded as `bytes`. A refused or failed call reverts with an `Error(string)` reason, and a revert anywhere in the EVM transaction undoes the CosmWasm execution too.
5. Keeps the calling EVM state in step with any bank transfer the contract makes, and refuses every attempt to enter the EVM again from inside the call.

The precompile is reachable only once governance has (a) added `0x...0901` to the x/vm `active_static_precompiles` list and (b) allowlisted at least one target with `MsgUpdateParams` (see Governance below).

### Synchronous: Native or CosmWasm → EVM / SVM / CosmWasm (MsgCrossVMCall)

An account (or a CosmWasm contract, by dispatching the message) sends `MsgCrossVMCall` with `async = false` (the default). The call executes inside the transaction, on the transaction's own gas, and its answer is returned in `MsgCrossVMCallResponse.data`; a failure reverts the whole transaction. The caller's lane is derived by the chain (a CosmWasm contract or a native account), never taken from `source_vm`. From the v3.2.5 `crossvm-lane-bounds` switch a call into a lane is refused while that lane's own entry message is tripped on the circuit breaker (`MsgEthereumTx` for the EVM, `MsgExecuteContract` for CosmWasm, `MsgExecuteProgram` for the SVM).

### Asynchronous (queue)

A `MsgCrossVMCall` with `async = true` is queued instead of executed:

1. A `MsgCrossVMCall` transaction is submitted with the target contract, payload and `async = true`. From the v3.2.5 `crossvm-lane-bounds` switch it prepays a fixed execution budget of 1,000,000 gas
2. The message is stored with `pending` status and added to the processing queue
3. The module's `EndBlocker` processes the queue (from `crossvm-lane-bounds`: each call runs on exactly its prepaid budget, at most 50 calls per block, the rest wait for the next block):
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
| `max_queue_size` | uint32 | 1,000 | Maximum pending messages in queue |
| `queue_timeout_blocks` | int64 | 100 | Blocks before a message times out |
| `enabled` | bool | true | Module enable/disable switch |
| `bridge_allowed_targets` | string[] | empty | Targets the EVM bridge precompile may call, each `cosmwasm:<qor1 contract address>`. Empty keeps the bridge shut. |

### Governance

From the v3.2.5 `crossvm-params-update` switch, `MsgUpdateParams` (signed by the governance module account, so it runs only from a passed proposal) replaces all parameters at once. It is refused unless every value is within bounds: `max_message_size` at most 1 MiB, `max_queue_size` at most 10,000, `queue_timeout_blocks` at most 100,000, and at most 64 distinct `bridge_allowed_targets`, each a valid CosmWasm contract address. Every field must be set, because a field left out becomes zero.

```json
{
  "@type": "/qorechain.crossvm.v1.MsgUpdateParams",
  "authority": "<gov module address>",
  "max_message_size": "65536",
  "max_queue_size": 1000,
  "queue_timeout_blocks": "100",
  "enabled": true,
  "bridge_allowed_targets": ["cosmwasm:qor1..."]
}
```

Allowlist only contracts that have been reviewed: a contract reached through the bridge acts with the EVM caller's identity as sender and runs inside an EVM transaction.

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

# Query module params (including the bridge allowlist)
qorechaind query crossvm params
# or over REST
curl https://<api>/qorechain/crossvm/v1/params
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

- **Public build** (`!internal`): Stub keeper that returns safe defaults. The module registers in genesis but cross-VM calls return "not available in public build" errors.
- **Internal build** (`internal`): Full keeper with precompile registration, CosmWasm execution via `PermissionedKeeper`, event queue processing, and PQC-signed cross-VM messages.

## Security

- Cross-VM messages are validated before queue insertion (valid VMs, non-empty payload, size limits)
- The precompile path runs within the EVM gas context - gas metering applies to CosmWasm execution
- Queue timeout prevents message accumulation from stalled VMs
- The module can be disabled via the `enabled` parameter without chain restart
- PQC signing is available for cross-VM attestations via the `x/pqc` module integration
