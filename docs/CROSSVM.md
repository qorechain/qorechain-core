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

From the v3.2.8 `crossvm-failed-call-charged` switch a cross-VM call that fails is charged for the work it did, as one that succeeds is: an EVM call that reverts or runs out of gas consumes the EVM gas it used from the caller's meter before the failure is reported, and a CosmWasm call made through the `0x0901` bridge that fails consumes the gas the contract used from the calling EVM frame. A caller that catches failures (a contract reply, an x/group proposal with `Exec=TRY`, a Solidity `try`) therefore cannot repeat a failing call on the same remaining gas; a contract that wants to survive an expensive failing callee sets `gas_limit` on the submessage. Successful calls cost exactly what they did.

From the v3.2.8 `crossvm-error-text-plain` switch the revert reason the bridge returns to an EVM caller, and the error recorded on a failed queued call, carry no source-file annotation (` [<path>.go:<line>]`), whose path depends on how the node binary was built.

From the v3.2.8 `crossvm-response-charged` switch the data a queued call returns is paid from the call's own 1,000,000-gas budget at the store's write price before it is stored; a call whose response does not fit fails (`out of gas`) and stores no data. A call that needs to return large data should be made synchronously. From `crossvm-record-prune` the records of finished queued calls (executed, failed, timed out) are deleted `record_retention_blocks` after they finish. From `crossvm-async-funds-checked` an async call with funds into a lane that cannot carry them is refused at submission. From `crossvm-queue-fifo` the queue runs in order of creation height, then message ID.

The pending-messages query (`/qorechain/crossvm/v1/pending`) is paginated (`pagination.key`, `pagination.limit`, at most 100 per page) and shows at most the first 1,024 bytes of each payload; the message query returns the whole payload.

From the v3.2.8 `wasm-block-gas-budget` switch the queue's CosmWasm calls count against the block wasm budget: a queued call into CosmWasm runs only while the budget can cover its whole prepaid gas, and otherwise stays pending for a later block (its timeout still applies).

