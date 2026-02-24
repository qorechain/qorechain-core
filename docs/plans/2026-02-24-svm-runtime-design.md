# SVM Runtime Design — QoreChain v0.8.0

**Date:** 2026-02-24
**Version:** v0.8.0
**Phase:** 8.0 — Solana Virtual Machine Runtime
**Estimated Effort:** 6-8 weeks

---

## 1. Overview

QoreChain v0.8.0 adds a Solana Virtual Machine (SVM) as the third runtime in the triple-VM architecture (EVM + CosmWasm + SVM). This makes QoreChain the first blockchain to natively support all three major smart contract ecosystems with shared PQC and AI infrastructure.

### Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scope | Full SVM (B.1–B.12) | Complete Solana developer experience |
| BPF engine | Hybrid: rbpf Rust FFI for execution, Go for state | Battle-tested execution + familiar Go patterns |
| Address format | Dual: 32-byte native + deterministic bech32 mapping | Solana compatibility + QoreChain interop |
| SPL programs | Full compat (Token, ATA, Memo) | Existing Solana DeFi programs work unmodified |
| JSON-RPC | Port 8899, comprehensive Solana RPC | Solana CLI, web3.js, Phantom all work |
| Rent model | Solana-compatible (lamports→uqor, 2-year exempt) | Solana programs' rent checks work unmodified |
| Signatures | Ed25519 primary + optional PQC upgrade | Solana wallet compat + quantum resistance path |

---

## 2. Architecture

```
┌─────────────────────────────────────────────────────┐
│                   QoreChain SDK Layer                 │
│  AnteHandler (triple routing: EVM / SVM / Cosmos)    │
├─────────────┬──────────────┬────────────────────────┤
│  x/vm (EVM) │  x/svm (SVM) │  x/wasm (CosmWasm)    │
│  go-ethereum │  Go keeper   │  wasmd                 │
│  cosmos-evm  │  + Rust exec │  wasmvm                │
├─────────────┴──────┬───────┴────────────────────────┤
│              x/crossvm (Cross-VM Bridge)             │
│  EVM↔CosmWasm (v0.5.0) + SVM↔EVM + SVM↔CosmWasm    │
├────────────────────┴─────────────────────────────────┤
│                 x/pqc    x/ai    x/reputation        │
│  (Available to all 3 VMs via precompiles/syscalls)   │
└──────────────────────────────────────────────────────┘
```

### Component Breakdown

| Component | Language | Build Tag | Purpose |
|-----------|----------|-----------|---------|
| Keeper (state) | Go | proprietary | Account store, program registry, rent, genesis |
| BPF Executor | Rust (rbpf) | proprietary | Execute BPF bytecode, syscall dispatch |
| FFI Bridge | Go+CGO | proprietary | Go↔Rust boundary for execution calls |
| Syscalls | Rust+Go | proprietary | Standard Solana + QoreChain extensions |
| SPL Programs | BPF | proprietary | Token, ATA, Memo (pre-deployed at genesis) |
| Ante Handler | Go | proprietary | SVM TX routing, signature verification, compute budget |
| JSON-RPC | Go | proprietary | Solana-compatible RPC server on port 8899 |
| Types/Interfaces | Go | (shared) | Public types, keeper interface, stubs |

### Data Flow (SVM Transaction)

```
1. User submits TX (Solana JSON-RPC or Cosmos TX)
2. AnteHandler detects SVM TX → routes to SVM path
3. SVM ante: Ed25519 sig verify → compute budget check → fee deduction
4. Keeper loads program BPF bytecode + input accounts from KVStore
5. FFI call to Rust: serialize(program, accounts, instruction_data)
6. rbpf executes BPF → syscalls callback into Go for:
   - sol_log_ (logging)
   - sol_invoke_signed_ (CPI → re-enter executor)
   - qor_pqc_verify (→ PQC keeper)
   - qor_call_evm (→ CrossVM keeper → EVM)
   - qor_call_cosmwasm (→ CrossVM keeper → CosmWasm)
7. Rust returns modified account data
8. Keeper writes updated accounts to KVStore
9. Events emitted, response returned
```

---

## 3. Account Model

