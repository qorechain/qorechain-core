# QoreChain Implementation Roadmap Design — v1.0.0 through v1.3.0

**Date:** 2026-02-25
**Target Versions:** v1.0.0, v1.1.0, v1.2.0, v1.3.0
**Status:** Approved
**Approach:** Critical-Path-First (Approach A)

## Overview

Four versioned releases covering 5 interconnected systems from the whitepaper: Tokenomics (Section 7), PQC Hybrid Signatures (Section 3), IBC/Bridges (Section 6), AI Interfaces (Section 5), and RDK Rollups (Section 1). Ordered by dependency chain — each release unlocks the next.

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Release order | Tokenomics → PQC → IBC → RDK | Tokenomics unblocks NilTokenomicsKeeper stubs in rlconsensus/qca; PQC hybrid needed before IBC relayer compat; IBC needed for DA routing |
| Approach | Critical-Path-First (A) | Resolves blocking stubs first, each release is independently shippable |
| IBC scope | On-chain code + configs | Build integration modules, lane configs, relayer configs, test with mocks |
| RL status | v0.9.0 complete | x/rlconsensus done, plan next versions fresh |
| Bridge count | 25 direct connections | 17 non-IBC bridges + 8 IBC connections |

---

## Section 1: Release Map

```
v1.0.0 Tokenomics ──┐
                     ├──→ v1.2.0 IBC/Bridges ──→ v1.3.0 RDK
v1.1.0 PQC Hybrid ──┘
```

v1.0.0 and v1.1.0 can proceed in parallel if needed — no cross-dependency between them. v1.2.0 depends on both (bridge fees need burn module; IBC relayer needs hybrid sig compat). v1.3.0 depends on v1.2.0 (Celestia DA routing needs IBC channel).

| Release | New Modules | Estimated Files | Key Deliverables |
|---------|-------------|-----------------|------------------|
| v1.0.0 | x/burn, x/xqore, x/inflation | ~45 | 9 burn mechanisms, lock/exit/rebase, epoch inflation |
| v1.1.0 | x/pqc extensions, AI interfaces | ~25 | Hybrid Ed25519+ML-DSA-87, SHAKE-256 merkle, TEE/FL specs |
| v1.2.0 | IBC, Skip lanes, 7 new bridges, x/babylon, x/abstractaccount | ~60 | 25 direct chains, 5-lane Block SDK, gas abstraction |
| v1.3.0 | x/rdk, x/multilayer extensions | ~35 | Rollup profiles, settlement, native DA, Celestia prep |

---

## Section 2: v1.0.0 — Tokenomics

### 2.1 x/burn Module

Central burn accounting. All QOR burns route through this module for unified tracking.

**9 Burn Mechanisms:**

| Source | Rate | Trigger |
|--------|------|---------|
| Gas fee | 30% of fee | Every TX (EndBlocker splits FeeCollector) |
| Contract create | Flat fee (configurable) | MsgCreateContract |
| AI service | 50% of service fee | AI sidecar inference calls |
| Bridge fee | 100% of bridge fee | Cross-chain transfers |
| Treasury buyback | Governance-set schedule | Periodic buyback-and-burn |
| Failed TX | Partial gas burn | TX execution failure |
| xQORE exit penalty | 50% of penalty | Early xQORE unlock |
| Auto buyback | Algorithmic | Surplus protocol revenue |
| TGE burn | One-time | Token generation event |

**Fee Distribution Flow (EndBlocker):**
```
FeeCollector balance each block:
  40% → Validators (staking rewards)
  30% → Burned (via x/burn)
  20% → Treasury (community pool)
  10% → Stakers (additional delegation rewards)
```

**Module structure:** Factory pattern (proprietary/stub). Types shared, keeper proprietary.

