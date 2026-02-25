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

## JSON-RPC (SVM — Solana-Compatible)

Port: 8899 (HTTP)

The SVM runtime exposes a Solana-compatible JSON-RPC interface. Existing Solana clients (e.g., `@solana/web3.js`) can connect directly.

| Method | Parameters | Description |
|--------|-----------|-------------|
| `getAccountInfo` | `pubkey (base58)` | Account data, owner, lamports, executable flag |
| `getBalance` | `pubkey (base58)` | Account balance in lamports |
| `getSlot` | (none) | Current slot number (block height + offset) |
| `getMinimumBalanceForRentExemption` | `dataLength (number)` | Minimum lamports for rent-exempt account |
| `getVersion` | (none) | Runtime version (`1.18.0-qorechain`) |
| `getHealth` | (none) | Health check (`"ok"`) |

### Example

```bash
curl -X POST http://localhost:8899 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getSlot","params":[]}'
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
