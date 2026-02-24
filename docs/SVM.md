# QoreChain SVM Runtime

## Overview

QoreChain v0.8.0 introduces the SVM (Solana Virtual Machine) runtime as the third execution environment in the triple-VM architecture. The SVM runtime enables BPF (Berkeley Packet Filter) program deployment and execution on QoreChain, with a Solana-compatible JSON-RPC interface that allows existing Solana clients and tooling to interact with QoreChain natively.

## Architecture

```
                    ┌────────────────────────────────────┐
                    │          QoreChain Node             │
                    │                                     │
  JSON-RPC (:8545) ─┤  eth_ / qor_ (EVM transactions)    │
  JSON-RPC (:8899) ─┤  Solana-compat (SVM queries)       │
  REST (:1317) ─────┤  QoreChain REST API                │
  gRPC (:9090) ─────┤  QoreChain gRPC API                │
                    │                                     │
                    │  ┌───────┐ ┌───────┐ ┌───────┐     │
                    │  │ x/vm  │ │x/wasm │ │ x/svm │     │
                    │  │ (EVM) │ │(Wasm) │ │ (BPF) │     │
                    │  └───┬───┘ └───┬───┘ └───┬───┘     │
                    │      └─────────┼─────────┘         │
                    │          x/crossvm                  │
                    │      (Precompile + Events)          │
                    └───────────┬────────────────────────┘
                                │
                    ┌───────────┴───────────┐
                    │      libqoresvm       │
                    │  (Rust BPF Executor)  │
                    └───────────────────────┘
```

## Module: x/svm

The SVM module manages the full lifecycle of BPF programs on QoreChain:

- **Program deployment** — Upload compiled BPF ELF binaries to the chain
- **Instruction execution** — Call deployed programs with instruction data and account references
- **Account management** — Create, fund, and manage SVM data accounts
- **Rent collection** — Automatic rent deduction from non-exempt accounts
- **Address mapping** — Bidirectional mapping between QoreChain (bech32) and SVM (base58) addresses
- **Slot tracking** — Block-height-derived slot numbering with configurable offset

## Rust Execution Engine (libqoresvm)

The BPF execution engine is implemented in Rust as the `qoresvm` crate and exposed to Go via CGO FFI. It provides:

- **rBPF-based execution** — JIT-compiled BPF program execution
- **Compute metering** — Instruction-level compute unit tracking with configurable budget
- **Cross-program invocation (CPI)** — Programs can call other programs up to a configurable depth
- **Syscall layer** — Solana-compatible syscalls including `sol_log`, `sol_invoke_signed`, `sol_get_clock_sysvar`, `sol_sha256`, and `sol_keccak256`
- **Memory management** — Stack + heap allocation with 32KB stack and 256KB heap per frame

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `max_program_size` | 10 MB | Maximum BPF ELF binary size |
| `max_account_data_size` | 10 MB | Maximum account data allocation |
| `compute_budget_max` | 1,400,000 | Maximum compute units per instruction |
| `lamports_per_byte` | 3,480 | Rent cost per byte per year |
| `rent_exemption_multi` | 2.0 | Multiplier for rent-exempt minimum balance |
| `enabled` | true | Whether SVM execution is enabled |
| `svm_slot_offset` | 0 | Offset added to block height for slot calculation |
| `default_sig_scheme` | 0 (Ed25519) | Default signature scheme for SVM accounts |
| `max_cpi` | 4 | Maximum cross-program invocation depth |

## JSON-RPC (Solana-Compatible)

Port: **8899** (HTTP)

The SVM runtime exposes a Solana-compatible JSON-RPC server. Existing Solana clients (such as `@solana/web3.js`) can connect directly.

### Supported Methods

| Method | Parameters | Description |
|--------|-----------|-------------|
| `getAccountInfo` | `pubkey (base58)` | Retrieve account data, owner, lamports, and executable flag |
| `getBalance` | `pubkey (base58)` | Get account balance in lamports |
| `getSlot` | (none) | Current slot number (derived from block height + offset) |
| `getMinimumBalanceForRentExemption` | `dataLength (number)` | Minimum lamports for rent-exempt account |
| `getVersion` | (none) | Runtime version info (`1.18.0-qorechain`) |
| `getHealth` | (none) | Health check (`"ok"`) |

### Response Format