```
x/burn/
├── interfaces.go          # BurnKeeper interface
├── module.go              # AppModuleBasic
├── module_proprietary.go  # //go:build proprietary
├── module_stub.go         # //go:build !proprietary
├── register.go            # //go:build proprietary — factory + adapter
├── keeper_stub.go         # //go:build !proprietary
├── keeper/                # //go:build proprietary
│   ├── keeper.go          # BurnFromSource, GetBurnStats, GetTotalBurned
│   ├── msg_server.go
│   ├── query_server.go
│   ├── fee_splitter.go    # EndBlocker fee distribution logic
│   └── genesis.go
├── types/
│   ├── keys.go            # Store key: "burn"
│   ├── burn_source.go     # BurnSource enum (9 constants)
│   ├── burn_record.go     # BurnRecord, BurnStats
│   ├── params.go          # GasBurnRate, ContractCreateFee, etc.
│   ├── msgs.go
│   ├── errors.go
│   ├── codec.go
│   └── genesis.go
└── client/cli/
```

**Key types:**
```go
type BurnSource string  // "gas_fee", "contract_create", "ai_service", etc.

type BurnRecord struct {
    Source    BurnSource
    Amount   math.Int
    Height   int64
    TxHash   string
}

type BurnParams struct {
    GasBurnRate       math.LegacyDec  // 0.30
    ContractCreateFee math.Int         // flat QOR amount
    AIServiceBurnRate math.LegacyDec  // 0.50
    BridgeBurnRate    math.LegacyDec  // 1.00
    FailedTxBurnRate  math.LegacyDec
    ValidatorShare    math.LegacyDec  // 0.40
    TreasuryShare     math.LegacyDec  // 0.20
    StakerShare       math.LegacyDec  // 0.10
}
```

### 2.2 x/xqore Module

Governance-boosted staking token. Lock QORE to mint xQORE (1:1). Exit with graduated penalty.

**Exit Penalty Schedule:**

| Duration Locked | Penalty |
|-----------------|---------|
| Immediate exit | 50% |
| After 1 month | 35% |
| After 3 months | 15% |
| After 6 months | 0% |

**Penalty distribution:** 50% burned (via x/burn, source "xqore_penalty"), 50% redistributed to remaining xQORE holders (PvP rebase mechanism).

**Governance:** xQORE holders get 2x voting power multiplier.

**Module structure:** Same factory pattern as x/burn.

```
x/xqore/
├── interfaces.go          # XQOREKeeper interface
├── module.go
├── module_proprietary.go
├── module_stub.go
├── register.go
├── keeper_stub.go
├── keeper/                # //go:build proprietary
│   ├── keeper.go          # Lock, Unlock, GetPosition, GetXQOREBalance
│   ├── msg_server.go      # MsgLock, MsgUnlock
│   ├── query_server.go
│   ├── rebase.go          # PvP rebase calculation
│   └── genesis.go
├── types/
│   ├── keys.go
│   ├── position.go        # XQOREPosition
│   ├── params.go          # XQOREParams with PenaltyTier[]
│   ├── msgs.go
│   ├── errors.go
│   ├── codec.go
│   └── genesis.go
└── client/cli/
```

**Key types:**
```go
type XQOREPosition struct {
    Owner      string
    Locked     math.Int    // QORE locked
    XBalance   math.Int    // xQORE minted
    LockHeight int64
    LockTime   time.Time
}

type PenaltyTier struct {
    MinDuration time.Duration
    PenaltyRate math.LegacyDec
}

type XQOREParams struct {
    GovernanceMultiplier math.LegacyDec     // 2.0
    MinLockAmount        math.Int
    ExitPenaltySchedule  []PenaltyTier
    PenaltyBurnRate      math.LegacyDec     // 0.50
    RebaseInterval       int64              // blocks between rebases
}
```

### 2.3 x/inflation Module

Epoch-based emission decay. Mints new QOR per epoch and sends to fee_collector for staking distribution.

**Emission Schedule:**

| Year | Inflation Rate |
|------|---------------|
| 1 | 15-20% |
| 2 | 10-12% |
| 3-4 | 6-8% |
| 5+ | 1-3% |

Module structure follows factory pattern. Keeper tracks current epoch, calculates mint amount, executes mint in BeginBlocker.

### 2.4 Cross-Module Wiring

**Replace NilTokenomicsKeeper:** Both x/rlconsensus and x/qca currently use `NilTokenomicsKeeper` (returns `math.ZeroInt()`). Replace with real `XQOREKeeper` adapter that satisfies the `TokenomicsKeeper` interface.

