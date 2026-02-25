# QoreChain Consensus Enhancements

## Overview

QoreChain v0.9.0 introduces four consensus-layer enhancements in the `x/qca` module: Triple-Pool Composite Proof-of-Stake (CPoS), a custom bonding curve for validator rewards, progressive slashing with temporal decay, and Quadratic Delegation with Reputation Weighting (QDRW) for governance. These features build on the existing reputation-weighted proposer selection and work together with the RL consensus tuning module (see [RL_CONSENSUS.md](./RL_CONSENSUS.md)).

## Triple-Pool Composite Proof-of-Stake (CPoS)

### Pool Classification

Validators are classified into three pools every `classification_interval` blocks (default: 1000). Classification is deterministic and based on on-chain metrics.

| Pool | Criteria | Default Weight |
|------|----------|----------------|
| **RPoS** (Reputation PoS) | Reputation >= 70th percentile AND stake >= median | 40% |
| **DPoS** (Delegated PoS) | Total delegation >= 10,000 QOR | 35% |
| **PoS** (Standard) | All remaining active validators | 25% |

Classification priority is RPoS > DPoS > PoS. A validator qualifying for multiple pools is assigned to the highest-priority pool. Unclassified validators (before the first classification epoch) default to PoS.

### Pool-Weighted Proposer Selection

Block proposers are selected in two stages:

1. **Pool selection** -- A pool is chosen via deterministic weighted random using `SHA256(lastBlockHash || height || "pool")`. The first 8 bytes of the hash produce a uniform random value in [0, 1), which selects a pool based on cumulative weights.
2. **Within-pool selection** -- The proposer is selected from the chosen pool using the existing reputation-weighted heuristic selector (cumulative distribution function over `reputation * stake`).

If the selected pool is empty, the selector falls back to the PoS pool, then to the full validator set.

### Pool Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `classification_interval` | uint64 | `1000` | Blocks between reclassification |
| `weight_rpos` | Dec | `0.40` | RPoS pool selection weight |
| `weight_dpos` | Dec | `0.35` | DPoS pool selection weight |
| `min_delegation_dpos` | uint64 | `10000000000` | Minimum delegation for DPoS (uqor, = 10k QOR) |
| `rep_percentile_rpos` | uint64 | `70` | Reputation percentile threshold for RPoS |

The PoS weight is implicitly `1.0 - weight_rpos - weight_dpos` (default: 0.25). Pool weights can also be dynamically adjusted by the RL consensus module when it is operating in conservative or autonomous mode.

## Custom Bonding Curve

### Formula

Validator staking rewards are computed using a multi-factor bonding curve:

```
R(v,t) = beta * S_v * (1 + alpha * log(1 + L_v)) * Q(r_v) * P(t)
```

| Symbol | Description |
|--------|-------------|
| `beta` | Base reward multiplier (default: 1.0) |
| `S_v` | Validator's self-bonded stake |
| `alpha` | Loyalty sensitivity coefficient (default: 0.1) |
| `L_v` | Loyalty duration in blocks since first bond |
| `log(1 + L_v)` | Natural logarithm, computed via deterministic Taylor series |
| `Q(r_v)` | Reputation quality factor |
| `P(t)` | Protocol phase multiplier |

### Loyalty Duration Bonus

The `log(1 + L_v)` term provides diminishing returns for longer bonding periods. It is computed using the `TaylorLn1PlusX` function from the `mathutil` package, which uses argument reduction followed by a 15-term Taylor series. All arithmetic uses `LegacyDec` for determinism.

### Reputation Quality Factor

```
Q(r_v) = 1 + 0.5 * (r_v - 0.5)
```

Where `r_v` is the validator's composite reputation score (from x/reputation). The result is clamped to [0.75, 1.25]:

- A reputation of 0.0 yields Q = 0.75 (25% penalty)
- A reputation of 0.5 yields Q = 1.0 (neutral)
- A reputation of 1.0 yields Q = 1.25 (25% bonus)

### Protocol Phase Multiplier

The multiplier `P(t)` allows the protocol to adjust rewards based on its lifecycle stage:

| Phase | Multiplier | Purpose |
|-------|------------|---------|
| Genesis | 1.5 | Incentivize early validators |
| Growth | 1.0 | Standard rewards |
| Mature | 0.8 | Reduce inflation |

The current phase multiplier is set via the `phase_multiplier` config parameter and can be updated through governance.

### Bonding Curve Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `alpha` | Dec | `0.1` | Loyalty sensitivity coefficient |
| `beta` | Dec | `1.0` | Base reward multiplier |
| `phase_multiplier` | Dec | `1.5` | Protocol phase multiplier (genesis phase) |

## Progressive Slashing

### Formula

Slashing penalties escalate based on a validator's infraction history, with past infractions decaying over time:

```
penalty = base_rate * escalation_factor^effective_count * severity_factor
```

The penalty is capped at `max_penalty` (default: 33%) per slashing event.

### Temporal Decay

Past infractions contribute to `effective_count` with half-life decay:

```
effective_count = SUM( 0.5^(blocks_since_i / decay_halflife) )
```

For each past infraction `i`, the contribution decays by half every `decay_halflife` blocks (default: 100,000 blocks, approximately 5.8 days at 5s block time). Recent infractions contribute close to 1.0; older infractions contribute progressively less.

The exponential `0.5^x` is computed as `exp(-ln(2) * x)` using the deterministic `ExpApprox` Taylor series.

### Escalation Calculation

The escalation is computed as:

```
escalation_factor^effective_count = exp(effective_count * ln(escalation_factor))
```

Where `ln(escalation_factor)` uses `TaylorLn1PlusX(escalation_factor - 1)`.

