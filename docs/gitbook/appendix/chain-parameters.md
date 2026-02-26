# Chain Parameters

Consolidated reference of all configurable module parameters in the QoreChain genesis. Parameters are grouped by module and can be queried at runtime with `qorechaind query <module> params`.

---

## PQC Module (`x/pqc`)

| Parameter                  | Type    | Default Value | Description                                           |
| -------------------------- | ------- | ------------- | ----------------------------------------------------- |
| `hybrid_signature_mode`    | uint    | `1`           | Enforcement mode: 0=off, 1=permissive, 2=mandatory   |
| `algorithm_registry`       | array   | ML-DSA-87, ML-KEM-1024 | Registered PQC algorithms with size constraints |
| `auto_register_enabled`    | bool    | `true`        | Auto-register PQC keys on first hybrid tx             |
| `migration_deadline_height`| uint64  | `0`           | Block height after which classical-only keys are rejected (0=disabled) |
| `migration_grace_period`   | uint64  | `100000`      | Blocks of warning before migration deadline            |

---

## AI Module (`x/ai`)

| Parameter                      | Type    | Default Value | Description                                       |
| ------------------------------ | ------- | ------------- | ------------------------------------------------- |
| `anomaly_weight_volume`        | string  | `0.30`        | Weight for volume anomaly in fraud scoring         |
| `anomaly_weight_velocity`      | string  | `0.25`        | Weight for velocity anomaly in fraud scoring       |
| `anomaly_weight_pattern`       | string  | `0.25`        | Weight for pattern anomaly in fraud scoring        |
| `anomaly_weight_network`       | string  | `0.20`        | Weight for network graph anomaly in fraud scoring  |
| `fraud_threshold_low`          | string  | `0.30`        | Score threshold for low-severity alert             |
| `fraud_threshold_medium`       | string  | `0.55`        | Score threshold for medium-severity alert          |
| `fraud_threshold_high`         | string  | `0.75`        | Score threshold for high-severity alert            |
| `fraud_threshold_critical`     | string  | `0.90`        | Score threshold for critical-severity alert        |
| `circuit_breaker_enabled`      | bool    | `true`        | Enable AI circuit breakers                         |

---

## Reputation Module (`x/reputation`)

| Parameter        | Type   | Default Value | Description                                              |
| ---------------- | ------ | ------------- | -------------------------------------------------------- |
| `alpha`          | string | `0.30`        | Weight for uptime score (S_i) in reputation formula      |
| `beta`           | string | `0.25`        | Weight for participation score (P_i)                     |
| `gamma`          | string | `0.25`        | Weight for community score (C_i)                         |
| `delta`          | string | `0.20`        | Weight for tenure score (T_i)                            |
| `decay_lambda`   | string | `0.01`        | Exponential time decay factor for historical scores      |

Reputation formula: `R_i = alpha * S_i + beta * P_i + gamma * C_i + delta * T_i` with exponential time decay applied per epoch.

---

## QCA Module (`x/qca`)

| Parameter                        | Type   | Default Value | Description                                        |
| -------------------------------- | ------ | ------------- | -------------------------------------------------- |
| `emerald_pool_weight`            | string | `0.50`        | Block proposal weight for Emerald pool              |
| `sapphire_pool_weight`           | string | `0.30`        | Block proposal weight for Sapphire pool             |
| `ruby_pool_weight`               | string | `0.20`        | Block proposal weight for Ruby pool                 |
| `emerald_min_reputation`         | string | `0.80`        | Minimum reputation score for Emerald pool           |
| `sapphire_min_reputation`        | string | `0.50`        | Minimum reputation score for Sapphire pool          |
| `bonding_curve_base_rate`        | string | `0.05`        | Base rate for staking bonding curve                  |
| `bonding_curve_multiplier`       | string | `1.50`        | Multiplier for bonding curve progression             |
| `slashing_downtime_window`       | int64  | `10000`       | Blocks to assess downtime                            |
| `slashing_downtime_threshold`    | string | `0.05`        | Minimum signed blocks fraction before slashing       |
| `slashing_downtime_penalty`      | string | `0.01`        | Slash fraction for downtime                          |
| `slashing_double_sign_penalty`   | string | `0.05`        | Slash fraction for double signing                    |
| `qdrw_enabled`                   | bool   | `true`        | Enable Dynamic Reward Weighting                      |
| `qdrw_throughput_weight`         | string | `0.40`        | QDRW weight for throughput metric                    |
| `qdrw_latency_weight`            | string | `0.30`        | QDRW weight for latency metric                       |
| `qdrw_security_weight`           | string | `0.20`        | QDRW weight for security metric                      |
| `qdrw_decentralization_weight`   | string | `0.10`        | QDRW weight for decentralization metric              |
| `qdrw_adjustment_cap`            | string | `0.10`        | Maximum single-epoch QDRW adjustment                 |
| `qdrw_adjustment_interval`       | int64  | `100`         | Blocks between QDRW adjustments                      |

