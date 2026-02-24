# SVM Runtime Implementation Plan — QoreChain v0.8.0

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a full Solana Virtual Machine as the third runtime in QoreChain's triple-VM architecture, enabling Solana program deployment and execution with PQC/AI integration.

**Architecture:** Hybrid Rust/Go — rbpf (solana-rbpf) handles BPF bytecode execution via FFI, Go handles state management, rent, ante handler, and JSON-RPC. Open-core split: shared types/interfaces/stubs in public repo, keeper/FFI/executor in proprietary.

**Tech Stack:** Go 1.26, Rust (solana-rbpf 0.8, solana-sdk 2.2, spl-token 7.0), CGO FFI, Cosmos SDK v0.53.6

**Design Doc:** `docs/plans/2026-02-24-svm-runtime-design.md`

---

## Standing Rules (READ FIRST)

These rules apply to EVERY file and EVERY commit in this plan:

1. **Forbidden terms:** Never use Cosmos SDK, CometBFT, Tendermint, Claude, Anthropic, AWS Bedrock, Ethermint, Evmos in any human-readable string. See CHANGELOG.md prompt section for full table.
2. **Exceptions:** Go import paths (`github.com/cosmos/...`, `cosmossdk.io/...`), function names from upstream SDK, API endpoint protocol paths, proto field names.
3. **Git config:** `user.name "Liviu Epure"`, `user.email "liviu.etty@gmail.com"` — verify before first commit.
4. **Build tags:** Proprietary files get `//go:build proprietary`, stubs get `//go:build !proprietary`.
5. **Both builds must pass** after every commit: `CGO_ENABLED=1 go build ./cmd/qorechaind/` AND `CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/`
6. **Working directory:** `/Users/liviu/Development/Qore/testnet/qorechain-core/`
7. **Proprietary files** exist in BOTH `qorechain-core/` (on disk, tracked in git) AND `qorechain-proprietary/` (backup). After creating proprietary files in core, copy to proprietary repo.
8. **Test command:** `cd /Users/liviu/Development/Qore/testnet/qorechain-core && go test ./x/svm/...`

---

## Task 1: SVM Types Package — Keys, Errors, Account Types

**Goal:** Create the foundational types shared between both builds.

**Files to create:**
- `x/svm/types/keys.go`
- `x/svm/types/errors.go`
- `x/svm/types/account.go`
- `x/svm/types/program.go`
- `x/svm/types/instruction.go`
- `x/svm/types/address.go`

**Pattern reference:** `x/multilayer/types/keys.go` for KVStore prefix pattern.

### Step 1: Create `x/svm/types/keys.go`

Module name `svm`, store key `svm`. KVStore prefixes:
```go
AccountKeyPrefix       = []byte{0x01} // 0x01 | 32-byte-addr → SVMAccount
ProgramKeyPrefix       = []byte{0x02} // 0x02 | 32-byte-addr → ProgramMeta
AddrMapKeyPrefix       = []byte{0x03} // 0x03 | 20-byte-cosmos-addr → [32]byte
RentEpochKey           = []byte{0x04} // single key → uint64
ParamsKey              = []byte{0x05} // single key → Params
SlotKey                = []byte{0x06} // single key → uint64
RecentBlockhashPrefix  = []byte{0x07} // 0x07 | height(8 bytes) → [32]byte hash
```

Helper functions: `AccountKey(addr [32]byte)`, `ProgramKey(addr [32]byte)`, `AddrMapKey(cosmosAddr []byte)`, `RecentBlockhashKey(height uint64)`.

### Step 2: Create `x/svm/types/errors.go`

Error codes: `ErrProgramNotFound`, `ErrAccountNotFound`, `ErrInvalidBytecode`, `ErrComputeBudgetExceeded`, `ErrInsufficientLamports`, `ErrRentNotExempt`, `ErrAccountAlreadyExists`, `ErrInvalidAccountOwner`, `ErrMaxCPIDepthExceeded`, `ErrSVMDisabled`, `ErrInvalidAddress`, `ErrProgramNotExecutable`, `ErrInvalidSignature`, `ErrInvalidInstruction`.

### Step 3: Create `x/svm/types/account.go`

`SVMAccount` struct (see design doc §3). Include `Validate() error` method that checks: non-zero address, Data length matches DataLen, Executable accounts must have non-zero Owner. Include JSON and protobuf-style serialization helpers (`Marshal()`/`Unmarshal()` using `encoding/json`).

### Step 4: Create `x/svm/types/program.go`

