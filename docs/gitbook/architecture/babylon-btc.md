# BTC Restaking (Babylon)

The `x/babylon` module integrates QoreChain with the Babylon Protocol to inherit Bitcoin's proof-of-work finality guarantees. Through BTC restaking, QoreChain gains a secondary finality layer backed by Bitcoin's hashrate -- without requiring any changes to the Bitcoin protocol itself.

---

## Overview

Babylon Protocol enables proof-of-stake chains to leverage Bitcoin's security through a timestamping and checkpointing mechanism. QoreChain's integration works as follows:

1. **BTC stakers** lock Bitcoin in Babylon staking transactions and register their positions on QoreChain.
2. **Epoch checkpoints** from QoreChain are periodically relayed to Babylon, which timestamps them on Bitcoin.
3. **Finality inheritance**: Once a QoreChain epoch is checkpointed on Bitcoin, the state covered by that epoch inherits Bitcoin's proof-of-work finality guarantees.

This provides a defense against long-range attacks and equivocation that is anchored to Bitcoin's accumulated hashrate rather than relying solely on QoreChain's own validator set.

---

## BTC Staking Positions

Users can register BTC staking positions on QoreChain by submitting a `MsgBTCRestake` transaction that references a Bitcoin staking transaction.

### Registration Requirements

| Parameter | Value | Description |
|---|---|---|
| **Minimum stake** | 100,000 satoshis (0.001 BTC) | Minimum BTC required per staking position |
| **Unbonding period** | 144 BTC blocks (~1 day) | Waiting period before staked BTC can be withdrawn |
| **Checkpoint interval** | Every 10 QoreChain epochs | How often state is checkpointed to Babylon |

### Staking Position Structure

Each BTC staking position tracks the following on-chain state:

| Field | Description |
|---|---|
| `staker_address` | QoreChain address of the staker (`qor1...`) |
| `btc_tx_hash` | Bitcoin transaction hash of the staking transaction |
| `amount_satoshis` | Amount of BTC staked in satoshis |
| `status` | Position lifecycle state: `active`, `unbonding`, or `withdrawn` |
| `staked_at` | Timestamp of position registration |
| `unbonding_height` | Block height at which unbonding was initiated (if applicable) |
| `validator_addr` | QoreChain validator address this stake is delegated to |

### Registration Flow

```
Bitcoin Network          QoreChain
      |                      |
      | 1. Create BTC        |
      |    staking TX         |
      |                      |
      |    [confirmed]        |
      |                      |
      |                      | 2. Submit MsgBTCRestake
      |                      |    with btc_tx_hash,
      |                      |    amount, validator
      |                      |
      |                      | 3. Position recorded
      |                      |    as "active"
```

---

## Epoch Checkpoints

QoreChain's epoch state roots are periodically checkpointed to Bitcoin through the Babylon relay chain.

### Checkpoint Flow

```
QoreChain              Babylon               Bitcoin
     |                    |                      |
     | 1. Validator       |                      |
     |    submits          |                      |
     |    MsgSubmitBTC     |                      |
     |    Checkpoint       |                      |
     |------------------->|                      |
     |                    | 2. IBC relay         |
     |                    |    epoch data        |
     |                    |--------------------->|
     |                    |                      | 3. Bitcoin
     |                    |                      |    timestamping
     |                    |                      |
     |                    | 4. Confirmation      |
     |                    |<---------------------|
     |                    |                      |
     | 5. Checkpoint      |                      |
     |    finalized        |                      |
     |<-------------------|                      |
```

1. **Submit**: A QoreChain validator submits `MsgSubmitBTCCheckpoint` containing the epoch number, BTC block hash, BTC block height, and QoreChain state root.
2. **IBC relay**: The checkpoint data is relayed to the Babylon chain via IBC.
3. **Timestamping**: Babylon includes the checkpoint in a Bitcoin transaction, anchoring QoreChain's state to Bitcoin's blockchain.
4. **Confirmation**: Once the Bitcoin transaction is confirmed, finality flows back through Babylon to QoreChain.
5. **Finalization**: The checkpoint status transitions from `pending` to `confirmed` to `finalized`.

