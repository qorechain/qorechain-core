# QoreChain API Reference

## Standard Cosmos SDK Endpoints

All standard Cosmos SDK REST and gRPC endpoints are available:
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

## AI Sidecar gRPC

Port: 50051

| Service | RPC | Description |
|---------|-----|-------------|
| AISidecar | AnalyzeTransaction | Fast-path heuristic analysis |
| AISidecar | DeepAnalyzeContract | Bedrock-powered contract analysis |
| AISidecar | DetectFraud | Deep fraud analysis |
| AISidecar | EstimateFee | Fee prediction |
| AISidecar | GenerateContract | AI contract generation (17 chains) |
| AISidecar | AuditContract | AI security audit |
| AISidecar | OptimizeNetwork | Network optimization advice |
| AISidecar | HealthCheck | Service health |
