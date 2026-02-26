# RL Consensus Engine

QoreChain embeds a **reinforcement learning (RL) agent** directly into the consensus layer via the `x/rlconsensus` module. The agent observes chain metrics every N blocks, runs inference through a fixed-point neural network, and proposes consensus parameter adjustments -- all deterministically, with no floating-point arithmetic in consensus-critical paths.

---

## Architecture Overview

The RL engine consists of four components:

1. **Observation Collector** -- Gathers 25-dimensional chain state vectors at configurable intervals.
2. **Policy Network (MLP)** -- A Go-native multi-layer perceptron that maps observations to actions.
3. **Reward Computer** -- Evaluates the quality of parameter changes using a weighted multi-objective function.
4. **Circuit Breaker** -- Monitors chain health and reverts all RL-tuned parameters if instability is detected.

All components operate within the ABCI lifecycle and produce deterministic, verifiable outputs across all validator nodes.

---

## Policy Network

The policy network is a feedforward multi-layer perceptron (MLP) implemented entirely in Go with **int64 fixed-point arithmetic** (scaled by 10^8).

### Network Architecture

| Property | Value |
|----------|-------|
| Input dimensions | 25 |
| Hidden layers | 2 |
| Hidden layer sizes | 256, 256 |
| Output dimensions | 5 |
| Activation (hidden) | ReLU |
| Activation (output) | tanh |
| Total parameters | 73,733 |
| Precision | int64 fixed-point (scaled by 10^8) |

### Parameter Count Breakdown

```
Layer 1: 25 * 256 + 256   =  6,656  (input -> hidden_1)
Layer 2: 256 * 256 + 256   = 65,792  (hidden_1 -> hidden_2)
Layer 3: 256 * 5 + 5       =  1,285  (hidden_2 -> output)
Total:                       73,733
```

### Fixed-Point Arithmetic

All MLP computations use `int64` values scaled by `FixedPointScale = 10^8`. This eliminates non-determinism from IEEE 754 floating-point rounding differences across hardware platforms.

- **Multiplication**: `fixMul(a, b) = (a / SCALE) * b + (a % SCALE) * b / SCALE` (split to prevent overflow)
- **ReLU**: `relu(x) = max(0, x)`
- **tanh**: Pade approximant `tanh(x) ~ x * (3*S - x^2) / (3*S + x^2)` for |x| <= 2.5*SCALE, clamped to +/- SCALE otherwise

Policy weights are stored on-chain as a flattened `[]int64` vector and can be updated via governance proposal.

---

## Observation Vector

The agent collects a 25-dimensional observation vector at each observation interval (default: every 10 blocks).

| Index | Dimension | Description |
|-------|-----------|-------------|
| 0 | `block_utilization` | Block gas used / block gas limit |
| 1 | `tx_count` | Number of transactions in the block |
| 2 | `avg_tx_size` | Mean transaction size in bytes |
| 3 | `block_time` | Time since previous block (ms) |
| 4 | `block_time_delta` | Block time minus target block time (ms) |
| 5 | `gas_price_50th` | Median gas price |
| 6 | `gas_price_95th` | 95th-percentile gas price |
| 7 | `mempool_size` | Number of pending transactions |
| 8 | `mempool_bytes` | Total bytes of pending transactions |
| 9 | `validator_count` | Active validator count |
| 10 | `validator_gini` | Gini coefficient of validator power distribution |
| 11 | `missed_block_ratio` | Fraction of validators that missed signing |
| 12 | `avg_commit_latency` | Average commit round latency (ms) |
| 13 | `max_commit_latency` | Maximum commit round latency (ms) |
| 14 | `precommit_ratio` | Fraction of precommits received |
| 15 | `failed_tx_ratio` | Fraction of failed transactions |
| 16 | `avg_gas_per_tx` | Mean gas consumed per transaction |
| 17 | `reward_per_validator` | Mean reward per validator (uqor) |
| 18 | `slash_count` | Number of slashing events in observation window |
| 19 | `jail_count` | Number of jail events in observation window |
| 20 | `inflation_rate` | Current inflation rate |
| 21 | `bonded_ratio` | Bonded tokens / total supply |
| 22 | `reputation_mean` | Mean reputation score across active validators |
| 23 | `reputation_stddev` | Standard deviation of reputation scores |
| 24 | `mev_estimate` | Estimated MEV extracted (heuristic) |

All values are stored as `LegacyDec` string representations and converted to int64 fixed-point before inference.

---

## Action Space

The MLP output is a 5-dimensional action vector, where each dimension represents a proposed change to a consensus parameter. The tanh activation constrains raw outputs to [-1, 1], which are then scaled by mode-specific bounds.

| Index | Action Dimension | Description |
|-------|-----------------|-------------|
| 0 | `block_time_delta` | Proposed change to target block time (ms) |
| 1 | `gas_price_delta` | Proposed change to base gas price |
| 2 | `validator_set_size_delta` | Proposed change to target validator set size (logged only, not applied) |
| 3 | `pool_weight_rpos_delta` | Proposed change to RPoS pool priority weight |
| 4 | `pool_weight_dpos_delta` | Proposed change to DPoS pool priority weight |

Actions are **clamped** to the maximum change bounds defined by the current agent mode before application.

---

## Reward Function

