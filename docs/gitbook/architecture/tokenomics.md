# Tokenomics

QoreChain uses a dual-token economic model centered on the native **QOR** token, with a sophisticated burn-and-lock mechanism that drives long-term deflationary pressure while rewarding active participants.

---

## Token Basics

| Property | Value |
|---|---|
| **Display token** | QOR |
| **Base denomination** | uqor |
| **Decimal precision** | 10^6 (1 QOR = 1,000,000 uqor) |
| **Chain ID** | `qorechain-diana` |
| **Bech32 prefix** | `qor` (accounts: `qor1...`, validators: `qorvaloper...`) |

---

## x/burn -- Multi-Channel Burn Engine

The `x/burn` module implements a 10-channel token burn system. Every burned token is permanently removed from circulating supply, creating sustained deflationary pressure as network usage grows.

### Burn Channels

| # | Channel | Source | Description |
|---|---------|--------|-------------|
| 1 | `gas_fee` | Transaction fees | 30% of all gas fees are burned |
| 2 | `contract_create` | Smart contract deployment | Flat 100 QOR fee burned per contract creation |
| 3 | `ai_service` | AI module usage fees | 50% of AI service fees burned |
| 4 | `bridge_fee` | Cross-chain bridge fees | 100% of bridge fees burned |
| 5 | `treasury_buyback` | Treasury operations | Periodic buyback-and-burn from treasury |
| 6 | `failed_tx` | Failed transaction gas | 10% of gas from failed transactions burned |
| 7 | `xqore_penalty` | xQORE early exit penalties | Penalty amounts routed through burn |
| 8 | `auto_buyback` | Automated buyback program | Protocol-level automated burns |
| 9 | `tge` | Token generation event | One-time genesis burns |
| 10 | `rollup_create` | Rollup deployment | 1% of rollup creation stake burned |

### Fee Distribution

All transaction fees collected by the network are split across four destinations:

| Recipient | Share | Description |
|---|---|---|
| **Validators** | 40% | Distributed to the active validator set proportional to stake |
| **Burned** | 30% | Permanently removed from supply via `gas_fee` burn channel |
| **Treasury** | 20% | Allocated to the community treasury for governance-directed spending |
| **Stakers** | 10% | Distributed to all QOR stakers proportional to delegation |

The shares are enforced on-chain and must always sum to exactly 100%.

### Burn Parameters

| Parameter | Default | Description |
|---|---|---|
| `gas_burn_rate` | 0.30 | Fraction of gas fees burned (30%) |
| `contract_create_fee` | 100,000,000 uqor (100 QOR) | Flat burn fee for contract creation |
| `ai_service_burn_rate` | 0.50 | Fraction of AI service fees burned (50%) |
| `bridge_burn_rate` | 1.00 | Fraction of bridge fees burned (100%) |
| `failed_tx_burn_rate` | 0.10 | Fraction of failed TX gas burned (10%) |

Each burn event is recorded on-chain with its source, amount, block height, and associated transaction hash. Aggregate statistics are queryable per channel and in total.

---

## x/xqore -- Locked Staking and Governance Amplification

The `x/xqore` module introduces **xQORE**, a non-transferable locked-staking derivative. Users lock QOR to mint xQORE at a 1:1 ratio. xQORE holders receive amplified governance power and a share of redistributed exit penalties.

### Lock Mechanism

- **Lock**: Send QOR to the xQORE module to mint xQORE at a 1:1 ratio.
- **Governance weight**: xQORE holders receive **2x governance voting power** compared to standard QOR stakers.
- **Non-transferable**: xQORE cannot be sent between accounts. It is bound to the locking address.

### Exit Penalty Schedule

Early withdrawal from xQORE incurs a penalty that decreases with lock duration:

| Lock Duration | Penalty Rate | Description |
|---|---|---|
| < 30 days | **50%** | Half of locked QOR is forfeited |
| 30 -- 90 days | **35%** | Significant penalty for short-term locks |
| 90 -- 180 days | **15%** | Reduced penalty for medium-term commitment |
| > 180 days | **0%** | Full withdrawal with no penalty |

### PvP Rebase Redistribution

Penalties collected from early exits are not simply destroyed. Instead, they follow a PvP (player-versus-player) rebase model:

1. **50%** of penalty amounts are burned (routed through `x/burn` via the `xqore_penalty` channel).
2. **50%** are redistributed pro-rata to all remaining xQORE holders.

This creates a positive-sum dynamic for long-term holders: every early exit increases the effective value of remaining xQORE positions. Rebases occur every **100 blocks**.

### xQORE Parameters

| Parameter | Default | Description |
|---|---|---|
| `governance_multiplier` | 2.0 | Voting power multiplier for xQORE holders |
| `min_lock_amount` | 1,000,000 uqor (1 QOR) | Minimum QOR required to lock |
| `penalty_burn_rate` | 0.50 | Fraction of exit penalties burned (50%) |
| `rebase_interval` | 100 blocks | Blocks between PvP rebase events |
| `enabled` | true | Module activation flag |

---

## x/inflation -- Epoch-Based Emission Schedule

The `x/inflation` module governs new QOR issuance through a declining emission schedule. Inflation is computed per epoch and distributed to stakers and validators.

### Emission Schedule

| Year | Annual Inflation Rate | Description |
|---|---|---|
| 1 | **17.5%** | Bootstrap phase -- high rewards to incentivize early validators |
| 2 | **11.0%** | Growth phase -- reduced emissions as network stabilizes |
| 3 | **7.0%** | Maturation phase |
| 4 | **7.0%** | Continued maturation |
| 5+ | **2.0%** | Perpetual tail emission for long-term security budget |

### Epoch Mechanics

- **Epoch length**: 17,280 blocks (~1 day at 5-second block times)
- **Blocks per year**: ~6,311,520
- New QOR is minted at the start of each epoch based on the current year's inflation rate and the total bonded supply.
- The epoch tracker records the current epoch number, current year, starting block, and cumulative tokens minted.

### Inflation Parameters

| Parameter | Default | Description |
|---|---|---|
| `schedule` | 5-tier declining | Year-indexed inflation rates (see table above) |
| `epoch_length` | 17,280 blocks | Blocks per emission epoch |
| `enabled` | true | Module activation flag |

---

## Deflationary Convergence

QoreChain is designed to reach a **net-deflationary crossover point** as network adoption grows:

```
Year 1-2:   Inflation (17.5% → 11%) dominates → net inflationary
Year 3-4:   Inflation (7%) declines, burn volume grows with usage → convergence
Year 5+:    Inflation (2%) is low; burn channels (gas, bridge, contracts, rollups)
            scale with transaction volume → net deflationary
```

The 10 burn channels ensure that every major network activity removes tokens from supply. As transaction volume, bridge usage, AI service calls, and rollup deployments increase, cumulative burn rates accelerate while inflation declines to a 2% floor.

---

## Module Lifecycle Order

The economic modules execute in a specific order during each block's `EndBlocker`:

```
x/burn → x/xqore → x/inflation → x/rlconsensus
```

1. **x/burn** processes pending burn records and updates aggregate statistics.
2. **x/xqore** executes PvP rebases (every `rebase_interval` blocks) and routes penalties to burn.
3. **x/inflation** mints new QOR per the emission schedule at epoch boundaries.
4. **x/rlconsensus** adjusts consensus parameters based on reinforcement learning signals.

This ordering ensures that burns are finalized before rebases, and rebases complete before new tokens are minted, maintaining consistent economic state transitions.
