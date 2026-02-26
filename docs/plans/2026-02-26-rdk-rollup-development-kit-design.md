# QoreChain v1.3.0 — RDK (Rollup Development Kit) Design

**Date:** 2026-02-26
**Version:** v1.3.0
**Status:** Approved
**Depends On:** v1.2.0 (IBC/Bridges), x/multilayer, x/burn, x/rlconsensus

---

## Overview

v1.3.0 adds the **Rollup Development Kit** — a module for deploying application-specific rollups that settle on QoreChain. Supports **optimistic rollups** (fraud proofs), **ZK rollups** (validity proofs), **based rollups** (L1-sequenced), and **sovereign rollups** (self-sequenced). Native and Celestia DA backends. AI-assisted profile selection via x/rlconsensus.

**Architecture decision:** x/rdk **composes** with x/multilayer (delegates layer registration + state anchoring) rather than extending or replacing it.

**DA decision:** Native backend fully functional; Celestia backend stubbed (interface + error) in v1.3.0. Full IBC MsgPayForBlobs integration deferred to v1.4.0.

---

## Section 1: Module Architecture

### Responsibility Split

| Concern | Owner | Notes |
|---------|-------|-------|
| Rollup config CRUD | **x/rdk** | RollupConfig, profiles, lifecycle |
| DA blob routing | **x/rdk** | DARouter with Native + Celestia stub backends |
| AI profile recommendation | **x/rdk** → x/rlconsensus | Advisory, creator can override |
| Rollup creation burn | **x/rdk** → x/burn | New `BurnSourceRollupCreate` |
| Layer registration | **x/rdk** → x/multilayer | Calls `RegisterSidechain` with rollup-specific config |
| State settlement | **x/rdk** → x/multilayer | Calls `AnchorState` with batch state roots |
| Challenge/finalization | **x/rdk** → x/multilayer | Uses existing challenge window |
| QOR bond escrow | **x/rdk** → bank | Lock QOR for rollup creation |

### Directory Structure

```
x/rdk/
├── types/
│   ├── keys.go              # Store key: "rdk"
│   ├── rollup_config.go     # RollupConfig, RollupProfile, SettlementMode, DABackend
│   ├── da_blob.go           # DABlob, DACommitment types
│   ├── settlement.go        # SettlementBatch, ChallengeWindow, ProofConfig
│   ├── genesis.go           # GenesisState
│   ├── errors.go            # 19 sentinel errors
│   ├── events.go            # 11 event types
│   └── msgs.go              # MsgCreateRollup, MsgPauseRollup, etc.
├── interfaces.go            # RDKKeeper interface (no build tag)
├── keeper_stub.go           # //go:build !proprietary
├── module_stub.go           # //go:build !proprietary
├── module.go                # //go:build proprietary — AppModule
├── register.go              # //go:build proprietary — factory + keeperAdapter
├── keeper/                  # //go:build proprietary
│   ├── keeper.go            # State CRUD, lifecycle methods
│   ├── profiles.go          # 4 preset profiles
│   ├── da_router.go         # DARouter: Native (KVStore) + Celestia (stub)
│   ├── lifecycle.go         # Create, Pause, Resume, Stop lifecycle
│   └── genesis.go           # InitGenesis/ExportGenesis
└── client/cli/
    ├── query.go             # CLI query commands
    └── tx.go                # CLI tx commands
```

---

## Section 2: Core Types

### Settlement Modes — All Rollup Paradigms

```go
type SettlementMode string
const (
    SettlementOptimistic SettlementMode = "optimistic"  // Fraud proofs, 7-day challenge window
    SettlementZK         SettlementMode = "zk"          // Validity proofs, instant finality on proof verification
    SettlementBased      SettlementMode = "based"       // L1-sequenced: QoreChain proposers order rollup TXs
    SettlementSovereign  SettlementMode = "sovereign"   // Self-sequenced, no settlement on QoreChain
)
```

### Rollup Paradigm Comparison

