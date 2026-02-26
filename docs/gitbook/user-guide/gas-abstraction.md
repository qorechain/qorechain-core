# Gas Abstraction

This guide covers QoreChain's gas abstraction feature, which allows users to pay transaction fees in non-native tokens instead of QOR.

---

## Overview

Gas abstraction removes the requirement to hold QOR tokens for paying transaction fees. Users who hold accepted alternative tokens (such as IBC-transferred USDC or ATOM) can use those tokens directly as fee payment. The protocol automatically converts the fee amount to its native equivalent before processing.

---

## Accepted Tokens

The following tokens are accepted for fee payment:

| Token | Denomination | Conversion Rate | Example Fee |
|-------|-------------|----------------|-------------|
| **QOR** | `uqor` | 1.0 (native) | `--fees 500uqor` |
| **USDC** (via IBC) | `ibc/USDC` | 1.0 | `--fees 500ibc/USDC` |
| **ATOM** (via IBC) | `ibc/ATOM` | 10.0 | `--fees 50ibc/ATOM` |

> **Note:** Conversion rates reflect the protocol-defined exchange ratio, not market prices. A rate of 10.0 for ATOM means 1 unit of ibc/ATOM is equivalent to 10 units of uqor for fee purposes.

---

## How It Works

QoreChain's `GasAbstractionDecorator` is integrated into the transaction processing pipeline. When a transaction includes fees in a non-native denomination, the following occurs:

1. **Fee Inspection** -- The decorator checks the fee denomination specified in the transaction.
2. **Rate Lookup** -- If the denomination is in the accepted tokens list, the protocol looks up the corresponding conversion rate.
3. **Conversion** -- The fee amount is converted to its native uqor equivalent using the conversion rate.
4. **Standard Processing** -- The converted fee is passed to the standard `DeductFee` handler for deduction from the sender's account.

The conversion is transparent to the rest of the transaction pipeline. All downstream fee processing (distribution to validators, burning, treasury allocation, staker rewards) operates on the native uqor equivalent.

---

## Usage Examples

### Pay Fees in USDC

Send a token transfer with fees paid in USDC:

```bash
qorechaind tx bank send mykey qor1recipient... 5000000uqor \
  --chain-id qorechain-diana \
  --fees 500ibc/USDC
```

Since USDC has a 1.0 conversion rate, 500 ibc/USDC is equivalent to 500 uqor.

### Pay Fees in ATOM

Send a token transfer with fees paid in ATOM:

```bash
qorechaind tx bank send mykey qor1recipient... 5000000uqor \
  --chain-id qorechain-diana \
  --fees 50ibc/ATOM
```

Since ATOM has a 10.0 conversion rate, 50 ibc/ATOM is equivalent to 500 uqor.

---

## Querying Accepted Tokens

Retrieve the list of tokens currently accepted for gas abstraction, along with their conversion rates:

```bash
qorechaind query gasabstraction accepted-tokens
```

**Sample output:**

```yaml
accepted_tokens:
- denom: uqor
  conversion_rate: "1.000000000000000000"
- denom: ibc/USDC
  conversion_rate: "1.000000000000000000"
- denom: ibc/ATOM
  conversion_rate: "10.000000000000000000"
```

---

## JSON-RPC Access

For applications integrating via JSON-RPC, query the gas abstraction configuration:

```
qor_getGasAbstractionConfig
```

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "qor_getGasAbstractionConfig",
  "params": [],
  "id": 1
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "accepted_tokens": [
      { "denom": "uqor", "conversion_rate": "1.0" },
      { "denom": "ibc/USDC", "conversion_rate": "1.0" },
      { "denom": "ibc/ATOM", "conversion_rate": "10.0" }
    ]
  }
}
```

---

## Tips

- Gas abstraction is ideal for users onboarding from other ecosystems who may not yet hold QOR.
- Conversion rates are set by governance and may be updated via parameter change proposals.
- If you hold multiple accepted tokens, any of them can be used for fees on any transaction type.
- The actual token specified in `--fees` is deducted from your account. The conversion is only used to validate that the fee meets the minimum requirement.
