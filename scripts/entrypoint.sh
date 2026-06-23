#!/bin/bash
set -euo pipefail
#ni11
export CHAIN_ID="${CHAIN_ID:-qorechain-diana}"
export MONIKER="${MONIKER:-qorechain-validator-1}"
export HOME_DIR="${HOME_DIR:-/home/qorechaind/.qorechaind}"

# Initialize if not already initialized
if [ ! -f "$HOME_DIR/config/genesis.json" ]; then
    echo "Initializing QoreChain node..."
    /scripts/init-testnet.sh || { echo "ERROR: init-testnet.sh failed"; exit 1; }
fi

# The REST API listen address has no start flag, so bind it to all interfaces in
# app.toml (default is localhost-only, unreachable from outside the container).
sed -i 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "$HOME_DIR/config/app.toml" 2>/dev/null || true

echo "Starting QoreChain node..."
exec qorechaind start \
    --home "$HOME_DIR" \
    --rpc.laddr "tcp://0.0.0.0:26657" \
    --p2p.laddr "tcp://0.0.0.0:26656" \
    --grpc.enable --grpc.address "0.0.0.0:9090" \
    --api.enable \
    --json-rpc.enable \
    --json-rpc.address "0.0.0.0:8545" \
    --json-rpc.ws-address "0.0.0.0:8546" \
    --minimum-gas-prices "${MIN_GAS_PRICE:-0.001uqor}"