| Property | Optimistic | ZK | Based | Sovereign |
|----------|-----------|-----|-------|-----------|
| Sequencer | Dedicated operator | Dedicated operator | QoreChain L1 proposer | Rollup's own set |
| Proof type | Fraud proof (reactive) | Validity proof (proactive) | Inclusion proof | None on QoreChain |
| Challenge window | 7 days (configurable) | 0 (instant on proof) | 0 (L1 finality = rollup finality) | N/A |
| Finality | L1 finality + challenge | L1 finality + proof verification | Same as L1 finality | Rollup's own |
| DA requirement | Required | Required | Required | Optional |
| Censorship resistance | Depends on sequencer | Depends on sequencer | Inherits L1's | Rollup's own |
| Liveness | Sequencer-dependent | Sequencer-dependent | Inherits L1's | Rollup's own |

### Sequencer Configuration

```go
type SequencerMode string
const (
    SequencerDedicated SequencerMode = "dedicated"  // Single operator sequences
    SequencerShared    SequencerMode = "shared"      // Shared sequencer set
    SequencerBased     SequencerMode = "based"       // L1 proposers sequence
)

type SequencerConfig struct {
    Mode              SequencerMode
    SequencerAddress  string          // For dedicated: operator address; empty for based
    SharedSetMinSize  uint32          // For shared: minimum sequencer set
    InclusionDelay    uint64          // For based: blocks before forced inclusion
    PriorityFeeShare  math.LegacyDec // For based: % of priority fees to L1 proposer
}
```

### Proof Configuration

```go
type ProofSystem string
const (
    ProofSystemFraud    ProofSystem = "fraud"     // Optimistic: interactive fraud proofs
    ProofSystemSNARK    ProofSystem = "snark"     // ZK: succinct proofs (Groth16, PLONK)
    ProofSystemSTARK    ProofSystem = "stark"     // ZK: transparent proofs (no trusted setup)
    ProofSystemNone     ProofSystem = "none"      // Based/Sovereign: no proofs needed
)

type ProofConfig struct {
    System              ProofSystem
    VerifierAddress     string    // On-chain verifier contract (ZK)
    ChallengeWindowSec  uint64    // Fraud proof window in seconds (Optimistic)
    ChallengeBond       math.Int  // Bond required to submit challenge
    MaxProofSize        uint64    // Max proof bytes
    RecursionDepth      uint32    // ZK: proof aggregation depth
}
```

### RollupConfig

```go
type RollupConfig struct {
    RollupID        string
    Creator         string
    Profile         RollupProfile
    SettlementMode  SettlementMode
    SequencerConfig SequencerConfig
    DABackend       DABackend
    BlockTimeMs     uint64
    MaxTxPerBlock   uint64
    GasConfig       RollupGasConfig
    VMType          string           // "evm", "cosmwasm", "svm", "custom"
    ProofConfig     ProofConfig
    Status          RollupStatus
    StakeAmount     math.Int
    LayerID         string
    CreatedHeight   int64
    CreatedAt       time.Time
}

type RollupGasConfig struct {
    GasModel     string          // "eip1559", "flat", "standard", "subsidized"
    BaseGasPrice math.LegacyDec
    MaxGasLimit  uint64
}
```

### Preset Profiles

| Profile | BlockTime | MaxTX | Gas | DA | Settlement | Sequencer | Proof |
|---------|-----------|-------|-----|------|-----------|-----------|-------|
| DeFi | 500ms | 10,000 | EIP-1559 | Native | ZK | Dedicated | SNARK |
| Gaming | 200ms | 50,000 | Flat | Native | Based | Based | None |
| NFT | 2,000ms | 5,000 | Standard | Celestia | Optimistic | Dedicated | Fraud |
| Enterprise | 1,000ms | 20,000 | Subsidized | Native | Based | Based | None |

### DA Types

