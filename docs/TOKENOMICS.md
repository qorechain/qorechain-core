# QoreChain Tokenomics

## Overview

QoreChain implements a three-module tokenomics engine designed to create long-term deflationary pressure while incentivizing early adoption and sustained participation. The system combines burn accounting (x/burn), governance-boosted staking (x/xqore), and controlled inflation (x/inflation) into a unified economic model.

## Token

- **Display denomination**: QOR
- **Base denomination**: uqor (1 QOR = 10^6 uqor)
- **Chain ID**: qorechain-diana (testnet)
- **Bech32 prefix**: qor (addresses: `qor1...`, validators: `qorvaloper...`)

---

## x/burn — Central Burn Accounting

All QOR burns flow through the x/burn module, providing a single source of truth for supply reduction.

### Burn Sources

Nine distinct channels feed the burn module:

| Source | Description |
|--------|-------------|
| `tx_fee` | Portion of every transaction fee |
| `governance_penalty` | Governance-related slashing |
| `slashing_burn` | Validator slashing burns |
| `bridge_fee` | Cross-chain transfer fees |
| `spam_deterrent` | Anti-spam mechanism burns |
| `epoch_excess` | Surplus epoch revenue |
| `manual_burn` | Governance-initiated burns |
| `contract_callback` | Smart contract burn callbacks |
| `cross_vm_fee` | Cross-VM messaging fees |

### Fee Distribution

The x/burn EndBlocker splits collected fees every block:

| Recipient | Share | Purpose |
|-----------|-------|---------|
| Validators | 40% | Block production rewards via staking module |
| Burn | 30% | Permanently removed from supply |
| Treasury | 20% | Protocol development and community pool |
| Stakers | 10% | Additional delegation rewards |

### Burn Statistics

Real-time tracking via the keeper:
- `GetTotalBurned(ctx)` — cumulative QOR burned since genesis
- `GetBurnStats(ctx)` — total burned, per-source breakdown, last burn height
- `GetBurnRecords(ctx, limit)` — recent burn event log

### Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `burn_enabled` | true | Global burn enable/disable |
| `validator_share` | 0.40 | Validator fee share |
| `burn_share` | 0.30 | Burn fee share |
| `treasury_share` | 0.20 | Treasury fee share |
| `staker_share` | 0.10 | Staker fee share |

---

## x/xqore — Governance-Boosted Staking

xQORE is QoreChain's governance-boosted staking mechanism. Users lock QOR to mint xQORE at a 1:1 ratio, gaining enhanced governance power through the QDRW (Quadratic Delegation with Reputation Weighting) system.

### How It Works

1. **Lock**: User sends QOR to the x/xqore module account. xQORE tokens are minted 1:1.
2. **Hold**: xQORE doubles the holder's voting weight in QDRW governance: `VP = sqrt(staked + 2 * xQORE) * ReputationMultiplier(r)`
3. **Unlock**: User requests withdrawal. A graduated exit penalty applies based on lock duration.
4. **Rebase**: Penalties from early exits are redistributed to remaining xQORE holders (PvP rebase).

### Exit Penalty Schedule

| Lock Duration | Penalty Rate | Effect |
|--------------|-------------|--------|
| < 30 days | 50% | Half of locked QOR is confiscated |
| 30–90 days | 35% | Moderate penalty for mid-term exit |
| 90–180 days | 15% | Reduced penalty approaching maturity |
| > 180 days | 0% | Full withdrawal, no penalty |

### PvP Rebase

All exit penalties are redistributed proportionally to remaining xQORE holders. This creates a game-theoretic incentive:
- **Patient holders** earn additional QOR from impatient exits
- **Early exiters** lose a portion of their capital
- **Net effect**: Conviction is rewarded; mercenary capital is punished

### TokenomicsKeeper Integration

The x/xqore module satisfies the `rlconsensus.TokenomicsKeeper` interface via `GetXQOREBalance(ctx, addr)`, providing real balance data for QDRW governance voting power calculations. This replaces the previous `NilTokenomicsKeeper` stub.

