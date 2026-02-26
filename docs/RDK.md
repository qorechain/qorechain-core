# QoreChain Rollup Development Kit (RDK)

## Overview

The Rollup Development Kit (RDK) is QoreChain's application-specific rollup deployment framework, introduced in **v1.3.0**. It enables developers to launch purpose-built rollup chains on QoreChain with configurable settlement, sequencing, data availability, and gas models — all secured by the host chain's PQC infrastructure and anchored via the Hierarchical Commitment Schemes (HCS) of the x/multilayer module.

The RDK supports four settlement paradigms, three sequencer modes, three data availability backends, and four proof systems, providing a flexible foundation for diverse application domains from DeFi to gaming to enterprise workloads.

## Settlement Paradigms

### Optimistic Settlement

Batches are submitted with an assumed-valid status and enter a configurable challenge window (default: 7 days / 604,800 seconds). During this window, any party may submit an interactive fraud proof to challenge the batch. If no valid challenge is submitted before the window expires, the batch is auto-finalized by the EndBlocker.

- **Proof system**: Interactive fraud proofs
- **Challenge window**: Configurable (default 604,800 seconds)
- **Challenge bond**: Required to submit a challenge (default 1,000 QOR)
- **Finality**: Delayed until challenge window expires
- **Auto-finalization**: EndBlocker checks each block and finalizes eligible batches

### ZK (Zero-Knowledge) Settlement

Batches include a validity proof (SNARK or STARK) that is verified on submission. If a non-empty proof is present, the batch is immediately finalized — no challenge window needed. This provides the fastest finality of any settlement mode.

- **Proof systems**: SNARK (Groth16, PLONK) or STARK (transparent, no trusted setup)
- **Recursive aggregation**: Configurable recursion depth for proof composition
- **Max proof size**: Configurable (default 1 MB)
- **Finality**: Instant upon proof verification
- **Note**: v1.3.0 uses stub verification (any non-empty proof accepted); full verifier integration is planned

### Based Settlement

The host chain's block proposers directly sequence rollup transactions. This is the most decentralized sequencing model, as it inherits the full validator set of the host chain. Based rollups finalize after a short confirmation delay (2 blocks) since L1 finality equals rollup finality.

- **Sequencer mode**: Must be `based` (enforced by validation)
- **Inclusion delay**: Configurable blocks before forced inclusion
- **Priority fee sharing**: Configurable percentage to L1 proposers
- **Proof system**: None required
- **Finality**: 2 blocks after submission (L1 finality proxy)

### Sovereign Settlement

Self-sequenced rollups that operate independently. No proofs are submitted and no challenge mechanism exists. The rollup anchors state to the host chain for verifiability but settles on its own terms.

- **Sequencer mode**: Any (dedicated or shared)
- **Proof system**: None
- **Finality**: Determined by the rollup's own consensus

## Preset Profiles

The RDK provides five preset profiles that configure optimal defaults for common use cases. Each profile pre-selects settlement mode, sequencer configuration, DA backend, block time, gas model, and VM type.

| Profile | Settlement | Sequencer | DA | Block Time | Gas Model | VM | Max TX/Block |
|---------|-----------|-----------|-----|-----------|-----------|-----|-------------|
| **DeFi** | ZK (SNARK) | Dedicated | Native | 500ms | EIP-1559 | EVM | 10,000 |
| **Gaming** | Based | Based | Native | 200ms | Flat | Custom | 50,000 |
| **NFT** | Optimistic | Dedicated | Celestia | 2,000ms | Standard | CosmWasm | 5,000 |
| **Enterprise** | Based | Based | Native | 1,000ms | Subsidized | EVM | 20,000 |
| **Custom** | Optimistic | Dedicated | Native | 1,000ms | Standard | EVM | 10,000 |

### Profile Details

**DeFi**: Optimized for financial applications requiring fast finality. ZK-SNARK proofs provide instant finality, EIP-1559 dynamic gas pricing handles congestion, and high throughput supports complex DeFi protocols.

**Gaming**: Ultra-low latency for real-time applications. L1-sequenced for decentralization, flat gas fee eliminates fee uncertainty, and the highest throughput ceiling supports game state updates.

**NFT**: Cost-efficient for asset-heavy workloads. Optimistic settlement reduces on-chain costs, Celestia DA offloads storage, and CosmWasm enables rich smart contract logic for NFT marketplaces.

**Enterprise**: Permissioned-friendly with zero-cost gas. Based settlement inherits L1 security, subsidized gas removes end-user costs, and moderate throughput suits enterprise workflows.

**Custom**: Full configuration control. Starts with sensible defaults that can be overridden field by field during rollup creation.

### AI-Assisted Profile Selection

The `SuggestProfile` function delegates to the RL consensus module (`x/rlconsensus`) for AI-assisted profile recommendation based on use-case descriptions. If the RL module is unavailable, it falls back to the DeFi profile as the most general-purpose option.

## Sequencer Modes