---

## Burn Module (`x/burn`)

| Parameter         | Type   | Default Value | Description                                         |
| ----------------- | ------ | ------------- | --------------------------------------------------- |
| `burn_enabled`    | bool   | `true`        | Enable fee burn mechanism                            |
| `validator_share` | string | `0.40`        | Fraction of fees distributed to block validators     |
| `burn_share`      | string | `0.30`        | Fraction of fees permanently burned                  |
| `treasury_share`  | string | `0.20`        | Fraction of fees sent to community treasury          |
| `staker_share`    | string | `0.10`        | Fraction of fees distributed to delegators           |

Shares must sum to `1.00`.

---

## xQORE Module (`x/xqore`)

| Parameter                | Type   | Default Value | Description                                      |
| ------------------------ | ------ | ------------- | ------------------------------------------------ |
| `min_lock_amount`        | string | `1000000uqor` | Minimum amount to lock as xQORE                  |
| `min_lock_duration`      | string | `7d`          | Minimum lock duration                             |
| `max_lock_duration`      | string | `365d`        | Maximum lock duration                             |
| `penalty_tier_1_pct`     | string | `0.50`        | Early unlock penalty: 0-25% of lock elapsed       |
| `penalty_tier_2_pct`     | string | `0.30`        | Early unlock penalty: 25-50% of lock elapsed      |
| `penalty_tier_3_pct`     | string | `0.15`        | Early unlock penalty: 50-75% of lock elapsed      |
| `penalty_tier_4_pct`     | string | `0.05`        | Early unlock penalty: 75-100% of lock elapsed     |
| `pvp_rebase_enabled`     | bool   | `true`        | Enable PvP rebase redistribution                  |

---

## Inflation Module (`x/inflation`)

| Parameter            | Type    | Default Value   | Description                                       |
| -------------------- | ------- | --------------- | ------------------------------------------------- |
| `epoch_length`       | uint64  | `100`           | Blocks per inflation epoch                         |
| `blocks_per_year`    | uint64  | `6311520`       | Estimated blocks per year (for rate calculation)   |
| `initial_rate`       | string  | `0.08`          | Initial annualized inflation rate (8%)             |
| `rate_decay`         | string  | `0.05`          | Decay factor applied each year                     |
| `min_rate`           | string  | `0.02`          | Floor inflation rate (2%)                          |
| `max_supply`         | string  | `1000000000000000uqor` | Maximum token supply cap                 |

Rate schedule: inflation decays annually from `initial_rate` by `rate_decay` until reaching `min_rate`.

---

## RL Consensus Module (`x/rlconsensus`)

| Parameter                    | Type    | Default Value | Description                                         |
| ---------------------------- | ------- | ------------- | --------------------------------------------------- |
| `observation_interval`       | uint64  | `10`          | Blocks between RL observation samples                |
| `agent_mode`                 | uint    | `0`           | Agent mode: 0=off, 1=observe, 2=suggest, 3=auto     |
| `circuit_breaker_enabled`    | bool    | `true`        | Enable RL circuit breaker                            |
| `circuit_breaker_max_change` | string  | `0.10`        | Maximum parameter change per action (10%)            |
| `circuit_breaker_cooldown`   | uint64  | `100`         | Blocks to wait after circuit breaker trips           |
| `reward_throughput_weight`   | string  | `0.40`        | Reward weight for throughput                         |
| `reward_latency_weight`      | string  | `0.30`        | Reward weight for latency                            |
| `reward_security_weight`     | string  | `0.20`        | Reward weight for security                           |
| `reward_stability_weight`    | string  | `0.10`        | Reward weight for stability                          |
| `ppo_learning_rate`          | string  | `0.0003`      | PPO learning rate                                    |
| `ppo_clip_range`             | string  | `0.20`        | PPO clipping range                                   |

---

## Bridge Module (`x/bridge`)

| Parameter                       | Type    | Default Value | Description                                     |
| ------------------------------- | ------- | ------------- | ----------------------------------------------- |
| `min_confirmations_ibc`         | uint64  | `1`           | Minimum confirmations for IBC transfers          |
| `min_confirmations_ethereum`    | uint64  | `12`          | Minimum confirmations for Ethereum bridge        |
| `min_confirmations_bitcoin`     | uint64  | `6`           | Minimum confirmations for Bitcoin bridge         |
| `circuit_breaker_enabled`       | bool    | `true`        | Enable bridge circuit breaker                    |
| `circuit_breaker_max_daily_usd` | string  | `10000000`    | Maximum daily bridge volume (USD equivalent)     |
| `circuit_breaker_max_single_tx` | string  | `1000000`     | Maximum single transfer amount (USD equivalent)  |

