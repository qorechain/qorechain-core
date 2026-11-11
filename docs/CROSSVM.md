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

From the v3.2.8 `crossvm-async-gas-priced` switch an async call into CosmWasm runs on a fixed amount of contract execution, `async_gas_limit` in base gas units (250,000 up to what the queue can run in a block, the queue reserve plus half the transactions' pool, 1,875,000 at the default budget; 500,000 when unset), and prepays it times the congestion factor in force (at most 50,000,000; above that the call is refused at submission, retry later or with a lower limit). Every async call also pays a queue-slot fee of 500,000 gas times ((capacity + occupied) / capacity) squared, from 500,000 into an empty queue to 2,000,000 into a full one. The queue runs a call on exactly its units or not at all: it waits while the block wasm budget cannot admit all of it (its timeout still applies), and from the v3.2.8 `wasm-upload-bounds` switch also while its units at the factor in force cost more than it prepaid (it runs once the factor is back at or below the price it paid, or times out as before); a prepay buys one run. The first call of a block is held to the same queue limit as the others. Under congestion fewer queued calls run per block, since the queue holds at most 50,000,000 gas per block.

From the v3.2.8 `crossvm-async-escrow` switch the prepay of an async call into CosmWasm is not charged as gas: it is held in escrow. The sender chooses the highest congestion factor it will pay with `async_max_factor` (in millionths, `--async-max-factor 2.5` on the CLI; default four times the factor at submission, capped at the chain's maximum factor and at what keeps the prepay within 50,000,000 gas; a value below the factor in force is refused at submission, since such a call could only wait). Its units times that factor, at the fee market's minimum gas price (0.1 uqor per gas on both networks), move from the sender to the `crossvm_async_escrow` module account, so the sender needs that much spendable uqor: locked vesting coins, fee grants and fees paid in other tokens through gas abstraction cannot fund an escrow, and such a sender's async call is refused (by design: the escrow is the sender's own money, and it is refunded to the sender). Only the queue-slot fee is charged as gas at submission. On a chain whose fee market has no minimum price (zero) there is nothing to hold an escrow at, and the prepay is charged as gas as below the switch. The call runs while the factor in force is at or below its maximum and otherwise waits (or times out). When it finishes, the gas it used at the factor in force is paid from the escrow to the fee collector (where it joins the block's fees and their usual split) and the rest goes back to the sender; a call that fails pays what it used, and a call that times out or is never run gets all of it back, its slot fee being the processing fee. Settlement and the call's final record are written together, so an escrow is paid out once. The escrow account holds exactly what pending calls record (the `escrow` field of each call, exported and imported with genesis); a finished call records the fee it paid (`escrow_fee`) and the `crossvm_escrow_settled` event gives the fee and refund. With the default maximum a call is never held back by the factor in its own block (the factor rises at most by half per block, and by half at most once in five blocks), the factor must more than quadruple before it waits, and a sender who wants no factor-based wait at all can set the chain's maximum factor and is refunded whatever the call did not use.

The end-of-block walk keeps, for every queued call, its creation height and target lane, and for an escrowed call its units, max factor and load cost, in a small index entry, so calls waiting for the factor to fall or for room in the block are passed over without reading their records. Only runnable calls count against any per-block bound on the calls the walk looks at: a call waiting for the factor to come down to its max is not runnable, takes no visit, and cannot keep a runnable call (on any lane) from running in its block (measured: EndBlock 1.0 ms with 1,000 waiting calls of 65 KB each, 0.3 ms with an empty queue). If the queue ever runs without the atomic walk (a switch turned off with calls pending), finished calls still settle, on the gas they used.

From the v3.2.8 `wasm-code-load-priced` switch every call to a code that is not pinned (instantiate, execute, migrate, sudo, reply, smart query, IBC entry points, also from another contract or the bridge) pays for loading the code before it runs: 1,000 + one budget unit per 450 bytes of the code's conservative compiled size (below) from the block wasm budget (like the call's own execution: the transaction's share and pool, the queue's, or the counter for calls outside transactions) and 18 gas per unit at the price in force. The gas is taken first: a call that cannot pay it fails before the budget or the block's set of loaded codes is touched. Calibrated on measured loads from the node's disk cache (2.7 to 8.8 compiled KB per microsecond on the measuring host, about three times slower on validator hardware): a unit stays within the budget's design bound of 265 ns, and a cold call costs about as much gas per nanosecond as executing code (within 2x). In block execution the load is charged on the first call of a code in the block only: later calls to the same code in the same block pay neither the units nor the gas (the module is already in the node's memory), whether the earlier transaction succeeded or failed. The set of loaded codes is kept in memory in block execution order and starts empty every block, so every node, a restarted one included (it executes the block again from its start), charges the same calls. Free repeats hold while the block's distinct loaded codes total at most 64 MiB of conservative compiled size, below the 100 MiB minimum memory cache; past that every call is charged again. A node refuses to start with `[wasm] memory_cache_size` below 100 MiB, and at startup loads the codes governance pinned into its VM's pinned cache. Simulation and CheckTx always charge as for a first call, so estimates are never too low. A smart query served by a node pays only for a code that is not hot (loaded in this block, or recently on this node within its memory cache size), and such cold loads share a node-local allowance of 1,000,000 units per second. The first cw20 transfer to an unpinned code in a block costs about 364,000 gas and 20,300 budget units; later ones in the same block about 80,500 gas and 4,600 units. Pinned codes (governance `MsgPinCodes`) are exempt altogether. An async call into CosmWasm records what loading its target costs at submission: its prepay (and escrow) includes the load gas, its admission the load units; when the load is free in its block the call may use that gas for execution and an escrowed call is refunded what it did not use.

