# xQORE Governance Staking

This guide covers the xQORE governance staking mechanism, which allows QOR holders to lock their tokens for enhanced governance power with a PvP rebase model that rewards long-term participants.

---

## Overview

xQORE is QoreChain's governance-staking token. When you lock QOR, you receive xQORE at a **1:1 ratio**. Holding xQORE provides a significant advantage in governance: xQORE tokens count at **double weight** in the QDRW voting power formula (see [Governance](governance.md)).

```
VP = sqrt(staked + 2 * xQORE) * ReputationMultiplier(r)
```

This means locking QOR into xQORE effectively doubles its governance impact compared to regular staking alone.

---

## Locking QOR for xQORE

Lock QOR tokens to mint xQORE at a 1:1 ratio:

```bash
qorechaind tx xqore lock <amount>uqor \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

**Example:** Lock 1,000 QOR:

```bash
qorechaind tx xqore lock 1000000000uqor \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

After this transaction, your account will hold 1,000,000,000 uxqore (1,000 xQORE).

---

## Unlocking xQORE

Burn xQORE to receive QOR back. An **exit penalty** may apply depending on how long the tokens have been locked:

```bash
qorechaind tx xqore unlock <amount>uqor \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

**Example:** Unlock 500 xQORE:

```bash
qorechaind tx xqore unlock 500000000uqor \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

---

## Exit Penalty Schedule

Early withdrawal from xQORE incurs a penalty. The longer you hold, the lower the penalty:

| Lock Duration | Exit Penalty |
|---------------|-------------|
| Less than 30 days | **50%** |
| 30 to 90 days | **35%** |
| 90 to 180 days | **15%** |
| More than 180 days | **0%** |

**Example:** If you locked 1,000 QOR and unlock after 45 days, you receive 650 QOR (35% penalty applied). The remaining 350 QOR is redistributed to other xQORE holders through the PvP rebase mechanism.

---

## PvP Rebase Mechanism

Penalties collected from early exits are **not burned**. Instead, they are redistributed proportionally to all remaining xQORE holders. This creates a "Player vs Player" dynamic where patient holders benefit from the impatience of others.

**How it works:**

1. A user unlocks xQORE before the 180-day zero-penalty threshold.
2. The exit penalty is deducted from their returned QOR.
3. The penalty amount is distributed proportionally across all remaining xQORE positions.
4. Each remaining holder's claimable QOR per xQORE increases.

This mechanism incentivizes long-term governance commitment and rewards holders who maintain their positions.

---

## Querying Your Position

Check your current xQORE position, lock duration, and applicable exit penalty:

```bash
qorechaind query xqore position <address>
```

**Example:**

```bash
qorechaind query xqore position qor1abc...xyz
```

**Sample output:**

```yaml
position:
  address: qor1abc...xyz
  locked_amount: "1000000000"
  xqore_balance: "1000000000"
  lock_timestamp: "2026-01-15T12:00:00Z"
  current_penalty_rate: "0.150000000000000000"
  accrued_rebase: "25000000"
```

---

## JSON-RPC Access

For applications integrating with QoreChain via JSON-RPC, the xQORE position can be queried using:

```
qor_getXQOREPosition
```

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "qor_getXQOREPosition",
  "params": ["qor1abc...xyz"],
  "id": 1
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "locked_amount": "1000000000",
    "xqore_balance": "1000000000",
    "lock_timestamp": "2026-01-15T12:00:00Z",
    "current_penalty_rate": "0.15",
    "accrued_rebase": "25000000"
  }
}
```

---

## Tips

- Lock QOR into xQORE well before important governance votes to maximize your voting power.
- The 180-day threshold for zero-penalty exits rewards patient governance participants.
- Monitor PvP rebase accruals. As others exit early, your position grows in value.
- xQORE is non-transferable. It can only be minted by locking QOR and burned by unlocking.
- Consider the exit penalty carefully before locking. Short-term locks carry significant penalties.