### Position Tracking

Each xQORE position tracks:
- `Owner` — account address
- `Locked` — QOR amount locked
- `XBalance` — xQORE minted
- `LockHeight` — block height at lock time
- `LockTime` — UTC timestamp at lock time

---

## x/inflation — Epoch-Based Emission

The x/inflation module controls new QOR supply creation through an epoch-based emission schedule with year-over-year decay.

### Emission Schedule

| Year | Inflation Rate | Description |
|------|---------------|-------------|
| 1 | 17.5% | Bootstrap — aggressive incentives for early validators |
| 2 | 11.0% | Growth — reduced emission as network matures |
| 3–4 | 7.0% | Stabilization — converging toward sustainability |
| 5+ | 2.0% | Long-term — minimal new supply |

### Epoch Mechanics

- **Epoch length**: Configurable (default: 100 blocks)
- **Blocks per year**: Configurable (default: 6,311,520 at ~5s block time)
- **Minting**: Each epoch, the module mints `(annual_rate / epochs_per_year) * total_supply` new QOR
- **Distribution**: Minted tokens are sent to the fee collector for standard distribution

### Epoch Info

The keeper tracks:
- `CurrentEpoch` — epoch counter since genesis
- `CurrentYear` — derived from epoch count and blocks-per-year
- `BlockStart` — first block of the current epoch
- `TotalMinted` — cumulative QOR minted since genesis

### Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `epoch_length` | 100 | Blocks per epoch |
| `blocks_per_year` | 6311520 | Expected blocks per year |
| `year1_rate` | 0.175 | Year 1 inflation rate |
| `year2_rate` | 0.110 | Year 2 inflation rate |
| `year3_rate` | 0.070 | Year 3-4 inflation rate |
| `long_term_rate` | 0.020 | Year 5+ inflation rate |

---

## Economic Model

### Deflationary Convergence

The tokenomics engine is designed to reach net-deflationary equilibrium:

1. **Inflation decreases** year-over-year (17.5% → 2%)
2. **Burns increase** with network usage (more transactions = more fees burned)
3. **Crossover point**: When 30% of transaction fees burned exceeds new minting, QOR supply contracts

### Governance Alignment

The xQORE mechanism aligns governance power with long-term commitment:
- Locked QOR earns 2x voting weight in QDRW governance
- Exit penalties create a cost for governance mercenaries
- PvP rebase rewards patient participants
- Combined with reputation multiplier, power concentrates with honest long-term validators

---

## JSON-RPC Endpoints

Four tokenomics-specific endpoints are available in the `qor_` JSON-RPC namespace:

| Method | Parameters | Description |
|--------|-----------|-------------|
| `qor_getBurnStats` | (none) | Total burned, per-source breakdown, last burn height |
| `qor_getXQOREPosition` | `address` | xQORE position: locked QOR, xQORE balance, lock time |
| `qor_getInflationRate` | (none) | Current rate, epoch number, year, total minted |
| `qor_getTokenomicsOverview` | (none) | Combined dashboard: burn + xQORE + inflation stats |

### Example

```bash
# Get burn statistics
curl -X POST http://localhost:8545 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qor_getBurnStats","params":[]}'

# Get xQORE position for an address
curl -X POST http://localhost:8545 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qor_getXQOREPosition","params":["qor1..."]}'

# Get current inflation rate
curl -X POST http://localhost:8545 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qor_getInflationRate","params":[]}'

# Get combined tokenomics overview
curl -X POST http://localhost:8545 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qor_getTokenomicsOverview","params":[]}'
```

## Module Lifecycle

The three tokenomics modules run in a defined order in the block lifecycle:

```
BeginBlockers: ... → burn → xqore → inflation → rlconsensus → ...
EndBlockers:   ... → burn → xqore → inflation → rlconsensus → ...
```

This ordering ensures burn accounting processes first, then xQORE rebase, then inflation minting — preventing circular dependencies.
