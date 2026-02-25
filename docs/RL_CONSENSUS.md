# QoreChain RL Consensus Module

## Overview

The `x/rlconsensus` module implements reinforcement learning-based dynamic consensus parameter tuning for QoreChain. It uses a Go-native fixed-point multi-layer perceptron (MLP) to observe chain state and propose adjustments to block time, gas pricing, and pool weights. All arithmetic is deterministic, using integer fixed-point math (scaled by 10^8) to avoid non-deterministic floating-point operations across validators.

The module operates on a periodic cycle: every N blocks (default 10), it collects a 25-dimensional observation vector, runs the MLP policy network forward pass, and applies (or logs) the resulting 5-dimensional action vector to consensus parameters.

## Architecture

```
                      +---------------------------+
                      |     EndBlocker (ABCI)     |
                      +---------------------------+
                                 |
                      +----------+-----------+
                      |  Observation Collector |
                      |  (25 dimensions)       |
                      +----------+-----------+
                                 |
                      +----------+-----------+
                      |   MLP Forward Pass    |
                      |  25 -> 256 -> 256 -> 5|
                      |  ReLU + Tanh output    |
                      +----------+-----------+
                                 |
                      +----------+-----------+
                      |   Action Applicator   |
                      |  (clamp + apply)       |
                      +----------+-----------+
                                 |
                 +---------------+----------------+
                 |                                |
      +----------+---------+          +-----------+----------+
      |   Reward Compute   |          |   Circuit Breaker    |
      | (multi-objective)  |          | (revert + pause)     |
      +--------------------+          +----------------------+
```

The MLP architecture is 25 input neurons, two hidden layers of 256 neurons each (ReLU activation), and 5 output neurons (tanh activation), totaling 73,733 trainable parameters. Weights are stored on-chain as int64 fixed-point values and can be updated via governance transaction.

## Observation Vector

The observation vector captures 25 dimensions of chain state collected at the current block height.

| Index | Name | Description |
|-------|------|-------------|
| 0 | `block_utilization` | Block gas used / block gas limit |
| 1 | `tx_count` | Number of transactions in block |
| 2 | `avg_tx_size` | Mean transaction size (bytes) |
| 3 | `block_time` | Time since previous block (ms) |
| 4 | `block_time_delta` | Block time minus target block time (ms) |
| 5 | `gas_price_50th` | Median gas price |
| 6 | `gas_price_95th` | 95th-percentile gas price |
| 7 | `mempool_size` | Number of pending transactions |
| 8 | `mempool_bytes` | Total bytes of pending transactions |
| 9 | `validator_count` | Active validator count |
| 10 | `validator_gini` | Gini coefficient of validator power |
| 11 | `missed_block_ratio` | Fraction of validators that missed signing |
| 12 | `avg_commit_latency` | Average commit round latency (ms) |
| 13 | `max_commit_latency` | Maximum commit round latency (ms) |
| 14 | `precommit_ratio` | Fraction of precommits received |
| 15 | `failed_tx_ratio` | Fraction of failed transactions |
| 16 | `avg_gas_per_tx` | Mean gas consumed per transaction |
| 17 | `reward_per_validator` | Mean reward per validator (uqor) |
| 18 | `slash_count` | Number of slashing events in window |
| 19 | `jail_count` | Number of jail events in window |
| 20 | `inflation_rate` | Current inflation rate |
| 21 | `bonded_ratio` | Bonded tokens / total supply |
| 22 | `reputation_mean` | Mean reputation score across validators |
| 23 | `reputation_std_dev` | Standard deviation of reputation scores |
| 24 | `mev_estimate` | Estimated MEV extracted (heuristic) |

All values are stored as `LegacyDec` string representations for deterministic serialization, then converted to int64 fixed-point for the MLP forward pass.

## Action Space

The MLP outputs a 5-dimensional action vector. Each value is in [-1, 1] (tanh output) and is interpreted as a proposed delta to the corresponding consensus parameter.

| Index | Name | Description |
|-------|------|-------------|
| 0 | `block_time_delta` | Proposed change to target block time (ms) |
| 1 | `gas_price_delta` | Proposed change to base gas price floor |
| 2 | `validator_set_size_delta` | Proposed change to validator set size (logged only, not directly applied) |
| 3 | `pool_weight_rpos_delta` | Proposed change to RPoS pool selection weight |
| 4 | `pool_weight_dpos_delta` | Proposed change to DPoS pool selection weight |