**Bridge fee to burn:** Modify existing x/bridge keeper to call `burnKeeper.BurnFromSource("bridge_fee", amount)`.

**Contract create to burn:** EVM module's contract creation hooks call burn keeper.

**Staking genesis params:**
- Min validator self-delegation: 100,000 QOR
- Min delegator: 10 QOR
- Unbonding period: 21 days
- Max validators: 175
- Min commission: 5%
- Downtime slash: 0.01%
- Double-sign slash: 5%

### 2.5 New RPC Endpoints

```
qor_getBurnStats()                    — Total burned, by source
qor_getXQOREPosition(address)        — Lock details, penalty estimate
qor_getInflationRate()                — Current epoch inflation
qor_getTokenomicsOverview()           — Combined burn + inflation + xQORE stats
```

---

## Section 3: v1.1.0 — PQC Hybrid Signatures

### 3.1 Hybrid Signature Format

Each TX carries two signatures:
- **Ed25519** (classical) — wallet-generated, standard signing
- **ML-DSA-87** (quantum-resistant) — attached as TX extension

TX extension approach (not multi-sig) preserves wallet compatibility. Wallets that don't support PQC simply omit the extension.

```go
type HybridSignatureMode uint8
const (
    HybridDisabled HybridSignatureMode = 0  // Classical only
    HybridOptional HybridSignatureMode = 1  // Default — PQC if registered
    HybridRequired HybridSignatureMode = 2  // Future governance upgrade
)

type PQCHybridSignature struct {
    AlgorithmID  uint32   // ML-DSA-87 = 1
    PQCSignature []byte   // Dilithium-5 signature
    PQCPublicKey []byte   // Optional — for auto-registration
}
```

### 3.2 AnteHandler Update

PQCVerifyDecorator updated flow:
1. Check for `PQCHybridSignature` TX extension
2. If account has registered PQC key → extension **required** (verify both sigs)
3. If no PQC key + extension present with PQCPublicKey → **auto-register** key, verify
4. If no PQC key + no extension → classical only, emit warning event

New proprietary file: `x/pqc/ante_hybrid.go` / stub: `x/pqc/ante_hybrid_stub.go`

### 3.3 Wallet Compatibility

| Wallet | Key Type | PQC Support |
|--------|----------|-------------|
| MetaMask | secp256k1 | EVM path only (no PQC) |
| Keplr | Ed25519 | Hybrid if PQC key registered |
| Phantom | Ed25519 | SVM path, future hybrid |
| QoreChain CLI | Ed25519+ML-DSA-87 | Full hybrid |

### 3.4 SHAKE-256 Merkle Foundation

Preparatory wrapper for future post-quantum IAVL tree replacement:
- Uses `sha3.NewShake256()` with 32-byte output
- Foundation layer only — does not replace IAVL in this version
- New file: `x/pqc/shake256.go` (no build tag — pure Go, shared)

### 3.5 AI Spec Interfaces

Pure Go interfaces for future TEE and Federated Learning integration. No implementation, no build tags — just type definitions:

```go
// x/ai/tee_interface.go
type TEEAttestation struct { ... }
type TEEVerifier interface { ... }

// x/ai/federated_interface.go
type FederatedUpdate struct { ... }
type FederatedCoordinator interface { ... }
```

### 3.6 New RPC Endpoints

```
qor_getPQCKeyStatus(address)          — Check if address has PQC key registered
qor_getHybridSignatureMode()          — Current chain-wide mode setting
```

---

## Section 4: v1.2.0 — IBC / Bridges

### 4.1 Skip Block SDK — 5 Lanes

| Lane | Priority | Purpose |
|------|----------|---------|
| PQC | 100 | PQC-signed TXs get highest priority |
| MEV Auction | 90 | Sealed-bid top-of-block auction |
| AI Flagged | 80 | TXs flagged as high-value by AI |
| Default | 50 | Normal transactions |
| Free | 10 | Gasless TXs (limited per block) |

RL agent from x/rlconsensus adjusts lane priorities dynamically. Factory pattern: `app/lanes.go` (proprietary, 5 lanes) / `app/lanes_stub.go` (single default lane).

