# RDK (Rollup Development Kit) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add x/rdk module supporting optimistic, ZK, based, and sovereign rollups with native DA, Celestia DA stub, settlement via x/multilayer, and AI-assisted profile selection.

**Architecture:** x/rdk composes with x/multilayer for settlement/layer registration and x/burn for creation burns. Factory pattern (proprietary/stub). 28 new files + 8 existing file modifications. Celestia DA stubbed.

**Tech Stack:** Go 1.26, QoreChain SDK v0.53, JSON-in-KVStore CRUD, open-core build tags

**Design doc:** `docs/plans/2026-02-26-rdk-rollup-development-kit-design.md`

**Build verification (run after every task):**
```bash
CGO_ENABLED=1 go build ./cmd/qorechaind/
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/
```

**Forbidden terms (NEVER use in any file):** Cosmos SDK, CometBFT, Tendermint, Claude, Anthropic, AWS Bedrock, Haiku, Sonnet, Opus, Baron Chain, Ethermint, Evmos

**No Co-Authored-By lines in commits.**

---

### Task 1: Add BurnSourceRollupCreate + LayerTypeRollup to existing modules

**Files:**
- Modify: `x/burn/types/burn_types.go`
- Modify: `x/multilayer/types/layer.go`

**Step 1: Add burn source constant**

In `x/burn/types/burn_types.go`, add after `BurnSourceTGE`:
```go
BurnSourceRollupCreate BurnSource = "rollup_create"
```

And add it to `ValidBurnSources()` return slice.

**Step 2: Add rollup layer type**

In `x/multilayer/types/layer.go`, add after `LayerTypePaychain`:
```go
LayerTypeRollup LayerType = "rollup" // Application-specific rollups (DeFi, Gaming, NFT, Enterprise)
```

**Step 3: Build both targets**

```bash
CGO_ENABLED=1 go build ./cmd/qorechaind/
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/
```

**Step 4: Commit**

```bash
git add x/burn/types/burn_types.go x/multilayer/types/layer.go
git commit -m "feat(burn,multilayer): add BurnSourceRollupCreate and LayerTypeRollup for v1.3.0"
```

---

### Task 2: Add advisory methods to x/rlconsensus interface + stub

**Files:**
- Modify: `x/rlconsensus/interfaces.go`
- Modify: `x/rlconsensus/keeper_stub.go`

**Step 1: Extend RLConsensusKeeper interface**

In `x/rlconsensus/interfaces.go`, add before the `// ABCI hooks` comment:
```go
// v1.3.0 RDK integration — advisory rollup configuration
SuggestRollupProfile(ctx sdk.Context, useCase string) (string, error)
OptimizeRollupGas(ctx sdk.Context, metrics map[string]uint64) (uint64, error)
```

**Step 2: Add stub implementations**

In `x/rlconsensus/keeper_stub.go`, add before the `Logger()` method:
```go
func (k *StubKeeper) SuggestRollupProfile(_ sdk.Context, _ string) (string, error) {
	return "defi", nil
}

func (k *StubKeeper) OptimizeRollupGas(_ sdk.Context, _ map[string]uint64) (uint64, error) {
	return 0, nil
}
```

**Step 3: Build both targets**

**Step 4: Commit**

```bash
git add x/rlconsensus/interfaces.go x/rlconsensus/keeper_stub.go
git commit -m "feat(rlconsensus): add SuggestRollupProfile and OptimizeRollupGas advisory methods"
```

---

### Task 3: Create x/rdk types — keys, rollup config, enums

**Files:**
- Create: `x/rdk/types/keys.go`
- Create: `x/rdk/types/rollup_config.go`

**Step 1: Create types directory and keys.go**

`x/rdk/types/keys.go`:
```go
package types

const (
	ModuleName = "rdk"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	RollupConfigPrefix     = []byte("rdk/rollup/")
	SettlementBatchPrefix  = []byte("rdk/batch/")
	LatestBatchPrefix      = []byte("rdk/lbatch/")
	DABlobPrefix           = []byte("rdk/blob/")
	LatestDAPrefix         = []byte("rdk/lda/")
	ParamsKey              = []byte("rdk/params")
)
```