The reward signal evaluates how well recent parameter changes improved chain performance. It is computed as a weighted sum of five objectives:

```
R = 0.30 * delta_throughput
  + 0.25 * delta_finality
  + 0.20 * delta_decentralization
  - 0.15 * mev_estimate
  - 0.10 * failed_tx_ratio
```

| Component | Weight | Direction | Source Metric |
|-----------|--------|-----------|---------------|
| Throughput | +0.30 | Maximize | Change in block utilization |
| Finality | +0.25 | Maximize | Change in precommit ratio |
| Decentralization | +0.20 | Maximize | Negative change in validator Gini coefficient |
| MEV | -0.15 | Minimize | Current MEV estimate |
| Failed Transactions | -0.10 | Minimize | Current failed transaction ratio |

The reward weights are governance-configurable and must sum to exactly 1.0.

---

## Agent Modes

The RL agent operates in one of four modes, controllable via governance:

| Mode | ID | Max Change | Behavior |
|------|----|-----------|----------|
| **Shadow** | 0 | 0% | Observe and log recommendations only. No parameters are changed. This is the default mode. |
| **Conservative** | 1 | +/- 10% | Apply parameter changes within tight bounds. Suitable for initial live deployment. |
| **Autonomous** | 2 | +/- 25% | Apply parameter changes within wider bounds. For mature networks with validated policies. |
| **Paused** | 3 | 0% | Agent is completely idle. No observations are collected and no inference runs. |

Mode transitions require a governance proposal. The recommended deployment path is: Shadow --> Conservative --> Autonomous.

---

## Circuit Breaker

The circuit breaker is a safety mechanism that monitors chain health and automatically reverts all RL-tuned parameters if instability is detected.

### Detection Logic

The circuit breaker evaluates the last **50 blocks** (configurable via `circuit_breaker_window`):

1. For each consecutive pair of block timestamps, compute the block time delta.
2. A block is considered **healthy** if its delta is positive and within 2x the target block time.
3. Compute the **healthy fraction** = healthy blocks / total deltas.

### Trigger Condition

If the healthy fraction falls below the threshold (default: **50%**), the circuit breaker triggers.

### Response

When triggered, the circuit breaker:

1. **Reverts** all RL-applied parameters (block time, gas price, pool weights) to their default values.
2. **Pauses** the RL agent (sets `CircuitBreakerActive = true`).
3. **Clears** the in-memory agent to force a fresh reload.
4. **Emits** a `circuit_breaker_triggered` event.

The circuit breaker automatically clears when the healthy fraction recovers above the threshold on subsequent evaluations.

---

## Rollup Advisory Functions

The RL module provides advisory functions for rollup parameter optimization:

- **`SuggestRollupProfile`** -- Analyzes current chain conditions and suggests optimal rollup configuration parameters (block time, gas limit, settlement frequency).
- **`OptimizeRollupGas`** -- Recommends gas pricing adjustments for rollup settlement transactions based on main chain congestion patterns.

These functions are informational only and do not modify chain state.

---

## Deterministic Math Library

All RL consensus calculations use the `mathutil` package, which provides deterministic alternatives to standard floating-point math:

| Function | Description | Method |
|----------|-------------|--------|
| `IntegerSqrt(x)` | Square root | Newton's method on `LegacyDec`, 100-iteration convergence |
| `TaylorLn1PlusX(x)` | Natural logarithm `ln(1+x)` | Argument reduction + 15-term Taylor series |
| `ExpApprox(x)` | Exponential `e^x` | 12-term Taylor series |
| `SigmoidApprox(x)` | Sigmoid `1/(1+e^-x)` | `ExpApprox` with symmetry for negative inputs |
| `ReputationMultiplier(r)` | Maps [0,1] to [0.5,2.0] | Sigmoid with scale and offset |

All functions operate on `cosmossdk.io/math.LegacyDec` values, ensuring identical results across all hardware platforms and Go compiler versions.

---

## Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable the RL consensus engine |
| `observation_interval` | uint64 | `10` | Blocks between observation collections |
| `agent_mode` | AgentMode | `0` (Shadow) | Current operating mode |
| `max_change_conservative` | LegacyDec | `0.10` | Maximum parameter change in Conservative mode |
| `max_change_autonomous` | LegacyDec | `0.25` | Maximum parameter change in Autonomous mode |
| `circuit_breaker_window` | uint64 | `50` | Number of recent blocks monitored by circuit breaker |
| `circuit_breaker_threshold` | LegacyDec | `0.50` | Minimum healthy block fraction before trigger |
| `default_block_time_ms` | int64 | `5000` | Default target block time (ms) |
| `default_base_gas_price` | LegacyDec | `100` | Default base gas price |
| `default_validator_set_size` | uint64 | `100` | Default target validator set size |
| `reward_weight_throughput` | LegacyDec | `0.30` | Reward weight for throughput improvement |
| `reward_weight_finality` | LegacyDec | `0.25` | Reward weight for finality improvement |
| `reward_weight_decentralization` | LegacyDec | `0.20` | Reward weight for decentralization improvement |
| `reward_weight_mev` | LegacyDec | `0.15` | Penalty weight for MEV extraction |
| `reward_weight_failed_txs` | LegacyDec | `0.10` | Penalty weight for failed transactions |
