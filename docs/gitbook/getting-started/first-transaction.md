# Your First Transaction

This guide walks through sending QOR tokens, querying transactions, and interacting with QoreChain across its native, EVM, and SVM interfaces.

---

## Check Your Balance

Before sending tokens, verify your account balance:

```bash
qorechaind query bank balances qor1youraddress... --output json
```

The response includes all token denominations held by the account. QOR balances are displayed in `uqor` (micro-QOR), where **1 QOR = 1,000,000 uqor**.

---

## Send QOR

Transfer tokens from your key to another address:

```bash
qorechaind tx bank send mykey qor1recipient... 1000000uqor \
  --chain-id qorechain-diana \
  --fees 500uqor
```

This sends **1 QOR** (1,000,000 uqor) to the recipient address, paying a fee of 500 uqor.

You will be prompted to confirm the transaction before it is broadcast. Once confirmed, the CLI returns a transaction hash.

---

## Query Transaction

Look up a completed transaction by its hash:

```bash
qorechaind query tx <txhash>
```

The output includes the transaction status, gas used, block height, and all events emitted during execution.

For JSON output:

```bash
qorechaind query tx <txhash> --output json
```

---

## Using JSON-RPC (EVM)

QoreChain's EVM execution environment exposes a standard Ethereum JSON-RPC interface on port `8545`.

### Get the Latest Block Number

```bash
curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "eth_blockNumber",
    "params": [],
    "id": 1
  }' | jq '.result'
```

### Get an Account Balance

```bash
curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "eth_getBalance",
    "params": ["0xYourEVMAddress", "latest"],
    "id": 1
  }' | jq '.result'
```

The balance is returned as a hex-encoded value in the smallest denomination.

---

## Using SVM RPC

QoreChain's SVM execution environment exposes a Solana-compatible RPC interface on port `8899`.

### Get the Current Slot

```bash
curl -s -X POST http://localhost:8899 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "getSlot",
    "id": 1
  }' | jq '.result'
```

### Get an Account Balance

```bash
curl -s -X POST http://localhost:8899 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "getBalance",
    "params": ["YourSVMPublicKey"],
    "id": 1
  }' | jq '.result'
```

---

## Common CLI Patterns

When working with the `qorechaind` CLI, these flags are used frequently:

| Flag | Description | Example |
|------|-------------|---------|
| `--chain-id` | Specifies the target chain | `--chain-id qorechain-diana` |
| `--fees` | Transaction fee in uqor | `--fees 500uqor` |
| `--from` | Signing key name or address | `--from mykey` |
| `--output` | Response format | `--output json` |
| `--node` | RPC endpoint to connect to | `--node tcp://localhost:26657` |
| `--gas` | Gas limit for the transaction | `--gas auto` |
| `--gas-adjustment` | Multiplier for estimated gas | `--gas-adjustment 1.3` |
| `-y` | Skip confirmation prompt | `-y` |

### Example: Full Command with All Common Flags

```bash
qorechaind tx bank send mykey qor1recipient... 500000uqor \
  --chain-id qorechain-diana \
  --fees 500uqor \
  --node tcp://localhost:26657 \
  --output json \
  -y
```

---

## Next Steps

Now that you have sent your first transaction, explore more of what QoreChain offers:

- [Staking and Delegation](../user-guide/staking-and-delegation.md) -- Stake QOR and earn rewards
- [Bridging Assets](../user-guide/bridging-assets.md) -- Move assets across chains
- [EVM Development](../developer-guide/evm-development.md) -- Deploy Solidity smart contracts on QoreChain