### Checkpoint Structure

| Field | Description |
|---|---|
| `epoch_num` | QoreChain epoch number being checkpointed |
| `btc_block_hash` | Bitcoin block hash containing the checkpoint |
| `btc_block_height` | Bitcoin block height |
| `state_root` | QoreChain state root at the epoch boundary |
| `submitted_at` | Timestamp of checkpoint submission |
| `status` | Checkpoint state: `pending`, `confirmed`, or `finalized` |

### Epoch Snapshots

At each checkpoint boundary, an epoch snapshot captures aggregate network state:

| Field | Description |
|---|---|
| `total_staked` | Total BTC staked across all positions (satoshis) |
| `active_positions` | Number of active staking positions |
| `validator_count` | Number of validators with BTC-backed delegations |
| `block_height` | QoreChain block height at snapshot |

---

## Secondary Finality Layer

The Babylon integration provides a **secondary finality guarantee** that complements QoreChain's native consensus finality:

| Finality Layer | Source | Speed | Security |
|---|---|---|---|
| **Primary** | QoreChain Consensus Engine | ~5 seconds | Backed by QOR stake + PQC signatures |
| **Secondary** | Babylon + Bitcoin | ~60 minutes | Backed by Bitcoin's cumulative hashrate |

The secondary layer is particularly valuable for:

- **Long-range attack prevention**: Even if an attacker accumulates significant QOR stake, they cannot rewrite history that has been checkpointed on Bitcoin.
- **Cross-chain bridge security**: Bridge operations involving large values can wait for Bitcoin-level finality before releasing funds.
- **Institutional confidence**: The Bitcoin timestamp provides an independently verifiable proof of QoreChain's state history.

---

## Configuration

| Parameter | Default | Description |
|---|---|---|
| `enabled` | `false` | Master switch for BTC restaking features |
| `min_stake_amount` | 100,000 satoshis | Minimum BTC per staking position |
| `unbonding_period` | 144 BTC blocks | BTC-denominated unbonding duration |
| `checkpoint_interval` | 10 epochs | Epochs between Babylon checkpoints |
| `babylon_chain_id` | `bbn-1` | Chain ID of the connected Babylon network |

---

## Events

The module emits the following on-chain events:

| Event Type | Attributes | Description |
|---|---|---|
| `btc_restake` | staker, btc_tx_hash, amount, validator | New BTC staking position registered |
| `btc_unbond` | staker, amount | BTC staking position entered unbonding |
| `btc_checkpoint` | epoch, checkpoint_id | Epoch checkpoint submitted to Babylon |
| `babylon_epoch_complete` | epoch | Babylon epoch finalized with Bitcoin timestamp |

---

## API Endpoints

### REST

| Method | Endpoint | Description |
|---|---|---|
| GET | `/babylon/v1/staking/{address}` | Get BTC staking positions for an address |
| GET | `/babylon/v1/checkpoint/{epoch}` | Get checkpoint data for a specific epoch |
| GET | `/babylon/v1/params` | Get module configuration parameters |

### JSON-RPC

| Method | Parameters | Description |
|---|---|---|
| `qor_getBTCStakingPosition` | `address` (string) | Returns the BTC staking position for the given QoreChain address |

---

## CLI Commands

### Query Commands

```bash
# Query BTC restaking configuration
qorechaind query babylon config

# Query a staking position by address
qorechaind query babylon position <staker-address>
```

### Transaction Commands

```bash
# Register a BTC restaking position
qorechaind tx babylon restake \
  --btc-tx-hash <hash> \
  --amount-satoshis <amount> \
  --validator <qorvaloper1...> \
  --from <key-name>

# Submit a BTC checkpoint
qorechaind tx babylon submit-checkpoint \
  --epoch <epoch-number> \
  --btc-block-hash <hash> \
  --btc-block-height <height> \
  --state-root <root> \
  --from <key-name>
```
