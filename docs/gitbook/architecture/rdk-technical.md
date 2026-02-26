# Rollup Development Kit

The `x/rdk` module provides a comprehensive Rollup Development Kit (RDK) that allows developers to deploy application-specific rollups on QoreChain. It supports four settlement paradigms, multiple sequencer modes, pluggable data availability backends, and AI-assisted configuration optimization.

---

## Settlement Paradigms

QoreChain RDK supports four distinct settlement modes, each with different trust assumptions, finality characteristics, and proof requirements.

### 1. Optimistic Settlement

Optimistic rollups assume transactions are valid by default and rely on fraud proofs for dispute resolution.

- **Proof system**: Interactive fraud proofs
- **Challenge window**: 7 days (604,800 seconds), configurable per rollup
- **Challenge bond**: 1,000 QOR (1,000,000,000 uqor) -- required to submit a fraud proof challenge
- **Finality**: Delayed until the challenge window expires with no valid challenge
- **Auto-finalization**: The `EndBlocker` automatically finalizes batches once the challenge window has passed without dispute

**Batch lifecycle**:
```
Submitted → [challenge window expires] → Finalized
Submitted → [fraud proof submitted] → Challenged → Rejected
```

### 2. ZK (Zero-Knowledge) Settlement

ZK rollups provide cryptographic validity proofs that guarantee state transition correctness.

- **Proof system**: SNARK (Groth16, PLONK) or STARK (transparent, no trusted setup)
- **Finality**: Instant on proof verification -- no challenge window required
- **Max proof size**: 1 MB (1,048,576 bytes)
- **Recursion depth**: Configurable proof aggregation depth (default: 1)
- **v1.3.0 note**: Uses stub verification that accepts any non-empty proof. Full SNARK/STARK verification is a planned upgrade.

**Batch lifecycle**:
```
Submitted + valid proof → Finalized (instant)
```

### 3. Based Settlement

Based rollups delegate transaction sequencing to L1 (QoreChain) proposers, inheriting the host chain's liveness and censorship resistance guarantees.

- **Proof system**: None required -- L1 proposers are the source of truth
- **Sequencer**: Must use `based` sequencer mode (enforced by validation)
- **Finality**: 2-block confirmation on QoreChain
- **Inclusion delay**: Configurable blocks before forced inclusion of rollup transactions
- **Priority fee sharing**: Configurable percentage of priority fees paid to L1 proposers

**Batch lifecycle**:
```
Submitted → [2 L1 blocks] → Finalized (auto)
```

### 4. Sovereign Settlement

Sovereign rollups operate with independent consensus and self-sequence their transactions. They anchor state to QoreChain for verifiability but do not depend on the host chain for finality.

- **Proof system**: None
- **Finality**: Independent -- determined by the rollup's own consensus
- **State anchoring**: State roots are posted to QoreChain for transparency and verifiability, but are not enforced
- **Auto-finalization**: None -- sovereign rollups manage their own finality

---

## Proof System Compatibility

| Settlement Mode | Fraud Proofs | SNARK | STARK | None |
|---|---|---|---|---|
| **Optimistic** | Required | -- | -- | -- |
| **ZK** | -- | Supported | Supported | -- |
| **Based** | -- | -- | -- | Required |
| **Sovereign** | -- | -- | -- | Required |

---

## Preset Profiles

Four preset profiles provide turnkey rollup configurations optimized for common use cases:

| Profile | Settlement | Sequencer | DA Backend | Block Time | Gas Model | VM | Max TX/Block |
|---|---|---|---|---|---|---|---|
| **DeFi** | ZK (SNARK) | Dedicated | Native | 500ms | EIP-1559 | EVM | 10,000 |
| **Gaming** | Based | Based | Native | 200ms | Flat | Custom | 50,000 |
| **NFT** | Optimistic | Dedicated | Celestia | 2,000ms | Standard | CosmWasm | 5,000 |
| **Enterprise** | Based | Based | Native | 1,000ms | Subsidized | EVM | 20,000 |

Developers can also select the **Custom** profile and configure every parameter individually.

---

## Sequencer Modes

The sequencer determines who orders transactions within a rollup block.

### Dedicated Sequencer

A single operator sequences all rollup transactions.

- **Operator**: Single designated address
- **Latency**: Lowest possible -- single-party ordering
- **Trust**: Requires trust in the sequencer operator for liveness and fair ordering

### Shared Sequencer

A set of sequencers collectively order transactions.

- **Minimum set size**: Configurable (default: 1)
- **Latency**: Slightly higher due to multi-party coordination
- **Trust**: Distributed across the sequencer set

### Based Sequencer