From the v3.2.8 `crossvm-async-gas-priced` switch an async call into CosmWasm runs on a fixed amount of contract execution, `async_gas_limit` in base gas units (250,000 up to what the queue can run in a block, the queue reserve plus half the transactions' pool, 1,875,000 at the default budget; 500,000 when unset), and prepays it times the congestion factor in force (at most 50,000,000; above that the call is refused at submission, retry later or with a lower limit). Every async call also pays a queue-slot fee of 500,000 gas times ((capacity + occupied) / capacity) squared, from 500,000 into an empty queue to 2,000,000 into a full one. The queue runs a call on exactly its units or not at all: it waits while the block wasm budget cannot admit all of it (its timeout still applies), runs once there is room whatever the factor did meanwhile, and a prepay buys one run. Under congestion fewer queued calls run per block, since the queue holds at most 50,000,000 gas per block.

Storing code costs wasmd's own 3 gas per byte whatever the calibrated price and congestion factor (an 800 KiB module about 2.4M gas), and is charged to the block wasm budget by size, one budget unit per byte (compiling measured 31 to 54 ns per byte on the measuring host). Gas estimates of contract work taken outside block execution (simulation, `eth_estimateGas` through the bridge) are priced at the congestion factor the next block will use plus one exhaustion step (x1.5), so an estimate survives one block's rise. Proposers do not drop contract transactions from a proposal once the budget is likely spent: a proposer cannot know a transaction's contract use without executing it, and a cut by declared gas limits would cap ordinary blocks at about a dozen contract transactions. Once governance opens code upload (`code_upload_access` other than Nobody) the mempool admits upload messages too, as contracts may already dispatch them; execution applies the access rule, the code filter and the budget to every route alike.

When the budget refuses a transaction's contract call, or cuts one short and the call runs out (counting its storage and query gas as well as its execution), the whole transaction fails ("block wasm gas budget") and can be retried in a later block; a submessage's reply cannot catch it, so an IBC packet receive is never turned into a permanent error acknowledgement by it.

**Block gas limit.** With `wasm-block-gas-budget` bounding contract execution separately, the consensus block `max_gas` (set by governance through x/consensus) should be 200,000,000, not 400,000,000: at the EVM's 3.7 to 3.9 ns per gas measured on validator hardware that bounds a block's EVM and native execution to under a second, plus at most about 0.85 s of wasm.

From the v3.2.8 `crossvm-queue-sender-cap` switch one sender may have at most `max_pending_per_sender` calls queued at a time (default 100, a tenth of the default queue), so a single sender cannot fill the shared queue and keep everyone else off the async path. A call over the bound is refused at submission with `ErrQueueFull`; once the sender's calls have run or timed out it may queue again. Synchronous calls are not affected. Calls queued before the switch do not count against the bound.

### Address rules and failure handling (from v3.2.7)

From the v3.2.7 `crossvm-address-shapes` switch a call is refused when it is submitted (queued or synchronous, from a transaction or from the `0x0901` bridge) if its target lane cannot represent the sender or the target:

| Target lane | Sender | `target_contract` |
|-------------|--------|-------------------|
| EVM | a 20-byte account, or a 32-byte derived account (a CosmWasm contract, a group policy); the EVM caller is the last 20 bytes | `0x` followed by 40 hex digits, not a precompile address (`0x...0000` to `0x...ffff`, the bridge `0x0901` included) |
| CosmWasm | a 20-byte or 32-byte account | a 32-byte `qor1...` contract address |
| SVM | a 20-byte account only (a 32-byte account has no SVM identity) | a canonical Base58 32-byte address |

Any other sender length is refused on every lane.

From the v3.2.7 `crossvm-panic-contained` switch a panic inside a call never reaches the block. A queued call that panics is recorded as `failed` with the error `cross-VM call aborted: the target VM panicked during execution`, its writes are dropped, its `crossvm_response` event carries an `error` attribute, and the queue goes on with the next call. On the synchronous path the panic becomes an `ErrExecutionFailed` error and the transaction fails; running out of gas still behaves as running out of gas. The async budget (1,000,000 gas) is charged at submission and is not refunded; coins attached to a CosmWasm call are only moved when the call succeeds.

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
| `crossvm_response` | message_id, status (and error for a failed call, from v3.2.7) | Message execution completed |
| `crossvm_timeout` | message_id | Message timed out |

## Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `max_message_size` | uint64 | 65,536 | Maximum payload size in bytes |
| `max_queue_size` | uint32 | 1,000 | Maximum pending messages in queue |
| `queue_timeout_blocks` | int64 | 100 | Blocks before a message times out |
| `enabled` | bool | true | Module enable/disable switch |
| `bridge_allowed_targets` | string[] | empty | Targets the EVM bridge precompile may call, each `cosmwasm:<qor1 contract address>`. Empty keeps the bridge shut. |
| `bridge_target_pins` | {target, code_id, history_len}[] | empty | From v3.2.8: the CosmWasm code ID each pinned allowlist entry must run, and the length of its code history when it was pinned, for the bridge to call it. |
| `max_pending_per_sender` | uint32 | 0 (= 100) | From v3.2.8: how many async calls one sender may have queued at a time. |
| `max_cosmos_tx_gas` | uint64 | 0 (= 50,000,000) | From v3.2.8 (`cosmos-tx-gas-cap`): the most gas one Cosmos-lane transaction may declare; 10,000,000 to 1,000,000,000 when set. A transaction above it is refused before its fee is taken. Kept here because these params are the chain's lane-level execution bounds. |
| `max_block_wasm_gas` | uint64 | 0 (= 3,000,000) | From v3.2.8 (`wasm-block-gas-budget`): the CosmWasm execution (VM gas, in SDK gas units) one block may use, failed transactions, the async queue and proposal execution included; 1,000,000 to 50,000,000 when set. Once spent, further contract calls in the block fail ("block wasm gas budget exhausted") and the next block starts full. On validator hardware (Intel c6i, AMD m6a) the slowest admitted operator costs about 265 ns per unit, so the default bounds a block's wasm execution to about 0.85 s; an ordinary cw20 transfer uses about 4,600. EVM and native transactions are not counted. |
| `max_tx_wasm_share_bps` | uint32 | 0 (= 5,000) | From v3.2.8: the share of the block wasm budget one transaction may use, in basis points (1 to 10,000). At the default one transaction may use up to 1,500,000; a single contract call needing more cannot run on this chain until governance raises the budget or the share. |
| `max_block_level_wasm_gas` | uint64 | 0 (= 10,000,000) | From v3.2.8: the CosmWasm execution all contract calls made outside transactions may use together in one block (a governance proposal executed at the end of the block, with every submessage and reply it makes); 1,000,000 to 100,000,000 when set. A separate counter, reset every block: transactions cannot spend it. A proposal whose contract calls run out of it fails; the chain goes on. |
| `queue_wasm_reserve_bps` | uint32 | 0 (= 2,500) | From v3.2.8: the share of the block wasm budget reserved for the async queue (1 to 9,000); transactions cannot spend it. A queued CosmWasm call is admitted on what the reserve and the transactions' leftover hold, its prepaid gas at most and 250,000 at least; below that it stays pending. |
| `wasm_congestion_max_factor` | uint32 | 0 (= 20) | From v3.2.8 (`wasm-congestion-pricing`): the most the congestion factor may reach (1 to 100). |
| `wasm_congestion_target_bps` | uint32 | 0 (= 5,000) | From v3.2.8: after a block that used more than this share of its wasm budget the congestion factor rises by 1/8 (by half when the transactions' pool was exhausted, less than 5 percent left, by at least two fee payers each using 5 percent of the budget or more), after one that used less it falls by 1/9. The factor (kept in state, shown by the params query in millionths) multiplies the SDK gas one unit of contract execution costs; storage, setup and code upload are not affected, so an ordinary cw20 transfer costs about 80,500 gas at factor 1, 85,000 at 2, 122,000 at 10 and 168,000 at 20. |
| `record_retention_blocks` | uint64 | 0 (= 100,000) | From v3.2.8 (`crossvm-record-prune`): how long the record of a finished queued call is kept before it is pruned (at most 200 records per block). |

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

#### Allowlist entries are bound to code (from v3.2.8)

An allowlist entry names a contract address, but what governance reviewed is the contract's code. A contract whose admin is an account can be migrated by that admin to any code it may instantiate, which would put unreviewed code behind a reviewed entry. From the v3.2.8 `crossvm-bridge-code-pin` switch:

- `bridge_target_pins` pins an entry to a code ID and to the length of the contract's code history (`qorechaind q wasm contract-history <addr>`: one entry for instantiation, one per migration). The bridge calls a pinned entry only while its contract runs that code with that history; after any migration the call is refused, including a migration to other code and back (which could have rewritten the contract's state).
- An entry whose contract has an admin other than the governance module account must be pinned. A contract with no admin, or with governance as admin, may be listed unpinned, as before; if its admin is later changed to an account, the bridge refuses it until it is pinned. An unpinned entry naming no contract is refused at call time.
- Every entry must name an existing contract, and a pin must name the code and history length the contract has when the update executes. To move a pinned contract to new code, put the `MsgMigrateContract` (or the admin's migration) first and then a `MsgUpdateParams` carrying the new code ID and the history length plus one; the same proposal can do both.
- The v3.2.8 upgrade pins every existing entry to the code and history its contract has at the upgrade height, so every target that answered before still answers, and removes entries that name no contract (event `crossvm_bridge_target_removed`). When resubmitting the parameters, copy `bridge_target_pins` from the params query along with the allowlist.

```json
{
  "@type": "/qorechain.crossvm.v1.MsgUpdateParams",
  "authority": "<gov module address>",
  "max_message_size": "65536",
  "max_queue_size": 1000,
  "queue_timeout_blocks": "100",
  "enabled": true,
  "bridge_allowed_targets": ["cosmwasm:qor1..."],
  "bridge_target_pins": [{"target": "cosmwasm:qor1...", "code_id": "5", "history_len": "1"}],
  "max_pending_per_sender": 100
}
```

Below their switches `bridge_target_pins` and `max_pending_per_sender` must be left empty; an update that sets them is refused.

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
| `crossvm/qsender/{length-prefixed sender address bytes}/{id}` | `0x01` | From v3.2.8: pending async call indexed by its sender (per-sender bound; one bucket per account however its address is spelled) |
| `crossvm/qdone/{8-byte height}{id}` | `0x01` | From v3.2.8 (`crossvm-record-prune`): finished queued call, by the height it finished at, for pruning |
| `crossvm/qcreated/{id}` | 8-byte height | From v3.2.8 (`crossvm-queue-lazy-decode`): creation height of a queued call, so the end-of-block walk decodes only the calls it times out or executes |

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
- A CosmWasm contract that calls the EVM (with `MsgCrossVMCall`) appears there as the last 20 bytes of its 32-byte address. Value, tokens and approvals an EVM contract gives to `msg.sender` in such a call are held by the native account with those 20 bytes, which only that contract can move (through the cross-VM module). Finding another key or contract whose address ends in the same 20 bytes takes a 2^160 search, or about 2^80 work for someone who grinds both a CosmWasm instantiate2 salt and an EVM CREATE2 salt to give two of their own contracts the same EVM identity. That only matters to a dApp that trusts such an address as "contract C": EVM contracts that authorize a CosmWasm caller should treat the 20-byte identity as an address, not as proof of which contract it is, and check the call's origin where it matters.
- A `MsgEthereumTx` executes only as a top-level message of an EVM-lane transaction. From v3.2.2 the handler refuses a context the EVM-lane ante handler did not mark; from the v3.2.8 `evm-dispatch-top-level-only` switch it also counts the transaction's own top-level `MsgEthereumTx`, so a contract reached through the bridge (whose context inherits the mark) cannot dispatch a previously signed Ethereum transaction either.