A module's conservative compiled size is the larger of 16 bytes per module byte plus 256 per function, and a per-operator estimate from its inspection (256 bytes per function, 180 per call, 250 per indirect call, 270 per block, loop, if, br_if or br_table, 70 per division or remainder, 16 per other operator). The VM's real compiled size of every release contract, every testnet code (407 distinct, worst 0.92 of the estimate) and adversarial shapes (worst 0.98) is at or below it.

From the v3.2.8 `wasm-upload-compiled-size` switch an upload may define at most 4,096 functions and its conservative compiled size may be at most 32 MiB (release contracts are 3.2 to 11.9 MB, the largest testnet code 11.9 MB; code already stored is never refused for either), and it pays 2 gas per byte of that size, which every node keeps on disk. cw20_base stores for about 14.7M gas, cw721_base for 37.7M, inside the 50M transaction cap. With the upload pool at its default (about 843,750 module bytes per block, just enough for one maximum-size upload) uploads can grow every node's disk by at most about 1.4 GB per 100 blocks, at a cost of at least about 430 QOR per 100 blocks at 0.1 uqor per gas; an upload refused at compile time pays all of it too.

When the block wasm budget refuses a transaction's contract work the transaction fails and pays its fee as any failed transaction does. Wallets should simulate first (the estimate carries the congestion margin) and can add a priority fee to land early in a block, before the budget is spent.

If an escrow cannot be paid out (a send the bank refuses), the call finishes as failed, its effects are dropped, and its whole escrow goes to the community pool, recorded as its fee; a call never keeps its queue slot for that reason. Genesis import refuses pending calls that record more escrow than the escrow account holds.

Storing code costs wasmd's own 3 gas per byte whatever the calibrated price and congestion factor, and is charged to the block wasm budget by size, one budget unit per byte (compiling measured 31 to 54 ns per byte on the measuring host). From the v3.2.8 `wasm-upload-bounds` switch an upload is checked in this order before anything is unpacked or compiled: the sender must be allowed by the upload access rule (the governance authority always is), then the transaction must have gas left for the declared module size (3 + 15 + 1 gas per uncompressed byte plus 200,000) or it fails out of gas at once, then the module is unpacked once, never past its declared size, and inspected in one pass (1 gas per byte); a gzip upload must be exactly one gzip member that unpacks to the size it declares, with nothing after it (an upload whose size field is not its member's own, or that unpacks to more or less, is refused after at most the declared size of work), a module whose custom sections total more than 128 KiB is refused, storage is charged at 15 gas per uncompressed byte, and only then is the budget charged and the module compiled. A module of wasmd's maximum 800 KiB costs about 15.6M gas, inside the 50M transaction cap. Upload units come from their own pool (`max_block_upload_bps`, default 37.5 percent of the transactions' share of the budget), so uploads never spend the execution pool and never count toward the congestion factor; an upload that is refused or runs out of gas before compiling takes nothing from any pool. Code already stored is never refused for its custom sections. The upload pool is shared by every sender and is charged when compiling starts, so uploads that the VM then refuses at compile time still use it: filling it with such uploads costs about 1.3 to 1.6 QOR per block (about 150 QOR per 100 blocks), about 4.2 QOR per block (about 420 QOR per 100 blocks) with the compiled-size charge of `wasm-upload-compiled-size` and delays only other code uploads, never contract execution or the congestion factor. This is accepted by design; if it happens, governance can raise `max_block_upload_bps`. The mempool admits upload messages only from senders the upload access rule allows. Gas estimates of contract work taken outside block execution (simulation, `eth_estimateGas` through the bridge) are priced at the congestion factor the next block will use plus two exhaustion steps (x2.25), so an estimate survives two blocks' rise; a node restored from a state-sync snapshot takes the factor from the restored state before it answers simulations. Proposers do not drop contract transactions from a proposal once the budget is likely spent: a proposer cannot know a transaction's contract use without executing it, and a cut by declared gas limits would cap ordinary blocks at about a dozen contract transactions. Once governance opens code upload (`code_upload_access` other than Nobody) the mempool admits upload messages too, as contracts may already dispatch them; execution applies the access rule, the code filter and the budget to every route alike.

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
| `max_block_upload_bps` | uint32 | 0 (= 3,750) | From v3.2.8 (`wasm-upload-bounds`): the block wasm budget units code uploads may use per block, as a share of the transactions' pool (1 to 10,000); separate from what contract execution uses. |
| `wasm_congestion_max_factor` | uint32 | 0 (= 20) | From v3.2.8 (`wasm-congestion-pricing`): the most the congestion factor may reach (1 to 100). |
| `wasm_congestion_target_bps` | uint32 | 0 (= 5,000) | From v3.2.8: after a block that used more than this share of its wasm budget the congestion factor rises by 1/8 (by half when the transactions' pool was exhausted, less than 5 percent left, by at least two fee payers each using 5 percent of the budget or more), after one that used less it falls by 1/9. The half step is taken at most once in five blocks; an exhausted block in between raises the factor by 1/8 (the height of the last half step is kept with the factor and exported in genesis as `wasm_congestion_exhausted_height`). The factor (kept in state, shown by the params query in millionths) multiplies the SDK gas one unit of contract execution costs; storage, setup and code upload are not affected, so an ordinary cw20 transfer costs about 80,500 gas at factor 1, 85,000 at 2, 122,000 at 10 and 168,000 at 20. |
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