### 4.2 IBC Connections (8 Chains)

Hermes relayer configs in `configs/hermes/chains/`:

| Chain | Config File |
|-------|-------------|
| Cosmos Hub | `cosmoshub.toml` |
| Osmosis | `osmosis.toml` |
| Noble | `noble.toml` |
| Celestia | `celestia.toml` |
| Stride | `stride.toml` |
| Oraichain | `oraichain.toml` |
| Akash | `akash.toml` |
| Babylon | `babylon.toml` |

Plus `configs/hermes/config.toml` (main config) and `configs/skipgo/chain_registry.json` (Skip:Go routing).

### 4.3 Non-IBC Bridges (17 Chains)

**Existing (8 chains, from v0.x):**

| Chain | ChainType |
|-------|-----------|
| Ethereum | `ChainTypeEVM` |
| Solana | `ChainTypeSolana` |
| TON | `ChainTypeTON` |
| BSC | `ChainTypeEVM` |
| Avalanche | `ChainTypeEVM` |
| Polygon | `ChainTypeEVM` |
| Arbitrum | `ChainTypeEVM` |
| Sui | `ChainTypeSui` |

**New (9 chains, v1.2.0):**

| Chain | ChainType | Notes |
|-------|-----------|-------|
| Aptos | `ChainTypeAptos` | Move VM, 32-byte hex addr. Adapt from Sui pattern |
| Optimism | `ChainTypeEVM` | EVM L2, same pattern as Arbitrum |
| Base | `ChainTypeEVM` | EVM L2, same pattern as Arbitrum |
| Bitcoin | `ChainTypeBitcoin` | UTXO model, bech32/base58 addr, 6-block confirmation, SPV verification |
| Near | `ChainTypeNear` | Implicit/named accounts, NEAR RPC |
| Cardano | `ChainTypeCardano` | bech32 addr, eUTXO/Plutus TX model |
| Polkadot | `ChainTypePolkadot` | SS58 addr, relay/para chain routing |
| Tezos | `ChainTypeTezos` | tz1/KT1 addr, Micheline encoding |
| Tron | `ChainTypeTron` | base58check addr, TVM (EVM fork) |

**New ChainType constants:**
```go
ChainTypeAptos    ChainType = "aptos"
ChainTypeBitcoin  ChainType = "bitcoin"
ChainTypeNear     ChainType = "near"
ChainTypeCardano  ChainType = "cardano"
ChainTypePolkadot ChainType = "polkadot"
ChainTypeTezos    ChainType = "tezos"
ChainTypeTron     ChainType = "tron"
```

Each new ChainType gets address validation in `ValidateAddress()` and a `DefaultChainConfig()` entry. Optimism and Base reuse `ChainTypeEVM`.

**Totals: 17 non-IBC + 8 IBC = 25 direct connections, 600+ accessible via IBC relaying.**

**Bitcoin vs Babylon distinction:** Babylon handles BTC restaking (stake BTC to secure QoreChain validators via x/babylon adapter). The direct Bitcoin bridge handles BTC transfers (move BTC value onto QoreChain as wrapped qorBTC). Complementary, not overlapping.

### 4.4 FairBlock tIBE

Encrypted mempool via threshold identity-based encryption. `FairBlockDecorator` in ante chain encrypts eligible TXs. Stub implementation for v1.2.0 — decrypt is a no-op pass-through.

### 4.5 Account & Gas Abstraction

**Account abstraction:** `x/abstractaccount/types/` — AbstractAccount struct with SessionKey support. Allows smart-contract-based accounts.

**Gas abstraction:** `GasAbstractionDecorator` — pay fees in any IBC token. Static 1:1 rate for testnet. Future: oracle-based conversion rates.

### 4.6 Babylon BTC Restaking

`x/babylon/` adapter module:
```go
type BabylonConfig struct {
    Enabled       bool
    BabylonChainID string
    IBCChannelID  string
    RewardShare   math.LegacyDec  // Share of staking rewards for BTC stakers
    MinBTCStake   math.Int
}
```