`ProgramMeta` struct: `ProgramAddress [32]byte`, `UpgradeAuthority [32]byte` (zero = immutable), `DeploySlot uint64`, `LastDeploySlot uint64`, `DataAccount [32]byte` (BPF data stored here). Include `ExecutionResult` struct: `Success bool`, `ReturnData []byte`, `ComputeUnitsUsed uint64`, `Logs []string`, `ModifiedAccounts []SVMAccount`.

### Step 5: Create `x/svm/types/instruction.go`

`AccountMeta` struct: `Address [32]byte`, `IsSigner bool`, `IsWritable bool`. `Instruction` struct: `ProgramID [32]byte`, `Accounts []AccountMeta`, `Data []byte`.

### Step 6: Create `x/svm/types/address.go`

Address mapping utilities:
- `SVMToCosmosAddress(svmAddr [32]byte) sdk.AccAddress` — SHA-256 truncate to 20 bytes
- `CosmosToSVMAddress(cosmosAddr sdk.AccAddress) [32]byte` — error: must use KVStore lookup
- `EVMToSVMAddress(evmAddr [20]byte) [32]byte` — SHA-256 of `(evmAddr || "qorechain-svm")`
- `SVMToEVMAddress(svmAddr [32]byte) [20]byte` — truncate to 20 bytes
- `Base58Encode(addr [32]byte) string`, `Base58Decode(s string) ([32]byte, error)`
- System program address constants (System, SPL Token, ATA, Memo, QorPQC, QorAI)

### Step 7: Write tests

Create `x/svm/types/types_test.go`:
- Test SVMAccount.Validate() with valid/invalid cases
- Test address mapping round-trips (SVMToCosmosAddress)
- Test Base58 encode/decode
- Test system program addresses are unique and non-zero
- Test KVStore key helpers produce expected prefixes
- Test error codes are unique

Run: `go test ./x/svm/types/ -v -count=1`

### Step 8: Commit

```bash
git add x/svm/types/
git commit -m "feat(svm): add SVM types package — keys, errors, accounts, addresses"
```

---

## Task 2: SVM Params, Genesis, Codec

**Files to create:**
- `x/svm/types/params.go`
- `x/svm/types/genesis.go`
- `x/svm/types/codec.go`

### Step 1: Create `x/svm/types/params.go`

`Params` struct per design doc §5. `DefaultParams()` returns defaults. `Validate() error` checks ranges (MaxProgramSize > 0, ComputeBudgetMax > 0, etc.).

### Step 2: Create `x/svm/types/genesis.go`

`GenesisState` struct: `Params Params`, `Accounts []SVMAccount`, `Programs []ProgramMeta`, `CurrentSlot uint64`. `DefaultGenesis()` returns default params + system program accounts (System Program, SPL Token, ATA, Memo, QorPQC, QorAI — all marked Executable). `Validate() error`.

### Step 3: Create `x/svm/types/codec.go`

Register message types for amino and interface registry. Messages: `MsgDeployProgram`, `MsgExecuteProgram`, `MsgCreateAccount`, `MsgRegisterSVMPQCKey`. Create these message types with `ValidateBasic()` and `GetSigners()` methods.

Pattern reference: `x/crossvm/types/` for message type patterns.

### Step 4: Write tests

Add to `x/svm/types/types_test.go`:
- Test DefaultParams().Validate() passes
- Test DefaultGenesis().Validate() passes
- Test invalid params (zero MaxProgramSize, etc.)
- Test message ValidateBasic() for each Msg type
- Test system program accounts in DefaultGenesis are Executable

Run: `go test ./x/svm/types/ -v -count=1`

### Step 5: Commit

```bash
git add x/svm/types/
git commit -m "feat(svm): add SVM params, genesis, codec, and message types"
```

---

## Task 3: SVMKeeper Interface + Stubs + Module Stubs

**Goal:** Create the shared interface and public-build stubs so the public build compiles.

**Files to create:**
- `x/svm/interfaces.go` (no build tag)
- `x/svm/keeper_stub.go` (`//go:build !proprietary`)
- `x/svm/module_stub.go` (`//go:build !proprietary`)
- `x/svm/ante_stub.go` (`//go:build !proprietary`)
- `x/svm/ffi/bridge_stub.go` (`//go:build !proprietary`)

### Step 1: Create `x/svm/interfaces.go`

SVMKeeper interface per design doc §5. Also define `SVMExecutor` interface for the FFI abstraction:
```go
type SVMExecutor interface {
    Execute(program []byte, instruction []byte, accounts []types.SVMAccount,
            computeBudget uint64) (*types.ExecutionResult, error)
    ValidateProgram(bytecode []byte) error
    Close()
}
```

Pattern: Follow `x/crossvm/interfaces.go` exactly.

### Step 2: Create `x/svm/keeper_stub.go`