All responses follow the Solana JSON-RPC 2.0 convention with `context.slot` for slot-aware queries:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "context": { "slot": 42 },
    "value": {
      "data": ["base64-encoded-data", "base64"],
      "executable": false,
      "lamports": 1000000,
      "owner": "11111111111111111111111111111111",
      "rentEpoch": 0
    }
  }
}
```

### Example Queries

```bash
# Get account info
curl -X POST http://localhost:8899 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getAccountInfo","params":["<base58-address>"]}'

# Get balance
curl -X POST http://localhost:8899 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getBalance","params":["<base58-address>"]}'

# Get current slot
curl -X POST http://localhost:8899 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getSlot","params":[]}'

# Get minimum balance for rent exemption (1024 bytes)
curl -X POST http://localhost:8899 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getMinimumBalanceForRentExemption","params":[1024]}'

# Get version
curl -X POST http://localhost:8899 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getVersion","params":[]}'
```

### Using Solana Web3.js

```javascript
const { Connection, PublicKey } = require("@solana/web3.js");

const connection = new Connection("http://localhost:8899");

// Get balance
const balance = await connection.getBalance(new PublicKey("<base58-address>"));
console.log("Balance:", balance, "lamports");

// Get account info
const accountInfo = await connection.getAccountInfo(new PublicKey("<base58-address>"));
console.log("Account:", accountInfo);

// Get slot
const slot = await connection.getSlot();
console.log("Current slot:", slot);
```

## CLI Commands

### Transactions

```bash
# Deploy a BPF program
qorechaind tx svm deploy-program ./my_program.so --from mykey

# Execute an instruction on a deployed program
qorechaind tx svm execute <program-id-base58> <data-hex> --from mykey

# Create an SVM data account
qorechaind tx svm create-account <owner-base58> <space> <lamports> --from mykey
```

### Queries

```bash
# Query an SVM account
qorechaind query svm account <base58-address>

# Query a deployed program
qorechaind query svm program <base58-address>

# Query SVM parameters
qorechaind query svm params

# Query current SVM slot
qorechaind query svm slot
```

## Address Mapping

QoreChain maintains bidirectional mapping between native bech32 addresses (`qor1...`) and SVM base58 addresses. When a QoreChain account deploys or interacts with SVM programs, the module automatically creates and manages the corresponding SVM address.

| Direction | Conversion |
|-----------|------------|
| QoreChain → SVM | `CosmosToSVMAddr(qor1...)` → base58 address |
| SVM → QoreChain | `SVMToCosmosAddr(base58)` → `qor1...` |

## Cross-VM Communication

SVM programs participate in the triple-VM cross-VM bridge via asynchronous event-based messaging. The `x/crossvm` module handles message routing between EVM, CosmWasm, and SVM runtimes.

| Path | Mechanism |
|------|-----------|
| EVM → SVM | Async event bridge via x/crossvm |
| CosmWasm → SVM | Async event bridge via x/crossvm |
| SVM → EVM | Async event bridge via x/crossvm |
| SVM → CosmWasm | Async event bridge via x/crossvm |

See [CROSSVM.md](./CROSSVM.md) for the full cross-VM communication protocol.

## Rent Model

SVM accounts are subject to rent charges based on their data size:

- **Rent per year** = `data_size_bytes * lamports_per_byte`
- **Rent-exempt minimum** = `rent_per_year * rent_exemption_multi`
- Accounts funded above the rent-exempt minimum are never charged rent
- Accounts below the minimum have rent deducted each epoch
- Accounts reaching zero lamports are purged

## Genesis Configuration

The SVM module is initialized in genesis under the `svm` key:

```json
{
  "svm": {
    "params": {
      "max_program_size": "10485760",
      "max_account_data_size": "10485760",
      "compute_budget_max": "1400000",
      "lamports_per_byte": "3480",
      "rent_exemption_multi": "2.0",
      "enabled": true,
      "svm_slot_offset": "0",
      "default_sig_scheme": 0,
      "max_cpi": 4
    },
    "accounts": [],
    "programs": [],
    "address_mappings": []
  }
}
```

## Building

The SVM runtime requires the Rust BPF execution engine (`libqoresvm`):

```bash
# Build the Rust crate
cd rust/qoresvm && cargo build --release

# Build QoreChain with SVM support (proprietary build)
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/

# Public build (SVM stubs — RPC returns "not available")
CGO_ENABLED=1 go build ./cmd/qorechaind/
```

The public community build includes the SVM module interface and CLI commands but uses stub implementations. Full BPF execution and the JSON-RPC server require the proprietary build.
