# QoreChain API Reference

## Standard QoreChain SDK Endpoints

All standard QoreChain SDK REST and gRPC endpoints are available:
- REST: `http://localhost:1317`
- gRPC: `localhost:9090`
- RPC: `http://localhost:26657`

## Custom Module Endpoints

### AI Module

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/ai/v1/config` | GET | AI module configuration |
| `/qorechain/ai/v1/stats` | GET | AI processing statistics |
| `/qorechain/ai/v1/fee-estimate` | GET | Fee estimation (query: urgency=fast\|normal\|slow) |
| `/qorechain/ai/v1/fraud/investigations` | GET | List fraud investigations |
| `/qorechain/ai/v1/fraud/investigations/{id}` | GET | Investigation details |
| `/qorechain/ai/v1/network/recommendations` | GET | Network optimization recommendations |
| `/qorechain/ai/v1/circuit-breakers` | GET | Active circuit breakers |

### Bridge Module

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/bridge/v1/chains` | GET | Supported chains |
| `/qorechain/bridge/v1/chains/{chain_id}` | GET | Chain details |
| `/qorechain/bridge/v1/validators` | GET | Bridge validator set |
| `/qorechain/bridge/v1/operations` | GET | Recent bridge operations |
| `/qorechain/bridge/v1/operations/{op_id}` | GET | Operation details |
| `/qorechain/bridge/v1/locked/{chain}/{asset}` | GET | Locked/minted amounts |
| `/qorechain/bridge/v1/limits/{chain}` | GET | Circuit breaker limits |
| `/qorechain/bridge/v1/estimate` | GET | AI-optimized route estimate |

### PQC Module

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/pqc/v1/params` | GET | PQC module parameters |
| `/qorechain/pqc/v1/accounts/{address}` | GET | PQC account info |
| `/qorechain/pqc/v1/stats` | GET | PQC verification statistics |

### Reputation Module

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/reputation/v1/validators` | GET | All validator reputation scores |
| `/qorechain/reputation/v1/validators/{address}` | GET | Specific validator score |

### Cross-VM Module

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/crossvm/v1/message/{id}` | GET | Cross-VM message by ID |
| `/qorechain/crossvm/v1/pending` | GET | Pending cross-VM messages |
| `/qorechain/crossvm/v1/params` | GET | Module parameters |

### Multilayer Module

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/multilayer/v1/layer/{id}` | GET | Layer info by ID |
| `/qorechain/multilayer/v1/layers` | GET | All registered layers |
| `/qorechain/multilayer/v1/anchor/{id}` | GET | State anchor by ID |
| `/qorechain/multilayer/v1/anchors` | GET | All state anchors |
| `/qorechain/multilayer/v1/routing-stats` | GET | Routing statistics |
| `/qorechain/multilayer/v1/params` | GET | Module parameters |

### SVM Module

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/svm/v1/params` | GET | SVM module parameters |
| `/qorechain/svm/v1/account/{address}` | GET | SVM account info by base58 address |
| `/qorechain/svm/v1/program/{address}` | GET | Deployed program info by base58 address |

### RL Consensus Module

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/rlconsensus/v1/agent` | GET | RL agent status (mode, epoch, circuit breaker state) |
| `/qorechain/rlconsensus/v1/observation` | GET | Latest 25-dimension observation vector |
| `/qorechain/rlconsensus/v1/rewards` | GET | Reward history with component breakdown |
| `/qorechain/rlconsensus/v1/params` | GET | Module parameters |
| `/qorechain/rlconsensus/v1/policy` | GET | Current policy network metadata |

### Burn Module

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/burn/v1/stats` | GET | Burn statistics: total burned, per-source breakdown, last burn height |
| `/qorechain/burn/v1/params` | GET | Burn module parameters (distribution weights, enable flag) |

### xQORE Module

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/xqore/v1/position/{address}` | GET | xQORE position for an address (locked, balance, lock time) |
| `/qorechain/xqore/v1/params` | GET | xQORE module parameters (penalty tiers, lock settings) |

### Inflation Module

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/inflation/v1/rate` | GET | Current inflation rate |
| `/qorechain/inflation/v1/epoch` | GET | Current epoch info (epoch number, year, total minted) |
| `/qorechain/inflation/v1/params` | GET | Inflation module parameters (epoch length, rate schedule) |

### RDK Module (v1.3.0)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/rdk/v1/rollup/{rollup_id}` | GET | Rollup configuration and status |
| `/qorechain/rdk/v1/rollups` | GET | List all registered rollups |
| `/qorechain/rdk/v1/batch/{rollup_id}/{batch_index}` | GET | Settlement batch details |
| `/qorechain/rdk/v1/batches/{rollup_id}` | GET | List batches for a rollup |
| `/qorechain/rdk/v1/blob/{rollup_id}/{blob_index}` | GET | DA blob details |
| `/qorechain/rdk/v1/params` | GET | RDK module parameters |