```go
type DABackend string
const (
    DANative   DABackend = "native"    // On-chain KVStore blob storage
    DACelestia DABackend = "celestia"  // IBC to Celestia (stub in v1.3.0)
    DABoth     DABackend = "both"      // Native + Celestia redundancy
)

type DABlob struct {
    RollupID    string
    BlobIndex   uint64
    Data        []byte
    Commitment  []byte
    Height      int64
    Namespace   []byte
    StoredAt    time.Time
    Pruned      bool
}

type DACommitment struct {
    RollupID   string
    BlobIndex  uint64
    Backend    DABackend
    Hash       []byte
    Size       uint64
    Confirmed  bool
}
```

### Settlement Types

```go
type BatchStatus string
const (
    BatchSubmitted  BatchStatus = "submitted"
    BatchChallenged BatchStatus = "challenged"
    BatchFinalized  BatchStatus = "finalized"
    BatchRejected   BatchStatus = "rejected"
)

type SettlementBatch struct {
    RollupID       string
    BatchIndex     uint64
    StateRoot      []byte
    PrevStateRoot  []byte
    TxCount        uint64
    DataHash       []byte
    ProofType      ProofSystem
    Proof          []byte
    SequencerMode  SequencerMode
    L1BlockRange   [2]int64      // For based rollups: L1 block range
    SubmittedAt    int64
    FinalizedAt    int64
    Status         BatchStatus
}
```

### Module Params

```go
type Params struct {
    MaxRollups              uint32         // 100
    MinStakeForRollup       math.Int       // 10,000 QOR
    RollupCreationBurnRate  math.LegacyDec // 0.01 (1%)
    DefaultChallengeWindow  time.Duration  // 7 days
    MaxDABlobSize           uint64         // 2MB
    BlobRetentionBlocks     uint64         // ~30 days
    MaxBatchesPerBlock      uint32         // 10
}
```

---

## Section 3: Keeper Interface & Cross-Module Wiring

### RDKKeeper Interface

```go
type RDKKeeper interface {
    // Rollup Lifecycle
    CreateRollup(ctx sdk.Context, config types.RollupConfig) (*types.RollupConfig, error)
    PauseRollup(ctx sdk.Context, rollupID string, reason string) error
    ResumeRollup(ctx sdk.Context, rollupID string) error
    StopRollup(ctx sdk.Context, rollupID string) error
    GetRollup(ctx sdk.Context, rollupID string) (*types.RollupConfig, error)
    ListRollups(ctx sdk.Context) ([]*types.RollupConfig, error)
    ListRollupsByCreator(ctx sdk.Context, creator string) ([]*types.RollupConfig, error)

    // Settlement
    SubmitBatch(ctx sdk.Context, batch types.SettlementBatch) error
    ChallengeBatch(ctx sdk.Context, rollupID string, batchIndex uint64, proof []byte) error
    FinalizeBatch(ctx sdk.Context, rollupID string, batchIndex uint64) error
    GetBatch(ctx sdk.Context, rollupID string, batchIndex uint64) (*types.SettlementBatch, error)
    GetLatestBatch(ctx sdk.Context, rollupID string) (*types.SettlementBatch, error)

    // DA Routing
    SubmitDABlob(ctx sdk.Context, blob types.DABlob) (*types.DACommitment, error)
    GetDABlob(ctx sdk.Context, rollupID string, blobIndex uint64) (*types.DABlob, error)
    PruneExpiredBlobs(ctx sdk.Context) (uint64, error)

    // AI-Assisted Configuration
    SuggestProfile(ctx sdk.Context, useCase string) (*types.RollupProfile, error)
    OptimizeGasConfig(ctx sdk.Context, rollupID string) (*types.RollupGasConfig, error)

    // Params / Genesis
    GetParams(ctx sdk.Context) types.Params
    SetParams(ctx sdk.Context, params types.Params) error
    InitGenesis(ctx sdk.Context, gs types.GenesisState)
    ExportGenesis(ctx sdk.Context) *types.GenesisState

    Logger() log.Logger
}
```

### Cross-Module Dependencies

