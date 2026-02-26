# QoreChain Multi-Layer Architecture

## Overview

The QoreChain Multi-Layer Architecture enables scalable transaction processing through a hierarchical four-tier system: **Main Chain**, **Sidechains**, **Paychains**, and **Rollups** (v1.3.0). Each layer is optimized for different workload profiles while maintaining security guarantees through Hierarchical Commitment Schemes (HCS) and PQC-signed state anchoring.

```
                    ┌─────────────────────────┐
                    │      Main Chain          │
                    │  (Settlement + Routing)  │
                    └──┬──────────┬─────────┬──┘
                       │          │         │
            ┌──────────┴──┐  ┌───┴──────┐  ┌┴─────────────┐
            │ Sidechains  │  │ Paychains│  │   Rollups    │
            │ (Compute)   │  │ (MicroTX)│  │ (App-Specific)│
            └─────────────┘  └──────────┘  └──────────────┘

  Main Chain:  Full consensus, state anchoring, cross-layer routing
  Sidechains:  Compute-heavy workloads (DeFi, contracts, analytics)
  Paychains:   High-frequency microtransactions (payments, streaming)
  Rollups:     Application-specific chains with configurable settlement (v1.3.0)
```

## Layer Types

### Main Chain (Settlement Layer)
- Full QoreChain Consensus Engine validation
- PQC-secured block production (Dilithium-5)
- State anchor storage for all subsidiary chains
- Cross-layer routing decisions via QCAI heuristic engine
- Acts as the trust root for all child layers

### Sidechains (Compute Layer)
- Designed for compute-intensive operations (DeFi protocols, smart contract execution, data analytics)
- Independent block production with configurable parameters
- State anchored to Main Chain at regular intervals
- Minimum 3 validators required
- Maximum 10 active sidechains (configurable via governance)
- Minimum stake: 1,000 QOR to register

### Paychains (Microtransaction Layer)
- Optimized for high-frequency, low-value transactions
- Sub-second target block time (500ms default)
- State anchored to Main Chain at regular intervals
- Minimum 3 validators required
- Maximum 50 active paychains (configurable via governance)
- Minimum stake: 100 QOR to register

### Rollups (Application-Specific Layer — v1.3.0)

Rollups are application-specific chains deployed via the **x/rdk** (Rollup Development Kit) module. They register as a `rollup` layer type in the multilayer system and anchor state to Main Chain using the same HCS infrastructure as sidechains and paychains.

- **Four settlement paradigms**: Optimistic (fraud proofs), ZK (validity proofs), Based (L1-sequenced), Sovereign (self-sequenced)
- **Preset profiles**: DeFi, Gaming, NFT, Enterprise, Custom — preconfigured templates for common use cases
- **Data availability**: Native KV-store blob storage with configurable retention and automatic pruning
- **Sequencer modes**: Dedicated (single operator), Shared (minimum sequencer set), Based (L1 proposers)
- **Proof systems**: Fraud proofs, SNARK, STARK, or none — matched to settlement mode
- **VM flexibility**: EVM, CosmWasm, SVM, or custom runtime
- State anchored to Main Chain via `MsgAnchorState` on each batch submission
- Rollup creation requires minimum 10,000 QOR stake (1% burned via x/burn)
- Maximum 100 rollups (configurable via governance)

For full RDK documentation, see [docs/RDK.md](RDK.md).

## QCAI Transaction Routing

The QCAI heuristic router analyzes incoming transactions and routes them to the optimal layer based on a weighted scoring model:

### Scoring Weights

| Factor | Weight | Description |
|--------|--------|-------------|
| Congestion | 0.30 | Current load on each layer |
| Capability | 0.40 | Match between TX type and layer specialization |
| Cost | 0.20 | Fee efficiency on each layer |
| Latency | 0.10 | Expected confirmation time |

### Routing Process