### SVMAccount Structure

```go
type SVMAccount struct {
    Address    [32]byte  // Solana-style 32-byte public key
    Lamports   uint64    // Balance (1 lamport = 1 uqor)
    DataLen    uint64    // Allocated data buffer size
    Data       []byte    // Account data (programs store BPF bytecode here)
    Owner      [32]byte  // Program that owns this account
    Executable bool      // true = program account, false = data account
    RentEpoch  uint64    // Last epoch rent was collected
}
```

### Lamport ↔ uqor Mapping

- 1 lamport = 1 uqor (1:1 mapping)
- 1 QOR = 1,000,000 lamports = 1,000,000 uqor

### Dual Address Mapping

```
SVM → Cosmos:  qor_addr = bech32("qor", sha256(svm_32byte_addr)[:20])
Cosmos → SVM:  Lookup table in KVStore (reverse mapping stored on first use)
```

- Every SVM account gets a deterministic `qor1...` address via SHA-256 truncation to 20 bytes
- Reverse mapping stored in KVStore index: `svm_addr_map/{qor_addr} → [32]byte`
- EVM addresses (20-byte) map to SVM via: `svm_addr = sha256(evm_20byte_addr || "qorechain-svm")`

### System Program Addresses (Genesis)

| Program | Address | Purpose |
|---------|---------|---------|
| System Program | `11111111111111111111111111111111` | Account creation, transfers |
| SPL Token | `TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA` | Token operations |
| Associated Token Account | `ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL` | ATA derivation |
| Memo Program | `MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr` | Transaction memos |
| QoreChain PQC Program | `QorPQC111111111111111111111111111111111111` | PQC operations |
| QoreChain AI Program | `QorAI1111111111111111111111111111111111111` | AI scoring |

### KVStore Layout

```
svm/account/{32-byte-addr}     → SVMAccount (protobuf)
svm/program/{32-byte-addr}     → ProgramMeta (upgrade authority, deploy slot)
svm/addr_map/{bech32-addr}     → [32]byte (reverse lookup)
svm/rent/epoch                 → uint64 (current rent epoch)
svm/params                     → Params (protobuf)
svm/slot                       → uint64 (current SVM slot = Cosmos block height)
```

---

## 4. Rust BPF Executor (qoresvm)

### Crate Structure

```
rust/qoresvm/
├── Cargo.toml
├── build.rs                  # cbindgen header generation
├── src/
│   ├── lib.rs                # FFI exports (C ABI)
│   ├── executor.rs           # BPF execution engine (wraps solana-rbpf)
│   ├── syscalls.rs           # Syscall registry + QoreChain extensions
│   ├── loader.rs             # ELF program loader/validator
│   ├── memory.rs             # Memory region management
│   ├── account.rs            # Account serialization for FFI boundary
│   ├── spl/
│   │   ├── mod.rs            # SPL program loader
│   │   ├── token.rs          # SPL Token program (BPF bytecode embedded)
│   │   ├── ata.rs            # Associated Token Account program
│   │   └── memo.rs           # Memo program
│   └── error.rs              # Error codes
```

### Key Dependencies

```toml
[dependencies]
solana-rbpf = "0.8"
solana-program = "2.2"
solana-sdk = "2.2"
spl-token = "7.0"
spl-associated-token-account = "5.0"
spl-memo = "5.0"
```

### FFI Interface

```rust
// Initialize executor (called once at app startup)
#[no_mangle]
pub extern "C" fn qore_svm_init() -> *mut SVMExecutor;

// Execute a BPF program
#[no_mangle]
pub extern "C" fn qore_svm_execute(
    executor: *mut SVMExecutor,
    program_data: *const u8,      // BPF ELF bytecode
    program_len: usize,
    instruction_data: *const u8,  // Serialized instruction
    instruction_len: usize,
    accounts_data: *const u8,     // Serialized input accounts
    accounts_len: usize,
    compute_budget: u64,          // Max compute units
    syscall_callback: extern "C" fn(
        syscall_id: u32, data: *const u8, len: usize,
        out: *mut u8, out_len: *mut usize
    ) -> i32,
    result_accounts: *mut u8,     // Modified accounts output
    result_accounts_len: *mut usize,
    compute_used: *mut u64,
    log_output: *mut u8,
    log_len: *mut usize,
) -> i32;

// Validate BPF ELF binary
#[no_mangle]
pub extern "C" fn qore_svm_validate_program(
    program_data: *const u8, program_len: usize
) -> i32;

// Free executor
#[no_mangle]
pub extern "C" fn qore_svm_free(executor: *mut SVMExecutor);
```

