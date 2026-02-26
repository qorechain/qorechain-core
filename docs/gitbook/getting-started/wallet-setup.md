# Wallet Setup

QoreChain supports multiple wallet types across its native, EVM, and SVM execution environments. Choose the wallet that matches your use case.

---

## Keplr Wallet

Keplr is the recommended wallet for native QoreChain transactions, staking, and governance.

### Add QoreChain as a Custom Chain

Open Keplr and navigate to **Settings > Add Custom Chain**, then enter:

| Field | Value |
|-------|-------|
| Chain Name | `QoreChain Diana Testnet` |
| Chain ID | `qorechain-diana` |
| RPC URL | `http://localhost:26657` |
| REST URL | `http://localhost:1317` |
| Bech32 Prefix | `qor` |
| Coin Denom | `QOR` |
| Coin Minimal Denom | `uqor` |
| Decimals | `6` |

After adding the chain, Keplr generates a `qor1...` address for your account. Use this address to receive testnet QOR tokens.

---

## MetaMask (EVM)

MetaMask enables interaction with QoreChain's EVM execution environment -- deploy Solidity contracts, manage ERC-20 tokens, and use familiar Ethereum tooling.

### Add QoreChain as a Custom Network

Open MetaMask and navigate to **Settings > Networks > Add Network**, then enter:

| Field | Value |
|-------|-------|
| Network Name | `QoreChain EVM` |
| RPC URL | `http://localhost:8545` |
| Currency Symbol | `QOR` |

Once connected, you can use MetaMask to sign EVM transactions, interact with deployed smart contracts, and manage ERC-20 tokens on QoreChain.

---

## Solana Wallets (SVM)

QoreChain's SVM execution environment is compatible with standard Solana tooling. Connect any Solana-compatible wallet or library to interact with SVM programs.

### Using @solana/web3.js

```javascript
import { Connection } from "@solana/web3.js";

const connection = new Connection("http://localhost:8899");
const slot = await connection.getSlot();
console.log("Current slot:", slot);
```

This enables deployment and interaction with SVM programs running on QoreChain.

---

## PQC-Enabled Wallets

QoreChain features hybrid post-quantum cryptography (PQC) support through ML-DSA-87 (Dilithium-5) signatures. This provides quantum-resistant security alongside standard Ed25519 signatures.

### How It Works

Wallets can attach an ML-DSA-87 PQC signature as a transaction extension alongside the standard Ed25519 signature. This hybrid approach ensures backward compatibility while enabling quantum resistance.

### Auto-Registration

When you include a PQC public key in your first transaction, QoreChain automatically registers it on-chain. No separate registration step is needed.

### PQC Modes

| Mode | Description |
|------|-------------|
| **Disabled** | PQC verification is turned off. Standard signatures only. |
| **Optional** (default) | Transactions may include PQC signatures. If present, they are verified. |
| **Required** | All transactions must include a valid PQC signature. |

The active mode is configured at the chain level and can be updated through governance.

---

## CLI Wallet

The `qorechaind` binary includes a built-in key management system for command-line usage.

### Create a New Key

```bash
qorechaind keys add mykey
```

This generates a new key pair and displays the mnemonic phrase. **Store the mnemonic securely** -- it is the only way to recover this key.

### View Your Address

```bash
qorechaind keys show mykey -a
```

This outputs your `qor1...` bech32 address.

### List All Keys

```bash
qorechaind keys list
```

### Import an Existing Key

```bash
qorechaind keys add mykey --recover
```

You will be prompted to enter your mnemonic phrase.

---

## Next Steps

- [Your First Transaction](first-transaction.md) -- Send QOR tokens using your new wallet
- [Connecting to Testnet](connecting-to-testnet.md) -- Join the live Diana testnet