Actions are clamped to the maximum change allowed by the current agent mode before application. Block time is bounded to [1000, 30000] ms. Pool weights are bounded to [0.05, 0.80].

## Reward Function

The reward signal is computed from consecutive observation pairs:

```
R = w1 * delta_throughput + w2 * delta_finality + w3 * delta_decentralization - w4 * mev - w5 * failed_txs
```

| Component | Weight (default) | Derivation |
|-----------|-----------------|------------|
| `delta_throughput` | 0.30 | Change in block utilization (higher is better) |
| `delta_finality` | 0.25 | Improvement in precommit ratio (higher is better) |
| `delta_decentralization` | 0.20 | Reduction in Gini coefficient (lower Gini = positive reward) |
| `mev` | 0.15 | Current MEV estimate (penalized) |
| `failed_txs` | 0.10 | Current failed transaction ratio (penalized) |

Weights must sum to 1.0 and can be updated via the `update-reward-weights` transaction.

## Agent Modes

The RL agent operates in one of four modes:

| Mode | Value | Behavior |
|------|-------|----------|
| **Shadow** | 0 | Collects observations, runs inference, logs proposed actions. No parameters are changed. This is the default mode. |
| **Conservative** | 1 | Applies actions with tight bounds (default max change: 10% per cycle). |
| **Autonomous** | 2 | Applies actions with wider bounds (default max change: 25% per cycle). |
| **Paused** | 3 | No observation collection or inference. Module is fully idle. |

Mode transitions are controlled via governance transactions (`MsgSetAgentMode`). The circuit breaker can automatically pause the agent.

## Circuit Breaker

The circuit breaker monitors block production health. It tracks the last N block time deltas (default window: 50 blocks) and counts how many are "healthy" (within 2x of target block time).

If the fraction of healthy blocks falls below the threshold (default: 50%), the circuit breaker triggers:

1. All RL-tuned parameters are reverted to their defaults
2. The agent status is flagged as circuit-breaker-active
3. An `rlconsensus_circuit_breaker_triggered` event is emitted

The agent remains paused until manually resumed via `MsgResumeAgent`.

## Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Master enable switch for the module |
| `observation_interval` | uint64 | `10` | Blocks between observation collections |
| `agent_mode` | uint8 | `0` (shadow) | Current operating mode |
| `max_change_conservative` | Dec | `0.10` | Maximum parameter change per cycle in conservative mode |
| `max_change_autonomous` | Dec | `0.25` | Maximum parameter change per cycle in autonomous mode |
| `circuit_breaker_window` | uint64 | `50` | Number of recent blocks monitored by circuit breaker |
| `circuit_breaker_threshold` | Dec | `0.50` | Minimum healthy block fraction before trigger |
| `default_block_time_ms` | int64 | `5000` | Default target block time (ms) |
| `default_base_gas_price` | Dec | `100` | Default base gas price |
| `default_validator_set_size` | uint64 | `100` | Default target validator set size |
| `reward_weights.throughput` | Dec | `0.30` | Reward weight for throughput |
| `reward_weights.finality` | Dec | `0.25` | Reward weight for finality |
| `reward_weights.decentralization` | Dec | `0.20` | Reward weight for decentralization |
| `reward_weights.mev` | Dec | `0.15` | Reward weight for MEV (penalized) |
| `reward_weights.failed_txs` | Dec | `0.10` | Reward weight for failed transactions (penalized) |

## Deterministic Math Utilities

The `mathutil` package provides deterministic implementations of mathematical functions used across both `x/rlconsensus` and `x/qca`. All functions use `cosmossdk.io/math.LegacyDec` to avoid non-deterministic floating-point arithmetic.

- **IntegerSqrt** -- Newton's method square root (100-iteration convergence)
- **TaylorLn1PlusX** -- Natural logarithm via argument reduction + 15-term Taylor series
- **ExpApprox** -- Exponential function via 12-term Taylor series
- **SigmoidApprox** -- Sigmoid function using ExpApprox, with symmetry for negative inputs
- **ReputationMultiplier** -- Maps reputation [0, 1] to multiplier [0.5, 2.0] via sigmoid curve

