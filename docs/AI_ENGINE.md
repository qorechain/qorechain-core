# AI Engine Documentation

## Overview

The QoreChain AI engine operates at three layers:
1. **Consensus level**: Transaction routing and validator selection
2. **Network level**: Parameter optimization and congestion prediction
3. **Application level**: Fraud detection and smart contract analysis

## Components

### Transaction Router

Optimizes routing using:
```
OptimalRoute = argmin_r(alpha * Latency(r) + beta * Cost(r) + gamma * Security(r)^-1)
```

Default weights: alpha=0.4, beta=0.3, gamma=0.3

### Fraud Detector

Multi-layered detection:
1. **Statistical Isolation Forest**: Z-score anomaly scoring across amount, gas, and sender frequency
2. **Sequence Analyzer**: Detects wash trading patterns (alternating sender/receiver)
3. **Sybil Detector**: Tracks new address spikes
4. **DDoS Detector**: Monitors per-sender transaction rates
5. **Flash Loan Detector**: Identifies same-block amount variance patterns
6. **Exploit Detector**: Flags abnormal gas consumption for contract calls

Response actions: alert, rate_limit, circuit_break, investigation

### Fee Optimizer

Predicts congestion using Exponential Moving Average (EMA) and estimates fees by urgency:
- **Fast**: Higher fee, estimated 1-2 blocks
- **Normal**: Standard fee, estimated 3-5 blocks
- **Slow**: Lower fee, estimated 6-10 blocks

### Network Optimizer

Monitors network state and recommends parameter adjustments using:
```
R(s,a,s') = alpha * DeltaPerformance + beta * DeltaLatency + gamma * DeltaEnergy - delta * StabilityPenalty
```

### AI Sidecar

The sidecar extends on-chain AI with AWS Bedrock:
- **Contract Generation**: Supports 17 blockchain platforms
- **Contract Auditing**: Hacken-style security audits
- **Deep Fraud Analysis**: Bedrock-powered threat assessment
- **Network Advice**: AI-driven optimization recommendations

Models: Claude Haiku 4.5 (fast path), Claude Sonnet 4.5 (balanced)

## API Endpoints

```
GET /qorechain/ai/v1/fee-estimate?urgency=fast|normal|slow
GET /qorechain/ai/v1/fraud/investigations
GET /qorechain/ai/v1/fraud/investigations/{id}
GET /qorechain/ai/v1/network/recommendations
GET /qorechain/ai/v1/circuit-breakers
```