### Syscall Callback Pattern

```
BPF calls qor_pqc_verify(...)
  → rbpf dispatches to registered syscall handler
  → Rust serializes args → calls syscall_callback(SYSCALL_PQC_VERIFY, ...)
  → Go callback receives via CGO → calls PQCKeeper.Verify()
  → Result serialized back → Rust returns to BPF program
```

### Syscall IDs

| ID | Name | Handler |
|----|------|---------|
| 0x01 | sol_log_ | Rust-only |
| 0x02 | sol_sha256_ | Rust-only |
| 0x03 | sol_keccak256_ | Rust-only |
| 0x04 | sol_invoke_signed_ | Rust re-entrant (CPI) |
| 0x05 | sol_create_program_address_ | Rust-only (PDA) |
| 0x06 | sol_get_clock_sysvar_ | Go callback |
| 0x07 | sol_get_rent_sysvar_ | Go callback |
| 0x10 | qor_pqc_verify | Go → PQC keeper |
| 0x11 | qor_pqc_key_status | Go → PQC keeper |
| 0x12 | qor_ai_risk_score | Go → AI keeper |
| 0x13 | qor_anomaly_check | Go → AI keeper |
| 0x14 | qor_call_evm | Go → CrossVM keeper |
| 0x15 | qor_call_cosmwasm | Go → CrossVM keeper |

### Build Outputs

- `lib/darwin_arm64/libqoresvm.dylib`
- `lib/darwin_amd64/libqoresvm.dylib`
- `lib/linux_amd64/libqoresvm.so`
- `lib/linux_arm64/libqoresvm.so`

---

## 5. Go Module Structure

### Directory Layout

```
x/svm/
├── interfaces.go              # SVMKeeper interface (shared)
├── module.go                  # AppModule (proprietary)
├── module_stub.go             # Stub AppModule (!proprietary)
├── register.go                # keeperAdapter + factories (proprietary)
├── keeper_stub.go             # StubKeeper (!proprietary)
├── ante.go                    # SVM AnteDecorator (proprietary)
├── ante_stub.go               # Stub ante (!proprietary)
├── ffi/
│   ├── bridge.go              # Go↔Rust FFI (proprietary)
│   ├── bridge.h               # C header
│   ├── bridge_stub.go         # Stub FFI (!proprietary)
│   └── callback.go            # Syscall callbacks (proprietary)
├── keeper/
│   ├── keeper.go              # Core keeper (proprietary)
│   ├── accounts.go            # Account CRUD (proprietary)
│   ├── programs.go            # Program deploy/upgrade (proprietary)
│   ├── executor.go            # Orchestrates FFI calls (proprietary)
│   ├── rent.go                # Rent collection (proprietary)
│   ├── genesis.go             # InitGenesis/ExportGenesis (proprietary)
│   ├── msg_server.go          # TX handlers (proprietary)
│   └── query_server.go        # Query handlers (proprietary)
├── rpc/
│   ├── server.go              # Solana JSON-RPC server (proprietary)
│   ├── handlers.go            # RPC method handlers (proprietary)
│   └── types.go               # RPC request/response types (shared)
├── types/
│   ├── keys.go
│   ├── params.go
│   ├── genesis.go
│   ├── errors.go
│   ├── account.go
│   ├── program.go
│   ├── instruction.go
│   ├── address.go
│   └── codec.go
├── client/cli/
│   ├── tx.go
│   └── query.go
└── README.md
```

### SVMKeeper Interface

