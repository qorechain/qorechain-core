# SVM Development

QoreChain includes a **Solana Virtual Machine (SVM)** execution environment, allowing developers to deploy and execute BPF programs using familiar Solana tooling. The SVM module exposes a Solana-compatible JSON-RPC interface on **port 8899**.

---

## Overview

The `x/svm` module provides:

- BPF program deployment and execution
- Data account creation and management
- A Solana-compatible JSON-RPC endpoint
- Bidirectional address mapping between QoreChain and Solana address formats
- Compute budget metering and rent-based storage economics

---

## JSON-RPC Endpoint

| Property | Value |
|----------|-------|
| Default URL | `http://localhost:8899` |
| Compatibility | Solana JSON-RPC (subset) |

### Supported Methods

| Method | Description |
|--------|-------------|
| `getAccountInfo` | Retrieve account data and lamport balance |
| `getBalance` | Get account balance in lamports |
| `getSlot` | Current slot number |
| `getMinimumBalanceForRentExemption` | Minimum balance for a given data size |
| `getVersion` | SVM runtime version info |
| `getHealth` | Health check for the SVM endpoint |

---

## Deploy a BPF Program

Compile your Solana program to a BPF shared object, then deploy it to QoreChain:

```bash
# Deploy the compiled program
qorechaind tx svm deploy-program ./my_program.so \
  --from mykey \
  --gas auto \
  --gas-adjustment 1.3 \
  -y
```

The transaction response includes the **program ID** in base58 format.

---

## Execute an Instruction

Call an on-chain BPF program with instruction data:

```bash
qorechaind tx svm execute <program-id-base58> <data-hex> \
  --from mykey \
  --gas auto \
  -y
```

| Parameter | Format | Description |
|-----------|--------|-------------|
| `program-id-base58` | Base58 string | The deployed program's address |
| `data-hex` | Hex-encoded bytes | Serialized instruction data |

---

## Create a Data Account

Programs often need accounts to store state. Create one with a specified size and owner:

```bash
qorechaind tx svm create-account <owner-base58> <space> <lamports> \
  --from mykey \
  --gas auto \
  -y
```

| Parameter | Description |
|-----------|-------------|
| `owner-base58` | The program that owns this account (base58) |
| `space` | Size of the data field in bytes |
| `lamports` | Initial balance (must meet rent exemption minimum) |

Query the minimum rent-exempt balance for a given size:

```bash
curl -X POST http://localhost:8899 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "getMinimumBalanceForRentExemption",
    "params": [1024]
  }'
```

---

## Using @solana/web3.js

The Solana JavaScript SDK works directly with the QoreChain SVM endpoint:

```javascript
import { Connection, PublicKey } from "@solana/web3.js";

const connection = new Connection("http://localhost:8899");

// Check health
const health = await connection.getHealth();
console.log("SVM health:", health);

// Get slot
const slot = await connection.getSlot();
console.log("Current slot:", slot);

// Get account info
const pubkey = new PublicKey("YourBase58ProgramId...");
const accountInfo = await connection.getAccountInfo(pubkey);
console.log("Account data:", accountInfo);

// Get balance
const balance = await connection.getBalance(pubkey);
console.log("Balance (lamports):", balance);
```

---

## Address Mapping

QoreChain maintains a **bidirectional address mapping** between native Bech32 addresses (`qor1...`) and Solana-style base58 addresses:

| Direction | Example |
|-----------|---------|
| Native to SVM | `qor1abc...xyz` maps to a deterministic base58 address |
| SVM to Native | Base58 program addresses map back to `qor1...` equivalents |

The mapping is deterministic and managed by the `x/svm` module. Both representations refer to the same underlying account.

---

## Rent Model

The SVM module uses a **rent-based storage model** to prevent state bloat:

| Parameter | Value |
|-----------|-------|
| Lamports per byte per year | `3,480` |
| Rent exemption multiplier | `2.0` |
| Collection frequency | Each epoch |

- Accounts with a balance **above** `2 * (data_size * 3480 / seconds_per_year)` in lamports are **rent-exempt** and are never charged.
- Accounts **below** the rent-exemption threshold are charged rent each epoch. If the balance reaches zero, the account is purged.

> **Best practice:** Always fund data accounts above the rent-exemption minimum to avoid unexpected account deletion.

---

## Compute Budget

Each instruction execution is metered with compute units:

| Parameter | Value |
|-----------|-------|
| Max compute units per instruction | `1,400,000` |
| Max CPI (cross-program invocation) depth | `4` |
| Max program size | `10 MB` |
| Max account data size | `10 MB` |

Programs that exceed the compute budget are halted and the transaction is reverted.

---

## Parameters Summary

| Parameter | Value |
|-----------|-------|
| `max_program_size` | 10 MB |
| `max_account_data_size` | 10 MB |
| `compute_budget_max` | 1,400,000 CU |
| `max_cpi_depth` | 4 |
| `lamports_per_byte_year` | 3,480 |
| `rent_exemption_multiplier` | 2.0 |
| JSON-RPC port | 8899 |

---

## Cross-VM Interoperability

SVM programs can communicate with EVM and CosmWasm contracts through the **asynchronous** cross-VM message path:

```bash
qorechaind tx crossvm call \
  --source-vm svm \
  --target-vm evm \
  --target-contract 0x1234...abcd \
  --payload '...' \
  --from mykey \
  -y
```

Messages are queued and processed by the EndBlocker. See [Cross-VM Interoperability](cross-vm-interop.md) for details on the message lifecycle and timeout behavior.

---

## Next Steps

- [Cross-VM Interoperability](cross-vm-interop.md) -- Communication between SVM, EVM, and CosmWasm
- [EVM Development](evm-development.md) -- Solidity smart contracts on QoreChain
- [CosmWasm Development](cosmwasm-development.md) -- Rust-based WebAssembly contracts
