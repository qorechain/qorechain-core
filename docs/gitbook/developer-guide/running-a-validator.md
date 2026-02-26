# Running a Validator

This guide covers how to create a validator on the QoreChain network, understand the pool classification system, register a PQC key for quantum-resistant security, and monitor your node.

---

## Prerequisites

- A fully synced `qorechaind` node (see [Connecting to Testnet](../getting-started/connecting-to-testnet.md))
- A funded account with at least **1,000 QOR** (1,000,000,000 uqor) for the initial self-delegation
- Familiarity with the [Staking and Delegation](../user-guide/staking-and-delegation.md) model

---

## Creating a Validator

```bash
qorechaind tx staking create-validator \
  --amount 1000000000uqor \
  --pubkey $(qorechaind comet show-validator) \
  --moniker "my-validator" \
  --commission-rate 0.10 \
  --commission-max-rate 0.20 \
  --commission-max-change-rate 0.01 \
  --min-self-delegation 1 \
  --from mykey \
  --gas auto \
  --gas-adjustment 1.3 \
  -y
```

| Parameter | Description |
|-----------|-------------|
| `--amount` | Self-delegation amount (minimum stake) |
| `--pubkey` | Validator consensus public key (ed25519) |
| `--moniker` | Human-readable name for your validator |
| `--commission-rate` | Initial commission rate (e.g., 0.10 = 10%) |
| `--commission-max-rate` | Maximum commission rate (immutable after creation) |
| `--commission-max-change-rate` | Maximum daily commission change rate |
| `--min-self-delegation` | Minimum tokens the operator must self-delegate |

After the transaction confirms, verify your validator:

```bash
qorechaind query staking validator $(qorechaind keys show mykey --bech val -a)
```

---

## Pool Classification

QoreChain uses a **three-pool classification system** managed by the `x/qca` (Quantum Consensus Allocation) module. Every **1,000 blocks**, validators are reclassified into one of three pools based on their reputation and stake:

| Pool | Criteria | Block Allocation |
|------|----------|-----------------|
| **RPoS** (Reputation Proof-of-Stake) | Reputation >= 70th percentile AND stake >= median | 40% of blocks |
| **DPoS** (Delegated Proof-of-Stake) | Total delegation >= 10,000 QOR | 35% of blocks |
| **PoS** (Proof-of-Stake) | All remaining active validators | 25% of blocks |

Within each pool, block proposers are selected using **weighted random selection** proportional to their effective stake. The classification ensures that both high-reputation and high-delegation validators receive fair representation, while still allowing smaller validators to participate.

### Query Your Pool Classification

```bash
qorechaind query qca pool-classification $(qorechaind keys show mykey --bech val -a)
```

Via JSON-RPC:

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "qor_getPoolClassification",
    "params": ["qorvaloper1..."],
    "id": 1
  }'
```

---

## Bonding Curve

The staking reward for a validator is determined by a bonding curve that incorporates multiple factors:

```
R = beta * S * (1 + alpha * log(1 + L)) * Q(r) * P(t)
```

| Variable | Description |
|----------|-------------|
| `R` | Reward amount |
| `beta` | Base reward rate |
| `S` | Effective stake |
| `alpha` | Loyalty scaling constant |
| `L` | Loyalty duration (continuous staking time) |
| `Q(r)` | Reputation quality factor, range [0.75 - 1.25] |
| `P(t)` | Protocol phase multiplier (adjusts over network lifecycle) |

**Key takeaways:**

- **Loyalty duration bonus:** Validators who stake continuously receive increasing rewards via the logarithmic loyalty term. This incentivizes long-term commitment.
- **Reputation quality factor:** Ranges from 0.75 (poor reputation) to 1.25 (excellent reputation). Reputation is computed from uptime, successful proposals, community participation, and transaction validation quality.
- **Protocol phase multiplier:** Adjusts as the network matures through different phases (bootstrap, growth, maturity).

---

## Progressive Slashing

QoreChain uses a **progressive slashing** model that escalates penalties for repeat offenders while allowing validators to recover over time:

```
penalty = base_rate * escalation^effective_count * severity
```

| Parameter | Value |
|-----------|-------|
| Maximum penalty per event | 33% of stake |
| Decay half-life | 100,000 blocks |
| Downtime severity | 1.0 |
| Double-sign severity | 2.0 |
| Light client attack severity | 3.0 |

**How it works:**

1. Each infraction increments the validator's **effective count**
2. The penalty escalates exponentially with the effective count
3. The effective count **decays** with a half-life of 100,000 blocks (~7 days at 6s blocks)
4. A validator with a clean record for a long period returns to near-zero effective count

This design means a single accidental downtime event results in a minor penalty, but repeated infractions trigger increasingly severe consequences.

---

## PQC Key Registration

Validators can optionally register a **post-quantum cryptographic (PQC) public key** using the ML-DSA-87 algorithm. This provides quantum-resistant security for validator identity and can be used for hybrid signing.

```bash
qorechaind tx pqc register-key <pubkey-hex> hybrid \
  --from mykey \
  --gas auto \
  -y