The MLP uses separate fixed-point arithmetic (`fixMul`) with overflow-safe split multiplication and a Pade-approximant tanh activation.

## CLI Commands

### Query Commands

```bash
# Query the current RL agent status
qorechaind query rlconsensus agent-status

# Query the latest observation vector
qorechaind query rlconsensus observation

# Query the latest reward signal
qorechaind query rlconsensus reward

# Query module parameters
qorechaind query rlconsensus params

# Query current policy network weights
qorechaind query rlconsensus policy
```

### Transaction Commands

```bash
# Set the agent operating mode
qorechaind tx rlconsensus set-agent-mode [shadow|conservative|autonomous|paused] --from mykey

# Resume the agent from paused state (returns to shadow mode)
qorechaind tx rlconsensus resume-agent --from mykey

# Update the policy network weights from a JSON file
qorechaind tx rlconsensus update-policy ./weights.json --from mykey

# Update the reward function weights (must sum to 1.0)
qorechaind tx rlconsensus update-reward-weights 0.30 0.25 0.20 0.15 0.10 --from mykey
```

## JSON-RPC Endpoints

The following `qor_` namespace JSON-RPC methods are available on port 8545:

### `qor_getRLAgentStatus`

Returns the current RL agent operational status.

**Parameters:** none

**Response:**

| Field | Type | Description |
|-------|------|-------------|
| `agent_mode` | string | Current mode: `shadow`, `conservative`, `autonomous`, or `paused` |
| `current_epoch` | uint64 | Policy training epoch |
| `is_active` | bool | Whether the agent is actively collecting observations |
| `circuit_breaker_active` | bool | Whether the circuit breaker has been triggered |

### `qor_getRLObservation`

Returns the latest observation vector with human-readable dimension names.

**Parameters:** none

**Response:**

| Field | Type | Description |
|-------|------|-------------|
| `height` | int64 | Block height at which the observation was collected |
| `dimensions` | map[string]string | Named observation dimensions and their LegacyDec values |

### `qor_getRLReward`

Returns the latest reward signal with per-component breakdown.

**Parameters:** none

**Response:**

| Field | Type | Description |
|-------|------|-------------|
| `height` | int64 | Block height |
| `total_reward` | string | Weighted total reward (LegacyDec) |
| `throughput_delta` | string | Change in block utilization |
| `finality_delta` | string | Change in precommit ratio |
| `decentralization_delta` | string | Negated change in Gini coefficient |
| `mev_estimate` | string | Current MEV estimate |
| `failed_tx_ratio` | string | Current failed transaction ratio |

### Example

```bash
curl -X POST http://localhost:8545 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qor_getRLAgentStatus","params":[]}'
```

## REST Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/rlconsensus/v1/agent` | GET | Agent status |
| `/qorechain/rlconsensus/v1/observation` | GET | Latest observation vector |
| `/qorechain/rlconsensus/v1/rewards` | GET | Reward history |
| `/qorechain/rlconsensus/v1/params` | GET | Module parameters |
| `/qorechain/rlconsensus/v1/policy` | GET | Current policy metadata |

## Genesis Configuration

The module is initialized in genesis under the `rlconsensus` key:

```json
{
  "rlconsensus": {
    "params": {
      "enabled": true,
      "observation_interval": 10,
      "agent_mode": 0,
      "max_change_conservative": "0.10",
      "max_change_autonomous": "0.25",
      "circuit_breaker_window": 50,
      "circuit_breaker_threshold": "0.50",
      "reward_weights": {
        "throughput": "0.30",
        "finality": "0.25",
        "decentralization": "0.20",
        "mev": "0.15",
        "failed_txs": "0.10"
      },
      "default_block_time_ms": 5000,
      "default_base_gas_price": "100",
      "default_validator_set_size": 100
    },
    "agent_status": {
      "mode": 0,
      "current_epoch": 0,
      "total_steps": 0,
      "last_observation_at": 0,
      "last_action_at": 0,
      "circuit_breaker_active": false,
      "blocks_since_revert": 0
    },
    "policy_weights": null
  }
}
```

The agent starts in shadow mode with no policy weights loaded. Once weights are uploaded via `MsgUpdatePolicy`, the agent begins inference on the next observation cycle.
