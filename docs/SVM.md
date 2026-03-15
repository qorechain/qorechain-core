# QoreChain SVM Runtime

## Overview

QoreChain's SVM (Solana Virtual Machine) runtime is the third execution environment in the triple-VM architecture. The SVM runtime enables BPF (Berkeley Packet Filter) program deployment and execution on QoreChain, with a Solana-compatible JSON-RPC interface that allows existing Solana clients and tooling to interact with QoreChain natively. As of v1.4.0, the runtime includes native built-in programs (System, SPL Token, ATA, Memo), account serialization, PDA derivation, CPI bridging, sysvar syscalls, and 20 Solana-compatible RPC methods.

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

### Supported Methods (20 Total)

| Method | Parameters | Description |
|--------|-----------|-------------|
| `getAccountInfo` | `pubkey (base58)` | Retrieve account data, owner, lamports, and executable flag |
| `getBalance` | `pubkey (base58)` | Get account balance in lamports |
| `getSlot` | (none) | Current slot number (derived from block height + offset) |
| `getMinimumBalanceForRentExemption` | `dataLength (number)` | Minimum lamports for rent-exempt account |
| `getVersion` | (none) | Runtime version info (`1.18.0-qorechain`) |
| `getHealth` | (none) | Health check (`"ok"`) |
| `sendTransaction` | `signedTx (base64)` | Submit a signed transaction for execution |
| `simulateTransaction` | `signedTx (base64)` | Simulate a transaction without submitting |
| `getProgramAccounts` | `programId (base58)`, `filters (optional)` | Get all accounts owned by a program |
| `getMultipleAccounts` | `pubkeys (base58[])` | Batch-fetch multiple accounts |
| `getSignaturesForAddress` | `address (base58)`, `limit (optional)` | Transaction signatures involving an address |
| `getTransaction` | `signature (base58)` | Full transaction details by signature |
| `getTokenAccountsByOwner` | `owner (base58)`, `mint or programId` | Token accounts for a wallet |
| `getTokenAccountsByDelegate` | `delegate (base58)`, `mint or programId` | Token accounts delegated to an address |
| `getBlockHeight` | (none) | Current block height |
| `getRecentBlockhash` | (none) | Recent blockhash for transaction signing |
| `getLatestBlockhash` | (none) | Latest blockhash with last valid block height |
| `getFeeForMessage` | `message (base64)` | Estimated fee for a serialized message |
| `isBlockhashValid` | `blockhash (base58)` | Check if a blockhash is still valid |
| `requestAirdrop` | `pubkey (base58)`, `lamports (number)` | Request an airdrop (testnet only) |

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

## Native Built-in Programs (v1.4.0)

QoreChain's SVM runtime includes four native built-in programs that execute without BPF interpretation, providing gas-efficient access to core Solana-compatible functionality.

| Program | Address | Description |
|---------|---------|-------------|
| **System Program** | `11111111111111111111111111111111` | Account creation, ownership assignment, SOL transfers, space allocation |
| **SPL Token Program** | `TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA` | Token mint/account initialization, transfers, approvals, minting, burning, account closure |
| **Associated Token Account (ATA)** | `ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL` | Deterministic token account creation (Create, CreateIdempotent) |
| **Memo Program** | `MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr` | On-chain memo logging attached to transactions |

### System Program Instructions

| Instruction | Description |
|-------------|-------------|
| `CreateAccount` | Create a new account with specified space and lamports, assigning ownership to a program |
| `Assign` | Change the owner of an account |
| `Transfer` | Transfer lamports between accounts |
| `Allocate` | Allocate data space for an account |

### SPL Token Program Instructions

| Instruction | Description |
|-------------|-------------|
| `InitializeMint` | Initialize a new token mint with decimals and authorities |
| `InitializeAccount` | Initialize a token account for a specific mint |
| `Transfer` | Transfer tokens between token accounts |
| `Approve` | Delegate token spending to another account |
| `Revoke` | Revoke a previously approved delegation |
| `MintTo` | Mint new tokens to a token account (mint authority required) |
| `Burn` | Burn tokens from a token account |
| `CloseAccount` | Close a token account and reclaim rent |
| `GetAccountDataSize` | Query the required data size for a token account |