`StubKeeper` struct that implements `SVMKeeper`. All methods return sensible defaults or `ErrSVMDisabled`. `NewStubKeeper(logger log.Logger) SVMKeeper`.

Pattern: Follow `x/crossvm/keeper_stub.go`.

### Step 3: Create `x/svm/module_stub.go`

`AppModule` and `AppModuleBasic` for stub build. `Name() = "svm"`. Genesis methods delegate to types package. RegisterServices is no-op.

Pattern: Follow `x/crossvm/module_stub.go`.

### Step 4: Create `x/svm/ante_stub.go`

`StubSVMAnteDecorator` that passes through (no-op). `NewStubSVMAnteDecorator() sdk.AnteDecorator`.

### Step 5: Create `x/svm/ffi/bridge_stub.go`

`StubExecutor` implementing `SVMExecutor`. `Execute()` returns error "SVM executor not available in community build". `ValidateProgram()` returns error. `NewStubExecutor() SVMExecutor`.

### Step 6: Write tests

`x/svm/stub_test.go` (`//go:build !proprietary`):
- Test StubKeeper.GetAccount returns ErrSVMDisabled
- Test StubKeeper.DeployProgram returns ErrSVMDisabled
- Test StubSVMAnteDecorator passes through

Run: `go test ./x/svm/... -v -count=1`

### Step 7: Commit

```bash
git add x/svm/
git commit -m "feat(svm): add SVMKeeper interface, stubs, and module stubs"
```

---

## Task 4: Wire SVM into App — Public Build Passes

**Goal:** Register SVM module in the app so `go build ./cmd/qorechaind/` passes.

**Files to modify:**
- `app/factory.go` — Add SVM factory declarations
- `app/factory_stub.go` — Add SVM stub factory assignments
- `app/app.go` — Add SVMKeeper field, storeKey, module registration
- `cmd/qorechaind/cmd/root.go` — Add SVMModuleBasic

**Pattern:** Follow the exact CrossVM module wiring pattern.

### Step 1: Modify `app/factory.go`

Add after the multilayer factories:
```go
// SVM module factories
svmmod "github.com/qorechain/qorechain-core/x/svm"

NewSVMKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey,
                       pqcKeeper pqcmod.PQCKeeper, aiKeeper aimod.AIKeeper,
                       crossvmKeeper crossvmmod.CrossVMKeeper,
                       logger log.Logger) svmmod.SVMKeeper
NewSVMAppModule   func(keeper svmmod.SVMKeeper) module.AppModule
NewSVMModuleBasic func() module.AppModuleBasic
```

### Step 2: Modify `app/factory_stub.go`

Add SVM stub factory init (follow CrossVM pattern):
```go
NewSVMKeeper = func(_ codec.Codec, _ storetypes.StoreKey, _ pqcmod.PQCKeeper,
    _ aimod.AIKeeper, _ crossvmmod.CrossVMKeeper, logger log.Logger) svmmod.SVMKeeper {
    return svmmod.NewStubKeeper(logger)
}
NewSVMAppModule = func(keeper svmmod.SVMKeeper) module.AppModule {
    return svmmod.NewAppModule(keeper)
}
NewSVMModuleBasic = func() module.AppModuleBasic {
    return svmmod.AppModuleBasic{}
}
```

### Step 3: Modify `app/app.go`

Add `SVMKeeper svmmod.SVMKeeper` field. After CrossVM keeper init:
```go
svmStoreKey := storetypes.NewKVStoreKey(svmtypes.StoreKey)
app.MountStores(svmStoreKey)
app.SVMKeeper = NewSVMKeeper(app.appCodec, svmStoreKey,
    app.PQCKeeper, app.AIKeeper, app.CrossVMKeeper, logger)
```
Register module: `NewSVMAppModule(app.SVMKeeper)`.

### Step 4: Modify `cmd/qorechaind/cmd/root.go`

Add after multilayer registration:
```go
svmBasic := app.NewSVMModuleBasic()
moduleBasicManager[svmBasic.Name()] = svmBasic
```

### Step 5: Verify public build

```bash
CGO_ENABLED=1 go build ./cmd/qorechaind/
```

### Step 6: Commit

```bash
git add app/ cmd/ x/svm/
git commit -m "feat(svm): wire SVM module into app — public build passes"
```

---

## Task 5: Rust qoresvm Crate — Scaffold + Error Types

**Goal:** Create the Rust crate skeleton with Cargo.toml, error types, and build.rs.

**Files to create (in proprietary repo, then copy to core):**
- `rust/qoresvm/Cargo.toml`
- `rust/qoresvm/build.rs`
- `rust/qoresvm/src/lib.rs`
- `rust/qoresvm/src/error.rs`

