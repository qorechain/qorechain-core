# AI Engine

QoreChain integrates AI capabilities at multiple levels of the protocol stack through the `x/ai` module. The on-chain layer provides deterministic heuristic-based analysis suitable for consensus-critical operations, while an off-chain sidecar extends capabilities with deep learning models for advisory and developer tooling.

---

## Three-Layer Architecture

The QCAI (QoreChain AI) engine operates across three layers:

| Layer | Scope | Execution | Deterministic |
|-------|-------|-----------|---------------|
| **Consensus Level** | Block production, parameter tuning | On-chain (x/rlconsensus) | Yes |
| **Network Level** | Transaction routing, fraud detection, fee optimization | On-chain (x/ai) | Yes |
| **Application Level** | Contract generation, auditing, deep analysis | Off-chain (sidecar) | No |

The consensus level is documented separately in [RL Consensus Engine](rl-consensus.md). This page covers the network and application levels.

---

## Transaction Router

The AI-enhanced router selects optimal validators and routes for each transaction using weighted multi-factor scoring.

### Optimization Formula

```
OptimalRoute = argmin_r( alpha * Latency(r) + beta * Cost(r) + gamma * Security(r)^-1 )
```

| Weight | Symbol | Default | Description |
|--------|--------|---------|-------------|
| Latency | alpha | 0.4 | Normalized response time (0=best, 1=worst). 0ms maps to 0.0, 1000ms maps to 1.0. |
| Cost | beta | 0.3 | Current load percentage as a proxy for cost. |
| Security | gamma | 0.3 | Inverse of reputation score. Higher reputation yields a lower (better) score. |

The router maintains a **metrics cache** (default TTL: 30 seconds) with per-validator performance data including average latency, uptime percentage, load percentage, and reputation score. When cached metrics are unavailable, the system falls back to the heuristic router.

### Routing Confidence

Confidence scales with the number of validators with available metrics:

| Validators with Metrics | Confidence |
|------------------------|------------|
| >= 10 | 0.95 |
| >= 5 | 0.85 |
| >= 2 | 0.75 |
| 1 | 0.60 |

---

## Fraud Detection

The fraud detector implements a **six-layer detection pipeline** that analyzes each transaction against recent history using statistical methods.

### Detection Layers

| Layer | Detector | Method | Trigger Threshold |
|-------|----------|--------|-------------------|
| 1 | **Isolation Forest** | Statistical Z-score across amount, gas, and sender frequency features | Anomaly score > 0.7 |
| 2 | **Sequence Analyzer** | Detects alternating send/receive patterns (wash trading) | > 3 alternating transfers between same pair |
| 3 | **Sybil Detector** | Tracks new unique addresses; flags spikes in new senders | > 30% of recent transactions from new addresses |
| 4 | **DDoS Detector** | Monitors per-sender transaction frequency | > 100 transactions per minute from a single sender |
| 5 | **Flash Loan Detector** | Identifies borrow-manipulate-repay patterns within a single block | >= 3 transactions in same block with > 10x amount variance |
| 6 | **Exploit Detector** | Flags abnormal gas consumption in contract calls | > 5x average gas for the same transaction type |

### Threat Classification

| Confidence Range | Threat Level |
|-----------------|--------------|
| >= 0.9 | Critical |
| >= 0.7 | High |
| >= 0.5 | Medium |
| >= 0.3 | Low |
| < 0.3 | None |

### Response Actions

| Threat Level | Confidence | Action |
|-------------|------------|--------|
| Critical | > 0.8 | `circuit_break` -- Pause specific contract executions |
| Critical | <= 0.8 | `rate_limit` -- Temporarily reduce TX acceptance from source |
| High | > 0.7 | `rate_limit` |
| High | <= 0.7 | `alert` -- Emit event for validators and operators |
| Medium | Any | `alert` |
| Low / None | Any | `allow` |

When an action other than `allow` is triggered, a fraud investigation record is created with a unique ID (format: `INV-{timestamp}-{txhash_prefix}`).

---

## Fee Optimizer

The fee optimizer predicts network congestion and suggests optimal fees for desired confirmation times using exponential moving average (EMA) congestion tracking.

### Congestion Prediction

- **EMA smoothing factor (alpha)**: 0.2
- **History window**: 100 blocks
- **Trend analysis**: Compares the most recent 5 blocks against the prior 5 blocks to detect congestion trends, then projects forward with 50% dampening.

### Urgency Tiers

| Urgency | Base Multiplier | Estimated Confirmation |
|---------|----------------|----------------------|
| `fast` | 2.0x | 1-2 blocks |
| `normal` | 1.0x | 3-5 blocks |
| `slow` | 0.5x | 6-10 blocks |

The final fee incorporates a **congestion multiplier** (1.0x at 0% congestion, up to 5.0x at 100% congestion) and a **trend premium** when predicted congestion exceeds current congestion. The minimum fee floor is 500 uqor (0.0005 QOR).

