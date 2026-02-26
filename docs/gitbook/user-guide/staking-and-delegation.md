# Staking and Delegation

This guide covers how to delegate QOR tokens to validators, redelegate between validators, unbond your stake, claim rewards, and understand QoreChain's Triple-Pool staking architecture.

---

## Delegating Tokens

Delegate QOR to a validator to earn staking rewards and participate in network security:

```bash
qorechaind tx staking delegate <validator_address> <amount>uqor \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

**Example:** Delegate 100 QOR to a validator:

```bash
qorechaind tx staking delegate qorvaloper1abc...xyz 100000000uqor \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

---

## Redelegating

Move your delegation from one validator to another without waiting for the unbonding period:

```bash
qorechaind tx staking redelegate <source_validator> <destination_validator> <amount>uqor \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

**Example:**

```bash
qorechaind tx staking redelegate qorvaloper1src... qorvaloper1dst... 50000000uqor \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

> **Note:** You cannot redelegate tokens that are already in a redelegation transit. Wait for the current redelegation to complete before initiating another.

---

## Unbonding

Withdraw your delegated tokens from a validator. Unbonding takes **21 days** to complete, during which the tokens do not earn rewards and cannot be transferred.

```bash
qorechaind tx staking unbond <validator_address> <amount>uqor \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

**Example:**

```bash
qorechaind tx staking unbond qorvaloper1abc...xyz 25000000uqor \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

After the 21-day unbonding period, tokens are automatically returned to your account.

---

## Claiming Rewards

Withdraw all accumulated staking rewards from every validator you have delegated to:

```bash
qorechaind tx distribution withdraw-all-rewards \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

To withdraw rewards from a specific validator only:

```bash
qorechaind tx distribution withdraw-rewards <validator_address> \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

---

## Triple-Pool Classification

QoreChain uses a **Triple-Pool** staking model that classifies validators into three pools based on their reputation and delegation levels. Each pool receives a weighted share of block rewards.

| Pool | Entry Criteria | Reward Weight |
|------|---------------|---------------|
| **RPoS** (Reputation Proof of Stake) | Reputation score >= 70th percentile **AND** stake >= median | 40% |
| **DPoS** (Delegated Proof of Stake) | Total delegation >= 10,000 QOR | 35% |
| **PoS** (Proof of Stake) | All remaining validators | 25% |

Validators are reclassified at every epoch boundary. A validator that builds a strong reputation and accumulates sufficient stake is promoted to the RPoS pool, earning the highest reward share.

---

## Bonding Curve Rewards

Individual staking rewards are computed using QoreChain's bonding curve formula:

```
R = beta * S * (1 + alpha * log(1 + L)) * Q(r) * P(t)
```

| Variable | Description |
|----------|-------------|
| `R` | Reward amount for the period |
| `beta` | Base reward rate (protocol parameter) |
| `S` | Staked amount |
| `alpha` | Loyalty coefficient (protocol parameter) |
| `L` | Lock duration in epochs |
| `Q(r)` | Quality multiplier derived from the validator's reputation score `r` |
| `P(t)` | Pool multiplier at time `t` (40%, 35%, or 25% depending on pool) |

Longer lock durations and higher reputation scores result in proportionally greater rewards, incentivizing long-term commitment and good validator behavior.

---

## Querying Validator Information

Look up details about any validator:

```bash
qorechaind query staking validator <validator_operator_address>
```

**Example:**

```bash
qorechaind query staking validator qorvaloper1abc...xyz
```

List all active validators:

```bash
qorechaind query staking validators --status bonded
```

Query your current delegations:

```bash
qorechaind query staking delegations <delegator_address>
```

---

## Tips

- Delegating to validators in the **RPoS pool** yields the highest rewards due to the 40% pool weight.
- Building validator reputation takes time. Consider the validator's track record before delegating.
- Redelegation is instant but has cooldown restrictions. Plan your moves carefully.
- The 21-day unbonding period is a security measure. During this time, slashing events can still affect your tokens.