**Step 2: Create rollup_config.go with all enums and config types**

`x/rdk/types/rollup_config.go` — Contains: `RollupProfile`, `SettlementMode`, `DABackend`, `RollupStatus`, `SequencerMode`, `ProofSystem`, `SequencerConfig`, `ProofConfig`, `RollupGasConfig`, `RollupConfig`. Also contains `DefaultRollupGasConfig()` and `DefaultSequencerConfig()`, `DefaultProofConfig()` functions.

Enums and their values:
- `RollupProfile`: defi, gaming, nft, enterprise, custom
- `SettlementMode`: optimistic, zk, based, sovereign
- `DABackend`: native, celestia, both
- `RollupStatus`: pending, active, paused, stopped
- `SequencerMode`: dedicated, shared, based
- `ProofSystem`: fraud, snark, stark, none

Include `Validate()` method on `RollupConfig` that enforces:
- Based settlement requires based sequencer
- ZK settlement requires snark or stark proof
- Optimistic settlement requires fraud proof
- Sovereign settlement requires none proof
- BlockTimeMs > 0
- MaxTxPerBlock > 0
- StakeAmount must be positive

**Step 3: Build both targets**

**Step 4: Commit**

```bash
git add x/rdk/
git commit -m "feat(rdk): add x/rdk types — keys, rollup config, enums, sequencer and proof config"
```

---

### Task 4: Create x/rdk types — settlement, DA, genesis, errors, events, msgs

**Files:**
- Create: `x/rdk/types/settlement.go`
- Create: `x/rdk/types/da_blob.go`
- Create: `x/rdk/types/genesis.go`
- Create: `x/rdk/types/errors.go`
- Create: `x/rdk/types/events.go`
- Create: `x/rdk/types/msgs.go`

**Step 1: Create settlement.go**

Types: `BatchStatus` (submitted/challenged/finalized/rejected), `SettlementBatch` struct with RollupID, BatchIndex, StateRoot, PrevStateRoot, TxCount, DataHash, ProofType (ProofSystem), Proof, SequencerMode, L1BlockRange [2]int64, SubmittedAt, FinalizedAt, Status.

**Step 2: Create da_blob.go**

Types: `DABlob` (RollupID, BlobIndex, Data, Commitment, Height, Namespace, StoredAt time.Time, Pruned bool), `DACommitment` (RollupID, BlobIndex, Backend DABackend, Hash, Size uint64, Confirmed bool).

**Step 3: Create genesis.go**

`GenesisState` with Params, Rollups []RollupConfig, Batches []SettlementBatch. `DefaultGenesisState()` returns default params + empty slices. `Validate()` checks params and validates each rollup config.

**Step 4: Create errors.go**

19 sentinel errors as specified in the design doc (ErrRollupNotFound through ErrChallengeBondRequired).

**Step 5: Create events.go**

11 event type constants as specified in the design doc.

**Step 6: Create msgs.go**

Empty placeholder file with package declaration — CLI commands will reference types directly.

**Step 7: Build both targets**

**Step 8: Commit**

```bash
git add x/rdk/types/
git commit -m "feat(rdk): add settlement, DA, genesis, errors, events, msgs types"
```

---

### Task 5: Create x/rdk interfaces + stub keeper + stub module

**Files:**
- Create: `x/rdk/interfaces.go`
- Create: `x/rdk/keeper_stub.go`
- Create: `x/rdk/module_stub.go`

**Step 1: Create interfaces.go (no build tag)**

Define `RDKKeeper` interface with all methods from the design: lifecycle (Create/Pause/Resume/Stop/Get/List/ListByCreator), settlement (SubmitBatch/ChallengeBatch/FinalizeBatch/GetBatch/GetLatestBatch), DA (SubmitDABlob/GetDABlob/PruneExpiredBlobs), AI (SuggestProfile/OptimizeGasConfig), params/genesis, Logger.

