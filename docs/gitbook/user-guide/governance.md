# Governance

This guide covers how on-chain governance works on QoreChain, including the Quadratic Delegation-Reputation Weighted (QDRW) voting system, how to submit proposals, and how to vote.

---

## Voting Power: QDRW Formula

QoreChain uses the **Quadratic Delegation-Reputation Weighted (QDRW)** formula to calculate voting power. This system prevents whale dominance while rewarding participants who have earned high reputation scores and committed to governance through xQORE staking.

```
VP = sqrt(staked + 2 * xQORE) * ReputationMultiplier(r)
```

| Variable | Description |
|----------|-------------|
| `VP` | Effective voting power |
| `staked` | Total QOR tokens staked by the voter |
| `xQORE` | Amount of xQORE governance tokens held (see [xQORE Governance Staking](xqore-staking.md)) |
| `r` | Voter's reputation score, normalized to [0, 1] |
| `ReputationMultiplier(r)` | Sigmoid function mapping reputation to a multiplier in the range [0.5, 2.0] |

### Key Properties

- **Quadratic dampening:** A holder with 100x the stake of another voter gains only ~10x the voting power, not 100x. This ensures that governance influence scales sub-linearly with wealth.
- **xQORE bonus:** xQORE tokens count at **2x weight** inside the square root, giving governance-committed participants a meaningful advantage.
- **Reputation multiplier:** Maps the voter's reputation score from [0, 1] to a multiplier in [0.5, 2.0] using a sigmoid curve. High-reputation participants can double their effective voting power, while low-reputation participants see their influence halved.

---

## Submitting a Proposal

Any QOR holder can submit a governance proposal. A minimum deposit is required for the proposal to enter the voting period.

```bash
qorechaind tx gov submit-proposal <proposal_file.json> \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

**Example proposal file** (`proposal.json`):

```json
{
  "title": "Increase Maximum Validator Count",
  "description": "This proposal increases the maximum active validator set from 100 to 150 to improve decentralization.",
  "type": "parameter_change",
  "changes": [
    {
      "subspace": "staking",
      "key": "MaxValidators",
      "value": "150"
    }
  ],
  "deposit": "10000000uqor"
}
```

---

## Voting on Proposals

Once a proposal enters the voting period, any staker can cast a vote:

```bash
qorechaind tx gov vote <proposal_id> <option> \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

**Vote options:**

| Option | Description |
|--------|-------------|
| `yes` | Support the proposal |
| `no` | Oppose the proposal |
| `abstain` | Acknowledge the proposal without taking a position |
| `no_with_veto` | Oppose the proposal and signal it should not have been submitted (burns the deposit if threshold is met) |

**Example:**

```bash
qorechaind tx gov vote 1 yes \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

---

## Proposal Types

QoreChain supports the following governance proposal types:

| Type | Description |
|------|-------------|
| **Text** | A signaling proposal with no automatic on-chain execution. Used for community sentiment checks. |
| **Parameter Change** | Modifies one or more on-chain protocol parameters (e.g., max validators, inflation rate). |
| **Software Upgrade** | Schedules a coordinated chain upgrade at a specified block height. |
| **Community Spend** | Requests funds from the community treasury for a specified recipient address. |

---

## Querying Proposals

List all proposals:

```bash
qorechaind query gov proposals
```

Query a specific proposal by ID:

```bash
qorechaind query gov proposal <proposal_id>
```

Check the current tally of votes on a proposal:

```bash
qorechaind query gov tally <proposal_id>
```

View your own vote on a proposal:

```bash
qorechaind query gov vote <proposal_id> <voter_address>
```

---

## Governance Parameters

Query the current governance parameters:

```bash
qorechaind query gov params
```

Key parameters include:

| Parameter | Description |
|-----------|-------------|
| `min_deposit` | Minimum deposit required for a proposal to enter voting |
| `max_deposit_period` | Time window for reaching the minimum deposit |
| `voting_period` | Duration of the voting period once a proposal is active |
| `quorum` | Minimum participation required for a valid vote |
| `threshold` | Minimum "yes" percentage to pass (excluding abstains) |
| `veto_threshold` | Minimum "no with veto" percentage to reject and burn the deposit |

---

## Tips

- Build reputation before major governance votes to maximize your voting power multiplier.
- Lock QOR into xQORE for a 2x governance weight bonus inside the QDRW formula.
- Use `no_with_veto` carefully. If the veto threshold is reached, the proposal deposit is burned.
- Proposals that do not reach the minimum deposit within the deposit period are automatically removed.
