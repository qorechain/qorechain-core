# MEV Protection (FairBlock)

The `x/fairblock` module implements QoreChain's defense against Maximal Extractable Value (MEV) attacks using threshold identity-based encryption. Combined with a 5-lane transaction prioritization system, this creates a comprehensive anti-MEV architecture that protects users from front-running, sandwich attacks, and other forms of mempool-based value extraction.

---

## The MEV Problem

MEV occurs when block proposers or observers exploit **information asymmetry** in the transaction mempool. Because pending transactions are visible before inclusion, adversaries can:

- **Front-run**: Place a transaction ahead of a detected profitable trade
- **Sandwich attack**: Place transactions before and after a victim's trade to extract value from price slippage
- **Back-run**: Place a transaction immediately after a detected opportunity

These attacks extract value from ordinary users and undermine fairness in DeFi, token swaps, and NFT minting.

---

## FairBlock tIBE Framework

QoreChain addresses MEV through **threshold Identity-Based Encryption (tIBE)**, a cryptographic scheme where:

1. **Encryption**: Users encrypt their transactions before broadcasting. Encrypted transactions are **opaque** -- proposers, validators, and mempool observers cannot read transaction contents.
2. **Inclusion**: Proposers include encrypted transactions in blocks without knowing their contents. Since the data is unreadable, information asymmetry is eliminated.
3. **Decryption**: After a transaction is committed to a block, a threshold number of validators contribute decryption shares. Once the threshold is met, the transaction is decrypted and executed.

This approach ensures that no single party can decrypt a transaction before it is irreversibly committed, eliminating the MEV attack vector at its root.

### Encrypted Transaction Structure

Each encrypted transaction contains:

| Field | Description |
|---|---|
| `encrypted_data` | tIBE-encrypted transaction payload |
| `sender` | Transaction sender address (visible for routing) |
| `target_height` | Block height at which decryption should occur |
| `submitted_at` | Timestamp of encryption |

### Decryption Shares

Validators contribute decryption shares for committed transactions:

| Field | Description |
|---|---|
| `validator` | Address of the contributing validator |
| `tx_id` | ID of the encrypted transaction |
| `share_data` | The validator's decryption key share |
| `height` | Block height of the share submission |

---

## Implementation Status (v1.2.0)

In the current testnet release, the FairBlock module is a **stub implementation**:

- The `FairBlockDecorator` ante handler is wired into the transaction processing pipeline but **passes through** all transactions without modification.
- When `enabled` is `false` (the default), the decorator immediately delegates to the next handler in the chain.
- Full tIBE activation is planned for a future release, pending a validator key ceremony to establish the threshold encryption parameters.

### FairBlock Configuration

| Parameter | Default | Description |
|---|---|---|
| `enabled` | `false` | Master switch for tIBE encryption |
| `tibe_threshold` | 5 | Number of validator decryption shares required |
| `decryption_delay` | 3 blocks | Blocks after inclusion before decryption begins |
| `max_encrypted_size` | 65,536 bytes | Maximum size of an encrypted transaction payload |

---

## 5-Lane Transaction Prioritization

QoreChain implements a 5-lane mempool architecture that categorizes transactions by type and assigns each lane a priority level and block space allocation.

### Lane Configuration

| Lane | Priority | Block Space | Transaction Type |
|---|---|---|---|
| **PQC** | 100 (highest) | 15% | Transactions with post-quantum hybrid signatures |
| **MEV** | 90 | 20% | FairBlock tIBE-encrypted transactions |
| **AI** | 80 | 15% | AI-scored and fee-optimized transactions |
| **Default** | 50 | 40% | Standard transactions |
| **Free** | 10 (lowest) | 10% | Gas-abstracted and sponsored transactions |

### Lane Descriptions

**PQC Lane** (Priority 100, 15% block space)
Transactions signed with hybrid post-quantum cryptographic signatures receive the highest priority. This incentivizes adoption of quantum-safe transaction signing and ensures PQC-protected operations are never crowded out during congestion.

**MEV Lane** (Priority 90, 20% block space)
tIBE-encrypted transactions receive the second-highest priority and the largest reserved allocation. This ensures that users who opt into MEV protection are guaranteed block space, encouraging widespread adoption of the encryption scheme.

**AI Lane** (Priority 80, 15% block space)
Transactions that have been scored or optimized by the AI anomaly detection system receive elevated priority. This includes transactions flagged as high-value legitimate operations or those with AI-optimized fee structures.

**Default Lane** (Priority 50, 40% block space)
Standard transactions without any special classification. This lane receives the largest absolute block space allocation to handle normal network traffic.

**Free Lane** (Priority 10, 10% block space)
Gas-abstracted and sponsored transactions. This lane enables zero-fee user experiences where a third party (application, protocol, or relayer) sponsors the gas cost. The low priority and limited block space prevent abuse while still supporting gas abstraction use cases.

### Implementation Status

Lane configuration is **data-only** in v1.2.0. The lane definitions (priority, block space allocation) are registered at application initialization, but actual mempool reordering via `PrepareProposal` and `ProcessProposal` is a future milestone. Currently, all transactions are processed in standard order regardless of lane assignment.

---

## Combined Anti-MEV Effect

When fully activated, the FairBlock and lane systems work together to create a layered defense:

```
Layer 1: Encryption (tIBE)
  Transactions are encrypted before entering the mempool.
  Proposers cannot read contents → no information to extract.

Layer 2: Prioritization (Lanes)
  Encrypted MEV-lane transactions get 20% reserved block space.
  Priority 90 ensures inclusion even during congestion.

Layer 3: Threshold Decryption
  Only after block commitment do validators reveal decryption shares.
  Threshold requirement prevents any single validator from early decryption.

Result: Information asymmetry is eliminated at every stage of the
        transaction lifecycle, from broadcast to execution.
```

This approach is strictly superior to time-delayed decryption or single-party commit-reveal schemes, because the threshold requirement distributes trust across the validator set.