### 4.7 Bridge Fee to Burn Wiring

Modify existing x/bridge keeper: on every cross-chain transfer, call `burnKeeper.BurnFromSource("bridge_fee", feeAmount)`. Requires x/burn from v1.0.0.

### 4.8 New RPC Endpoints

```
qor_getIBCChannels()                  — List active IBC channels
qor_getBridgeStatus(chainId)          — Bridge status for a chain
qor_getLaneConfig()                   — Current Block SDK lane configuration
qor_estimateCrossChainFee(src, dst)   — Fee estimate for cross-chain transfer
```

---

## Section 5: v1.3.0 — RDK (Rollup Development Kit)

### 5.1 x/rdk Module (PROPRIETARY)

Central orchestrator for application-specific rollups.

```
x/rdk/
├── interfaces.go              # No build tag — RDKKeeper interface
├── module.go                  # No build tag — AppModuleBasic
├── module_proprietary.go      # //go:build proprietary
├── module_stub.go             # //go:build !proprietary
├── register.go                # //go:build proprietary — factory
├── keeper_stub.go             # //go:build !proprietary
├── keeper/                    # //go:build proprietary
│   ├── keeper.go              # State: rollup configs, status
│   ├── msg_server.go          # CreateRollup, UpdateRollup, PauseRollup
│   ├── query_server.go        # GetRollup, ListRollups, GetRollupStatus
│   ├── profiles.go            # Preset profiles (DeFi, gaming, NFT, enterprise)
│   ├── lifecycle.go           # Rollup lifecycle management
│   ├── da_router.go           # Routes data blobs to DA backend
│   └── genesis.go
├── types/
│   ├── keys.go                # Store key: "rdk"
│   ├── rollup_config.go       # RollupConfig, RollupProfile, DAConfig
│   ├── msgs.go
│   ├── params.go              # MaxRollups, MinStakeForRollup
│   ├── errors.go
│   ├── codec.go
│   └── genesis.go
└── client/cli/
```

**RollupConfig type:**
```go
type RollupConfig struct {
    RollupID       string
    Creator        string
    Profile        RollupProfile     // DeFi, Gaming, NFT, Enterprise, Custom
    SettlementMode SettlementMode    // Optimistic, ZK, Sovereign
    DABackend      DABackend         // Native, Celestia, Both
    BlockTime      time.Duration
    MaxTxPerBlock  uint64
    GasConfig      RollupGasConfig
    Status         RollupStatus      // Pending, Active, Paused, Stopped
    StakeAmount    math.Int          // QOR staked for rollup operation
    CreatedHeight  int64
}
```

### 5.2 Preset Profiles

AI-assisted selection — RL agent recommends profile based on declared use case.

| Profile | Block Time | Max TX/Block | Gas Model | DA Default |
|---------|------------|--------------|-----------|------------|
| DeFi | 500ms | 10,000 | EIP-1559 | Native |
| Gaming | 200ms | 50,000 | Flat fee | Native |
| NFT | 2s | 5,000 | Standard | Celestia |
| Enterprise | 1s | 20,000 | Subsidized | Native |

### 5.3 Settlement Layer Extensions (x/multilayer)

Extend existing x/multilayer for rollup settlement:

```go
type SettlementBatch struct {
    RollupID      string
    BatchIndex    uint64
    StateRoot     []byte        // Rollup state root after batch
    TxCount       uint64
    DataHash      []byte        // Hash of DA blob
    Proof         []byte        // Fraud proof (optimistic) or validity proof (ZK)
    SubmittedAt   int64
    FinalizedAt   int64         // 0 until challenge window passes
    Status        BatchStatus   // Submitted, Challenged, Finalized
}

type ChallengeWindow struct {
    Duration      time.Duration // Default 7 days optimistic, 0 for ZK
    BondAmount    math.Int      // Required bond to challenge
}
```

New keeper methods (proprietary): `SubmitBatch`, `ChallengeBatch`, `FinalizeBatch`, `GetRollupState`.

### 5.4 Native DA Layer

QoreChain stores data availability blobs on-chain for rollups choosing Native DA:

