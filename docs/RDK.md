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

Batches include a validity proof (SNARK or STARK) that is verified on submission. If the proof verifies, the batch is immediately finalized — no challenge window needed. This provides the fastest finality of any settlement mode.

- **Proof systems**: SNARK (Groth16, PLONK) or STARK (transparent, no trusted setup)
- **Recursive aggregation**: Configurable recursion depth for proof composition
- **Max proof size**: Configurable (default 1 MB)
- **Finality**: Instant upon proof verification
- **STARK verifier (v3.1.33+)**: A real, from-scratch transparent STARK verifier lives in `x/rdk/stark` (prime field, Fiat-Shamir transcript, Merkle commitments, FRI low-degree test, AIR + DEEP composition). It is wired into settlement behind an opt-in `QSTK` verification-key gate — rollups configured for STARK verification have their ZK batches cryptographically verified on-chain; others fall back to the optimistic challenge path. (Earlier releases up to v1.3.0 accepted any non-empty proof as a stub.)

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

### Posted Rollup Data and Based Sequencing (from v3.2.8)

`MsgPostRollupData` (`tx rdk post-data`) puts rollup data on chain. The payload
stays in the transaction (so it is in the block); the chain keeps a commitment
(SHA-256 of the data, size, poster, height) under the rollup's next inbox
sequence and extends the rollup's inbox accumulator. A batch of a native-DA
rollup may reference a retained post by `data_hash = SHA-256(data)`.

- Who may post: the rollup's sequencer (its creator when no sequencer address
  is set); anyone on a based rollup.
- Caps: `max_posted_data_bytes` per post (default 128 KiB, hard ceiling 1 MiB)
  and `max_posted_data_bytes_per_block` for all rollups together (default
  512 KiB of billable bytes).
- Fee: `posted_data_fee_per_byte` uqor (default 10) per billable byte, where a
  post is billed at least 1 KiB, paid into the fee collector and split like a
  transaction fee. Gas is charged on top.
- Retention: commitments are pruned after `posted_data_retention_blocks`
  (default 201,600). The bytes stay in the block history.

Based sequencing (`create-rollup --sequencing based`, or governance
`MsgSetRollupSequencing` for an existing rollup) orders a rollup's
transactions by their inclusion on this chain. Each post gets the next inbox
sequence, and every batch of a based rollup sets `inbox_end` and carries as
`data_hash` the inbox range commitment from the consumed position to
`inbox_end` (`query rdk inbox` returns it). A based zk rollup proving with
Groth16 (`defi` profile) accepts batches from any account, since the proof
binds the roots, the inbox range and the withdrawals root; a based-settlement
rollup (`gaming`, `enterprise`) keeps its sequencer as the only submitter.
Optimistic and STARK rollups cannot be based in this release.

The accumulator and range commitment are:

```
acc_n = SHA-256("qorechain/rdk/inbox/v1" | lp(chain_id) | lp(rollup_id) | acc_{n-1}
                | be64(n) | be64(height) | lp(poster) | SHA-256(data))
range = SHA-256("qorechain/rdk/inbox-range/v1" | lp(chain_id) | lp(rollup_id)
                | be64(from) | be64(to) | acc_from | acc_to)
```

with `acc_0` = 32 zero bytes and `lp` a 4-byte big-endian length prefix.


### Settlement Modes (from v3.2.8)

Three protocol features, all enabled by the v3.2.8 upgrade.

**`rdk-optimistic-reexec`: optimistic rollups with fraud proofs that check execution.**
An optimistic rollup (`nft`, `custom`) created with `--execution-vm qledger-v1 --sequencing based` runs the
qledger-v1 model (`x/rdk/qledger`): a uqor account ledger (sparse Merkle tree, SHA-256) with signed transfers
and withdrawals (ML-DSA-44 or Ed25519 account keys). Its inputs are its inbox: user posts (one transaction each,
at most 6 KiB) and deposits (`MsgDepositToRollup`, which the chain appends itself). Any account holding a
submitter bond (`MsgBondRollupSubmitter`, at least `min_submitter_bond`, default 1,000 QOR) may submit a batch of
at most 64 inbox entries. During the challenge window anyone may send `MsgChallengeBatch` with a qledger-v1 fraud
proof (the batch's inputs and a Merkle witness of the accounts they touch); the chain re-executes the batch and,
if its state root or withdrawals root is wrong, rejects it and every batch built on it, rolls the inbox back and
pays the submitter's whole bond to the challenger. A proof that does not show fraud fails and changes nothing.
Withdrawals are paid from finalized batches only, from the rollup's deposit pool (never from the creator's
escrow); the creator cannot stop the rollup while the pool holds deposits. A bond can be withdrawn
(`MsgUnbondRollupSubmitter`) a full challenge window after its holder's last batch. Optimistic rollups without an
execution VM are unchanged: their batches are as good as their sequencer.