### Step 1: Create `Cargo.toml`

```toml
[package]
name = "qoresvm"
version = "0.8.0"
edition = "2024"

[lib]
crate-type = ["cdylib", "staticlib"]

[dependencies]
solana-rbpf = "0.8"
solana-program = "2.2"
solana-sdk = "2.2"
spl-token = "7.0"
spl-associated-token-account = "5.0"
spl-memo = "5.0"
sha2 = "0.10"
bs58 = "0.5"

[profile.release]
opt-level = 3
lto = "fat"
codegen-units = 1
strip = "symbols"
panic = "abort"
```

NOTE: Exact dependency versions may need adjustment based on compatibility. Check latest crates.io for solana-rbpf and resolve any conflicts with solana-sdk. If solana-rbpf 0.8 is not available, use the latest available version and adjust API calls accordingly.

### Step 2: Create `build.rs`

If using cbindgen for header generation. Otherwise minimal build script.

### Step 3: Create `src/error.rs`

Error codes as `#[repr(i32)]` enum: `Success = 0`, `InvalidProgram = -1`, `ComputeBudgetExceeded = -2`, `MemoryError = -3`, `SyscallError = -4`, `AccountError = -5`, `SerializationError = -6`, `InvalidELF = -7`.

### Step 4: Create `src/lib.rs`

Module declarations (`mod executor; mod syscalls; mod loader; mod memory; mod account; mod spl; mod error;`). Placeholder FFI exports that return error codes (stubs that will be filled in later tasks).

### Step 5: Verify compilation

```bash
cd /Users/liviu/Development/Qore/testnet/qorechain-proprietary/rust/qoresvm
cargo build 2>&1 | head -50
```

If dependency resolution fails, adjust versions in Cargo.toml. The solana ecosystem has many inter-dependent crates — pinning compatible versions is critical.

### Step 6: Run Rust tests

```bash
cargo test
```

### Step 7: Commit (proprietary repo)

```bash
cd /Users/liviu/Development/Qore/testnet/qorechain-proprietary
git add rust/qoresvm/
git commit -m "feat(svm): scaffold qoresvm Rust crate with error types"
```

---

## Task 6: Rust — Memory Management + Account Serialization

**Files to create:**
- `rust/qoresvm/src/memory.rs`
- `rust/qoresvm/src/account.rs`

### Step 1: Create `memory.rs`

Memory region management for BPF VM. Define `MemoryRegion` for input data, program data, stack, heap. Max heap size: 32KB (Solana default). Stack size: 4096 frames. Helper functions to create memory maps for rbpf execution.

### Step 2: Create `account.rs`

Serialization format for passing accounts across the FFI boundary. Use a simple length-prefixed binary format:
- `serialize_accounts(accounts: &[SerializedAccount]) -> Vec<u8>`
- `deserialize_accounts(data: &[u8]) -> Result<Vec<SerializedAccount>, Error>`

`SerializedAccount`: address [32], lamports u64, data_len u64, data [data_len], owner [32], executable u8, rent_epoch u64.

### Step 3: Write Rust tests

Test serialization round-trip, test memory region allocation, test edge cases (empty data, max size).

```bash
cargo test
```

### Step 4: Commit

---

## Task 7: Rust — ELF Loader + Program Validator

**Files to create:**
- `rust/qoresvm/src/loader.rs`

### Step 1: Create `loader.rs`

Use `solana_rbpf::elf::Executable` to load and verify BPF ELF binaries:
- `load_program(bytecode: &[u8]) -> Result<Executable, Error>` — parse ELF, verify BPF instructions
- `validate_program(bytecode: &[u8]) -> bool` — quick validation without full load
- Check max program size (10MB default)
- Verify ELF has required sections (.text)

### Step 2: Implement FFI export `qore_svm_validate_program`

In `lib.rs`, implement the validation FFI function that calls `loader::validate_program`.

### Step 3: Rust tests

Test with a minimal valid BPF program (hand-crafted or compiled from a trivial Solana program). Test invalid ELF rejection. Test oversized program rejection.

### Step 4: Commit

---

## Task 8: Rust — Core BPF Executor

**Files to create:**
- `rust/qoresvm/src/executor.rs`
- `rust/qoresvm/src/syscalls.rs`

### Step 1: Create `executor.rs`

Core execution engine wrapping `solana_rbpf`:
- `SVMExecutor` struct holding rbpf configuration
- `execute()` method: load program → create VM → register syscalls → run → return results
- Compute meter integration (track CU consumption, abort on budget exceeded)
- Log collection during execution

### Step 2: Create `syscalls.rs`

