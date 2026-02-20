# QoreChain Bridge (QCB) Documentation

## Overview

The QoreChain Bridge implements a hub-and-spoke multi-protocol bridge with PQC-secured operations.

## Supported Chains

| Chain | Type | Protocol | Status |
|-------|------|----------|--------|
| Ethereum | EVM | QCB Native + IBC | Testnet |
| Solana | Solana | QCB Native + IBC | Testnet |
| TON | TON | QCB Native + IBC | Testnet |
| BSC | EVM | QCB Native + IBC | Testnet |
| Avalanche | EVM | QCB Native + IBC | Testnet |
| Polygon | EVM (PoS) | QCB Native + IBC | Testnet |
| Arbitrum | Ethereum L2 | QCB Native + IBC | Testnet |
| Sui | Move VM | QCB Native + IBC | Testnet |
| IBC-compatible chains | IBC | QCB Native + IBC | Testnet |

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