**`rdk-stark-statement`: the demonstration STARK AIR is refused.** The only AIR the STARK verifier implements
proves iterated squaring between two 64-bit values, which anyone can prove for any roots and which does not
constrain withdrawals or data. From v3.2.8 a STARK key cannot be set and a STARK rollup's batches are refused;
governance can move such a rollup to a Groth16 key. STARK settlement returns with a real rollup statement.

**`rdk-private-rollups`: private rollups.** A `defi` (Groth16) rollup created with `--privacy encapsulated` takes
only posts that are ML-KEM-1024 envelopes (`x/rdk/privpost`) to its members: the data encrypted once (AES-256-GCM)
under a fresh key, that key encapsulated to every member's key in the x/pqc KEM registry, with a key commitment so
all members open the same content. The creator sets the members (`MsgSetRollupMembers`, at most 32, each with a
registered ML-KEM-1024 key); each change starts a new epoch. The chain checks the envelope's structure and that
every entry names the member's current key; it never decrypts. Posts, poster, timing, sizes, members, roots and
withdrawals (recipient and amount) stay public. Use `tx rdk post-private-data` to encrypt and post, and
`tx rdk decrypt-private-data` (offline) to open a post.

**Based rollups are paused and resumed by governance only** (from `rdk-based-sequencing`).

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
| `max_batches_per_block` | 10 | Maximum settlement batches one rollup can settle in one block (enforced from v3.2.8) |
| `finalized_batch_retention_blocks` | 0 (= 432,000, ~30 days) | Blocks a finalized batch keeps its stored proof before it is compacted (from v3.2.8). Compaction removes only the proof; withdrawals against the batch keep working |
| `max_posted_data_bytes` | 0 (= 131,072) | Largest `MsgPostRollupData` payload (from v3.2.8) |
| `max_posted_data_bytes_per_block` | 0 (= 524,288) | Billable bytes all posts of one block may total (from v3.2.8) |
| `posted_data_fee_per_byte` | 0 (= 10 uqor) | Fee per billable byte, at least 1 KiB billed per post (from v3.2.8) |
| `posted_data_retention_blocks` | 0 (= 201,600) | Blocks a posted-data commitment stays in state (from v3.2.8) |
| `min_submitter_bond` | 0 (= 1,000,000,000 uqor) | Bond a qledger-v1 batch submitter must hold (from v3.2.8) |

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
| GET | `/qorechain/rdk/v1/posted_data/{rollup_id}/{sequence}` | Posted-data commitment (v3.2.8) |
| GET | `/qorechain/rdk/v1/inbox/{rollup_id}?inbox_end=N` | Inbox head, consumed position and the range data_hash for a based batch (v3.2.8) |
| GET | `/qorechain/rdk/v1/submitter_bond/{rollup_id}/{submitter}` | A qledger-v1 submitter's bond and when it unlocks (v3.2.8) |

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

From v3.2.8 an export also carries `escrow_ledger`, each rollup's share of the
pooled module escrow (`{"rollup_id": "...", "amount": "<uqor>"}`), and an import
restores it. A rollup imported without a ledger entry is treated as a legacy
record: stopping it refunds its recorded escrow (or its stake, whichever is
smaller; the stake when no escrow figure is recorded).

It also carries `posted_data` (the retained posted-data commitments) and
`inboxes` (each rollup's inbox head and consumed position); an import rebuilds
the hash index and the pruning queue from them.

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
| `rdk_data_posted` | Rollup data posted (rollup_id, sequence, data_hash, size, billable_bytes, fee, poster, inbox_acc), from v3.2.8 |
| `rdk_inbox_consumed` | A based batch consumed an inbox range (rollup_id, batch_index, inbox_start, inbox_end), from v3.2.8 |
| `rdk_sequencing_set` | A rollup was made based at creation or by governance, from v3.2.8 |
| `rdk_posted_data_pruned` | Expired posted-data commitments pruned this block (count), from v3.2.8 |

## Security Considerations

- **PQC State Anchoring**: All rollup state roots are anchored to the Main Chain through the x/multilayer HCS infrastructure, inheriting PQC-signed integrity guarantees
- **Challenge Windows**: Optimistic rollups enforce configurable challenge periods (default 7 days) with bond requirements to prevent frivolous challenges
- **Stake Requirements**: Minimum 10,000 QOR stake with 1% burn prevents spam rollup creation
- **Lifecycle Enforcement**: Only the rollup creator can manage lifecycle transitions; strict state machine prevents invalid transitions
- **DA Blob Limits**: Maximum blob size (2 MB) and automatic pruning prevent state bloat
- **Settlement Validation**: Proof system and settlement mode compatibility enforced at creation time
- **Non-Fatal Anchoring**: Multilayer anchor failures do not block batch submission, ensuring rollup liveness
- **EndBlocker Safety**: Auto-finalization runs within the EndBlocker with error recovery to prevent chain halts