QoreChain L1 proposers sequence rollup transactions.

- **Inclusion delay**: Configurable blocks before forced inclusion (default: 10)
- **Priority fee share**: Configurable percentage of priority fees paid to L1 proposers (e.g., 50% for Gaming, 30% for Enterprise)
- **Trust**: Inherits QoreChain's validator set security and censorship resistance
- **Requirement**: Based settlement mode requires based sequencer (enforced at validation)

---

## Data Availability Backends

### Native DA

On-chain KV-store blob storage within QoreChain itself.

| Parameter | Value |
|---|---|
| **Max blob size** | 2 MB (2,097,152 bytes) |
| **Retention period** | 432,000 blocks (~30 days at 6-second blocks) |
| **Auto-pruning** | Expired blobs are pruned in the `EndBlocker` -- data is removed but commitment metadata is retained |
| **Commitment** | SHA-256 hash of blob data |

### Celestia DA

IBC-based data availability using Celestia's dedicated DA layer.

- **Status in v1.3.0**: Stubbed -- returns an error if selected as the sole backend
- **Namespace support**: Rollup-specific namespaces are supported in the blob schema
- **Planned**: Full IBC integration with Celestia's blob submission and verification

### Both (Redundant)

Stores blobs on both Native and Celestia backends simultaneously.

- In v1.3.0, only the native blob is actually stored; a warning is logged for the Celestia component.

---

## Rollup Lifecycle

```
Pending → Active → Paused → Active → Stopped
                      ↑                  |
                      └──────────────────┘
                      (can resume from paused,
                       stopped is permanent)
```

| State | Description |
|---|---|
| **Pending** | Rollup registered but not yet activated |
| **Active** | Rollup is live and processing batches |
| **Paused** | Temporarily halted by creator (can resume) |
| **Stopped** | Permanently decommissioned -- stake bond returned to creator |

On creation, rollup status is set to `Active` immediately after stake escrow and layer registration succeed.

---

## Batch Lifecycle

Settlement batches track the state progression of rollup state roots:

```
Submitted → Finalized                              (happy path)
Submitted → Challenged → Rejected                  (fraud detected)
```

| State | Description |
|---|---|
| **Submitted** | Batch posted to QoreChain, awaiting finalization |
| **Challenged** | Fraud proof challenge submitted (optimistic only) |
| **Finalized** | Batch accepted as canonical |
| **Rejected** | Batch invalidated by successful challenge |

### Auto-Finalization Rules

| Settlement Mode | Finalization Trigger |
|---|---|
| **Optimistic** | Challenge window expires (~7 days) with no valid challenge |
| **ZK** | Instant on valid proof submission |
| **Based** | 2 L1 blocks after submission |
| **Sovereign** | None -- managed by the rollup's own consensus |

Auto-finalization is executed in the `EndBlocker` for optimistic and based rollups. ZK batches are finalized inline during batch submission.

---

## Module Parameters

| Parameter | Default | Description |
|---|---|---|
| `max_rollups` | 100 | Maximum number of rollups that can be registered |
| `min_stake_for_rollup` | 10,000,000,000 uqor (10,000 QOR) | Minimum stake required to create a rollup |
| `rollup_creation_burn_rate` | 0.01 (1%) | Fraction of creation stake burned via `x/burn` |
| `default_challenge_window` | 604,800 seconds (7 days) | Default optimistic challenge window |
| `max_da_blob_size` | 2,097,152 bytes (2 MB) | Maximum data availability blob size |
| `blob_retention_blocks` | 432,000 (~30 days) | Blocks before DA blobs are pruned |
| `max_batches_per_block` | 10 | Maximum settlement batches processed per block |

---

## Multilayer Integration

The RDK module integrates with `x/multilayer` for cross-layer state management:

### Layer Registration

When a rollup is created, it is automatically registered as a sidechain layer via `RegisterSidechain`. The registration includes:

- Layer ID (matches rollup ID)
- Target block time and max transactions per block
- Supported VM types and domains
- Settlement interval

Registration is **non-fatal**: if `x/multilayer` registration fails, the rollup is still created and a warning is logged.

### State Anchoring

Every settlement batch submitted to the RDK is anchored to `x/multilayer` via `AnchorState`. This records:

- Layer ID and layer height (batch index)
- State root
- Transaction count

Anchoring is **non-fatal**: failures are logged but do not prevent batch processing.

---

## Burn Integration

On rollup creation, **1% of the stake amount** is burned via the `x/burn` module through the `rollup_create` burn channel. For example, creating a rollup with the minimum 10,000 QOR stake burns 100 QOR permanently. The remaining 9,900 QOR is held in escrow and returned when the rollup is stopped.
