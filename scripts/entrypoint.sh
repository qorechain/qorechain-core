#!/bin/bash
set -euo pipefail

export CHAIN_ID="${CHAIN_ID:-qorechain-diana}"
export MONIKER="${MONIKER:-qorechain-validator-1}"
export HOME_DIR="${HOME_DIR:-/home/qorechaind/.qorechaind}"

# Initialize if not already initialized
if [ ! -f "$HOME_DIR/config/genesis.json" ]; then
    echo "Initializing QoreChain node..."
    /scripts/init-testnet.sh || { echo "ERROR: init-testnet.sh failed"; exit 1; }
fi

echo "Starting QoreChain node..."
exec qorechaind start \
    --home "$HOME_DIR" \
    --rpc.laddr "tcp://0.0.0.0:26657" \
    --grpc.address "0.0.0.0:9090" \
    --api.enable true \
    --api.address "tcp://0.0.0.0:1317" \
    --api.swagger ${ENABLE_SWAGGER:-false} \
    --p2p.laddr "tcp://0.0.0.0:26656" \
    --minimum-gas-prices "${MIN_GAS_PRICE:-0.001uqor}"
