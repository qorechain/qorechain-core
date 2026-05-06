# QoreChain Bridge (QCB) Documentation

## Overview

The QoreChain Bridge implements a hub-and-spoke multi-protocol bridge with PQC-secured operations.

## Supported Chains

The bridge supports **37 default chain configurations** registered in `x/bridge/types/DefaultChainConfigs()` plus 8 IBC chains. All bridge operations are gated by per-chain on-chain licenses (see `x/license`).

### Baseline EVM and major non-EVM (10)

| Chain | Type | Protocol | Status |
|-------|------|----------|--------|
| Ethereum | EVM | QCB Native + IBC | Testnet |
| Solana | Solana | QCB Native + IBC | Testnet |
| TON | TON | QCB Native + IBC | Testnet |
| BSC | EVM | QCB Native + IBC | Testnet |
| Avalanche | EVM | QCB Native + IBC | Testnet |
| Polygon | EVM (PoS) | QCB Native + IBC | Testnet |
| Arbitrum | Ethereum L2 | QCB Native + IBC | Testnet |
| Optimism | Ethereum L2 (OP Stack) | QCB Native | Testnet |
| Base | Ethereum L2 (OP Stack) | QCB Native | Testnet |
| Sui | Move VM | QCB Native + IBC | Testnet |

### Cross-network expansion v2.24.0–v2.34.0 — EVM-family (14)

zkSync Era (L2 ZK), Linea (L2 ZK), Scroll (L2 ZK), Blast (L2 Optimistic, yield-bearing), Mantle (L2), Hyperliquid (HyperEVM L1), Berachain (L1, PoL), Sonic (L1), Sei (parallel EVM L1, dual EVM+IBC), Monad (parallel EVM L1, 30-block finality), Plasma (L1, stablecoin-focused, BTC-anchored), Filecoin FVM, Cronos, Kaia.

### Cross-network expansion v2.24.0–v2.34.0 — non-EVM (5)

| Chain | Architecture | Protocol | Default Confirmations |
|-------|--------------|----------|------------------------|
| Starknet | Cairo VM L2 | Dedicated handler + L1 state-update + STARK proof | 12 (L1) |
| XRP Ledger | UNL consensus | Dedicated handler | 4 ledger closes |
| Stellar | SCP | Dedicated handler | 5 ledger closes |
| Hedera | Hashgraph | Dedicated handler + HCS subscription | 4 consensus rounds |
| Algorand | Pure PoS | Dedicated handler | 4 rounds |

### IBC-connected chains (8)

Cosmos Hub, Osmosis, Noble, Celestia, Stride, Akash, Babylon, Injective. Packet flow via Hermes relayer; no sidecar container required.

### Other (NEAR, Bitcoin, Cardano, Polkadot, Tezos, Tron, Aptos)

These 7 chains have bridge configs but are flagged as `Pending` until production handlers ship.

### License surface

- **74 license feature IDs total** in `x/license/types/feature_ids.go`:
  - 1 umbrella (`qcb_bridge`)
  - 36 per-chain `bridge_*`
  - 37 per-chain `validator_*` (10 baseline + 19 non-IBC v2.27.0 + 8 IBC v2.27.0)
- Helper functions: `AllBridgeFeatureIDs()`, `AllValidatorFeatureIDs()`, `AllFeatureIDs()`, `IsValidFeatureID()`, `ChainFromFeature()`

## Architecture

```
                     PQC Security Perimeter
      +----------------------------------------------+
      |                                              |
ETH <-+-> ETH Bridge    +                           |
      |   (QCB Native)  |                           |
SOL <-+-> SOL Bridge    +-> AI Path -> QoreChain    |
      |   (QCB Native)  |   Optimizer  Hub          |
TON <-+-> TON Bridge    |                           |
      |   (QCB Native)  |                           |
BSC <-+-> EVM Bridge    +                           |
      |   (QCB Native)                               |
POLY<-+-> Polygon Bridge                             |
      |   (QCB Native)                               |
ARB <-+-> Arbitrum Bridge                            |
      |   (QCB Native)                               |
SUI <-+-> Sui Bridge                                 |
      |   (QCB Native)                               |
      |                                              |
IBC chains <-> IBC Module (native, no bridge needed)     |
      |                                              |
      +----------------------------------------------+
```

## Security

1. **7-of-10 Multisig**: Bridge validator set requires 7 PQC signatures from 10 validators
2. **24-hour Challenge Period**: Large withdrawals (>100K QOR equivalent) have a 24h delay
3. **Circuit Breaker**: Per-chain limits on single transfers and daily aggregates
4. **PQC Attestations**: All bridge validator signatures use Dilithium-5
5. **ML-KEM Commitments**: Quantum-safe commitment scheme for bridge operations
6. **AI Path Optimization**: Optimal route selection across bridge paths

## Operations

### Deposit (External -> QoreChain)
1. User locks assets on external chain
2. Bridge validators observe the lock transaction
3. Validators submit PQC-signed attestations
4. When threshold (7/10) is met, minted assets are created on QoreChain
5. Large deposits enter 24h challenge period

### Withdrawal (QoreChain -> External)
1. User initiates withdrawal on QoreChain
2. Minted assets are burned
3. Bridge validators attest to the withdrawal
4. When threshold is met, assets are unlocked on external chain

## API Endpoints

```
GET /qorechain/bridge/v1/chains
GET /qorechain/bridge/v1/chains/{chain_id}
GET /qorechain/bridge/v1/validators
GET /qorechain/bridge/v1/operations
GET /qorechain/bridge/v1/operations/{op_id}
GET /qorechain/bridge/v1/locked/{chain}/{asset}
GET /qorechain/bridge/v1/limits/{chain}
GET /qorechain/bridge/v1/estimate?from=ethereum&to=qorechain&asset=ETH&amount=1.0
```