### Severity Factors

The `severity_factor` is set based on the infraction type:

| Infraction | Typical Severity |
|------------|-----------------|
| `downtime` | 1.0 |
| `double_sign` | 2.0 |
| `light_client_attack` | 3.0 |

### Example

For a validator with 2 past downtime infractions (one at 50,000 blocks ago, one at 150,000 blocks ago) committing a new downtime infraction:

- Infraction 1 decay: `0.5^(50000/100000) = 0.707`
- Infraction 2 decay: `0.5^(150000/100000) = 0.354`
- Effective count: `0.707 + 0.354 = 1.061`
- Penalty: `0.01 * 1.5^1.061 * 1.0 = 0.01 * 1.512 = 0.01512` (1.5%)

### Slashing Records

Each slashing event is recorded in the KV store with:
- Validator address
- Infraction height
- Infraction type
- Severity factor
- Computed penalty

Records are used for temporal decay calculations and can be pruned after a configurable retention period.

### Slashing Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `base_rate` | Dec | `0.01` | Base slash rate (1%) |
| `escalation_factor` | Dec | `1.5` | Progressive multiplier base |
| `max_penalty` | Dec | `0.33` | Maximum penalty per event (33%) |
| `decay_halflife` | uint64 | `100000` | Blocks for half-life decay |

## QDRW Governance

### Overview

Quadratic Delegation with Reputation Weighting (QDRW) modifies governance voting power to reduce plutocratic influence and reward reputable validators.

### Voting Power Formula

```
VP(v) = sqrt(staked + 2 * xQORE) * ReputationMultiplier(r)
```

| Symbol | Description |
|--------|-------------|
| `staked` | Validator's staked QOR amount |
| `xQORE` | Validator's xQORE governance token balance (from future tokenomics module) |
| `2` | xQORE multiplier (configurable, default: 2.0) |
| `r` | Validator's reputation score [0, 1] |

### Square Root (Quadratic Component)

The square root provides sublinear scaling: doubling stake does not double voting power. This reduces the influence of large stakers relative to a linear model. The `IntegerSqrt` function uses Newton's method with 100-iteration convergence, fully deterministic via `LegacyDec`.

### Reputation Multiplier

The reputation multiplier maps a score in [0, 1] to a multiplier in [0.5, 2.0] using a sigmoid curve:

```
ReputationMultiplier(r) = 0.5 + 1.5 * sigmoid(6 * (r - 0.5))
```

| Reputation | Multiplier (approx) |
|------------|---------------------|
| 0.0 | 0.53 |
| 0.25 | 0.68 |
| 0.50 | 1.25 |
| 0.75 | 1.82 |
| 1.0 | 1.97 |

The sigmoid is computed via the `SigmoidApprox` function using `ExpApprox` (12-term Taylor series). The result is clamped to [0.5, 2.0].

### xQORE Integration

The xQORE governance token is provided by a future tokenomics module. The `TokenomicsKeeper` interface has a single method:

```
GetXQOREBalance(ctx, voterAddr) math.Int
```

A stub implementation returns zero for all addresses. When the tokenomics module is deployed, it will satisfy this interface.

### Activation

QDRW starts disabled (`enabled: false`). It can be enabled via a governance parameter change. When disabled, standard linear voting power is used.

### QDRW Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `false` | Enable QDRW tally |
| `xqore_multiplier` | Dec | `2.0` | xQORE weight relative to staked tokens |
| `rep_min_multiplier` | Dec | `0.5` | Minimum reputation multiplier |
| `rep_max_multiplier` | Dec | `2.0` | Maximum reputation multiplier |

## Full QCA Configuration Reference

The complete `x/qca` configuration combines all subsystems:

| Section | Parameter | Type | Default |
|---------|-----------|------|---------|
| **Core** | `use_reputation_weighting` | bool | `true` |
| **Core** | `min_reputation_score` | float64 | `0.1` |
| **Pool** | `classification_interval` | uint64 | `1000` |
| **Pool** | `weight_rpos` | Dec | `0.40` |
| **Pool** | `weight_dpos` | Dec | `0.35` |
| **Pool** | `min_delegation_dpos` | uint64 | `10000000000` |
| **Pool** | `rep_percentile_rpos` | uint64 | `70` |
| **Bonding** | `alpha` | Dec | `0.1` |
| **Bonding** | `beta` | Dec | `1.0` |
| **Bonding** | `phase_multiplier` | Dec | `1.5` |
| **Slashing** | `base_rate` | Dec | `0.01` |
| **Slashing** | `escalation_factor` | Dec | `1.5` |
| **Slashing** | `max_penalty` | Dec | `0.33` |
| **Slashing** | `decay_halflife` | uint64 | `100000` |
| **QDRW** | `enabled` | bool | `false` |
| **QDRW** | `xqore_multiplier` | Dec | `2.0` |
| **QDRW** | `rep_min_multiplier` | Dec | `0.5` |
| **QDRW** | `rep_max_multiplier` | Dec | `2.0` |

## JSON-RPC Endpoint

### `qor_getPoolClassification`

Returns the current pool classification for a validator.

**Parameters:**

| Name | Type | Description |
|------|------|-------------|
| `validator` | string | Validator operator address |

**Response:**

| Field | Type | Description |
|-------|------|-------------|
| `validator` | string | Validator address |
| `pool` | string | Pool assignment: `rpos`, `dpos`, `pos`, or `unclassified` |
| `assigned_at` | int64 | Block height of last classification |

**Example:**

```bash
curl -X POST http://localhost:8545 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qor_getPoolClassification","params":["qorvaloper1..."]}'
```
