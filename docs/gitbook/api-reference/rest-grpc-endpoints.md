# REST / gRPC Endpoints

QoreChain exposes three primary interfaces for programmatic access:

| Interface | Default Port | Protocol   | Description                          |
| --------- | ------------ | ---------- | ------------------------------------ |
| REST      | `1317`       | HTTP/1.1   | LCD (Light Client Daemon) REST API   |
| gRPC      | `9090`       | HTTP/2     | Protobuf-encoded gRPC service        |
| RPC       | `26657`      | HTTP + WS  | QoreChain Consensus Engine RPC       |

All REST endpoints return JSON. gRPC endpoints use Protocol Buffers and can be consumed with any gRPC client. The RPC interface provides consensus-level queries and transaction broadcast.

---

## Base URLs

```
REST:  http://localhost:1317
gRPC:  localhost:9090
RPC:   http://localhost:26657
```

---

## AI Module

| Method | Endpoint                                 | Description                                      |
| ------ | ---------------------------------------- | ------------------------------------------------ |
| GET    | `/ai/v1/config`                          | Returns current AI module configuration           |
| GET    | `/ai/v1/stats`                           | Aggregated AI processing statistics               |
| GET    | `/ai/v1/fee-estimate`                    | AI-assisted gas fee estimation for a transaction  |
| GET    | `/ai/v1/fraud/investigations`            | Lists all active fraud investigations             |
| GET    | `/ai/v1/fraud/investigations/{id}`       | Returns details for a specific fraud investigation|
| GET    | `/ai/v1/network/recommendations`         | AI-generated network optimization recommendations |
| GET    | `/ai/v1/circuit-breakers`                | Current circuit breaker states and thresholds     |

---

## Bridge Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/bridge/v1/chains`                      | Lists all registered bridge chains                 |
| GET    | `/bridge/v1/chains/{id}`                 | Details for a specific bridged chain               |
| GET    | `/bridge/v1/validators`                  | Lists active bridge validators                     |
| GET    | `/bridge/v1/operations`                  | Lists recent bridge operations                     |
| GET    | `/bridge/v1/operations/{id}`             | Details for a specific bridge operation             |
| GET    | `/bridge/v1/locked/{chain}/{asset}`      | Total locked value for a chain/asset pair          |
| GET    | `/bridge/v1/limits/{chain}`              | Rate limits and thresholds for a bridged chain     |
| GET    | `/bridge/v1/estimate`                    | Estimates bridge fee and transfer time              |

---

## PQC Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/pqc/v1/params`                         | Current PQC module parameters                      |
| GET    | `/pqc/v1/accounts/{address}`             | PQC key status for a specific account              |
| GET    | `/pqc/v1/stats`                          | Aggregate PQC registration and migration stats     |

---

## Reputation Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/reputation/v1/validators`              | Reputation scores for all validators               |
| GET    | `/reputation/v1/validators/{address}`    | Reputation score for a specific validator           |

---

## Cross-VM Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/crossvm/v1/message/{id}`               | Retrieves a cross-VM message by ID                 |
| GET    | `/crossvm/v1/pending`                    | Lists pending cross-VM messages in queue           |
| GET    | `/crossvm/v1/params`                     | Current Cross-VM module parameters                 |

---

## Multilayer Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/multilayer/v1/layer/{id}`              | Details for a specific layer                       |
| GET    | `/multilayer/v1/layers`                  | Lists all registered layers                        |
| GET    | `/multilayer/v1/anchor/{id}`             | Details for a specific anchor record               |
| GET    | `/multilayer/v1/anchors`                 | Lists recent anchor submissions                    |
| GET    | `/multilayer/v1/routing-stats`           | Transaction routing statistics across layers       |
| GET    | `/multilayer/v1/params`                  | Current Multilayer module parameters               |

---

## SVM Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/svm/v1/params`                         | Current SVM module parameters                      |
| GET    | `/svm/v1/account/{address}`              | SVM account info for a given address               |
| GET    | `/svm/v1/program/{address}`              | Deployed program info for a given program address  |

---

## RL Consensus Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/rlconsensus/v1/agent`                  | Current RL agent status and mode                   |
| GET    | `/rlconsensus/v1/observation`            | Latest observation vector                          |
| GET    | `/rlconsensus/v1/rewards`                | Cumulative reward metrics                          |
| GET    | `/rlconsensus/v1/params`                 | Current RL Consensus module parameters             |
| GET    | `/rlconsensus/v1/policy`                 | Active policy configuration and weights            |

---

## Burn Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/burn/v1/stats`                         | Burn statistics across all channels                |
| GET    | `/burn/v1/params`                        | Current Burn module parameters                     |

---

## xQORE Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/xqore/v1/position/{address}`           | xQORE staking position for a given address         |
| GET    | `/xqore/v1/params`                       | Current xQORE module parameters                   |

---

## Inflation Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/inflation/v1/rate`                     | Current annualized inflation rate                  |
| GET    | `/inflation/v1/epoch`                    | Current epoch number and progress                  |
| GET    | `/inflation/v1/params`                   | Current Inflation module parameters                |

---

## RDK Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/rdk/v1/rollup/{id}`                    | Details for a specific rollup                      |
| GET    | `/rdk/v1/rollups`                        | Lists all registered rollups                       |
| GET    | `/rdk/v1/batch/{id}/{index}`             | Retrieves a specific settlement batch              |
| GET    | `/rdk/v1/batches/{id}`                   | Lists batches for a specific rollup                |
| GET    | `/rdk/v1/blob/{id}/{index}`              | Retrieves a specific DA blob                       |
| GET    | `/rdk/v1/params`                         | Current RDK module parameters                     |

---

## Babylon Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/babylon/v1/staking/{address}`          | BTC staking position for a given address           |
| GET    | `/babylon/v1/checkpoint/{epoch}`         | BTC checkpoint data for a given epoch              |
| GET    | `/babylon/v1/params`                     | Current Babylon module parameters                  |

---

## Abstract Account Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/abstractaccount/v1/account/{address}`  | Abstract account details for a given address       |
| GET    | `/abstractaccount/v1/params`             | Current Abstract Account module parameters         |

---

## FairBlock Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/fairblock/v1/config`                   | Current FairBlock encryption configuration         |
| GET    | `/fairblock/v1/params`                   | Current FairBlock module parameters                |

---

## Gas Abstraction Module

| Method | Endpoint                                 | Description                                       |
| ------ | ---------------------------------------- | ------------------------------------------------- |
| GET    | `/gasabstraction/v1/accepted-tokens`     | Lists tokens accepted for gas payment              |
| GET    | `/gasabstraction/v1/params`              | Current Gas Abstraction module parameters          |

---

## gRPC Reflection

gRPC server reflection is enabled by default, allowing tools such as `grpcurl` to discover available services:

```bash
grpcurl -plaintext localhost:9090 list
```

To query a specific service:

```bash
grpcurl -plaintext localhost:9090 qorechain.pqc.v1.Query/Params
```

---

## Authentication

All REST and gRPC endpoints are unauthenticated by default. For production deployments, place a reverse proxy (e.g., Nginx or Caddy) in front of the node to handle TLS termination and access control.