```go
type SVMKeeper interface {
    GetAccount(ctx sdk.Context, addr [32]byte) (*types.SVMAccount, error)
    SetAccount(ctx sdk.Context, account *types.SVMAccount) error
    DeleteAccount(ctx sdk.Context, addr [32]byte) error
    DeployProgram(ctx sdk.Context, deployer [32]byte, bytecode []byte) ([32]byte, error)
    ExecuteProgram(ctx sdk.Context, programID [32]byte, instruction []byte,
                   accounts []types.AccountMeta, signers [][32]byte) (*types.ExecutionResult, error)
    SVMToCosmosAddr(svmAddr [32]byte) sdk.AccAddress
    CosmosToSVMAddr(cosmosAddr sdk.AccAddress) ([32]byte, error)
    CollectRent(ctx sdk.Context, addr [32]byte) error
    GetMinimumBalance(dataLen uint64) uint64
    GetParams(ctx sdk.Context) types.Params
    SetParams(ctx sdk.Context, params types.Params) error
    InitGenesis(ctx sdk.Context, gs types.GenesisState)
    ExportGenesis(ctx sdk.Context) *types.GenesisState
    Logger() log.Logger
}
```

### Module Parameters

```go
type Params struct {
    MaxProgramSize      uint64  // default: 10MB
    MaxAccountDataSize  uint64  // default: 10MB
    ComputeBudgetMax    uint64  // default: 1,400,000
    LamportsPerByte     uint64  // default: 3,480
    RentExemptionMulti  float64 // default: 2.0 (2 years of rent)
    Enabled             bool    // default: true
    SVMSlotOffset       int64   // default: 0
    DefaultSigScheme    uint8   // default: 0 (Ed25519)
    MaxCPI              uint8   // default: 4
}
```

### Transaction Messages

- **MsgDeployProgram**: Deploy BPF ELF binary as a program account
- **MsgExecuteProgram**: Execute instruction against a deployed program
- **MsgCreateAccount**: Create an SVM data account with allocated space
- **MsgRegisterSVMPQCKey**: Register Dilithium-5 key for optional PQC upgrade

---

## 6. Ante Handler (Triple Routing)

### Routing Extension

```go
case "/qorechain.svm.v1.ExtensionOptionsSVMTx":
    anteHandler = newSVMAnteHandler(options)
```

### SVM Ante Decorator Chain

```
SetUpContext
→ SVMRejectCosmosMessages
→ SVMSignatureVerify (Ed25519 or Dilithium-5)
→ SVMComputeBudgetCheck
→ SVMRentCheck
→ SVMDeductFee (compute_units × price → uqor)
→ SVMIncrementNonce
```

### Nonce/Blockhash Mapping

Solana uses "recent blockhash" instead of sequential nonces:
- Cosmos block hash at height H is valid for 150 blocks
- SVM keeper maintains a rolling window of 150 recent hashes
- `getRecentBlockhash` RPC returns latest Cosmos block hash

---

## 7. Cross-VM Extensions

### New VMType

```go
const VMTypeSVM VMType = "svm"
```

### Cross-VM Paths

| Path | Mechanism | Latency |
|------|-----------|---------|
| SVM → EVM | `qor_call_evm` syscall → CrossVM → EVM | Synchronous |
| SVM → CosmWasm | `qor_call_cosmwasm` syscall → CrossVM → Wasm | Synchronous |
| EVM → SVM | CrossVM precompile (0x0901) target=SVM | Async (EndBlocker) |
| CosmWasm → SVM | CrossVM message target=SVM | Async (EndBlocker) |

### Address Translation

- SVM → EVM: `evm_addr = svm_32byte_addr[:20]` (truncate)
- EVM → SVM: `svm_addr = sha256(evm_20byte_addr || "qorechain-svm")`
- SVM → CosmWasm: Use `SVMToCosmosAddr()` dual mapping
- CosmWasm → SVM: Use `CosmosToSVMAddr()` dual mapping

---

## 8. Solana JSON-RPC Server

**Port:** 8899 (standard Solana port)
**Flag:** `--svm-rpc.enable` on the start command

### Methods