1. **Transaction Analysis**: The router examines the transaction payload size and type
2. **Layer Scoring**: Each active layer (including Main Chain) receives a composite score
3. **Confidence Check**: If the best score exceeds the `routing_confidence_threshold` (default: 0.6), routing proceeds
4. **Selection**: The highest-scoring layer is selected, with optional preferred layer hints
5. **Fallback**: If no layer meets the confidence threshold, the transaction stays on Main Chain

### Payload Heuristics

- **< 256 bytes**: Classified as microtransaction → Paychains preferred
- **256-1024 bytes**: Standard transaction → Main Chain or best-scoring layer
- **> 1024 bytes**: Complex operation → Sidechains preferred

## Hierarchical Commitment Schemes (HCS)

State anchoring ensures that subsidiary chain state is periodically committed to the Main Chain, providing a verifiable trust chain.

### Anchoring Process

1. The subsidiary chain produces a **state root hash** at a designated block height
2. The state root is signed with a **PQC aggregate signature** (Dilithium-5)
3. The anchor is submitted to Main Chain via `MsgAnchorState`
4. Main Chain validates the anchor:
   - Layer must be in `active` status
   - Minimum anchor interval (100 blocks) must have elapsed since last anchor
   - PQC signature must be present and non-empty
5. The anchor is stored and indexed by layer + height

### Anchor Data

Each state anchor contains:
- **Layer ID**: Which subsidiary chain this anchor represents
- **Height**: Block height on the subsidiary chain
- **State Root**: Cryptographic hash of the subsidiary chain's state
- **PQC Signature**: Dilithium-5 aggregate signature from layer validators
- **Timestamp**: When the anchor was created
- **Validator Set Hash**: Hash of the validator set that produced the anchor

### Challenge Mechanism

State anchors can be challenged within the **challenge period** (default: 24 hours):

1. A challenger submits `MsgChallengeAnchor` with a fraud proof
2. If the fraud proof is valid, the anchor is marked as `challenged`
3. Challenged anchors trigger a rollback investigation on the subsidiary chain
4. After the challenge period expires without challenge, anchors are considered finalized

## Cross-Layer Fee Bundling (CLFB)

CLFB allows users to pay a single fee on the source layer that covers execution across all layers in the transaction path.

### Fee Calculation

```
avgMultiplier = sum(layer_multiplier for each layer) / num_layers
bundledFee = (totalGas / 1000) * avgMultiplier
```

- Main Chain base multiplier: 1.0
- Each subsidiary chain has a configurable `base_fee_multiplier`
- Minimum fee: 1 uqor
- Fee denomination: uqor

### Example

A cross-layer transaction touching Main Chain (1.0x) and a sidechain (0.5x) with 50,000 gas:
```
avgMultiplier = (1.0 + 0.5) / 2 = 0.75
bundledFee = (50000 / 1000) * 0.75 = 37.5 → 37 uqor
```

## Layer Lifecycle

Layers follow a strict state machine:

```
PROPOSED → ACTIVE → SUSPENDED → DECOMMISSIONED
              ↕
          SUSPENDED
```

### Valid Transitions

| From | To |
|------|-----|
| Proposed | Active |
| Active | Suspended |
| Active | Decommissioned |
| Suspended | Active |
| Suspended | Decommissioned |

Invalid transitions (e.g., `Active → Proposed` or `Decommissioned → Active`) are rejected.

## Genesis Configuration

The multilayer module initializes with the following default parameters:

```json
{
  "multilayer": {
    "params": {
      "max_sidechains": 10,
      "max_paychains": 50,
      "min_anchor_interval": 100,
      "max_anchor_interval": 1000,
      "default_challenge_period": 86400,
      "min_sidechain_stake": "1000000000",
      "min_paychain_stake": "100000000",
      "routing_enabled": true,
      "routing_confidence_threshold": "0.6",
      "cross_layer_fee_bundling": true
    },
    "layers": [],
    "anchors": []
  }
}
```