```go
type DABlob struct {
    RollupID    string
    BlobIndex   uint64
    Data        []byte          // Raw rollup block data
    Commitment  []byte          // KZG or Merkle commitment
    Height      int64           // QoreChain height where stored
    Namespace   []byte          // Celestia namespace (if routed there)
}
```

DARouter in keeper dispatches blobs to Native (on-chain store) or Celestia (IBC channel from v1.2.0). Blobs pruned after finalization + retention period (default 30 days).

### 5.5 AI-Assisted Rollup Config

RL agent from x/rlconsensus provides advisory recommendations:
- `SuggestProfile(useCase) -> RollupProfile`
- `OptimizeGas(rollupID) -> GasConfig`
- Creator can override. Wired via `RDKKeeper.SetRLKeeper(rlKeeper)`.

### 5.6 Cross-Module Dependencies

| Dependency | Source | Target | Wiring |
|------------|--------|--------|--------|
| Rollup staking | x/rdk | bank | Escrow QOR for rollup creation |
| Settlement | x/rdk | x/multilayer | SubmitBatch, FinalizeBatch |
| DA routing | x/rdk | IBC (Celestia) | CelestiaDA sends blobs via IBC |
| Burn on create | x/rdk | x/burn | BurnFromSource("contract_create") |
| AI recommendations | x/rdk | x/rlconsensus | SuggestProfile, OptimizeGas |
| PQC for batches | settlement TXs | x/pqc | Hybrid sigs on settlement batches |

### 5.7 New RPC Endpoints

```
qor_createRollup(config)              — Deploy a new rollup
qor_getRollupStatus(rollupID)         — Current rollup state
qor_listRollups(creator?)             — List rollups with filters
qor_getSettlementBatch(batchID)       — Query batch status
qor_suggestRollupProfile(useCase)     — AI-recommended profile
```

---

## Cross-Cutting Patterns

### Open-Core Split

All new modules follow the established factory pattern:
- `interfaces.go` — No build tag, defines keeper interface
- `module.go` — No build tag, AppModuleBasic
- `module_proprietary.go` — `//go:build proprietary`, full AppModule
- `module_stub.go` — `//go:build !proprietary`, stub AppModule
- `register.go` — `//go:build proprietary`, factory + keeperAdapter
- `keeper_stub.go` — `//go:build !proprietary`, returns zero values
- `keeper/` directory — `//go:build proprietary` on all files

### Existing Patterns Reused

| Pattern | Source | Reuse In |
|---------|--------|----------|
| Factory function vars | `app/factory.go` | x/burn, x/xqore, x/inflation, x/rdk factories |
| keeperAdapter | `x/bridge/register.go` | All new module register.go files |
| Stub keeper | `x/bridge/keeper_stub.go` | All new module keeper_stub.go files |
| Module interface | `x/bridge/interfaces.go` | All new module interfaces.go files |
| AppModuleBasic registration | `cmd/qorechaind/cmd/root.go:64-76` | New module basics |

### Build Verification

```bash
# Public build
CGO_ENABLED=1 go build ./cmd/qorechaind/

# Proprietary build
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/

# Both must pass at every version boundary
```

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| NilTokenomicsKeeper replacement breaks rlconsensus/qca | Adapter satisfies same interface, returns real values instead of zero |
| Hybrid sig extension breaks existing wallets | HybridOptional mode — extension only checked if PQC key registered |
| 7 new ChainTypes add surface area | Each follows existing DefaultChainConfig pattern, address validation only |
| IBC version conflicts | Pin ibc-go version, test with mock channels |
| DA blob storage growth | Pruning after finalization + retention period |
| EVM + PQC interaction | Classical-only EVM path (modular upgrade later) |

---

## Module Count Progression

| Version | New Modules | Running Total |
|---------|-------------|---------------|
| v0.9.0 (current) | — | ~37 genesis modules |
| v1.0.0 | x/burn, x/xqore, x/inflation | ~40 |
| v1.1.0 | (extensions only) | ~40 |
| v1.2.0 | ibc, transfer, x/babylon, x/abstractaccount, wasm (if needed) | ~44 |
| v1.3.0 | x/rdk | ~45 |
