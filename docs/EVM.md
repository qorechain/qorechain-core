# QoreChain EVM Runtime

## Overview

QoreChain v0.5.0 introduces a dual VM runtime enabling full EVM (Ethereum Virtual Machine) compatibility alongside the existing Cosmos SDK runtime. This allows developers to deploy Solidity smart contracts on QoreChain while benefiting from the chain's PQC security, AI-native processing, and cross-VM communication capabilities.

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
- **Cosmos SDK transactions**: Routed through the standard Cosmos SDK path with PQC verification, AI anomaly detection, and CosmWasm decorators

```
              ┌─ Has EthereumTx extension? ─┐
              │                              │
             YES                            NO
              │                              │
     EVM Path:                    Cosmos SDK Path:
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
- `eth_` - Ethereum state and transaction queries
- `web3_` - Web3 utility methods
- `net_` - Network information
- `txpool_` - Transaction pool queries (off in a new app.toml, which enables
  `eth,net,web3`; it publishes the node's pending transactions)

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
// Network Name:      QoreChain Diana Testnet
// RPC URL:           http://localhost:8545   (prod: https://evm.<domain>)
// Chain ID:          9800   (0x2648)         (mainnet: 9801 / 0x2649)
// Currency Symbol:   QOR
// Currency Decimals: 18    <- EVM native currency is the 18-decimal `aqor` view
//                            of QOR; do NOT use 6 (the bank `uqor` denom) or
//                            MetaMask shows balances 1e12x off.

// Ethers.js
const provider = new ethers.JsonRpcProvider("http://localhost:8545");
const balance = await provider.getBalance("0x...");
```

> `@qorechain/wallet-adapter` exports `qoreEvmChainParams({rpcUrl, explorerUrl})`
> and `addQoreEvmToWallet(provider, …)` which emit the correct EIP-3085 params
> (18 decimals baked in) so apps don't hand-roll this.

## Gas and Denomination

QoreChain uses `uqor` (6 decimals) as its base denomination. The EVM layer handles the decimal conversion transparently through the `x/precisebank` module.

- **QoreChain side**: 1 QOR = 1,000,000 uqor
- **EVM side**: Gas prices and values are expressed in the EVM's native precision

## Precompiles

The standard cosmos/evm precompiles sit at fixed addresses (cosmos/evm v0.6.3,
`x/vm/types/precompiles.go`). A precompile answers only when it is active:

| Address | Precompile | Notes |
|---------|-----------|-------|
| `0x...0100` | P256 | signature verification, stateless |
| `0x...0400` | Bech32 | address conversion, stateless |
| `0x...0800` | Staking | delegation management |
| `0x...0801` | Distribution | reward claims |
| `0x...0802` | ICS20 | IBC transfers |
| `0x...0803` | Vesting | reserved upstream, not implemented in this build |
| `0x...0804` | Bank | balances and supply, read-only |
| `0x...0805` | Gov | proposals and votes |
| `0x...0806` | Slashing | validator unjail and signing info |

None of these is active on either network: the EVM `active_static_precompiles`
parameter is empty on testnet and mainnet. ERC-20 token pairs are not a fixed
precompile address; there are none registered.

QoreChain's own precompiles are switched on by protocol features. On testnet
the CrossVM bridge (`0x...0901`), pqcVerify (`0x...0A01`), pqcKeyStatus
(`0x...0A02`) and rlConsensusParams (`0x...0C01`) are active; on mainnet none
is. Their interfaces and gas are in [EVM_PRECOMPILES.md](./EVM_PRECOMPILES.md);
the CrossVM bridge lets Solidity contracts call CosmWasm contracts
synchronously, see [CROSSVM.md](./CROSSVM.md).

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

The EVM modules are initialized in genesis in this order (after the IBC
modules, before CosmWasm):
1. `feemarket` (base-fee parameters)
2. `evm` (must come before `precisebank`: its InitGenesis sets the EVM coin
   denom and decimals that precisebank reads for its conversion factor)
3. `precisebank`
4. `erc20`

Default EVM genesis parameters can be customized in `genesis.json` under the respective module keys.