Pattern: follow `x/babylon/interfaces.go` exactly.

**Step 2: Create keeper_stub.go (build tag: !proprietary)**

`StubKeeper` struct with logger. All methods return zero values/nil/errors. `NewStubKeeper(logger)` constructor.

Pattern: follow `x/babylon/keeper_stub.go` exactly.

**Step 3: Create module_stub.go (build tag: !proprietary)**

`AppModuleBasic` + `AppModule` structs. AppModuleBasic.Name() returns "rdk". DefaultGenesis/ValidateGenesis use types.DefaultGenesisState()/Validate(). AppModule wraps RDKKeeper interface. InitGenesis/ExportGenesis delegate to keeper.

Pattern: follow `x/babylon/module_stub.go` exactly (same interface vars, same structure).

**Step 4: Build both targets**

**Step 5: Commit**

```bash
git add x/rdk/interfaces.go x/rdk/keeper_stub.go x/rdk/module_stub.go
git commit -m "feat(rdk): add RDKKeeper interface, stub keeper, and stub module"
```

---

### Task 6: Create x/rdk proprietary keeper — core CRUD + lifecycle

**Files:**
- Create: `x/rdk/keeper/keeper.go` (build tag: proprietary)
- Create: `x/rdk/keeper/lifecycle.go` (build tag: proprietary)

**Step 1: Create keeper.go**

Struct: `Keeper` with `cdc`, `storeKey`, `burnKeeper` (burn.BurnKeeper), `multilayerKeeper` (multilayer.MultilayerKeeper), `rlKeeper` (rlconsensus.RLConsensusKeeper), `bankKeeper` (bankkeeper.Keeper), `logger`.

`NewKeeper(...)` constructor. Implement: GetRollup, ListRollups, ListRollupsByCreator (iterate prefix), setRollup (JSON marshal to KVStore), GetParams, SetParams, InitGenesis, ExportGenesis.

Pattern: JSON-in-KVStore like `x/babylon/keeper/keeper.go`.

**Step 2: Create lifecycle.go**

Implement: CreateRollup (validate config, check MaxRollups, escrow QOR via bank, burn via BurnFromSource, register layer via multilayer, set status=active), PauseRollup (check creator, update status), ResumeRollup (check creator, update status), StopRollup (check creator, return bond via bank, update status to stopped).

**Step 3: Build proprietary target only**

```bash
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/
```

**Step 4: Commit**

```bash
git add x/rdk/keeper/
git commit -m "feat(rdk): add proprietary keeper with CRUD and lifecycle management"
```

---

### Task 7: Create x/rdk proprietary keeper — profiles + DA router

**Files:**
- Create: `x/rdk/keeper/profiles.go` (build tag: proprietary)
- Create: `x/rdk/keeper/da_router.go` (build tag: proprietary)

**Step 1: Create profiles.go**

`GetPresetProfile(profile RollupProfile) RollupConfig` returning 4 hardcoded profiles per the design table:
- DeFi: 500ms, 10000 TX, EIP-1559, Native DA, ZK, Dedicated, SNARK
- Gaming: 200ms, 50000 TX, Flat, Native DA, Based, Based, None
- NFT: 2000ms, 5000 TX, Standard, Celestia DA, Optimistic, Dedicated, Fraud
- Enterprise: 1000ms, 20000 TX, Subsidized, Native DA, Based, Based, None

Also implement `SuggestProfile` (delegates to rlKeeper.SuggestRollupProfile) and `OptimizeGasConfig` (delegates to rlKeeper.OptimizeRollupGas).

**Step 2: Create da_router.go**

Implement `SubmitDABlob`:
- If backend=native or both: store blob in KVStore, compute Merkle commitment (SHA-256 hash of data), return DACommitment
- If backend=celestia or both: return `ErrCelestiaDAStubed` for celestia-only; for both, store native + log celestia stub warning

Implement `GetDABlob` (retrieve from KVStore), `PruneExpiredBlobs` (iterate blobs, mark pruned if height + retention < current height).