---

## Multilayer Module (`x/multilayer`)

| Parameter                   | Type    | Default Value | Description                                        |
| --------------------------- | ------- | ------------- | -------------------------------------------------- |
| `max_sidechains`            | uint    | `10`          | Maximum number of registered sidechains             |
| `max_paychains`             | uint    | `50`          | Maximum number of registered paychains              |
| `anchor_interval_sidechain` | uint64  | `100`         | Mandatory anchor interval for sidechains (blocks)   |
| `anchor_interval_paychain`  | uint64  | `50`          | Mandatory anchor interval for paychains (blocks)    |
| `challenge_period`          | string  | `7d`          | Duration for fraud challenges on anchors            |
| `min_sidechain_stake`       | string  | `100000000000uqor` | Minimum stake to register a sidechain         |
| `min_paychain_stake`        | string  | `10000000000uqor`  | Minimum stake to register a paychain          |
| `routing_threshold`         | string  | `0.80`        | Load threshold to trigger automatic routing         |

---

## Cross-VM Module (`x/crossvm`)

| Parameter            | Type    | Default Value | Description                                       |
| -------------------- | ------- | ------------- | ------------------------------------------------- |
| `max_message_size`   | uint64  | `65536`       | Maximum cross-VM message size in bytes (64 KB)     |
| `max_queue_size`     | uint    | `1000`        | Maximum pending messages in the cross-VM queue     |
| `queue_timeout`      | uint64  | `100`         | Blocks before a pending message is timed out       |

---

## SVM Module (`x/svm`)

| Parameter                      | Type    | Default Value | Description                                      |
| ------------------------------ | ------- | ------------- | ------------------------------------------------ |
| `max_program_size`             | uint64  | `10485760`    | Maximum program binary size in bytes (10 MB)      |
| `compute_budget`               | uint64  | `1400000`     | Default compute units per transaction (1.4M)      |
| `rent_lamports_per_byte_year`  | uint64  | `3480`        | Annual rent cost per byte in lamports              |
| `rent_exemption_threshold`     | string  | `2.0`         | Years of rent required for exemption               |
| `max_accounts_per_tx`          | uint    | `64`          | Maximum accounts referenced per transaction        |

---

## RDK Module (`x/rdk`)

| Parameter              | Type    | Default Value     | Description                                       |
| ---------------------- | ------- | ----------------- | ------------------------------------------------- |
| `max_rollups`          | uint    | `100`             | Maximum registered rollups                         |
| `min_stake`            | string  | `10000000000uqor` | Minimum operator stake (10,000 QOR)                |
| `burn_rate`            | string  | `0.01`            | Percentage of rollup fees burned (1%)              |
| `challenge_window`     | string  | `7d`              | Duration of the fraud challenge window             |
| `max_blob_size`        | uint64  | `2097152`         | Maximum DA blob size in bytes (2 MB)               |
| `blob_retention`       | uint64  | `432000`          | Blocks to retain DA blobs before pruning           |
| `max_batches_pending`  | uint    | `10`              | Maximum unfinalized batches per rollup             |
| `auto_finalize`        | bool    | `true`            | Enable EndBlocker auto-finalization                |
| `settlement_types`     | array   | optimistic, zk, pessimistic, sovereign | Allowed settlement paradigms |
| `preset_profiles`      | array   | defi, gaming, nft, social, general     | Available rollup presets     |

---

## FairBlock Module (`x/fairblock`)

| Parameter            | Type    | Default Value | Description                                       |
| -------------------- | ------- | ------------- | ------------------------------------------------- |
| `enabled`            | bool    | `false`       | Enable FairBlock tIBE encryption                   |
| `tibe_threshold`     | uint    | `2`           | Minimum decryption key shares required              |
| `decryption_delay`   | uint64  | `1`           | Blocks after finalization before decryption         |
| `max_encrypted_size` | uint64  | `4096`        | Maximum encrypted payload size in bytes             |

---

## Gas Abstraction Module (`x/gasabstraction`)

| Parameter         | Type  | Default Value | Description                                          |
| ----------------- | ----- | ------------- | ---------------------------------------------------- |
| `accepted_tokens` | array | (see below)   | Tokens accepted for gas payment with conversion rates |

**Default accepted tokens:**

| Token Denom  | Conversion Rate | Description               |
| ------------ | --------------- | ------------------------- |
| `uqor`       | `1.0`           | Native QOR token (1:1)    |
| `ibc/USDC`   | `1.0`           | IBC-bridged USDC          |
| `ibc/ATOM`   | `10.0`          | IBC-bridged ATOM          |

Conversion rates represent the number of gas units per token unit. Higher rates mean each token unit covers more gas.
