# Bridge Architecture

The `x/bridge` module connects QoreChain to the broader blockchain ecosystem through **25 cross-chain connections**: 8 IBC (Inter-Blockchain Communication) channels and 17 QCB (QoreChain Bridge) endpoints. Every bridge operation is secured by post-quantum cryptography.

---

## Connection Overview

QoreChain supports two bridge protocols operating in parallel:

| Protocol | Connections | Security Model | Use Case |
|---|---|---|---|
| **IBC** | 8 channels | Standard IBC + PQC packet signatures | QoreChain SDK-compatible chains |
| **QCB** | 17 endpoints | 7-of-10 Dilithium-5 multisig | Non-IBC chains (EVM, Solana, TON, etc.) |

**Total**: 25 active bridge connections across 12 distinct chain types.

---

## IBC Channels

QoreChain maintains IBC connections to the following chains, relayed via Hermes v1.x:

| Chain | Description |
|---|---|
| Cosmos Hub | Primary hub connection |
| Osmosis | DEX liquidity routing |
| Noble | USDC native issuance |
| Celestia | Data availability layer |
| Stride | Liquid staking |
| Akash | Decentralized compute |
| Babylon | BTC restaking protocol |
| Loopback | Internal testing channel |

### IBC Relayer Configuration

- **Relayer software**: Hermes v1.x
- **Client updates**: Automatic light client refresh
- **Misbehaviour detection**: Enabled -- relayer monitors for equivocation
- **Packet clearing**: Every 100 blocks, pending IBC packets are cleared
- **PQC enhancement**: Every IBC packet originating from QoreChain includes an optional Dilithium-5 signature for forward quantum security. PQC-aware receiving chains can verify this signature alongside standard IBC verification.

---

## QCB (QoreChain Bridge) Protocol

The QCB protocol uses a hub-and-spoke architecture secured by post-quantum cryptography. QoreChain acts as the hub, with spoke connections to each external chain.

### Supported Chain Types

The bridge supports 12 distinct chain architectures:

| Chain Type | Chains | Address Format |
|---|---|---|
| `evm` | Ethereum, BNB Smart Chain, Avalanche, Polygon, Arbitrum, Optimism, Base | `0x` + 40 hex characters |
| `solana` | Solana | Base58, 32-44 characters |
| `ton` | TON | `EQ` + base64 encoded |
| `sui_move` | Sui | `0x` + 64 hex characters |
| `aptos_move` | Aptos | `0x` + 64 hex characters |
| `bitcoin` | Bitcoin | Bech32 (`bc1`), P2SH (`3...`), or legacy (`1...`) |
| `near` | NEAR Protocol | `.near` suffix or implicit |
| `cardano` | Cardano | `addr1` (payment) or `stake1` (staking) |
| `polkadot` | Polkadot | SS58 encoded |
| `tezos` | Tezos | `tz1`/`tz2`/`tz3` (implicit) or `KT1` (originated) |
| `tron` | TRON | `T` + base58, 34 characters |

### QCB Chain Connections (17 Endpoints)

| Chain | Type | Min Confirmations | Supported Assets |
|---|---|---|---|
| Ethereum | EVM | 12 | ETH, USDC, USDT, WBTC |
| Solana | Solana | 32 | SOL, USDC |
| TON | TON | 10 | TON, USDT |
| BNB Smart Chain | EVM | 15 | BNB, USDC, USDT |
| Avalanche C-Chain | EVM | 12 | AVAX, USDC |
| Polygon PoS | EVM | 128 | POL, USDC, USDT, WETH |
| Arbitrum One | EVM | 64 | ETH, USDC, ARB, USDT |
| Sui | Sui | 3 | SUI, USDC |
| Optimism | EVM | 10 | ETH, USDC, OP |
| Base | EVM | 10 | ETH, USDC |
| Aptos | Aptos | 6 | APT, USDC |
| Bitcoin | Bitcoin | 6 | BTC |
| NEAR Protocol | NEAR | 3 | NEAR, USDC |
| Cardano | Cardano | 15 | ADA |
| Polkadot | Polkadot | 12 | DOT |
| Tezos | Tezos | 2 | XTZ |
| TRON | TRON | 20 | TRX, USDT |

---

## Deposit Flow (External to QoreChain)