## CLI Commands

### Transaction Commands

```bash
# Register a new sidechain
qorechaind tx multilayer register-sidechain \
  --name "defi-sidechain" \
  --description "DeFi-optimized sidechain" \
  --from <key>

# Register a new paychain
qorechaind tx multilayer register-paychain \
  --name "payments-paychain" \
  --description "High-frequency payment channel" \
  --from <key>

# Anchor subsidiary chain state to Main Chain
qorechaind tx multilayer anchor-state \
  --layer-id <layer-id> \
  --height <block-height> \
  --state-root <hex-hash> \
  --pqc-signature <hex-sig> \
  --from <key>

# Route a transaction through QCAI engine
qorechaind tx multilayer route-tx \
  --payload <hex-data> \
  --from <key>

# Update layer status
qorechaind tx multilayer update-layer-status \
  --layer-id <layer-id> \
  --new-status <active|suspended|decommissioned> \
  --from <key>

# Challenge a state anchor
qorechaind tx multilayer challenge-anchor \
  --layer-id <layer-id> \
  --anchor-height <height> \
  --fraud-proof <hex-proof> \
  --from <key>
```

### Query Commands

```bash
# Query a specific layer
qorechaind query multilayer layer <layer-id>

# List all layers (with optional type filter)
qorechaind query multilayer layers [--type sidechain|paychain]

# Query latest anchor for a layer
qorechaind query multilayer anchor <layer-id>

# List all anchors for a layer
qorechaind query multilayer anchors <layer-id>

# Query routing statistics
qorechaind query multilayer routing-stats

# Simulate routing for a transaction
qorechaind query multilayer simulate-route --payload <hex-data>

# Query module parameters
qorechaind query multilayer params
```

## REST/gRPC API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/qorechain/multilayer/v1/layer/{layer_id}` | Get layer details |
| GET | `/qorechain/multilayer/v1/layers` | List all layers |
| GET | `/qorechain/multilayer/v1/anchor/{layer_id}` | Get latest anchor |
| GET | `/qorechain/multilayer/v1/anchors/{layer_id}` | List all anchors |
| GET | `/qorechain/multilayer/v1/routing_stats` | Routing statistics |
| GET | `/qorechain/multilayer/v1/simulate_route` | Simulate routing |
| GET | `/qorechain/multilayer/v1/params` | Module parameters |

## Module Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `max_sidechains` | 10 | Maximum number of active sidechains |
| `max_paychains` | 50 | Maximum number of active paychains |
| `min_anchor_interval` | 100 | Minimum blocks between state anchors |
| `max_anchor_interval` | 1000 | Maximum blocks between state anchors |
| `default_challenge_period` | 86400 | Challenge period in seconds (24h) |
| `min_sidechain_stake` | 1,000,000,000 uqor | Minimum stake to register a sidechain (1,000 QOR) |
| `min_paychain_stake` | 100,000,000 uqor | Minimum stake to register a paychain (100 QOR) |
| `routing_enabled` | true | Enable QCAI transaction routing |
| `routing_confidence_threshold` | 0.6 | Minimum confidence for routing decisions |
| `cross_layer_fee_bundling` | true | Enable CLFB for cross-layer operations |

## Security Considerations

- **PQC-Signed Anchors**: All state anchors require Dilithium-5 signatures, ensuring quantum-resistant integrity
- **Challenge Period**: 24-hour window for fraud proof submission prevents undetected state corruption
- **Minimum Validators**: Both sidechains and paychains require a minimum of 3 validators
- **Stake Requirements**: Registration requires bonded stake, preventing spam layer creation
- **Status Transitions**: Strict state machine prevents invalid lifecycle changes
- **Anchor Intervals**: Minimum interval prevents anchor spam; maximum interval ensures freshness
- **Routing Confidence**: Transactions only routed when QCAI confidence exceeds threshold
