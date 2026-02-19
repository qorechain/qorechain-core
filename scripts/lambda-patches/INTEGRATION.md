# Lambda → Testnet Integration Guide

## Overview

These patches add a `USE_TESTNET` feature flag to the existing Lambda functions,
enabling them to query real blockchain data from the QoreChain testnet node.

## Environment Variables

Add to each modified Lambda:

```
TESTNET_REST_URL=http://<ec2-ip>:1317
TESTNET_RPC_URL=http://<ec2-ip>:26657
USE_TESTNET=true
```

## Integration Steps

### 1. Copy `testnet-client.mjs` into each Lambda

```bash
for lambda in qorechain-explorer-api qorechain-wallet-api qorechain-faucet-api qorechain-dashboard-stats; do
  cp testnet-client.mjs ../../current/lambda/$lambda/
done
```

### 2. Apply patches

Each `*-patch.mjs` file contains replacement functions. The pattern is:

1. Import testnet client at top of `index.mjs`
2. Rename original function to `*_original`
3. Add patched function that tries testnet first, falls back to original

### 3. Lambda Environment Variables (Cloud CLI)

```bash
EC2_IP="<your-ec2-public-ip>"

for fn in qorechain-explorer-api qorechain-wallet-api qorechain-faucet-api qorechain-dashboard-stats; do
  qor-cli lambda update-function-configuration \
    --function-name $fn \
    --environment "Variables={USE_TESTNET=true,TESTNET_REST_URL=http://${EC2_IP}:1317,TESTNET_RPC_URL=http://${EC2_IP}:26657}" \
    --region us-east-1
done
```

## Architecture

```
Dashboard → API Gateway → Lambda Function
                              │
                              ├── if USE_TESTNET=true:
                              │     └── Testnet Node REST (real blockchain data)
                              │         http://<ec2-ip>:1317
                              │
                              └── else (or on error):
                                  └── DynamoDB / mock data (existing behavior)
```

## Functions Modified

| Lambda | Modification |
|--------|-------------|
| `qorechain-explorer-api` | Blocks, txs, addresses, stats from real node |
| `qorechain-wallet-api` | Real balance queries via `/cosmos/bank/v1beta1/balances/` |
| `qorechain-faucet-api` | Real token distribution (requires CosmJS) |
| `qorechain-dashboard-stats` | Real block height, validator count |