```
External Chain          QoreChain Validators           QoreChain
     |                         |                          |
     | 1. Lock assets on       |                          |
     |    bridge contract      |                          |
     |------------------------>|                          |
     |                         | 2. Observe & attest      |
     |                         |    (7/10 PQC sigs)       |
     |                         |------------------------->|
     |                         |                          | 3. Mint wrapped
     |                         |                          |    tokens
     |                         |                          |
     |                         |    [If > 100K QOR]       |
     |                         |    24h challenge period   |
     |                         |    before execution       |
```

1. **Lock**: User locks assets in the bridge contract on the external chain.
2. **Attest**: Bridge validators observe the lock transaction and submit Dilithium-5 signed attestations. A minimum of **7 out of 10** validator attestations are required.
3. **Mint**: Once the attestation threshold is met, wrapped tokens are minted on QoreChain.
4. **Challenge period**: For transfers exceeding 100,000 QOR equivalent, a **24-hour challenge period** applies before execution. During this window, validators can flag suspicious activity.

---

## Withdrawal Flow (QoreChain to External)

```
QoreChain               QoreChain Validators           External Chain
     |                         |                          |
     | 1. Burn wrapped tokens  |                          |
     |------------------------>|                          |
     |                         | 2. Attest burn           |
     |                         |    (7/10 PQC sigs)       |
     |                         |------------------------->|
     |                         |                          | 3. Unlock original
     |                         |                          |    assets
```

1. **Burn**: User burns wrapped tokens on QoreChain.
2. **Attest**: Validators attest to the burn event with Dilithium-5 signatures.
3. **Unlock**: Once the threshold is reached, original assets are unlocked on the external chain.

All bridge fees collected during withdrawals are routed to the `x/burn` module via the `bridge_fee` burn channel (100% of bridge fees are burned).

---

## Security Architecture

### PQC Multisig

All QCB bridge operations require a **7-of-10 threshold** of Dilithium-5 post-quantum signatures from registered bridge validators. Each bridge validator registers with:

- A QoreChain validator address
- A Dilithium-5 public key (2,592 bytes)
- A list of supported chains
- A reputation score (maintained by `x/reputation`)

### Circuit Breakers

Each connected chain has independent circuit breaker protections:

| Protection | Description |
|---|---|
| **Single transfer limit** | Maximum amount for any individual bridge operation per chain |
| **Daily aggregate limit** | Total volume cap per chain per 24-hour window |
| **Manual pause** | Governance or validator-triggered emergency halt per chain |
| **Anomaly detection** | Automatic pause if >50 operations in a short window or volume exceeds 5x daily limit |

Circuit breaker state is tracked per chain and includes: max single transfer, daily limit, current daily usage, last reset height, and pause status with reason.

### Challenge Period

For large transfers (>100,000 QOR equivalent, configurable via `large_transfer_threshold`):

- A **24-hour challenge period** (86,400 seconds) applies after attestation threshold is met.
- During this window, any validator can flag the operation.
- If unchallenged, the operation executes automatically after the period expires.
- Challenged operations are frozen for governance review.

### AI Path Optimization

The bridge module integrates with the AI subsystem for route optimization. For transfers that can traverse multiple paths (e.g., chain A to chain B via intermediary), the path optimizer evaluates:

- Estimated fees across routes
- Estimated completion time
- Security score per path
- Confidence level of the estimate

---

## REST API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/bridge/v1/chains` | List all supported chain configurations |
| GET | `/bridge/v1/chains/{chain_id}` | Get configuration for a specific chain |
| GET | `/bridge/v1/validators` | List all registered bridge validators |
| GET | `/bridge/v1/operations` | List all bridge operations (most recent first) |
| GET | `/bridge/v1/operations/{operation_id}` | Get details of a specific operation |
| GET | `/bridge/v1/locked/{chain}/{asset}` | Get locked/minted amounts for a chain/asset pair |
| GET | `/bridge/v1/circuit-breakers` | List all circuit breaker states |
| GET | `/bridge/v1/estimate/{from}/{to}/{asset}/{amount}` | Get AI-optimized route estimate |

---

## Bridge Events

The bridge module emits the following on-chain events:

| Event Type | Description |
|---|---|
| `bridge_deposit` | New deposit operation created |
| `bridge_withdraw` | New withdrawal operation created |
| `bridge_attestation` | Validator attestation submitted |
| `bridge_operation_executed` | Operation finalized and executed |
| `bridge_circuit_breaker_trip` | Circuit breaker activated or deactivated |
| `bridge_validator_registered` | New bridge validator registered |
| `bridge_pqc_verification` | PQC signature verification result (IBC packets) |