Register standard Solana syscalls with rbpf:
- `sol_log_` — write to log buffer (Rust-only, no callback needed)
- `sol_sha256_` / `sol_keccak256_` — hash functions (Rust-only)
- `sol_create_program_address_` — PDA derivation (Rust-only)
- Syscall callback dispatch for Go-dependent syscalls (0x06-0x15): serialize args, call the `syscall_callback` function pointer, deserialize result

### Step 3: Implement FFI exports in `lib.rs`

Complete `qore_svm_init`, `qore_svm_execute`, `qore_svm_free` — the main FFI boundary.

### Step 4: Rust tests

Test executor with a minimal BPF program that:
1. Logs "hello" (tests sol_log_)
2. Returns success
3. Exceeds compute budget (tests CU metering)
4. Calls sha256 syscall

### Step 5: Build dylib

```bash
cargo build --release
cp target/release/libqoresvm.dylib ../../../qorechain-core/lib/darwin_arm64/
```

### Step 6: Commit

---

## Task 9: Rust — SPL Programs (Token, ATA, Memo)

**Files to create:**
- `rust/qoresvm/src/spl/mod.rs`
- `rust/qoresvm/src/spl/token.rs`
- `rust/qoresvm/src/spl/ata.rs`
- `rust/qoresvm/src/spl/memo.rs`

### Step 1: Embed SPL Token BPF bytecode

The SPL Token program is a pre-compiled BPF binary. Options:
- **Option A:** Include the official SPL Token v7 BPF binary as a `include_bytes!` resource
- **Option B:** Compile from spl-token source to BPF

Use Option A — download the official SPL Token program `.so` from Solana releases and embed it.

### Step 2: Create `spl/mod.rs`

`get_builtin_programs() -> Vec<(Pubkey, Vec<u8>)>` — returns list of (address, BPF bytecode) for all built-in programs.

### Step 3: Implement ATA and Memo

Same pattern — embed official BPF binaries.

### Step 4: Add FFI export

`qore_svm_get_builtin_programs(out: *mut u8, out_len: *mut usize) -> i32` — serializes all built-in program data for Go to load into genesis.

### Step 5: Rust tests + Commit

---

## Task 10: Go FFI Bridge

**Goal:** Create the Go↔Rust FFI bridge using CGO.

**Files to create:**
- `x/svm/ffi/bridge.h`
- `x/svm/ffi/bridge.go` (`//go:build proprietary`)
- `x/svm/ffi/callback.go` (`//go:build proprietary`)

### Step 1: Create `bridge.h`

C header declaring all FFI functions. Either hand-written or generated by cbindgen.

### Step 2: Create `bridge.go`

CGO directives linking to `libqoresvm`:
```go
//go:build proprietary

package ffi

/*
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/../../../lib/darwin_arm64 -lqoresvm
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/../../../lib/darwin_amd64 -lqoresvm
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/../../../lib/linux_amd64 -lqoresvm
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/../../../lib/linux_arm64 -lqoresvm
#include "bridge.h"
*/
import "C"
```

Implement `FFIExecutor` struct satisfying the `SVMExecutor` interface from `interfaces.go`:
- `NewFFIExecutor() *FFIExecutor` — calls `C.qore_svm_init()`
- `Execute(...)` — serializes accounts, calls `C.qore_svm_execute()`, deserializes results
- `ValidateProgram(...)` — calls `C.qore_svm_validate_program()`
- `Close()` — calls `C.qore_svm_free()`

Pattern reference: `x/pqc/ffi/bridge.go`

### Step 3: Create `callback.go`

The `syscall_callback` exported C function that Rust calls back into:
```go
//export goSVMSyscallCallback
func goSVMSyscallCallback(syscallID C.uint32_t, data *C.uint8_t, dataLen C.size_t,
    out *C.uint8_t, outLen *C.size_t) C.int32_t {
    // Dispatch based on syscallID to registered Go handlers
    // Uses a global handler registry set by the keeper during execution
}
```

Global handler pattern: Before each FFI call, the keeper sets a `syscallHandlers` context (using sync.Mutex for thread safety) that maps syscall IDs to Go functions.

### Step 4: Verify proprietary build compiles with FFI

```bash
CGO_ENABLED=1 go build -tags proprietary ./x/svm/ffi/
```

### Step 5: Commit

```bash
git add x/svm/ffi/
git commit -m "feat(svm): add Go↔Rust FFI bridge for BPF executor"
```

---

## Task 11: Proprietary Keeper — Core + Account CRUD

**Files to create:**
- `x/svm/keeper/keeper.go` (`//go:build proprietary`)
- `x/svm/keeper/accounts.go` (`//go:build proprietary`)