| Mode | Description | Key Parameters |
|------|-------------|----------------|
| **Dedicated** | Single operator sequences transactions | `sequencer_address` — operator's address |
| **Shared** | Distributed sequencer set | `shared_set_min_size` — minimum set size |
| **Based** | L1 proposers sequence rollup TXs | `inclusion_delay` — blocks before forced inclusion; `priority_fee_share` — % to L1 proposer |

**Validation constraint**: Based settlement mode requires the `based` sequencer mode. This is enforced at rollup creation.

## Proof Systems

| System | Settlement Mode | Description |
|--------|----------------|-------------|
| **Fraud** | Optimistic | Interactive fraud proofs within challenge window |
| **SNARK** | ZK | Succinct proofs (Groth16, PLONK) — trusted setup required |
| **STARK** | ZK | Transparent proofs — no trusted setup, larger proof size |
| **None** | Based, Sovereign | No proofs needed |

**Compatibility matrix** (enforced by `RollupConfig.Validate()`):

| Settlement | Valid Proof Systems |
|-----------|-------------------|
| Optimistic | Fraud only |
| ZK | SNARK or STARK |
| Based | None only |
| Sovereign | None only |

## Data Availability

### Native Backend

On-chain KV-store blob storage within the QoreChain state tree. Blobs are stored with configurable retention and automatically pruned by the EndBlocker.

- **Max blob size**: 2 MB (configurable)
- **Retention**: 432,000 blocks (~30 days at 6-second blocks)
- **Pruning**: Automatic via EndBlocker on each block

### Celestia Backend

IBC-based data availability via Celestia. Stubbed in v1.3.0 — submissions return `ErrCelestiaDAStubed`. Full IBC integration planned for a future release.

### Both Backend

Redundant storage on both native and Celestia backends simultaneously.

## Gas Models

| Model | Description | Use Case |
|-------|-------------|----------|
| **standard** | Fixed base gas price | General purpose |
| **eip1559** | Dynamic base fee with priority tips | DeFi, congestion-sensitive |
| **flat** | Single flat fee per transaction | Gaming, predictable costs |
| **subsidized** | Zero or near-zero gas cost | Enterprise, sponsored chains |

Gas configuration is optimizable via the RL consensus module's `OptimizeRollupGas` advisory function, which analyzes block time and throughput metrics to recommend gas limit adjustments.

## Rollup Lifecycle

Rollups follow a strict state machine:

```
PENDING → ACTIVE → PAUSED → STOPPED
                ↕
            PAUSED
```

### Valid Transitions

| From | To | Action |
|------|-----|--------|
| Pending | Active | Automatic on creation (with sufficient stake) |
| Active | Paused | `PauseRollup` — creator only |
| Paused | Active | `ResumeRollup` — creator only |
| Active | Stopped | `StopRollup` — creator only |
| Paused | Stopped | `StopRollup` — creator only |

- **Pending**: Initial state during validation
- **Active**: Fully operational, accepting batches
- **Paused**: Temporarily suspended, no new batches accepted
- **Stopped**: Permanently decommissioned

### Multilayer Integration

When a rollup is created, it is registered as a `rollup` layer type in the x/multilayer module via `RegisterSidechain`. Each batch submission triggers `AnchorState` to commit the state root to the Main Chain. This is a non-fatal operation — anchor failures are logged but do not block batch submission.

### Burn Integration

Rollup creation burns 1% of the required stake through the x/burn module's `rollup_create` source channel. This is the 10th burn channel in the unified burn accounting system.

## Settlement Batch Lifecycle

```
SUBMITTED → FINALIZED
    ↓
CHALLENGED → REJECTED
```

### Batch States

| State | Description |
|-------|-------------|
| **Submitted** | Batch accepted, awaiting finalization |
| **Challenged** | Fraud proof submitted (optimistic only) |
| **Finalized** | Settlement complete, state committed |
| **Rejected** | Challenge upheld, batch invalidated |

### Auto-Finalization

The EndBlocker runs settlement logic each block:

- **Optimistic batches**: Auto-finalized when `currentBlock - submittedAt >= challengeWindowBlocks`
- **Based batches**: Auto-finalized after 2 blocks (L1 finality proxy)
- **ZK batches**: Immediately finalized on submission if proof is present
- **Sovereign batches**: No auto-finalization by the host chain

## Module Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `max_rollups` | 100 | Maximum number of registered rollups |
| `min_stake_for_rollup` | 10,000,000,000 uqor (10,000 QOR) | Minimum stake to create a rollup |
| `rollup_creation_burn_rate` | 0.01 (1%) | Percentage of stake burned on creation |
| `default_challenge_window` | 604,800 seconds (7 days) | Default optimistic challenge window |
| `max_da_blob_size` | 2,097,152 bytes (2 MB) | Maximum data availability blob size |
| `blob_retention_blocks` | 432,000 (~30 days) | Blocks before expired blobs are pruned |
| `max_batches_per_block` | 10 | Maximum settlement batches per block |

## CLI Commands

### Transaction Commands

