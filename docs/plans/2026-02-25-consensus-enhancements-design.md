# QoreChain Consensus Enhancements Design — v0.9.0

**Date:** 2026-02-25
**Target Version:** v0.9.0
**Status:** Approved

## Overview

Three sequential enhancements to the QoreChain consensus layer:

- **Part A (Launch):** x/rlconsensus — RL-based dynamic parameter tuning via PPO agent
- **Part B (Phase 2):** Triple-Pool CPoS + custom bonding curve + progressive slashing (integrated into x/qca)
- **Part C (Phase 2):** QDRW governance — quadratic delegation with reputation weighting (integrated into x/qca)

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Version | v0.9.0 | v0.8.0 already tagged for SVM Runtime |
| PPO model | Go-native MLP, int64 fixed-point | Deterministic on-chain inference, no external deps |
| Slashing | Integrated into x/qca | QCA already owns proposer selection + reputation dep |
| xQORE | Stub TokenomicsKeeper interface | Ready for future tokenomics module, returns zero now |
| x/rlconsensus pattern | Factory indirection | Consistent with PQC, AI, SVM modules |
| x/qca extensions | Direct (no factory) | Already wired directly in app.go |
| Float64 boundary | LegacyDec conversion at edge | x/reputation uses float64; convert with fmt.Sprintf("%.18f") |

---

## Section 1: Architecture Overview

### Module Topology

```
x/rlconsensus (NEW — factory pattern, proprietary/stub)
    depends on: x/reputation, x/ai, x/feemarket, x/staking, consensus params
    implements: RLConsensusParamsProvider interface (drop-in for StaticRLProvider)

x/qca (EXTENDED — direct wiring, proprietary files added)
    new deps: x/staking (for pool classification + bonding curve)
    new files: pool_classifier, pool_selector, bonding_curve, progressive_slashing, qdrw_tally

x/reputation (UNCHANGED — read-only dependency)
x/ai (UNCHANGED — read-only dependency for anomaly counts)
x/feemarket (UNCHANGED — read-only dependency for base fee)
```

### Data Flow

```
BeginBlock:
  Every 10 blocks → RLKeeper.CollectObservation()
    reads: block stats, mempool depth, validator participation,
           latency percentiles, base fee, anomaly counts, current params
    writes: observation to KV store

EndBlock:
  Every 10 blocks → RLKeeper.StepAgent()
    1. ComputeReward() from last observation pair
    2. agent.Infer(observation) → action vector (5 dims)
    3. Clamp actions to ±MaxChangePerAction
    4. Apply new params (or log-only in shadow mode)
    5. Store experience tuple for off-chain training
    6. CircuitBreaker check → revert if <50% blocks on time
```

### RLConsensusParamsProvider Drop-In

The interface already exists at `x/vm/precompiles/rl_interface.go` with a `StaticRLProvider`. The new x/rlconsensus keeper implements this interface directly. In `app.go`, the EVM precompile receives the RL keeper instead of the static provider once the module is wired.

### Factory Pattern for x/rlconsensus

```go
// app/factory.go — add:
var NewRLConsensusKeeper func(...) rlconsensus.RLConsensusKeeper

// app/factory_proprietary.go — add:
func init() { NewRLConsensusKeeper = proprietary.NewKeeper }

// app/factory_stub.go — add:
func init() { NewRLConsensusKeeper = stub.NewKeeper }
```

---

## Section 2: x/rlconsensus Module Design

### Determinism Strategy

All on-chain inference uses **int64 fixed-point** with scale factor `10^8` (1.0 = 100,000,000). No `float64` in the inference or reward path. `math.LegacyDec` for parameter storage and API boundaries.

**Float64 boundary with x/reputation:** x/reputation stores scores as `float64`. The observation collector converts at the boundary:

```go
dec := math.LegacyNewDecFromStr(fmt.Sprintf("%.18f", reputationFloat64))
```