```
x/rdk
  ├──► x/multilayer (MultilayerKeeper)
  │      CreateRollup → RegisterSidechain (LayerType="rollup")
  │      SubmitBatch  → AnchorState (state root settlement)
  │      GetLatestBatch → GetLatestAnchor
  │      PauseRollup  → UpdateLayerStatus
  ├──► x/burn (BurnKeeper)
  │      CreateRollup → BurnFromSource("rollup_create", stake * burnRate)
  ├──► x/rlconsensus (RLConsensusKeeper)
  │      SuggestProfile   → SuggestRollupProfile (advisory)
  │      OptimizeGasConfig → OptimizeRollupGas (advisory)
  ├──► bank (BankKeeper)
  │      CreateRollup → SendCoinsFromAccountToModule (QOR bond escrow)
  │      StopRollup   → SendCoinsFromModuleToAccount (bond return)
  └──► x/pqc (indirect via ante handler chain)
```

### Changes to Existing Modules

1. **x/burn/types/burn_types.go** — add `BurnSourceRollupCreate = "rollup_create"`
2. **x/multilayer/types/layer.go** — add `LayerTypeRollup LayerType = "rollup"`
3. **x/rlconsensus/interfaces.go** — add 2 advisory methods:
   - `SuggestRollupProfile(ctx, useCase string) (string, error)`
   - `OptimizeRollupGas(ctx, metrics map[string]uint64) (uint64, error)`
4. **x/rlconsensus/keeper_stub.go** — stub the 2 new methods

### Factory Wiring

```go
var (
    NewRDKKeeper func(
        cdc         codec.Codec,
        storeKey    storetypes.StoreKey,
        burnKeeper  burnmod.BurnKeeper,
        multiKeeper multilayermod.MultilayerKeeper,
        rlKeeper    rlconsensusmod.RLConsensusKeeper,
        bankKeeper  bankkeeper.Keeper,
        logger      log.Logger,
    ) rdkmod.RDKKeeper
    NewRDKAppModule   func(keeper rdkmod.RDKKeeper) module.AppModule
    NewRDKModuleBasic func() module.AppModuleBasic
)
```

### KVStore Layout

```
Prefix  Key                              Value
0x01    rollup_id                        RollupConfig (JSON)
0x02    rollup_id | "/" | batch_idx(8B)  SettlementBatch (JSON)
0x03    rollup_id                        SettlementBatch (latest)
0x04    rollup_id | "/" | blob_idx(8B)   DABlob (JSON)
0x05    rollup_id                        DACommitment (latest)
0x06    (singleton)                      Params (JSON)
```

### BeginBlocker / EndBlocker

**BeginBlocker:** No-op in v1.3.0.

**EndBlocker:**
1. Auto-finalize batches past challenge window (optimistic)
2. Auto-finalize based rollup batches when L1 blocks finalize
3. Prune expired DA blobs past retention period
4. Emit rollup status events

---

## Section 4: RPC Endpoints & CLI

### New `qor_` JSON-RPC Methods (5)

| Method | Params | Description |
|--------|--------|-------------|
| `qor_getRollupStatus` | `rollupID` | Full rollup dashboard |
| `qor_listRollups` | `creator?` | List all or by creator |
| `qor_getSettlementBatch` | `rollupID`, `batchIndex?` | Specific or latest batch |
| `qor_suggestRollupProfile` | `useCase` | AI-assisted profile selection |
| `qor_getDABlobStatus` | `rollupID`, `blobIndex?` | DA blob storage status |

### CLI Query Commands

```bash
qorechaind query rdk rollup <rollup-id>
qorechaind query rdk list-rollups [--creator <address>]
qorechaind query rdk batch <rollup-id> [--index <batch-index>]
qorechaind query rdk latest-batch <rollup-id>
qorechaind query rdk da-blob <rollup-id> [--index <blob-index>]
qorechaind query rdk config
qorechaind query rdk suggest-profile <use-case>
```

### CLI Transaction Commands