```bash
# Create a rollup from a preset profile
qorechaind tx rdk create-rollup \
  --rollup-id "my-defi-rollup" \
  --profile defi \
  --from <key>

# Create a custom rollup
qorechaind tx rdk create-rollup \
  --rollup-id "my-rollup" \
  --profile custom \
  --settlement optimistic \
  --sequencer dedicated \
  --da-backend native \
  --vm-type evm \
  --block-time 1000 \
  --from <key>

# Submit a settlement batch
qorechaind tx rdk submit-batch \
  --rollup-id "my-rollup" \
  --state-root <hex-hash> \
  --tx-count 500 \
  --proof <hex-proof> \
  --from <key>

# Challenge a batch (optimistic only)
qorechaind tx rdk challenge-batch \
  --rollup-id "my-rollup" \
  --batch-index 42 \
  --proof <hex-fraud-proof> \
  --from <key>

# Finalize a batch manually
qorechaind tx rdk finalize-batch \
  --rollup-id "my-rollup" \
  --batch-index 42 \
  --from <key>

# Pause a rollup
qorechaind tx rdk pause-rollup \
  --rollup-id "my-rollup" \
  --from <key>

# Resume a rollup
qorechaind tx rdk resume-rollup \
  --rollup-id "my-rollup" \
  --from <key>

# Stop a rollup (permanent)
qorechaind tx rdk stop-rollup \
  --rollup-id "my-rollup" \
  --from <key>
```

### Query Commands

```bash
# Query a specific rollup
qorechaind query rdk rollup <rollup-id>

# List all rollups
qorechaind query rdk rollups

# Query a settlement batch
qorechaind query rdk batch <rollup-id> <batch-index>

# Query latest batch for a rollup
qorechaind query rdk latest-batch <rollup-id>

# Get AI-suggested profile for a use case
qorechaind query rdk suggest-profile --use-case "defi lending protocol"

# Query DA blob
qorechaind query rdk blob <rollup-id> <blob-index>

# Query module parameters
qorechaind query rdk params
```

## REST/gRPC API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/qorechain/rdk/v1/rollup/{rollup_id}` | Rollup configuration and status |
| GET | `/qorechain/rdk/v1/rollups` | List all registered rollups |
| GET | `/qorechain/rdk/v1/batch/{rollup_id}/{batch_index}` | Settlement batch details |
| GET | `/qorechain/rdk/v1/batches/{rollup_id}` | List batches for a rollup |
| GET | `/qorechain/rdk/v1/blob/{rollup_id}/{blob_index}` | DA blob details |
| GET | `/qorechain/rdk/v1/params` | Module parameters |

## JSON-RPC (`qor_` Namespace)

| Method | Parameters | Description |
|--------|-----------|-------------|
| `qor_getRollupStatus` | `rollupId` | Rollup configuration, status, and settlement mode |
| `qor_listRollups` | (none) | All registered rollups with status summary |
| `qor_getSettlementBatch` | `rollupId`, `batchIndex` | Settlement batch details and finalization status |
| `qor_suggestRollupProfile` | `useCase` | AI-assisted rollup profile recommendation |
| `qor_getDABlobStatus` | `rollupId`, `blobIndex` | Data availability blob storage status |

## Genesis Configuration

```json
{
  "rdk": {
    "params": {
      "max_rollups": 100,
      "min_stake_for_rollup": "10000000000",
      "rollup_creation_burn_rate": "0.01",
      "default_challenge_window": 604800,
      "max_da_blob_size": 2097152,
      "blob_retention_blocks": 432000,
      "max_batches_per_block": 10
    },
    "rollups": [],
    "batches": []
  }
}
```

## Events

| Event | Description |
|-------|-------------|
| `rollup_created` | New rollup registered |
| `rollup_paused` | Rollup paused by creator |
| `rollup_resumed` | Rollup resumed by creator |
| `rollup_stopped` | Rollup permanently stopped |
| `batch_submitted` | Settlement batch submitted |
| `batch_challenged` | Fraud proof challenge submitted |
| `batch_finalized` | Batch settlement finalized |
| `batch_rejected` | Challenged batch rejected |
| `da_blob_stored` | DA blob stored on native backend |
| `da_blob_pruned` | Expired DA blob pruned |
| `profile_suggested` | AI profile recommendation made |

## Security Considerations

- **PQC State Anchoring**: All rollup state roots are anchored to the Main Chain through the x/multilayer HCS infrastructure, inheriting PQC-signed integrity guarantees
- **Challenge Windows**: Optimistic rollups enforce configurable challenge periods (default 7 days) with bond requirements to prevent frivolous challenges
- **Stake Requirements**: Minimum 10,000 QOR stake with 1% burn prevents spam rollup creation
- **Lifecycle Enforcement**: Only the rollup creator can manage lifecycle transitions; strict state machine prevents invalid transitions
- **DA Blob Limits**: Maximum blob size (2 MB) and automatic pruning prevent state bloat
- **Settlement Validation**: Proof system and settlement mode compatibility enforced at creation time
- **Non-Fatal Anchoring**: Multilayer anchor failures do not block batch submission, ensuring rollup liveness
- **EndBlocker Safety**: Auto-finalization runs within the EndBlocker with error recovery to prevent chain halts
