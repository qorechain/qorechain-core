# Cross-VM Interoperability

QoreChain's **triple-VM architecture** (EVM, CosmWasm, SVM) allows smart contracts on any virtual machine to communicate with contracts on any other VM. The `x/crossvm` module provides both synchronous and asynchronous messaging paths.

---

## Architecture Overview

```
 EVM (Solidity)          CosmWasm (Rust/Wasm)         SVM (BPF)
      |                        |                        |
      |--- sync (precompile) ->|                        |
      |                        |                        |
      |<-- async (EndBlocker) -|-- async (EndBlocker) ->|
      |                        |                        |
      |<------------ async (EndBlocker) ----------------|
```

| Path | Direction | Timing | Mechanism |
|------|-----------|--------|-----------|
| **Synchronous** | EVM to CosmWasm | Same transaction | Precompile at `0x0000...0901` |
| **Asynchronous** | CosmWasm to EVM | Next block | `MsgCrossVMCall` via EndBlocker |
| **Asynchronous** | SVM to any VM | Next block | `MsgCrossVMCall` via EndBlocker |
| **Asynchronous** | Any to SVM | Next block | `MsgCrossVMCall` via EndBlocker |

---

## Synchronous Path (EVM to CosmWasm)

The synchronous path uses an EVM **precompile** at address `0x0000000000000000000000000000000000000901`. This allows Solidity contracts to call CosmWasm contracts and receive a response within the same transaction.

### Solidity Example

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface ICrossVM {
    function call(bytes calldata payload) external returns (bytes memory);
}

contract CrossVMCaller {
    ICrossVM constant CROSSVM = ICrossVM(0x0000000000000000000000000000000000000901);

    function callCosmWasmContract(
        string memory cosmwasmAddr,
        string memory executeMsg,
        uint256 funds
    ) external returns (bytes memory) {
        bytes memory payload = abi.encode(cosmwasmAddr, executeMsg, funds);
        return CROSSVM.call(payload);
    }
}
```

The precompile executes the CosmWasm contract immediately and returns the result. Gas cost: **50,000 base + execution cost**.

---

## Asynchronous Path

All other cross-VM directions use the asynchronous message queue. Messages are submitted in one block and processed by the **EndBlocker** in the next block.

### CLI

```bash
# CosmWasm to EVM
qorechaind tx crossvm call \
  --source-vm cosmwasm \
  --target-vm evm \
  --target-contract 0x1234...abcd \
  --payload '{"method":"transfer","params":["0xRecipient",100]}' \
  --from mykey \
  -y

# SVM to CosmWasm
qorechaind tx crossvm call \
  --source-vm svm \
  --target-vm cosmwasm \
  --target-contract qor1contractaddr... \
  --payload '{"execute":{"action":{}}}' \
  --from mykey \
  -y

# EVM to SVM (async)
qorechaind tx crossvm call \
  --source-vm evm \
  --target-vm svm \
  --target-contract <program-id-base58> \
  --payload '0a0b0c...' \
  --from mykey \
  -y
```

---

## Message Lifecycle

Every cross-VM message transitions through a defined set of states:

```
Submitted --> Pending --> Executed
                |
                +--> Failed
                |
                +--> Timed Out
```

| State | Description |
|-------|-------------|
| **Submitted** | Message accepted into the queue |
| **Pending** | Awaiting execution in the next EndBlocker pass |
| **Executed** | Target contract called successfully; response recorded |
| **Failed** | Target contract execution reverted; error recorded |
| **Timed Out** | Message exceeded `queue_timeout_blocks` without execution |

---

## Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| `max_message_size` | 65,536 bytes | Maximum payload size per message |
| `max_queue_size` | 1,000 | Maximum pending messages in the queue |
| `queue_timeout_blocks` | 100 | Blocks before an unprocessed message times out |

---

## Events

The `x/crossvm` module emits the following events:

| Event | Attributes | Description |
|-------|-----------|-------------|
| `crossvm_request` | `message_id`, `source_vm`, `target_vm`, `target_contract`, `sender` | New cross-VM message submitted |
| `crossvm_response` | `message_id`, `status`, `result` | Message executed (success or failure) |
| `crossvm_timeout` | `message_id`, `source_vm`, `target_vm` | Message expired without execution |

Subscribe to events via WebSocket:

```bash
wscat -c ws://localhost:26657/websocket
> {"jsonrpc":"2.0","method":"subscribe","params":["tm.event='Tx' AND crossvm_request.message_id EXISTS"],"id":1}
```

---

## Querying Messages

### CLI

```bash
# Query a specific message by ID
qorechaind query crossvm message <message-id>

# List all pending messages
qorechaind query crossvm pending

# List messages by sender
qorechaind query crossvm messages-by-sender <address>
```

### JSON-RPC

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

### Response Format

```json
{
  "message_id": "crossvm-00000042",
  "source_vm": "cosmwasm",
  "target_vm": "evm",
  "target_contract": "0x1234...abcd",
  "sender": "qor1sender...",
  "payload": "...",
  "status": "executed",
  "result": "0x...",
  "submitted_height": 12345,
  "executed_height": 12346
}
```

---

## Design Considerations

**Atomicity:** Synchronous calls (EVM to CosmWasm via precompile) are atomic -- if either side reverts, the entire transaction reverts. Asynchronous calls are **not atomic** across blocks; design your contracts to handle the `Failed` and `Timed Out` states gracefully.

**Ordering:** Messages in the queue are processed FIFO within each EndBlocker pass. There is no guaranteed ordering across different source VMs.

**Payload encoding:** The payload format depends on the target VM:
- **EVM targets:** ABI-encoded function calls
- **CosmWasm targets:** JSON-encoded execute messages
- **SVM targets:** Hex-encoded BPF instruction data

---

## Next Steps

- [EVM Precompiles](evm-precompiles.md) -- The synchronous CrossVM precompile and other custom precompiles
- [EVM Development](evm-development.md) -- Solidity development on QoreChain
- [CosmWasm Development](cosmwasm-development.md) -- Rust/Wasm contract development
- [SVM Development](svm-development.md) -- BPF program deployment