This is deterministic because the float64 value is already deterministic (produced by deterministic on-chain math in x/reputation's EndBlocker).

### MLP Architecture

| Layer | Shape | Parameters |
|-------|-------|------------|
| Input | 25 neurons | 0 |
| Hidden 1 | 256 neurons, ReLU | 25 x 256 + 256 bias = 6,656 |
| Hidden 2 | 256 neurons, ReLU | 256 x 256 + 256 bias = 65,792 |
| Output | 5 neurons (tanh) | 256 x 5 + 5 bias = 1,285 |
| **Total** | | **73,733 int64 params (~589KB)** |

Weights stored on-chain as `repeated int64` in protobuf. Updated via governance `MsgUpdatePolicy` (imports a new weight blob from off-chain training).

### Observation Vector (~25 dimensions)

```
[0]  block_utilization        — gas_used / gas_limit (last 10 blocks avg)
[1]  mempool_depth            — pending tx count
[2]  validator_participation   — signed / total validators (last 10 blocks)
[3]  latency_p50             — 50th percentile block time (ms)
[4]  latency_p95             — 95th percentile block time (ms)
[5]  latency_p99             — 99th percentile block time (ms)
[6]  base_fee                — current EIP-1559 base fee
[7]  base_fee_velocity       — delta from 10 blocks ago
[8]  anomaly_count           — flagged txs in last 10 blocks
[9]  failed_tx_ratio         — failed / total txs
[10] avg_reputation          — mean validator reputation score
[11] reputation_std          — std deviation of reputation scores
[12-16] current_params       — block_time, gas_limit, gas_price_floor, pool_weight_rpos, pool_weight_dpos
[17-21] param_deltas         — change from 100 blocks ago
[22] epoch                   — current RL training epoch
[23] blocks_since_revert     — blocks since last circuit breaker trigger
[24] mev_estimate            — estimated MEV (heuristic from tx ordering)
```

### Action Space (5 dimensions)

Output neurons use `tanh` scaled to `[-MaxChangePerAction, +MaxChangePerAction]`:

| Index | Parameter | Default Max Change |
|-------|-----------|-------------------|
| 0 | block_time_delta | +/-10% (shadow), +/-25% (full) |
| 1 | gas_limit_delta | +/-10% / +/-25% |
| 2 | gas_price_floor_delta | +/-10% / +/-25% |
| 3 | pool_weight_rpos_delta | +/-5% / +/-15% |
| 4 | pool_weight_dpos_delta | +/-5% / +/-15% |

### Reward Function

```
R = w1*delta_throughput + w2*delta_finality + w3*delta_decentralization
    - w4*mev_estimate - w5*failed_tx_ratio
```

Default weights: `w1=0.30, w2=0.25, w3=0.20, w4=0.15, w5=0.10` (governance-adjustable).

### Circuit Breaker

If fewer than 50% of the last 50 blocks were produced within the target block time:
1. Revert all RL-tuned parameters to their pre-RL defaults
2. Set `AgentMode = PAUSED`
3. Emit `rl_circuit_breaker_triggered` event
4. Recovery requires governance `MsgResumeAgent`

### Shadow Mode Rollout

| Phase | Mode | Max Change | Description |
|-------|------|------------|-------------|
| 1 | SHADOW | N/A | Observe + log recommendations only |
| 2 | CONSERVATIVE | +/-10% | Apply with tight bounds |
| 3 | AUTONOMOUS | +/-25% | Full autonomy with circuit breaker |

Mode transitions via governance `MsgSetAgentMode`.

### Keeper Dependencies

```go
type Keeper struct {
    storeService   store.KVStoreService
    reputationKeeper ReputationReader      // GetAllValidatorReputations, GetValidatorReputation
    aiKeeper         AIStatsReader          // GetStats (for anomaly counts)
    feeMarketKeeper  FeeMarketReader        // GetBaseFee
    stakingKeeper    StakingReader          // GetAllValidators, GetValidatorDelegations
    consensusKeeper  ConsensusParamsReader  // GetConsensusParams
    agent            *PPOAgent              // MLP + inference logic
    circuitBreaker   *CircuitBreaker
}
```

### Module Files

```
x/rlconsensus/
    interfaces.go              — NO build tag: RLConsensusKeeper interface
    module.go                  — NO build tag: AppModuleBasic
    module_proprietary.go      — //go:build proprietary: full AppModule
    module_stub.go             — //go:build !proprietary: stub AppModule
    register.go                — //go:build proprietary: factory + adapter
    keeper_stub.go             — //go:build !proprietary: stub keeper
    abci.go                    — NO build tag: ABCI hooks interface
    types/
        keys.go, params.go, genesis.go, errors.go, codec.go, events.go
        observation.go         — ObservationVector type
        action.go              — ActionVector type
        reward.go              — RewardConfig type
        policy.go              — PolicyWeights (MLP weights) protobuf
    keeper/                    — //go:build proprietary (ALL files)
        keeper.go              — Main keeper
        observation.go         — CollectObservation logic
        reward.go              — ComputeReward logic
        agent.go               — PPOAgent (inference only)
        mlp.go                 — Fixed-point MLP forward pass
        policy.go              — PolicyWeights storage + MsgUpdatePolicy
        circuit_breaker.go     — Circuit breaker logic
        params_applicator.go   — Apply actions to consensus params
        msg_server.go          — MsgSetAgentMode, MsgResumeAgent, MsgUpdatePolicy, etc.
        query_server.go        — QueryObservation, QueryAgentStatus, QueryReward, etc.
        genesis.go             — InitGenesis / ExportGenesis
        abci.go                — BeginBlock / EndBlock implementations
    cli/
        tx.go                  — CLI tx commands
        query.go               — CLI query commands
```

### Off-Chain Training Sidecar

```
sidecar/rl_training/
    service.go    — gRPC service exposing Train RPC
    ppo.go        — Full PPO implementation (float64 OK — off-chain)
    buffer.go     — Experience replay buffer
```

The sidecar reads experience tuples from the chain (via gRPC query), runs PPO training with float64, and produces updated int64 weight blobs. A validator submits `MsgUpdatePolicy` with the new weights; other validators verify the blob hash matches the governance-approved training run.

---

## Section 3: x/qca Extensions (Part B)

### Triple-Pool Classification

Every 1000 blocks, validators are reclassified into three pools:

| Pool | Criteria | Default Weight |
|------|----------|----------------|
| **RPoS** | Reputation >= 70th percentile AND stake >= median | 40% |
| **DPoS** | Total delegation >= 10,000 QOR | 35% |
| **PoS** | All remaining active validators | 25% |

A validator can qualify for multiple pools but is placed in the highest-priority pool (RPoS > DPoS > PoS).

Pool weights are RL-adjustable (Part A action dimensions 3-4). The PoS weight is always `1.0 - rpos_weight - dpos_weight`.

### Pool-Weighted Selection

Extends the existing `HeuristicSelector` in x/qca:

```go
// x/qca/keeper/pool_selector.go — //go:build proprietary
type PoolWeightedSelector struct {
    inner           *HeuristicSelector  // existing selector for within-pool ranking
    reputationKeeper reputationkeeper.Keeper
    stakingKeeper    stakingkeeper.Keeper
    rlKeeper         rlconsensus.RLConsensusKeeper  // for dynamic weights, nil-safe
}
```

Selection algorithm:
1. Classify all active validators into pools
2. Select pool via weighted random (seed = SHA256(blockHash || height), same as existing)
3. Within the selected pool, use existing reputation-weighted selection from `HeuristicSelector`

### Custom Bonding Curve

```
R(v,t) = beta * S_v * (1 + alpha * log(1 + L_v)) * Q(r_v) * P(t)
```

Where:
- `S_v` = self-bonded stake
- `L_v` = loyalty duration (blocks since first bond)
- `Q(r_v)` = reputation quality factor = `1 + 0.5 * (r_v - 0.5)` clamped to [0.75, 1.25]
- `P(t)` = protocol phase multiplier (genesis=1.5, growth=1.0, mature=0.8) — governance-set
- `alpha` = loyalty sensitivity (default 0.1)
- `beta` = base multiplier (default 1.0)

**Deterministic log:** `log(1+x)` approximated via Taylor series on `math.LegacyDec`:

```
log(1+x) = x - x^2/2 + x^3/3 - x^4/4 + ...  (10 terms)
```

For large x (loyalty > e^2 - 1 ~ 6.39), use `log(x) = log(x/k) + log(k)` with precomputed `log(k)` constants to keep the Taylor argument in (0, 1].

### Progressive Slashing

```
penalty = base_rate * 1.5^effective_count * severity_factor
```

Capped at 33% per slash event.

**Temporal decay:** Each historical infraction's weight decays with half-life at 100,000 blocks:

```
effective_count = sum(0.5^(blocks_since_i / 100000)) for each past infraction i
```

The `0.5^x` (where x is fractional) is computed via the identity `0.5^x = exp(-ln2 * x)`, with `exp` approximated by Taylor series (12 terms) on `math.LegacyDec`.

**Storage:**

```go
// x/qca/types/slashing_record.go
type SlashingRecord struct {
    ValidatorAddr string
    InfractionHeight int64
    InfractionType string   // "double_sign", "downtime", "light_client_attack"
    SeverityFactor math.LegacyDec
    Penalty math.LegacyDec
}
```

Stored in x/qca's KV store, keyed by `validator_addr + infraction_height`. Pruned after 1,000,000 blocks (decay makes them negligible).

### Slashing Integration

x/qca listens to slashing events from the SDK's `x/slashing` and `x/evidence` modules via `BeginBlock` event inspection or keeper hooks. When a slash event is detected:

1. Load validator's `SlashingRecord` history from KV
2. Compute `effective_count` with temporal decay
3. Compute penalty via progressive formula
4. Update the validator's reputation in x/reputation (negative adjustment)
5. Store new `SlashingRecord`

The actual token slashing still happens via `x/slashing` — x/qca only computes the _enhanced penalty_ and applies reputation consequences.

### New x/qca Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `pool_classification_interval` | 1000 | Blocks between reclassification |
| `pool_weight_rpos` | 0.40 | RPoS pool selection weight |
| `pool_weight_dpos` | 0.35 | DPoS pool selection weight |
| `pool_min_delegation_dpos` | 10000000000 | Min delegation for DPoS (uqor) |
| `pool_rep_percentile_rpos` | 70 | Reputation percentile for RPoS |
| `bonding_alpha` | 0.1 | Loyalty sensitivity |
| `bonding_beta` | 1.0 | Base multiplier |
| `bonding_phase_multiplier` | 1.5 | Protocol phase (genesis) |
| `slashing_base_rate` | 0.01 | Base slash rate (1%) |
| `slashing_escalation_factor` | 1.5 | Progressive multiplier base |
| `slashing_max_penalty` | 0.33 | Maximum slash (33%) |
| `slashing_decay_halflife` | 100000 | Blocks for half-life decay |

### New x/qca Files

```
x/qca/keeper/pool_classifier.go         — //go:build proprietary
x/qca/keeper/pool_selector.go           — //go:build proprietary
x/qca/keeper/bonding_curve.go           — //go:build proprietary
x/qca/keeper/progressive_slashing.go    — //go:build proprietary
x/qca/keeper/math_utils.go              — NO build tag (shared math)
x/qca/types/pool.go                     — NO build tag
x/qca/types/bonding_curve.go            — NO build tag
x/qca/types/slashing_record.go          — NO build tag
```

---

## Section 4: QDRW Governance (Part C)

### Approach: x/gov Hooks via Custom Tally Handler

Rather than creating a separate governance module, we extend the existing SDK `x/gov` through a custom `TallyHandler`. This keeps governance upgrades minimal and avoids reimplementing the proposal lifecycle.

### Voting Power Formula

```
VP(v) = sqrt(staked_v + 2 * xQORE_v) * ReputationMultiplier(r_v)
```

Where:
- `staked_v` = delegated QOR in uqor — read from x/staking
- `xQORE_v` = governance-boost token balance — read from TokenomicsKeeper (stubbed to 0)
- `r_v` = composite reputation score from x/reputation

**Square root** uses integer Newton's method on `math.LegacyDec`. No stdlib `math.Sqrt`.

### Reputation Multiplier

Maps reputation score [0, 1] to multiplier [0.5, 2.0] via sigmoid:

```
ReputationMultiplier(r) = 0.5 + 1.5 * sigmoid(6 * (r - 0.5))
```

Sigmoid approximated via 5th-order Pade approximant for determinism:

```
sigmoid(x) ~ (1/2) + x*(1/4 - x^2/48) / (1 + x^2/12)
```

Less than 0.1% error on [-3, 3], which covers the full input range.

### TokenomicsKeeper Stub Interface

```go
type TokenomicsKeeper interface {
    GetXQOREBalance(ctx sdk.Context, addr sdk.AccAddress) math.Int
}

type NilTokenomicsKeeper struct{}
func (NilTokenomicsKeeper) GetXQOREBalance(_ sdk.Context, _ sdk.AccAddress) math.Int {
    return math.ZeroInt()
}
```

Until xQORE is implemented, VP simplifies to `sqrt(staked) * ReputationMultiplier(r)`.

### Implementation Files

```
x/qca/keeper/qdrw_tally.go              — //go:build proprietary
    QDRWTallyHandler struct
    CalculateVotingPower(voter) method
    Tally(proposal) method

x/qca/keeper/qdrw_tally_stub.go         — //go:build !proprietary
    Returns default SDK tally behavior

x/qca/keeper/math_utils.go              — NO build tag (shared)
    IntegerSqrt(dec) — Newton's method
    SigmoidApprox(dec) — Pade approximant
    ReputationMultiplier(score) — [0.5, 2.0] mapping
```

### QDRW Parameters (added to x/qca params)

| Parameter | Default | Description |
|-----------|---------|-------------|
| `qdrw_enabled` | false | Enable QDRW tally |
| `qdrw_xqore_multiplier` | 2.0 | xQORE weight relative to staked tokens |
| `qdrw_rep_min_multiplier` | 0.5 | Minimum reputation multiplier |
| `qdrw_rep_max_multiplier` | 2.0 | Maximum reputation multiplier |

QDRW starts disabled. A governance proposal can enable it. When disabled, the standard SDK tally runs unchanged.

---

## RPC Extensions

### New JSON-RPC Methods (qor_ namespace)

| Method | Description |
|--------|-------------|
| `qor_getRLAgentStatus` | Agent mode, epoch, last observation, circuit breaker state |
| `qor_getRLObservation` | Current/historical observation vectors |
| `qor_getRLReward` | Reward history and component breakdown |
| `qor_getPoolClassification` | Current validator pool assignments |

### New REST Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/rlconsensus/v1/agent` | GET | Agent status |
| `/qorechain/rlconsensus/v1/observation` | GET | Latest observation |
| `/qorechain/rlconsensus/v1/rewards` | GET | Reward history |
| `/qorechain/rlconsensus/v1/params` | GET | Module parameters |
| `/qorechain/rlconsensus/v1/policy` | GET | Current policy metadata |

---

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| RL agent destabilizes consensus | Shadow mode first; circuit breaker auto-reverts; conservative bounds |
| Fixed-point overflow in MLP | int64 scale 10^8 gives range +/-92 billion; clamp activations |
| Pool classification gaming | Reclassify only every 1000 blocks; reputation has time decay |
| Progressive slashing too aggressive | 33% cap; temporal decay half-life; governance-adjustable params |
| Float64 non-determinism from x/reputation | Convert at boundary with %.18f format; reputation itself is deterministic |
| QDRW whale dominance | Quadratic root limits large-stake advantage; reputation multiplier rewards honest behavior |
