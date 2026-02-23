# QoreChain EVM Runtime

## Overview

QoreChain v0.5.0 introduces a dual VM runtime enabling full EVM (Ethereum Virtual Machine) compatibility alongside the existing QoreChain SDK runtime. This allows developers to deploy Solidity smart contracts on QoreChain while benefiting from the chain's PQC security, AI-native processing, and cross-VM communication capabilities.

## Architecture

```
                    ┌─────────────────────────────────┐
                    │        QoreChain Node            │
                    │                                  │
  JSON-RPC (:8545) ─┤  eth_ namespace (EVM txs)       │
                    │  qor_ namespace (custom queries) │
                    │                                  │
  REST (:1317) ─────┤  QoreChain REST API                 │
  gRPC (:9090) ─────┤  QoreChain gRPC API                 │
                    │                                  │
                    │  ┌──────────┐  ┌──────────────┐  │
                    │  │  x/vm    │  │  x/wasm      │  │
                    │  │  (EVM)   │  │  (CosmWasm)  │  │
                    │  └────┬─────┘  └──────┬───────┘  │
                    │       │               │          │
                    │  ┌────┴───────────────┴───────┐  │
                    │  │       x/crossvm            │  │
                    │  │   (Precompile + Events)    │  │
                    │  └────────────────────────────┘  │
                    └─────────────────────────────────┘
```

## Modules

### x/vm (EVM)
The core EVM execution engine based on go-ethereum. Processes Ethereum-format transactions, manages EVM state, and provides full EVM opcode compatibility.

### x/feemarket (EIP-1559)
Dynamic base fee calculation following EIP-1559 for EVM transactions. Manages gas pricing with elastic block sizes.

### x/erc20
Automatic ERC-20 token pair registration. Enables native QoreChain tokens to be used as ERC-20 tokens in the EVM context and vice versa.

### x/precisebank
Wraps the bank module to handle the decimal precision difference between QoreChain (6 decimals for uqor) and EVM (18 decimals expected by tooling).

## Transaction Routing

QoreChain uses a dual ante handler that automatically routes transactions:

- **Ethereum transactions** (containing `ExtensionOptionsEthereumTx`): Routed through the EVM mono decorator → `EVMMonoDecorator` → `TxListenerDecorator`
- **QoreChain SDK transactions**: Routed through the standard QoreChain SDK path with PQC verification, AI anomaly detection, and CosmWasm decorators

```
              ┌─ Has EthereumTx extension? ─┐
              │                              │
             YES                            NO
              │                              │
     EVM Path:                    QoreChain SDK Path:
     EVMMonoDecorator             SetUpContext
     TxListenerDecorator          WasmLimitSim
                                  WasmCountTX
                                  WasmGasRegister
                                  WasmTxContracts
                                  CircuitBreaker
                                  PQCVerify
                                  AIAnomaly
                                  RejectEVMMessages
                                  ExtensionOptions
                                  ValidateBasic
                                  ...
                                  SigVerify
                                  IncrementSequence
```

## JSON-RPC

### Standard Ethereum Namespaces
QoreChain supports the standard Ethereum JSON-RPC namespaces:
- `eth_` — Ethereum state and transaction queries
- `web3_` — Web3 utility methods
- `net_` — Network information
- `txpool_` — Transaction pool queries

### Configuration

The JSON-RPC server is configured in `app.toml`:

```toml
[json-rpc]
enable = true
address = "127.0.0.1:8545"
ws-address = "127.0.0.1:8546"
api = "eth,web3,net,txpool,qor"
```

### Connecting with MetaMask / Ethers.js

```javascript
// MetaMask: Add custom network
// Network Name: QoreChain Diana Testnet
// RPC URL: http://localhost:8545
// Chain ID: (from genesis)
// Currency Symbol: QOR

// Ethers.js
const provider = new ethers.JsonRpcProvider("http://localhost:8545");
const balance = await provider.getBalance("0x...");
```

## Gas and Denomination

QoreChain uses `uqor` (6 decimals) as its base denomination. The EVM layer handles the decimal conversion transparently through the `x/precisebank` module.

- **QoreChain side**: 1 QOR = 1,000,000 uqor
- **EVM side**: Gas prices and values are expressed in the EVM's native precision

## Precompiles

QoreChain registers the standard QoreChain EVM precompiles for interacting with SDK modules from Solidity:

| Address | Precompile | Description |
|---------|-----------|-------------|
| `0x...0800` | Bank | Native token transfers |
| `0x...0801` | Staking | Delegation management |
| `0x...0802` | Distribution | Reward claims |
| `0x...0803` | IBC Transfer | Cross-chain transfers |
| `0x...0804` | Governance | Proposal voting |
| `0x...0900` | ERC-20 | Token pair operations |
| `0x...0901` | CrossVM | Cross-VM contract calls |

The CrossVM precompile (`0x...0901`) enables Solidity contracts to call CosmWasm contracts synchronously. See [CROSSVM.md](./CROSSVM.md) for details.

## Deploying Contracts

### Using Hardhat

```javascript
// hardhat.config.js
module.exports = {
  networks: {
    qorechain: {
      url: "http://localhost:8545",
      accounts: ["0x..."],
      chainId: 1234,  // Replace with actual chain ID
    }
  }
};
```

### Using Foundry

```bash
forge create --rpc-url http://localhost:8545 \
  --private-key 0x... \
  src/MyContract.sol:MyContract
```

## PQC Considerations

In the current release (v0.5.0), EVM transactions use classical ECDSA signatures (secp256k1). This is required for compatibility with existing Ethereum tooling (MetaMask, Hardhat, etc.).

The architecture is modular to support future PQC integration with the EVM path:
- Extension point in the EVM ante handler for a future `PQCVerifyEVM` decorator
- The `x/crossvm` module already signs cross-VM messages with Dilithium-5 when both VMs are involved

## Genesis Configuration

The EVM modules are initialized in genesis with the following order:
1. `feemarket` (must come before `evm`)
2. `precisebank` (must come before `evm`)
3. `evm`
4. `erc20`

Default EVM genesis parameters can be customized in `genesis.json` under the respective module keys.
