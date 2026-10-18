# x/stakingtier

Delegation locks with a bonus yield, from v3.2.4 (protocol feature
`staking-tiers`). Below the v3.2.4 plan height the module writes nothing, emits
nothing and changes no gas, so history replays byte-identically. A chain that
starts on this binary has the feature on from genesis.

The ordinary staking yield is untouched: it still flows through the fee split
and x/distribution. This module pays a BONUS on top of it, per inflation epoch,
from its own module account.

## The bonuses

| Bonus | Default | Applies to |
| --- | --- | --- |
| 3 month lock | +1.5 APY points | the locked amount |
| 6 month lock | +3 APY points | the locked amount |
| 12 month lock | +5 APY points | the locked amount |
| Light node | +2 APY points | the holder's locked amount |
| Validator | +3 APY points | the operator's own self-bond, locked or not |

All rates are yearly, in basis points, and net: the module pays the holder
directly and no commission is taken from the bonus.

- Only locked stake earns a bonus, with one exception: the validator bonus. A
  light node operator therefore locks to get the light node bonus.
- The light node bonus needs, at settlement: an active `lightnode_operator`
  licence held by the lock's delegator, and a light node registered by that
  address with status `active` and an uptime (heartbeats over expected
  heartbeats) at least the light node module's `min_uptime_for_rewards`.
- The validator bonus is paid while the validator is bonded and not jailed. On
  the locked part of a self-bond it stacks with the lock bonus (and the light
  node bonus); the unlocked part of the self-bond earns the validator bonus
  alone.
- The ceiling: base yield plus bonus never exceeds `max_total_apy_bps` (25%).
  The base yield is what the epoch's base emission gives the bonded stake
  through the fee split (`validator_share + staker_share` of x/burn), net of
  the validator's commission for a delegator; for an operator's self-bond it is
  the gross yield, since the commission on their own stake comes back to them.
  The bonus of each stake is cut to the room left above its base.
- The light node and validator conditions, and the ceiling, are read at each
  settlement and apply to the whole period since the previous one. A lock's
  term bonus is fixed when the lock is made, so a later change to the tiers
  never alters a commitment already made.

## Locks

`MsgLockDelegation{delegator, validator, amount, months}` commits `amount` of
an existing delegation to `validator` until `months` calendar months later
(3, 6 or 12; the block time plus the months, in UTC). While locked, that amount
cannot be undelegated or redelegated. A lock covers at least `min_lock_amount`
(10 QOR by default), and one delegator holds at most 16 open locks; both bound
the work of the epoch settlement, which reads every open lock.

- **Enforcement.** The staking message server refuses a `MsgUndelegate` or
  `MsgBeginRedelegate` that would leave the delegation below its locked part.
  The check runs inside the message server, the same place as the validator
  self-bond floor, so it holds for every dispatch path: a transaction, authz
  `MsgExec`, x/group, a contract, a precompile, governance.
- **Slashing.** A lock is recorded in delegation shares. A slash lowers the
  tokens behind every share, so the lock shrinks exactly with the delegation.
  If a redelegation slash removes shares from the delegation itself, the locks
  are capped at what the delegation still holds, oldest first.
- **Clock.** Each lock keeps its own start height, start time and end time.
  x/distribution's `DelegatorStartingInfo` is reset by every reward withdrawal
  and cannot be used for this.
- **Expiry.** Locks are released automatically in the end blocker, which reads
  only the locks that are due (a queue ordered by end time). A released lock no
  longer restricts the delegation; its final period is credited at the next
  settlement, and its bonus stays claimable.
- **Early unlock.** `MsgUnlockEarly{delegator, lock_id}` ends a lock before it
  expires. The lock's unpaid bonus is forfeited (it stays in the module account
  and funds the next settlement before anything is minted), and
  `early_unlock_penalty_bps` (5%) of the locked stake is taken out of the
  delegation and burned through x/burn under the `staking_tier_penalty` source.
  The rest of the stake stays delegated, unlocked.

## Funding

The bonus is minted by x/inflation, once per epoch, into this module's account,
against the SAME cumulative cap (`max_total_emission`) as the base emission. The
base is minted first and is never reduced. Each settlement:

1. computes the bonus owed for the period since the previous settlement:
   `APY points x amount x period_seconds / seconds_per_year`;
2. spends the module's surplus (forfeits, rounding) first;
3. asks x/inflation to mint the rest, at most what the cap still allows;
4. if the funding falls short, scales every bonus down by the same factor.

So total emission never exceeds the cap.

## Payment: accrue, then claim

Bonuses are credited to the lock (or, for the validator bonus and expired
locks, to the holder) and paid by `MsgClaimBonus{holder}`. This follows the
light node module, and it keeps the epoch settlement, which runs in the begin
blocker for every lock at once, to store writes: a bank transfer and its
events per holder per epoch would cost far more and grow with every holder.
It is also what makes "forfeits the unpaid bonus" well defined.

## Parameters

Set only by governance (`MsgUpdateParams`, checked with `govauth.Assert`):
`lock_tiers` (months to bps), `light_node_bonus_bps` (200),
`validator_bonus_bps` (300), `max_total_apy_bps` (2500),
`early_unlock_penalty_bps` (500), `min_lock_amount` (10,000,000 uqor), `enabled`. With `enabled` false no new lock is
made and no bonus accrues; existing locks still bind and credited bonus can
still be claimed.

## Queries

- `/qorechain/stakingtier/v1/params`
- `/qorechain/stakingtier/v1/locks/{holder}`: each lock with its current
  amount, accrued bonus, the bonuses that would apply and the projected bonus,
  plus what the holder can claim.
- `/qorechain/stakingtier/v1/pool`: balance, outstanding, surplus, totals and
  the last settlement.

## Storage

The module keeps its records in the x/inflation store under the
`stakingtier/` prefix rather than in a store of its own. Every mounted store is
part of the app hash from the first block, empty or not, so a new store would
stop this binary from replaying the blocks below the plan height. x/inflation's
own keys all start with `inflation/`, and nothing iterates that store as a
whole.