**Step 3: Build proprietary target**

**Step 4: Commit**

```bash
git add x/rdk/keeper/profiles.go x/rdk/keeper/da_router.go
git commit -m "feat(rdk): add preset profiles and DA router with native backend"
```

---

### Task 8: Create x/rdk proprietary keeper — settlement (optimistic, ZK, based, sovereign)

**Files:**
- Add to: `x/rdk/keeper/keeper.go` or create `x/rdk/keeper/settlement.go` (build tag: proprietary)

If separate file, create `x/rdk/keeper/settlement.go`.

**Step 1: Implement SubmitBatch**

- Validate rollup is active
- Validate proof matches settlement mode (ZK requires proof bytes, optimistic doesn't require proof at submission, based auto-constructs)
- Store SettlementBatch with status=submitted
- Call multilayer AnchorState with state root
- For ZK: if proof present, attempt verification (stub: accept any non-empty proof), auto-finalize on success
- Emit EventBatchSubmitted

**Step 2: Implement ChallengeBatch**

- Validate batch exists and status=submitted (not finalized)
- Validate within challenge window (only for optimistic)
- For ZK/based/sovereign: return ErrChallengeWindowClosed (no challenges)
- Set status=challenged
- Emit EventBatchChallenged

**Step 3: Implement FinalizeBatch**

- Validate batch exists
- For optimistic: check challenge window expired and status != challenged
- Set status=finalized, FinalizedAt=current height
- Emit EventBatchFinalized

**Step 4: Implement GetBatch, GetLatestBatch**

Standard KVStore reads.

**Step 5: Implement EndBlocker settlement auto-finalization**

In keeper.go or lifecycle.go, add `EndBlockSettlement(ctx)`:
- Iterate all active rollups
- For each optimistic rollup: check submitted batches past challenge window → auto-finalize
- For based rollups: finalize when L1 blocks finalize (use current height as proxy)
- Call PruneExpiredBlobs

**Step 6: Build proprietary target**

**Step 7: Commit**

```bash
git add x/rdk/keeper/
git commit -m "feat(rdk): add settlement logic for optimistic, ZK, based, sovereign rollups"
```

---

### Task 9: Create x/rdk proprietary module + register

**Files:**
- Create: `x/rdk/module.go` (build tag: proprietary)
- Create: `x/rdk/register.go` (build tag: proprietary)

**Step 1: Create module.go**

Proprietary `AppModule` using concrete `keeper.Keeper`. EndBlocker calls `keeper.EndBlockSettlement(ctx)`.

Pattern: follow `x/babylon/module.go` but add EndBlocker.

The module needs to implement `appmodule.HasEndBlocker` by adding:
```go
func (am AppModule) EndBlock(ctx sdk.Context) error {
	return am.keeper.EndBlockSettlement(ctx)
}
```

**Step 2: Create register.go**

`keeperAdapter` wrapping `keeper.Keeper` to satisfy `RDKKeeper` interface. All methods delegate to concrete keeper.

`RealNewRDKKeeper(...)` creates concrete keeper + wraps in adapter.
`RealNewAppModule(k RDKKeeper)` unwraps adapter and creates proprietary AppModule.

Pattern: follow `x/babylon/register.go` exactly.

**Step 3: Build both targets**

**Step 4: Commit**

```bash
git add x/rdk/module.go x/rdk/register.go
git commit -m "feat(rdk): add proprietary module with EndBlocker and keeperAdapter"
```

---

### Task 10: Register RDK factories (factory.go, factory_stub.go, factory_proprietary.go)

**Files:**
- Modify: `app/factory.go`
- Modify: `app/factory_stub.go`
- Modify: `app/factory_proprietary.go`

**Step 1: Add factory vars to factory.go**

Add import: `rdkmod "github.com/qorechain/qorechain-core/x/rdk"`

Add after GasAbstraction factory vars:
```go
// RDK module factories (v1.3.0 — Rollup Development Kit)
NewRDKKeeper func(
	cdc          codec.Codec,
	storeKey     storetypes.StoreKey,
	burnKeeper   burnmod.BurnKeeper,
	multiKeeper  multilayermod.MultilayerKeeper,
	rlKeeper     rlconsensusmod.RLConsensusKeeper,
	bankKeeper   bankkeeper.Keeper,
	logger       log.Logger,
) rdkmod.RDKKeeper
NewRDKAppModule   func(keeper rdkmod.RDKKeeper) module.AppModule
NewRDKModuleBasic func() module.AppModuleBasic
```

**Step 2: Add stub assignments to factory_stub.go**

Add import: `rdkmod "github.com/qorechain/qorechain-core/x/rdk"`

Add after GasAbstraction stub block:
```go
// RDK — stub factories
NewRDKKeeper = func(_ codec.Codec, _ storetypes.StoreKey, _ burnmod.BurnKeeper, _ multilayermod.MultilayerKeeper, _ rlconsensusmod.RLConsensusKeeper, _ bankkeeper.Keeper, logger log.Logger) rdkmod.RDKKeeper {
	return rdkmod.NewStubKeeper(logger)
}
NewRDKAppModule = func(keeper rdkmod.RDKKeeper) module.AppModule {
	return rdkmod.NewAppModule(keeper)
}
NewRDKModuleBasic = func() module.AppModuleBasic {
	return rdkmod.AppModuleBasic{}
}
```

**Step 3: Add real assignments to factory_proprietary.go**

Add import: `rdkmod "github.com/qorechain/qorechain-core/x/rdk"`

Add after GasAbstraction real factory block:
```go
// RDK — real factories
NewRDKKeeper = func(cdc codec.Codec, storeKey storetypes.StoreKey, burnKeeper burnmod.BurnKeeper, multiKeeper multilayermod.MultilayerKeeper, rlKeeper rlconsensusmod.RLConsensusKeeper, bankKeeper bankkeeper.Keeper, logger log.Logger) rdkmod.RDKKeeper {
	return rdkmod.RealNewRDKKeeper(cdc, storeKey, burnKeeper, multiKeeper, rlKeeper, bankKeeper, logger)
}
NewRDKAppModule = func(keeper rdkmod.RDKKeeper) module.AppModule {
	return rdkmod.RealNewAppModule(keeper)
}
NewRDKModuleBasic = func() module.AppModuleBasic {
	return rdkmod.AppModuleBasic{}
}
```

**Step 4: Build both targets**

**Step 5: Commit**

```bash
git add app/factory.go app/factory_stub.go app/factory_proprietary.go
git commit -m "feat(factory): register rdk module factories for v1.3.0"
```

---

### Task 11: Wire RDK into app.go + app_config.go + root.go

**Files:**
- Modify: `app/app.go`
- Modify: `app/app_config.go`
- Modify: `cmd/qorechaind/cmd/root.go`

**Step 1: Add RDKKeeper to QoreChainApp struct in app.go**

Add import: `rdkmod "github.com/qorechain/qorechain-core/x/rdk"` and `rdktypes "github.com/qorechain/qorechain-core/x/rdk/types"`

Add keeper field after GasAbstractionKeeper:
```go
RDKKeeper rdkmod.RDKKeeper // v1.3.0 — Rollup Development Kit
```

**Step 2: Initialize RDK keeper in NewQoreChainApp**

After GasAbstraction keeper initialization, add:
```go
// --- Initialize RDK module (via factory, v1.3.0 — Rollup Development Kit) ---
rdkStoreKey := storetypes.NewKVStoreKey(rdktypes.StoreKey)
app.MountStores(rdkStoreKey)

app.RDKKeeper = NewRDKKeeper(
	app.appCodec,
	rdkStoreKey,
	app.BurnKeeper,
	app.MultilayerKeeper,
	app.RLConsensusKeeper,
	app.BankKeeper,
	logger,
)
```

**Step 3: Register in RegisterModules**

Add after NewGasAbstractionAppModule:
```go
NewRDKAppModule(app.RDKKeeper),
```

**Step 4: Update app_config.go**

Add module account permission:
```go
{Account: "rdk", Permissions: []string{authtypes.Minter, authtypes.Burner}},
```

Add "rdk" to BeginBlockers (after "babylon"), EndBlockers (after "babylon", before EVM), InitGenesis (after "gasabstraction"), ExportGenesis (after "gasabstraction").

**Step 5: Update root.go moduleBasicManager**

After gasabstraction basic manager registration, add:
```go
rdkBasic := app.NewRDKModuleBasic()
moduleBasicManager[rdkBasic.Name()] = rdkBasic
```

**Step 6: Build both targets**

**Step 7: Commit**

```bash
git add app/app.go app/app_config.go cmd/qorechaind/cmd/root.go
git commit -m "feat(app): wire x/rdk into app, app_config, and root.go"
```

---

### Task 12: Add RPC endpoints

**Files:**
- Modify: `rpc/qor/api.go`
- Modify: `rpc/qor_stub/api.go`

**Step 1: Add RDKKeeper field + 5 new endpoints to rpc/qor/api.go**

Add keeper field, constructor param. Add methods:
- `GetRollupStatus(rollupID string) (map[string]interface{}, error)` — calls GetRollup + GetLatestBatch
- `ListRollups(creator string) ([]map[string]interface{}, error)`
- `GetSettlementBatch(rollupID string, batchIndex int64) (map[string]interface{}, error)`
- `SuggestRollupProfile(useCase string) (map[string]interface{}, error)`
- `GetDABlobStatus(rollupID string, blobIndex int64) (map[string]interface{}, error)`

**Step 2: Add stubs to rpc/qor_stub/api.go**

5 new methods returning `errNotAvailable`.

**Step 3: Build both targets**

**Step 4: Commit**

```bash
git add rpc/qor/api.go rpc/qor_stub/api.go
git commit -m "feat(rpc): add qor_ RPC endpoints for RDK rollup module"
```

---

### Task 13: Add CLI commands

**Files:**
- Create: `x/rdk/client/cli/query.go`
- Create: `x/rdk/client/cli/tx.go`

**Step 1: Create query.go**

`GetQueryCmd()` returning cobra command with subcommands:
- `CmdQueryRollup` — query rdk rollup <rollup-id>
- `CmdQueryListRollups` — query rdk list-rollups [--creator addr]
- `CmdQueryBatch` — query rdk batch <rollup-id> [--index N]
- `CmdQueryConfig` — query rdk config
- `CmdSuggestProfile` — query rdk suggest-profile <use-case>

**Step 2: Create tx.go**

`GetTxCmd()` returning cobra command with subcommands:
- `CmdCreateRollup` — tx rdk create-rollup <profile> [flags]
- `CmdPauseRollup` — tx rdk pause-rollup <rollup-id>
- `CmdResumeRollup` — tx rdk resume-rollup <rollup-id>
- `CmdStopRollup` — tx rdk stop-rollup <rollup-id>
- `CmdSubmitBatch` — tx rdk submit-batch <rollup-id> <state-root> <data-hash>
- `CmdChallengeBatch` — tx rdk challenge-batch <rollup-id> <batch-index> <proof>

**Step 3: Build both targets**

**Step 4: Commit**

```bash
git add x/rdk/client/
git commit -m "feat(rdk): add CLI query and transaction commands"
```

---

### Task 14: Unit tests

**Files:**
- Create: `x/rdk/types/rollup_config_test.go`
- Create: `x/rdk/types/settlement_test.go`
- Create: `x/rdk/types/da_blob_test.go`
- Create: `x/rdk/types/genesis_test.go`
- Create: `x/rdk/types/params_test.go`
- Create: `x/rdk/keeper/profiles_test.go`
- Create: `x/rdk/keeper/da_router_test.go`
- Create: `x/rdk/keeper/lifecycle_test.go`

**Step 1: Create rollup_config_test.go**

Tests:
- TestDefaultRollupGasConfig — verify defaults
- TestRollupProfileValues — 5 profiles exist
- TestSettlementSequencerProofCompatibility — based+based=ok, based+dedicated=fail, zk+snark=ok, zk+fraud=fail, optimistic+fraud=ok, optimistic+snark=fail, sovereign+none=ok
- TestRollupConfigValidation — valid config passes, zero block time fails, zero stake fails

**Step 2: Create settlement_test.go**

Tests:
- TestBatchStatusValues — 4 statuses
- TestSettlementBatchFields — verify struct fields

**Step 3: Create da_blob_test.go**

Tests:
- TestDABlobStruct — verify DABlob fields
- TestDACommitmentStruct — verify DACommitment fields
- TestDABackendValues — 3 backends

**Step 4: Create genesis_test.go**

Tests:
- TestDefaultGenesisState — defaults are valid
- TestGenesisStateValidation — valid passes, invalid fails

**Step 5: Create params_test.go**

Tests:
- TestDefaultParams — verify all defaults match spec

**Step 6: Create profiles_test.go** (in keeper/ directory, no build tag since it just tests types)

Tests:
- TestGetPresetProfileDeFi — verify DeFi profile config
- TestGetPresetProfileGaming — 200ms block time, based sequencer
- TestGetPresetProfileNFT — optimistic + celestia DA
- TestGetPresetProfileEnterprise — based + subsidized gas

**Step 7: Create da_router_test.go**

Tests:
- TestDANativeBackendStoresBlob — commitment returned
- TestDACelestiaBackendStubbed — returns ErrCelestiaDAStubed

**Step 8: Create lifecycle_test.go**

Tests:
- TestRollupLifecycleStateMachine — create→active→pause→resume→stop transitions

**Step 9: Run all tests**

```bash
CGO_ENABLED=1 go test ./x/rdk/...
```

**Step 10: Commit**

```bash
git add x/rdk/types/*_test.go x/rdk/keeper/*_test.go
git commit -m "test: add unit tests for v1.3.0 RDK module"
```

---

### Task 15: Final verification + CHANGELOG + tag

**Files:**
- Modify: `CHANGELOG.md`

**Step 1: Build both targets**

```bash
CGO_ENABLED=1 go build ./cmd/qorechaind/
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/
```

**Step 2: Run all tests**

```bash
CGO_ENABLED=1 go test ./x/rdk/...
```

**Step 3: Check for forbidden terms**

```bash
grep -rinE "Cosmos SDK|CometBFT|Tendermint|Claude|Anthropic|AWS Bedrock|Haiku|Sonnet|Opus|Baron Chain|Ethermint|Evmos" x/rdk/ || echo "CLEAN"
```

**Step 4: Update CHANGELOG.md**

Add v1.3.0 section at the top with all changes.

**Step 5: Commit and tag**

```bash
git add CHANGELOG.md
git commit -m "docs: v1.3.0 RDK CHANGELOG"
git tag v1.3.0
```

---

## Summary

| Task | Description | New Files | Modified Files |
|------|-------------|-----------|----------------|
| 1 | Burn source + layer type | 0 | 2 |
| 2 | RL consensus advisory methods | 0 | 2 |
| 3 | RDK types: keys + rollup config | 2 | 0 |
| 4 | RDK types: settlement, DA, genesis, errors, events, msgs | 6 | 0 |
| 5 | RDK interface + stub keeper + stub module | 3 | 0 |
| 6 | Proprietary keeper: CRUD + lifecycle | 2 | 0 |
| 7 | Proprietary keeper: profiles + DA router | 2 | 0 |
| 8 | Proprietary keeper: settlement | 1 | 0 |
| 9 | Proprietary module + register | 2 | 0 |
| 10 | Factory registration | 0 | 3 |
| 11 | App wiring | 0 | 3 |
| 12 | RPC endpoints | 0 | 2 |
| 13 | CLI commands | 2 | 0 |
| 14 | Unit tests | 8 | 0 |
| 15 | CHANGELOG + tag | 0 | 1 |
| **Total** | | **28** | **13** |