### Step 1: Create `keeper.go`

```go
type Keeper struct {
    cdc           codec.Codec
    storeKey      storetypes.StoreKey
    logger        log.Logger
    executor      svm.SVMExecutor  // FFI executor
    pqcKeeper     pqcmod.PQCKeeper
    aiKeeper      aimod.AIKeeper
    crossvmKeeper crossvmmod.CrossVMKeeper
}
```

Constructor: `NewKeeper(...)`. Params getter/setter using KVStore + JSON marshal (follow multilayer pattern).

### Step 2: Create `accounts.go`

Account CRUD operations:
- `GetAccount(ctx, addr) → (*SVMAccount, error)` — read from KVStore
- `SetAccount(ctx, account) error` — write to KVStore + update addr_map
- `DeleteAccount(ctx, addr) error` — delete from KVStore + addr_map
- `GetAccountByCosmosAddr(ctx, cosmosAddr) → (*SVMAccount, error)` — reverse lookup
- `IterateAccounts(ctx, fn)` — iterate all accounts (for queries)

Use `json.Marshal`/`json.Unmarshal` for serialization (consistent with other modules).

### Step 3: Write tests

`x/svm/keeper/keeper_test.go` (`//go:build proprietary`):
- Test account CRUD round-trip
- Test address mapping consistency
- Test params get/set

Run: `CGO_ENABLED=1 go test -tags proprietary ./x/svm/keeper/ -v -count=1`

### Step 4: Commit

---

## Task 12: Proprietary Keeper — Programs, Rent, Executor

**Files to create:**
- `x/svm/keeper/programs.go` (`//go:build proprietary`)
- `x/svm/keeper/rent.go` (`//go:build proprietary`)
- `x/svm/keeper/executor.go` (`//go:build proprietary`)

### Step 1: Create `programs.go`

- `DeployProgram(ctx, deployer, bytecode) → ([32]byte, error)` — validate via FFI, create program account + data account, return address
- `GetProgramMeta(ctx, programAddr) → (*ProgramMeta, error)`
- `SetProgramMeta(ctx, meta) error`

### Step 2: Create `rent.go`

Solana rent model:
- `GetMinimumBalance(dataLen uint64) uint64` — `(dataLen + 128) * lamportsPerByte * rentExemptionMulti` (128 = account header overhead)
- `CollectRent(ctx, addr) error` — deduct rent or garbage-collect if below minimum
- `IsRentExempt(account *SVMAccount) bool`

### Step 3: Create `executor.go`

Orchestrates BPF execution:
- `ExecuteProgram(ctx, programID, instruction, accounts, signers) → (*ExecutionResult, error)`
- Sets up syscall handlers (registers Go callbacks for PQC/AI/CrossVM)
- Loads program bytecode from KVStore
- Serializes input accounts
- Calls FFI executor
- Deserializes modified accounts, writes back to KVStore
- Collects logs and events

### Step 4: Tests + Commit

---

## Task 13: Proprietary Keeper — Genesis, MsgServer, QueryServer

**Files to create:**
- `x/svm/keeper/genesis.go` (`//go:build proprietary`)
- `x/svm/keeper/msg_server.go` (`//go:build proprietary`)
- `x/svm/keeper/query_server.go` (`//go:build proprietary`)

### Step 1: Create `genesis.go`

- `InitGenesis(ctx, gs)` — set params, deploy system programs from genesis, set initial slot
- `ExportGenesis(ctx) → *GenesisState` — export all accounts, programs, params

### Step 2: Create `msg_server.go`

Handle all 4 message types:
- `DeployProgram(ctx, msg)` — validate, call keeper.DeployProgram
- `ExecuteProgram(ctx, msg)` — validate, call keeper.ExecuteProgram
- `CreateAccount(ctx, msg)` — validate, create account with allocated space
- `RegisterSVMPQCKey(ctx, msg)` — validate Ed25519 sig, register PQC key

### Step 3: Create `query_server.go`

Query handlers:
- `Account(ctx, req) → AccountResponse`
- `Program(ctx, req) → ProgramResponse`
- `SimulateTransaction(ctx, req) → SimulateResponse`
- `Params(ctx, req) → ParamsResponse`

### Step 4: Tests + Commit

---

## Task 14: Proprietary Module + Register + Factory

**Goal:** Complete the proprietary module so `go build -tags proprietary` passes.

**Files to create:**
- `x/svm/module.go` (`//go:build proprietary`)
- `x/svm/register.go` (`//go:build proprietary`)

**Files to modify:**
- `app/factory_proprietary.go`

### Step 1: Create `module.go`