```

| Parameter | Description |
|-----------|-------------|
| `<pubkey-hex>` | 2592-byte ML-DSA-87 public key in hex encoding |
| `hybrid` | Registration mode (hybrid = both classical + PQC) |

Verify registration:

```bash
qorechaind query pqc key <account-address>
```

> **Recommendation:** PQC key registration is optional but strongly recommended for validators operating on the mainnet. It provides a forward-looking defense against quantum computing threats.

---

## Monitoring

### Prometheus Metrics

QoreChain exposes Prometheus metrics on port **26660**:

```
http://localhost:26660/metrics
```

Key metrics to monitor:

| Metric | Description |
|--------|-------------|
| `qorechain_missed_blocks_total` | Total blocks missed by your validator |
| `qorechain_validator_uptime` | Uptime percentage over the last N blocks |
| `qorechain_reputation_score` | Current reputation score |
| `qorechain_pool_classification` | Current pool assignment (0=PoS, 1=DPoS, 2=RPoS) |
| `qorechain_consecutive_signed` | Consecutive blocks signed |
| `consensus_height` | Current block height |
| `consensus_rounds` | Consensus rounds for current height |

### Query Reputation Score

```bash
qorechaind query reputation score $(qorechaind keys show mykey --bech val -a)
```

Via JSON-RPC:

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "qor_getReputationScore",
    "params": ["qorvaloper1..."],
    "id": 1
  }'
```

### Health Checks

```bash
# Node status
qorechaind status | jq '.sync_info'

# Validator signing info (uptime, missed blocks)
qorechaind query slashing signing-info $(qorechaind comet show-validator)

# Check if your validator is in the active set
qorechaind query staking validators --status bonded | grep "my-validator"
```

---

## Operational Best Practices

1. **Use a sentry node architecture:** Run your validator behind sentry nodes to protect it from DDoS attacks. Only expose sentry nodes to the public network.

2. **Set up alerting:** Configure alerts for missed blocks, low uptime, and unexpected restarts. A few missed blocks are normal; sustained misses will trigger slashing.

3. **Maintain high uptime:** The reputation system rewards consistent uptime. Extended downtime degrades your reputation quality factor, reducing rewards.

4. **Keep software updated:** Track QoreChain releases and apply updates promptly. Coordinate with the validator community for chain upgrades.

5. **Secure your keys:** Use a hardware security module (HSM) or remote signer for the validator consensus key. Never store keys on the same machine as the node.

6. **Register a PQC key:** Future-proof your validator against quantum threats by registering an ML-DSA-87 key.

7. **Monitor your pool:** Track your pool classification every 1,000 blocks. Improving your reputation can move you from PoS to RPoS, significantly increasing block proposal opportunities.

---

## Validator Commands Reference

```bash
# Edit validator metadata
qorechaind tx staking edit-validator \
  --moniker "new-name" \
  --website "https://myvalidator.com" \
  --details "Description of my validator" \
  --from mykey -y

# Unjail after downtime slashing
qorechaind tx slashing unjail --from mykey -y

# Delegate additional stake
qorechaind tx staking delegate $(qorechaind keys show mykey --bech val -a) \
  500000000uqor --from mykey -y

# Withdraw rewards
qorechaind tx distribution withdraw-rewards $(qorechaind keys show mykey --bech val -a) \
  --commission --from mykey -y
```

---

## Next Steps

- [Building from Source](building-from-source.md) -- Build the `qorechaind` binary
- [EVM Development](evm-development.md) -- Deploy smart contracts on QoreChain
- [Account Abstraction](account-abstraction.md) -- Programmable accounts for your validator operations