### Babylon Module (v1.2.0)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/babylon/v1/staking/{address}` | GET | BTC staking position |
| `/qorechain/babylon/v1/checkpoint/{epoch}` | GET | BTC checkpoint for epoch |
| `/qorechain/babylon/v1/params` | GET | Babylon module parameters |

### Abstract Account Module (v1.2.0)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/abstractaccount/v1/account/{address}` | GET | Abstract account details |
| `/qorechain/abstractaccount/v1/params` | GET | Module parameters |

### FairBlock Module (v1.2.0)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/fairblock/v1/config` | GET | FairBlock tIBE configuration |
| `/qorechain/fairblock/v1/params` | GET | Module parameters |

### Gas Abstraction Module (v1.2.0)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/qorechain/gasabstraction/v1/accepted-tokens` | GET | Accepted fee tokens and conversion rates |
| `/qorechain/gasabstraction/v1/params` | GET | Module parameters |

## JSON-RPC (EVM)

Port: 8545 (HTTP), 8546 (WebSocket)

### Standard Ethereum Namespaces

| Namespace | Description |
|-----------|-------------|
| `eth_` | Ethereum state, transactions, blocks |
| `web3_` | Web3 utility methods |
| `net_` | Network information |
| `txpool_` | Transaction pool queries |

### Custom `qor_` Namespace

| Method | Parameters | Description |
|--------|-----------|-------------|
| `qor_getPQCKeyStatus` | `address` | PQC key registration status |
| `qor_getHybridSignatureMode` | (none) | Current hybrid signature mode (disabled/optional/required) |
| `qor_getAIStats` | (none) | AI module statistics and configuration |
| `qor_getCrossVMMessage` | `messageId` | Cross-VM message status by ID |
| `qor_getReputationScore` | `validator` | Validator reputation score breakdown |
| `qor_getLayerInfo` | `layerId` | Multilayer chain layer information |
| `qor_getBridgeStatus` | `chainId` | Bridge connection status for a chain |
| `qor_getRLAgentStatus` | (none) | RL agent mode, epoch, active state, circuit breaker |
| `qor_getRLObservation` | (none) | Latest 25-dimension observation vector with named dimensions |
| `qor_getRLReward` | (none) | Latest reward signal with per-component breakdown |
| `qor_getPoolClassification` | `validator` | Validator pool assignment (rpos/dpos/pos) |
| `qor_getBurnStats` | (none) | Total burned, per-source breakdown, last burn height |
| `qor_getXQOREPosition` | `address` | xQORE position: locked QOR, xQORE balance, lock time |
| `qor_getInflationRate` | (none) | Current inflation rate, epoch, year, total minted |
| `qor_getTokenomicsOverview` | (none) | Combined tokenomics dashboard (burn + xQORE + inflation) |
| `qor_getRollupStatus` | `rollupId` | Rollup configuration, status, and settlement mode |
| `qor_listRollups` | (none) | All registered rollups with status summary |
| `qor_getSettlementBatch` | `rollupId`, `batchIndex` | Settlement batch details and finalization status |
| `qor_suggestRollupProfile` | `useCase` | AI-assisted rollup profile recommendation |
| `qor_getDABlobStatus` | `rollupId`, `blobIndex` | Data availability blob storage status |
| `qor_getBTCStakingPosition` | `address` | BTC restaking position via Babylon adapter |
| `qor_getAbstractAccount` | `address` | Abstract account details and spending rules |
| `qor_getFairBlockStatus` | (none) | FairBlock tIBE module status and configuration |
| `qor_getGasAbstractionConfig` | (none) | Accepted fee tokens and conversion rates |
| `qor_getLaneConfiguration` | (none) | Transaction lane priorities and block space allocation |

## JSON-RPC (SVM — Solana-Compatible)

Port: 8899 (HTTP)

The SVM runtime exposes a Solana-compatible JSON-RPC interface with 20 methods. Existing Solana clients (e.g., `@solana/web3.js`) can connect directly.

### Account & State Queries