## Account Serialization (v1.4.0)

The SVM runtime uses a Solana-compatible binary account serialization format for the FFI boundary between Go and Rust. This ensures that BPF programs receive account data in the exact layout they expect.

Each account is serialized as a contiguous byte region containing:

| Field | Size | Description |
|-------|------|-------------|
| `is_signer` | 1 byte | Whether the account signed the transaction |
| `is_writable` | 1 byte | Whether the account is writable |
| `key` | 32 bytes | Account public key |
| `lamports` | 8 bytes (LE) | Account balance in lamports |
| `data_len` | 8 bytes (LE) | Length of account data |
| `data` | variable | Account data bytes |
| `owner` | 32 bytes | Program that owns the account |
| `executable` | 1 byte | Whether the account is an executable program |
| `rent_epoch` | 8 bytes (LE) | Rent epoch |

After BPF execution, modified accounts are deserialized back using the same format, and balance/data changes are applied to the on-chain state.

## CPI Bridge (v1.4.0)

Cross-Program Invocation (CPI) enables BPF programs to call native built-in programs via the `sol_invoke_signed` syscall. In v1.4.0, the CPI bridge supports the BPF-to-Native direction:

- BPF programs can invoke System Program, SPL Token, ATA, and Memo instructions
- Signer seeds are supported for PDA-based signing
- CPI depth is limited by the `max_cpi` parameter (default: 4)
- Native-to-BPF and BPF-to-BPF CPI paths are planned for a future release

### Example CPI Flow

```
BPF Program
  └─ sol_invoke_signed(SPL Token, TransferInstruction, accounts, signer_seeds)
       └─ Native SPL Token Program executes Transfer
            └─ Account balances updated in-place
```

## PDA Derivation (v1.4.0)

Program Derived Addresses (PDAs) are deterministic addresses derived from a program ID and a set of seeds. The SVM runtime provides two syscalls for PDA operations:

| Syscall | Description |
|---------|-------------|
| `sol_create_program_address` | Derive a PDA from seeds and program ID. Returns error if the derived address falls on the Ed25519 curve. |
| `sol_try_find_program_address` | Find a valid PDA by iterating bump seeds (255 down to 0). Returns the address and the bump seed that produces an off-curve address. |

PDAs are used extensively for:
- Token account authority (e.g., vault signers in DeFi programs)
- Program-owned data accounts with deterministic addresses
- CPI signing without private keys (programs sign via their PDA seeds)

## Sysvar Syscalls (v1.4.0)

BPF programs can access on-chain state variables via sysvar syscalls:

| Syscall | Description |
|---------|-------------|
| `sol_get_clock_sysvar` | Returns current slot, epoch, unix timestamp, and leader schedule epoch |
| `sol_get_rent_sysvar` | Returns lamports per byte-year, exemption threshold, and burn percentage |

## SPL Token Support (v1.4.0)

QoreChain's SVM runtime provides full SPL token lifecycle support, enabling Solana-compatible token operations:

### Creating a Token

1. **Create Mint** — Use System Program `CreateAccount` to allocate space, then SPL Token `InitializeMint` to set decimals and authorities
2. **Create Token Account** — Use ATA Program `Create` (or `CreateIdempotent`) for deterministic associated token accounts
3. **Mint Tokens** — Use SPL Token `MintTo` with the mint authority signer

### Transferring Tokens

- Direct: SPL Token `Transfer` between token accounts
- Delegated: `Approve` a delegate, then the delegate calls `Transfer`
- Revoke: `Revoke` removes an existing delegation

### Closing Token Accounts

Use SPL Token `CloseAccount` to close a zero-balance token account and reclaim the rent-exempt lamports to a destination account.

### Compatibility

Existing Solana programs compiled for the SPL Token Program interface work on QoreChain without modification. The native implementation matches Solana's instruction encoding and account layout.

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
