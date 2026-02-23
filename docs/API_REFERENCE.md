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
| `qor_getAIStats` | (none) | AI module statistics and configuration |
| `qor_getCrossVMMessage` | `messageId` | Cross-VM message status by ID |
| `qor_getReputationScore` | `validator` | Validator reputation score breakdown |
| `qor_getLayerInfo` | `layerId` | Multilayer chain layer information |
| `qor_getBridgeStatus` | `chainId` | Bridge connection status for a chain |

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