```bash
qorechaind tx rdk create-rollup <profile> [--settlement <mode>] [--sequencer <mode>] \
  [--da <backend>] [--vm <type>] [--stake <amount>] --from mykey
qorechaind tx rdk pause-rollup <rollup-id> --reason "maintenance" --from creator
qorechaind tx rdk resume-rollup <rollup-id> --from creator
qorechaind tx rdk stop-rollup <rollup-id> --from creator
qorechaind tx rdk submit-batch <rollup-id> <state-root-hex> <data-hash-hex> \
  [--proof <proof-hex>] [--proof-type <type>] --from sequencer
qorechaind tx rdk challenge-batch <rollup-id> <batch-index> <fraud-proof-hex> --from challenger
qorechaind tx rdk submit-blob <rollup-id> <data-file> [--backend <backend>] --from sequencer
```

### Sentinel Errors (19)

```go
ErrRollupNotFound, ErrRollupAlreadyExists, ErrRollupNotActive, ErrMaxRollupsReached,
ErrInsufficientStake, ErrUnauthorized, ErrBatchNotFound, ErrBatchAlreadyFinalized,
ErrChallengeWindowClosed, ErrInvalidProof, ErrProofRequired, ErrDABlobTooLarge,
ErrDABlobNotFound, ErrCelestiaDAStubed, ErrInvalidSettlementMode, ErrInvalidSequencerMode,
ErrInvalidProofSystem, ErrBasedSequencerOnly, ErrChallengeBondRequired
```

### Validation Rules

| Rule | Enforced At |
|------|-------------|
| Based settlement requires based sequencer | CreateRollup |
| ZK settlement requires SNARK or STARK proof system | CreateRollup |
| Optimistic settlement requires fraud proof system | CreateRollup |
| Sovereign settlement requires none proof system | CreateRollup |
| Only creator can pause/resume/stop | All lifecycle methods |
| Challenge bond must be posted | ChallengeBatch |
| Batch must be within challenge window | ChallengeBatch |
| Proof required for ZK batch submission | SubmitBatch |
| DA blob under MaxDABlobSize | SubmitDABlob |
| Rollup must be active for submissions | SubmitBatch, SubmitDABlob |

---

## Section 5: Testing Strategy

### Unit Tests (8 files)

| File | Tests |
|------|-------|
| `x/rdk/types/rollup_config_test.go` | Defaults, profile presets, settlement↔sequencer↔proof compatibility |
| `x/rdk/types/settlement_test.go` | BatchStatus transitions, ProofConfig validation |
| `x/rdk/types/da_blob_test.go` | DABlob, DACommitment, size limits |
| `x/rdk/types/genesis_test.go` | GenesisState Validate(), default genesis, round-trip |
| `x/rdk/types/params_test.go` | Defaults, validation bounds |
| `x/rdk/keeper/profiles_test.go` | 4 presets return correct configs, custom override |
| `x/rdk/keeper/da_router_test.go` | Native store/retrieve, Celestia stub error, pruning |
| `x/rdk/keeper/lifecycle_test.go` | Create→Active→Pause→Resume→Stop state machine |

### File Count

| Category | Files | Build Tag |
|----------|-------|-----------|
| Types | 8 | none |
| Module/interface | 2 | none |
| Stub keeper + module | 2 | !proprietary |
| Proprietary module + register | 2 | proprietary |
| Proprietary keeper | 4 | proprietary |
| CLI | 2 | none |
| Tests | 8 | none |
| **Total new** | **28** | |
| **Existing modified** | **8** | |

### Settlement Flow Per Mode

**Optimistic:** Sequencer submits batch → 7-day window → finalized (or challenged → fraud proof → accepted/rejected)

**ZK:** Prover submits batch + validity proof → on-chain verification → immediately finalized (or rejected)

**Based:** L1 proposer includes rollup TXs → batch auto-constructed from L1 block range → finalized when L1 finalizes

**Sovereign:** Rollup sequences independently → optionally posts DA blobs → no settlement on QoreChain
