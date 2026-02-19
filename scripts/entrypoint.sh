#!/bin/bash
set -euo pipefail

CHAIN_ID="${CHAIN_ID:-qorechain-testnet-1}"
MONIKER="${MONIKER:-qorechain-validator-1}"
HOME_DIR="/root/.qorechaind"

# Initialize if not already initialized
if [ ! -f "$HOME_DIR/config/genesis.json" ]; then
    echo "Initializing QoreChain node..."
    /scripts/init-testnet.sh
fi

echo "Starting QoreChain node..."
exec qorechaind start \
    --home "$HOME_DIR" \
    --rpc.laddr "tcp://0.0.0.0:26657" \
    --grpc.address "0.0.0.0:9090" \
    --api.enable true \
    --api.address "tcp://0.0.0.0:1317" \
    --api.swagger true \
    --p2p.laddr "tcp://0.0.0.0:26656" \
    --minimum-gas-prices "0uqor"