---

## Network Optimizer

The network optimizer continuously monitors performance metrics and generates governance parameter recommendations using a multi-objective reward function.

### Reward Function

```
R(s, a, s') = alpha * DeltaPerformance + beta * DeltaLatency + gamma * DeltaEnergy - delta * StabilityPenalty
```

| Weight | Value | Objective |
|--------|-------|-----------|
| alpha | 0.35 | Performance improvement |
| beta | 0.30 | Latency reduction |
| gamma | 0.15 | Energy/resource savings |
| delta | 0.20 | Stability preservation |

### Recommendation Types

The optimizer generates recommendations for:
- **Block gas limit**: Increase when utilization > 80%, decrease when < 20%
- **Minimum commission rate**: Lower when validator count is below 5
- **Maximum validators**: Increase when block times are healthy and >= 10 validators active
- **Block time target**: Alert when average block time exceeds 8 seconds

Each recommendation includes the current value, suggested value, expected impact, confidence score, and reasoning.

---

## AI Sidecar

The QCAI Sidecar extends on-chain AI with off-chain deep learning models, accessible via gRPC on port **50051**. The sidecar is optional and non-consensus-critical.

### Capabilities

| Capability | Description |
|------------|-------------|
| **Contract Generation** | Generates smart contracts from natural language specifications across 17 platforms |
| **Contract Auditing** | Deep security analysis of smart contract code |
| **Deep Fraud Analysis** | Extended fraud investigation using trained models (supplements on-chain heuristics) |
| **Network Advice** | Advanced parameter optimization recommendations |

### Models

| Model Name | Use Case |
|------------|----------|
| QCAI Fast | Low-latency responses for fee estimation and routing |
| QCAI Balanced | Deeper analysis for auditing and fraud investigation |

The sidecar maintains its own Go module (`qorechain-core/sidecar/`) with a separate dependency graph.

---

## EVM Precompiles

Two precompiled contracts expose on-chain AI capabilities to EVM smart contracts:

| Precompile | Address | Description |
|------------|---------|-------------|
| `aiRiskScore` | `0x0B01` | Returns a risk score (0-100) for a given address or transaction hash |
| `aiAnomalyCheck` | `0x0B02` | Returns a boolean anomaly flag and confidence score for a transaction |

**Important**: EVM precompiles use the **deterministic heuristic engine only**. They never call the sidecar, ensuring all EVM execution remains fully deterministic and reproducible.

---

## TEE Attestation (v1.1.0)

The AI module defines interfaces for **Trusted Execution Environment** attestation, enabling future verifiable AI model execution inside secure hardware enclaves.

### Supported Platforms

| Platform | Identifier | Description |
|----------|-----------|-------------|
| Intel SGX | `sgx` | Software Guard Extensions |
| Intel TDX | `tdx` | Trust Domain Extensions |
| AMD SEV-SNP | `sev-snp` | Secure Encrypted Virtualization - Secure Nested Paging |
| ARM CCA | `arm-cca` | Confidential Compute Architecture |

### Attestation Flow

1. The sidecar loads AI model weights into a TEE enclave.
2. Inference runs inside the enclave's protected memory.
3. The enclave produces an attestation report binding the model hash, input hash, and output hash.
4. Validators verify the attestation on-chain before accepting inference results.

TEE attestation is currently at the interface specification stage. Implementation is planned for a future release.

---

## Federated Learning (v1.1.0)

The AI module defines interfaces for **on-chain federated learning** coordination, where validator nodes train local models and submit gradient updates that are aggregated into a global model without sharing raw training data.

### Aggregation Methods

| Method | Description |
|--------|-------------|
| `fedavg` | Federated Averaging -- weighted average of gradients by sample count |
| `fedprox` | Federated Proximal -- adds a proximal term to handle heterogeneous data |
| `scaffold` | SCAFFOLD -- uses control variates to correct for client drift |

### Round Lifecycle

```
Pending --> Training --> Aggregating --> Complete
                                    \-> Failed (timeout or insufficient participants)
```

Each round is configured with minimum/maximum participants, timeout, learning rate, gradient clipping norm, and an optional differential privacy noise multiplier. All gradient submissions are signed with PQC (Dilithium-5) signatures.

Federated learning is currently at the interface specification stage. Implementation is planned for a future release.

---

## REST Endpoints

| Endpoint | Description |
|----------|-------------|
| `/ai/v1/fee-estimate` | Returns fee estimates for fast, normal, and slow urgency tiers |
| `/ai/v1/fraud/investigations` | Lists active and resolved fraud investigations |
| `/ai/v1/network/recommendations` | Returns current network parameter optimization recommendations |
| `/ai/v1/circuit-breakers` | Lists active circuit breaker states for contracts |
