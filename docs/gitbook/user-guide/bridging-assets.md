# Bridging Assets

This guide covers how to move assets between QoreChain and other blockchain networks. QoreChain supports **25 cross-chain connections**: 8 IBC channels for compatible chains and 17 QCB (QoreChain Bridge) endpoints for heterogeneous networks.

---

## Connection Overview

QoreChain provides two bridging protocols:

| Protocol | Connections | Use Case |
|----------|-------------|----------|
| **IBC** (Inter-Blockchain Communication) | 8 channels | Native interoperability with IBC-enabled chains |
| **QCB** (QoreChain Bridge) | 17 endpoints | Cross-chain transfers with non-IBC networks via PQC-secured attestations |

---

## IBC Channels

The following IBC-enabled chains have established channels with QoreChain:

| Chain | Channel | Status |
|-------|---------|--------|
| Cosmos Hub | `channel-0` | Active |
| Osmosis | `channel-1` | Active |
| Noble | `channel-2` | Active |
| Celestia | `channel-3` | Active |
| Stride | `channel-4` | Active |
| Akash | `channel-5` | Active |
| Babylon | `channel-6` | Active |
| QoreChain (loopback) | `channel-7` | Active |

IBC transfers use the standard `ibc-transfer` module:

```bash
qorechaind tx ibc-transfer transfer transfer <channel> <recipient> <amount>uqor \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

---

## QCB Bridge Endpoints

The QoreChain Bridge connects to 17 external chains spanning multiple ecosystem types:

| Chain | Chain Type | Supported Assets |
|-------|-----------|-----------------|
| Ethereum | EVM | ETH, USDC, WBTC |
| BSC | EVM | BNB, USDC |
| Solana | Solana | SOL, USDC |
| Avalanche | EVM | AVAX, USDC |
| Polygon | EVM | MATIC, USDC |
| Arbitrum | EVM | ETH, ARB, USDC |
| TON | TON | TON |
| Sui | Sui Move | SUI |
| Optimism | EVM | ETH, USDC, OP |
| Base | EVM | ETH, USDC |
| Aptos | Aptos | APT, USDC |
| Bitcoin | Bitcoin | BTC |
| NEAR | NEAR | NEAR, USDC |
| Cardano | Cardano | ADA |
| Polkadot | Polkadot | DOT |
| Tezos | Tezos | XTZ |
| Tron | Tron | TRX, USDT |

---

## Deposit Flow (External Chain to QoreChain)

Depositing assets from an external chain into QoreChain follows this sequence:

1. **Lock** tokens on the external chain by sending them to the QCB bridge contract or address.
2. **Attestation** -- Bridge validators observe the lock transaction and produce PQC-signed attestations.
3. **Threshold** -- Once **7 out of 10** validator attestations are collected, the bridge finalizes the deposit.
4. **Mint** -- The equivalent wrapped tokens are minted on QoreChain and credited to your `qor1...` address.

**CLI command:**

```bash
qorechaind tx bridge deposit \
  --chain ethereum \
  --amount 1000000 \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

---

## Withdraw Flow (QoreChain to External Chain)

Withdrawing assets from QoreChain to an external chain:

1. **Burn** the wrapped tokens on QoreChain.
2. **Attestation** -- Bridge validators observe the burn and produce PQC-signed attestations.
3. **Threshold** -- Once **7 out of 10** attestations are collected, the withdrawal is finalized.
4. **Unlock** -- The original tokens are released on the external chain to the specified destination address.

**CLI command:**

```bash
qorechaind tx bridge withdraw \
  --chain ethereum \
  --amount 1000000 \
  --to 0xYourEthereumAddress \
  --from mykey \
  --chain-id qorechain-diana \
  --fees 500uqor
```

---

## Security Model

The QoreChain Bridge is secured by multiple defense layers:

| Mechanism | Description |
|-----------|-------------|
| **7-of-10 PQC Multisig** | Every bridge operation requires attestations from at least 7 of 10 bridge validators, each using post-quantum cryptographic signatures. |
| **24-Hour Challenge Period** | Withdrawals exceeding a configurable threshold enter a 24-hour challenge window during which validators or watchers can flag fraudulent transactions. |
| **Circuit Breakers** | Automated rate limiters halt bridge operations if abnormal volume or suspicious patterns are detected. Bridge operations resume after manual review. |

---

## Querying Bridge Status

Check the status of a pending bridge operation:

```bash
qorechaind query bridge pending-deposits --address <your_qor_address>
```

```bash
qorechaind query bridge pending-withdrawals --address <your_qor_address>
```

List all active bridge connections:

```bash
qorechaind query bridge connections
```

---

## Tips

- Bridge deposits typically finalize within minutes once the required 7-of-10 attestations are gathered.
- Large withdrawals trigger the 24-hour challenge period automatically. Plan ahead for time-sensitive transfers.
- Always verify the destination address format matches the target chain (e.g., `0x...` for EVM chains, base58 for Solana).
- IBC transfers are generally faster than QCB transfers since they use native protocol-level communication.
- Bridge fees are burned through the `bridge_fee` burn channel (see [Token Operations](token-operations.md)).