| Method | Maps To |
|--------|---------|
| `getAccountInfo` | SVMKeeper.GetAccount() |
| `getBalance` | SVMKeeper.GetAccount().Lamports |
| `sendTransaction` | Broadcast Cosmos TX (MsgExecuteProgram) |
| `simulateTransaction` | SVMKeeper.ExecuteProgram(simulate=true) |
| `getProgramAccounts` | KVStore iteration with owner filter |
| `getTransaction` | Cosmos TX query by hash |
| `getBlock` | Cosmos block query |
| `getSlot` | Current Cosmos block height |
| `getEpochInfo` | Computed: epoch = height / 432,000 |
| `getRecentBlockhash` | Latest Cosmos block hash |
| `getMinimumBalanceForRentExemption` | SVMKeeper.GetMinimumBalance() |
| `getTokenAccountsByOwner` | KVStore scan for SPL Token accounts |
| `getSignatureStatuses` | Cosmos TX status query |
| `getHealth` | Node health check |

---

## 9. Testing Strategy

| Test Type | Scope | ~Count |
|-----------|-------|--------|
| Unit (types) | Account, instruction, address mapping, params, genesis | 30 |
| Unit (keeper) | Account CRUD, program deploy, rent collection | 20 |
| Unit (FFI) | BPF execution with mock programs | 15 |
| Integration | Deploy → execute → verify state | 10 |
| Cross-VM | SVM↔EVM, SVM↔CosmWasm | 10 |
| SPL | Token mint/transfer/burn, ATA creation | 10 |
| JSON-RPC | Each RPC method, valid/invalid inputs | 20 |
| Ante handler | Sig verify, compute budget, rent | 10 |
| **Total** | | **~125** |

---

## 10. Files to Create/Modify

### New Files (Public — qorechain-core)

```
x/svm/interfaces.go
x/svm/module_stub.go
x/svm/keeper_stub.go
x/svm/ante_stub.go
x/svm/ffi/bridge_stub.go
x/svm/types/keys.go
x/svm/types/params.go
x/svm/types/genesis.go
x/svm/types/errors.go
x/svm/types/account.go
x/svm/types/program.go
x/svm/types/instruction.go
x/svm/types/address.go
x/svm/types/codec.go
x/svm/rpc/types.go
x/svm/client/cli/tx.go
x/svm/client/cli/query.go
x/svm/README.md
```

### New Files (Proprietary — both repos)

```
x/svm/module.go
x/svm/register.go
x/svm/ante.go
x/svm/ffi/bridge.go
x/svm/ffi/bridge.h
x/svm/ffi/callback.go
x/svm/keeper/keeper.go
x/svm/keeper/accounts.go
x/svm/keeper/programs.go
x/svm/keeper/executor.go
x/svm/keeper/rent.go
x/svm/keeper/genesis.go
x/svm/keeper/msg_server.go
x/svm/keeper/query_server.go
x/svm/rpc/server.go
x/svm/rpc/handlers.go
rust/qoresvm/Cargo.toml
rust/qoresvm/build.rs
rust/qoresvm/src/lib.rs
rust/qoresvm/src/executor.rs
rust/qoresvm/src/syscalls.rs
rust/qoresvm/src/loader.rs
rust/qoresvm/src/memory.rs
rust/qoresvm/src/account.rs
rust/qoresvm/src/spl/mod.rs
rust/qoresvm/src/spl/token.rs
rust/qoresvm/src/spl/ata.rs
rust/qoresvm/src/spl/memo.rs
rust/qoresvm/src/error.rs
```

### Modified Files

```
app/app.go               — Add SVMKeeper, wire dependencies
app/app_config.go         — Add SVM to genesis ordering
app/ante.go              — Triple routing (EVM + SVM + Cosmos)
app/factory.go           — Add SVM factory declarations
app/factory_stub.go      — Add SVM stub factory
app/factory_proprietary.go — Add SVM real factory
cmd/qorechaind/cmd/root.go — Add SVMModuleBasic
x/crossvm/types/cross_vm_message.go — Add VMTypeSVM
x/crossvm/keeper/ (proprietary) — Add SVM bridge methods
docker-compose.yml       — Add SVM RPC port 8899
docs/ARCHITECTURE.md     — Update with triple-VM
docs/CROSSVM.md          — Update with SVM paths
docs/API_REFERENCE.md    — Add SVM endpoints
```