Full `AppModule` implementation with `RegisterServices()`, `InitGenesis()`, `ExportGenesis()`. Follow `x/crossvm/module.go` pattern exactly.

### Step 2: Create `register.go`

`keeperAdapter` wrapping `keeper.Keeper` to satisfy `SVMKeeper` interface. `RealNewSVMKeeper(...)`, `RealNewAppModule(...)` constructors.

### Step 3: Modify `app/factory_proprietary.go`

Add SVM real factory init:
```go
NewSVMKeeper = func(...) svmmod.SVMKeeper {
    return svmmod.RealNewSVMKeeper(...)
}
```

### Step 4: Verify both builds

```bash
CGO_ENABLED=1 go build ./cmd/qorechaind/
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/
```

### Step 5: Commit

```bash
git add x/svm/ app/
git commit -m "feat(svm): add proprietary SVM module, keeper, and app wiring"
```

---

## Task 15: SVM Ante Handler + Triple Routing

**Files to create:**
- `x/svm/ante.go` (`//go:build proprietary`)

**Files to modify:**
- `app/ante.go` — Add SVM routing case + HandlerOptions field

### Step 1: Create `x/svm/ante.go`

SVM ante decorators (can be in a single file):
- `SVMSignatureVerifyDecorator` — verify Ed25519 or Dilithium-5 based on account's PQC registration
- `SVMComputeBudgetDecorator` — check compute units against max
- `SVMDeductFeeDecorator` — deduct compute_units × price from sender

### Step 2: Modify `app/ante.go`

Add to `HandlerOptions`:
```go
SVMKeeper svmmod.SVMKeeper
```

Add SVM routing case in the switch:
```go
case "/qorechain.svm.v1.ExtensionOptionsSVMTx":
    anteHandler = newSVMAnteHandler(options)
```

Create `newSVMAnteHandler()` function that chains the SVM decorators.

### Step 3: Verify both builds + Commit

---

## Task 16: CrossVM SVM Extensions

**Files to modify:**
- `x/crossvm/types/cross_vm_message.go` — Add `VMTypeSVM`
- CrossVM keeper (proprietary) — Add SVM as valid target

### Step 1: Add VMTypeSVM

In `x/crossvm/types/cross_vm_message.go`, add:
```go
VMTypeSVM VMType = "svm"
```

Update `Validate()` to accept SVM as source/target.

### Step 2: Update CrossVM keeper

The proprietary crossvm keeper needs to handle SVM targets in `ExecuteSyncCall()` and `ProcessQueue()`. For sync calls targeting SVM, call `SVMKeeper.ExecuteProgram()`. For async messages from EVM/CosmWasm targeting SVM, process in EndBlocker.

NOTE: This requires the crossvm keeper to have access to SVMKeeper. Add it as a dependency or use a callback pattern.

### Step 3: Tests + Commit

---

## Task 17: Solana JSON-RPC Server

**Files to create:**
- `x/svm/rpc/types.go` (no build tag — shared types)
- `x/svm/rpc/server.go` (`//go:build proprietary`)
- `x/svm/rpc/handlers.go` (`//go:build proprietary`)

### Step 1: Create `rpc/types.go`

Solana JSON-RPC request/response types:
- `RPCRequest` (standard JSON-RPC 2.0)
- `GetAccountInfoRequest/Response`
- `SendTransactionRequest/Response`
- `GetBalanceRequest/Response`
- `GetSlotResponse`, `GetEpochInfoResponse`, `GetRecentBlockhashResponse`
- etc.

### Step 2: Create `rpc/server.go`

HTTP server on configurable port (default 8899). Standard JSON-RPC 2.0 dispatch. Method registry pattern. Server starts with `Start()` and stops with `Stop()`.

Wire into the node via start command flag `--svm-rpc.enable` and `--svm-rpc.address` (default `0.0.0.0:8899`).

### Step 3: Create `rpc/handlers.go`

Implement all methods from design doc §8. Each handler reads from the SVM keeper's KVStore or broadcasts Cosmos transactions.

### Step 4: Tests

`x/svm/rpc/rpc_test.go`:
- Test each RPC method with mock keeper
- Test invalid method names
- Test malformed requests

### Step 5: Commit

---

## Task 18: CLI Commands

**Files to create:**
- `x/svm/client/cli/tx.go`
- `x/svm/client/cli/query.go`

### Step 1: Create TX CLI commands

- `qorechaind tx svm deploy-program [bytecode-file]`
- `qorechaind tx svm execute [program-id] [instruction-data-hex] --accounts [addr1:signer:writable,...]`
- `qorechaind tx svm create-account [address] [space] [owner]`
- `qorechaind tx svm register-pqc-key [svm-address] [algorithm-id] [pubkey-hex]`