| Method | Parameters | Returns | Description |
|--------|-----------|---------|-------------|
| `getAccountInfo` | `pubkey (base58)` | `{data, executable, lamports, owner, rentEpoch}` | Account data, owner, lamports, executable flag |
| `getBalance` | `pubkey (base58)` | `{value: number}` | Account balance in lamports |
| `getMultipleAccounts` | `pubkeys (base58[])` | `{value: AccountInfo[]}` | Batch-fetch multiple accounts in a single call |
| `getProgramAccounts` | `programId (base58)`, `filters? (object[])` | `[{pubkey, account}]` | All accounts owned by a program, with optional memcmp/dataSize filters |
| `getSlot` | (none) | `number` | Current slot number (block height + offset) |
| `getBlockHeight` | (none) | `number` | Current block height |
| `getMinimumBalanceForRentExemption` | `dataLength (number)` | `number` | Minimum lamports for rent-exempt account |
| `getVersion` | (none) | `{solana-core, feature-set}` | Runtime version (`1.18.0-qorechain`) |
| `getHealth` | (none) | `"ok"` | Health check |

### Transaction Methods

| Method | Parameters | Returns | Description |
|--------|-----------|---------|-------------|
| `sendTransaction` | `signedTx (base64)`, `options? {encoding, skipPreflight, preflightCommitment}` | `signature (base58)` | Submit a signed transaction for on-chain execution |
| `simulateTransaction` | `signedTx (base64)`, `options? {sigVerify, replaceRecentBlockhash}` | `{err, logs, accounts, unitsConsumed}` | Simulate a transaction without committing state changes |
| `getTransaction` | `signature (base58)`, `options? {encoding, commitment}` | `{slot, transaction, meta, blockTime}` | Full transaction details and execution metadata by signature |
| `getSignaturesForAddress` | `address (base58)`, `options? {limit, before, until}` | `[{signature, slot, err, memo, blockTime}]` | Transaction signatures involving an address (newest first) |

### Blockhash & Fee Methods

| Method | Parameters | Returns | Description |
|--------|-----------|---------|-------------|
| `getRecentBlockhash` | (none) | `{blockhash, feeCalculator}` | Recent blockhash for transaction signing |
| `getLatestBlockhash` | `options? {commitment}` | `{blockhash, lastValidBlockHeight}` | Latest blockhash with validity window |
| `isBlockhashValid` | `blockhash (base58)`, `options? {commitment}` | `{value: boolean}` | Check if a blockhash is still within its validity window |
| `getFeeForMessage` | `message (base64)`, `options? {commitment}` | `{value: number}` | Estimated fee in lamports for a serialized transaction message |

### Token Methods

| Method | Parameters | Returns | Description |
|--------|-----------|---------|-------------|
| `getTokenAccountsByOwner` | `owner (base58)`, `{mint (base58)}` or `{programId (base58)}` | `[{pubkey, account}]` | All token accounts owned by a wallet, filtered by mint or token program |
| `getTokenAccountsByDelegate` | `delegate (base58)`, `{mint (base58)}` or `{programId (base58)}` | `[{pubkey, account}]` | Token accounts where the given address has been approved as delegate |

### Testnet Utilities

| Method | Parameters | Returns | Description |
|--------|-----------|---------|-------------|
| `requestAirdrop` | `pubkey (base58)`, `lamports (number)` | `signature (base58)` | Request a testnet airdrop of lamports to an account |

### Example

```bash
# Get account info
curl -X POST http://localhost:8899 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getAccountInfo","params":["<base58-address>"]}'

# Send a transaction
curl -X POST http://localhost:8899 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"sendTransaction","params":["<base64-signed-tx>"]}'

# Get token accounts for a wallet
curl -X POST http://localhost:8899 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getTokenAccountsByOwner","params":["<owner-base58>",{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"}]}'

# Get latest blockhash
curl -X POST http://localhost:8899 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getLatestBlockhash","params":[]}'
```

## AI Sidecar gRPC

Port: 50051

| Service | RPC | Description |
|---------|-----|-------------|
| AISidecar | AnalyzeTransaction | Fast-path heuristic analysis |
| AISidecar | DeepAnalyzeContract | QCAI Backend-powered contract analysis |
| AISidecar | DetectFraud | Deep fraud analysis |
| AISidecar | EstimateFee | Fee prediction |
| AISidecar | GenerateContract | AI contract generation (17 chains) |
| AISidecar | AuditContract | AI security audit |
| AISidecar | OptimizeNetwork | Network optimization advice |
| AISidecar | HealthCheck | Service health |
