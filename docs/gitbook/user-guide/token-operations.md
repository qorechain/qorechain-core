# Token Operations

This guide covers the QOR token, how to send and receive tokens, query balances, and understand the fee distribution model on QoreChain.

---

## Token Information

| Property | Value |
|----------|-------|
| **Display Denomination** | QOR |
| **Base Denomination** | uqor |
| **Conversion** | 1 QOR = 1,000,000 uqor (10^6) |
| **Chain ID** | `qorechain-diana` |
| **Bech32 Prefix** | `qor` (e.g., `qor1abc...xyz`) |

All on-chain amounts are denominated in **uqor**. When submitting transactions, always specify amounts in uqor.

---

## Sending Tokens

To transfer QOR tokens from one account to another:

```bash
qorechaind tx bank send <from_address> <to_address> <amount>uqor \
  --chain-id qorechain-diana \
  --fees 500uqor
```

**Example:** Send 5 QOR (5,000,000 uqor) to another address:

```bash
qorechaind tx bank send qor1sender... qor1recipient... 5000000uqor \
  --chain-id qorechain-diana \
  --fees 500uqor
```

You can also use a key name instead of a raw address for the sender:

```bash
qorechaind tx bank send mykey qor1recipient... 5000000uqor \
  --chain-id qorechain-diana \
  --fees 500uqor
```

---

## Querying Balances

Check the balance of any account:

```bash
qorechaind query bank balances <address>
```

**Example:**

```bash
qorechaind query bank balances qor1abc...xyz
```

**Sample output:**

```yaml
balances:
- amount: "15000000"
  denom: uqor
pagination:
  next_key: null
  total: "0"
```

This indicates the account holds 15 QOR (15,000,000 uqor).

---

## Fee Structure

Transaction fees on QoreChain are distributed across four destinations to align network incentives:

| Destination | Share | Purpose |
|-------------|-------|---------|
| **Validators** | 40% | Rewards block producers and secures the network |
| **Burned** | 30% | Permanently removed from supply, creating deflationary pressure |
| **Treasury** | 20% | Funds protocol development and ecosystem grants |
| **Stakers** | 10% | Distributed proportionally to all delegators |

---

## Burn Channels

QoreChain implements a multi-channel burn mechanism. QOR tokens are permanently removed from circulation through 10 distinct channels:

| Channel | Description |
|---------|-------------|
| `tx_fee` | The 30% burn portion of every transaction fee |
| `governance_penalty` | Burned when governance proposals fail to reach quorum or are vetoed |
| `slashing_burn` | Burned portion of slashed validator stakes |
| `bridge_fee` | Fee burned on cross-chain bridge transfers |
| `spam_deterrent` | Additional burn applied to flagged spam transactions |
| `epoch_excess` | Excess inflation beyond target burned at epoch boundaries |
| `manual_burn` | Community-initiated token burns via governance proposal |
| `contract_callback` | Fees burned on smart contract callback executions |
| `cross_vm_fee` | Burned when executing cross-VM (e.g., EVM to CosmWasm) calls |
| `rollup_create` | 1% of the minimum stake burned when deploying a new rollup |

You can query the total burned amount across all channels:

```bash
qorechaind query bank total --denom uqor
```

---

## Tips

- Always double-check recipient addresses before sending tokens. Transactions on QoreChain are irreversible.
- Use the `--dry-run` flag to simulate a transaction without broadcasting it.
- Use `--gas auto` to let the node estimate the required gas for your transaction.
- The minimum fee for a standard transfer is **500 uqor**.