### Step 2: Create Query CLI commands

- `qorechaind query svm account [address]`
- `qorechaind query svm program [address]`
- `qorechaind query svm params`
- `qorechaind query svm simulate [program-id] [instruction-hex]`

### Step 3: Commit

---

## Task 19: Documentation

**Files to create:**
- `x/svm/README.md`
- `docs/SVM.md`

**Files to modify:**
- `docs/ARCHITECTURE.md` — Update with triple-VM diagram
- `docs/CROSSVM.md` — Add SVM cross-VM paths
- `docs/API_REFERENCE.md` — Add SVM endpoints + JSON-RPC methods
- `docker-compose.yml` — Add SVM RPC port 8899

### Step 1: Create docs

Full documentation covering: architecture, deployment guide, Solana compatibility notes, QoreChain-specific syscalls, address mapping, rent model, JSON-RPC API, cross-VM communication.

### Step 2: Commit

---

## Task 20: Final Verification + Tag

### Step 1: Run all tests

```bash
CGO_ENABLED=1 go test ./x/svm/... -v -count=1
CGO_ENABLED=1 go test -tags proprietary ./x/svm/... -v -count=1
```

### Step 2: Verify both builds

```bash
CGO_ENABLED=1 go build ./cmd/qorechaind/
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/
```

### Step 3: Forbidden term scan

```bash
grep -riE "Cosmos SDK|CometBFT|Tendermint|AWS Bedrock|Claude Haiku|Claude Sonnet|Claude Opus|Anthropic" \
  --include='*.go' --include='*.md' --include='*.rs' --include='*.toml' \
  x/svm/ rust/qoresvm/ docs/SVM.md | grep -vE "github\.com|cosmossdk\.io|cometbft/|/tendermint/|Co-Authored"
```

### Step 4: Copy proprietary files to proprietary repo

```bash
# Copy all proprietary SVM files + Rust crate to proprietary repo
rsync -av --include='*/' --include='*.go' --exclude='*' x/svm/ ../qorechain-proprietary/x/svm/
cp -r rust/qoresvm/ ../qorechain-proprietary/rust/qoresvm/
```

### Step 5: Tag + Push

```bash
# Core repo
git tag -a v0.8.0 -m "v0.8.0: SVM Runtime — triple-VM architecture with Solana program support"
git push origin main --tags

# Proprietary repo
cd ../qorechain-proprietary
git add x/svm/ rust/qoresvm/
git commit -m "feat(svm): add proprietary SVM keeper, FFI bridge, and BPF executor"
git tag -a v0.8.0 -m "v0.8.0: Proprietary SVM executor and keeper"
git push origin main --tags
```

### Step 6: Update internal CHANGELOG

Add v0.8.0 entry to `/Users/liviu/Development/Qore/testnet/CHANGELOG.md`.

---

## Task Dependency Graph

```
Task 1 (types/keys,errors,account) ──┐
Task 2 (types/params,genesis,codec) ──┼─→ Task 3 (interfaces+stubs) → Task 4 (wire app, public build)
                                      │
Task 5 (Rust scaffold) ──────────────┤
Task 6 (Rust memory+account) ────────┼─→ Task 7 (Rust loader) → Task 8 (Rust executor) → Task 9 (SPL)
                                      │                                    │
                                      │                                    ▼
                                      │                          Task 10 (Go FFI bridge)
                                      │                                    │
                                      ▼                                    ▼
                              Task 11 (keeper core) → Task 12 (programs,rent,exec) → Task 13 (genesis,msg,query)
                                                                                            │
                                                                                            ▼
                                                                                    Task 14 (module+register+factory)
                                                                                            │
                                                                            ┌───────────────┼───────────────┐
                                                                            ▼               ▼               ▼
                                                                    Task 15 (ante)  Task 16 (crossvm)  Task 17 (JSON-RPC)
                                                                            │               │               │
                                                                            └───────────────┼───────────────┘
                                                                                            ▼
                                                                                    Task 18 (CLI)
                                                                                            │
                                                                                            ▼
                                                                                    Task 19 (docs)
                                                                                            │
                                                                                            ▼
                                                                                    Task 20 (verify+tag)
```

## Parallelization Notes

Tasks 1-2 (Go types) and Tasks 5-9 (Rust crate) can run in parallel — they have no dependencies on each other. This is the biggest parallelization opportunity:

- **Track A (Go):** Tasks 1 → 2 → 3 → 4
- **Track B (Rust):** Tasks 5 → 6 → 7 → 8 → 9

Both tracks converge at Task 10 (Go FFI bridge) which depends on both the Go interfaces (Task 3) and the Rust dylib (Task 8).
