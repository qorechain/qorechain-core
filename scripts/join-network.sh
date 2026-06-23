#!/bin/bash
set -euo pipefail

# Join an existing QoreChain network as a full node.
#
# For exchanges and integrators who only need to follow QoreChain: sync the
# chain, query balances/blocks, and submit transactions. Unlike init-testnet.sh
# (which bootstraps a fresh single-node chain), this script joins the live
# network using its published genesis and peers. It never creates a new chain.
#
# No validator key, AI sidecar, bridge relayer, or external-network component is
# required — this is a plain full node.

CHAIN_ID="${CHAIN_ID:-qorechain-diana}"
MONIKER="${MONIKER:-qorechain-node}"
HOME_DIR="${HOME_DIR:-/home/qorechaind/.qorechaind}"
CONFIG_DIR="$HOME_DIR/config"
GENESIS_FILE="$CONFIG_DIR/genesis.json"

# 1. One-time initialization. On a fresh volume there is no node key yet; create
#    the home directory and config. On restarts this is skipped so existing keys
#    and state are preserved.
FIRST_RUN=false
if [ ! -f "$CONFIG_DIR/node_key.json" ]; then
    FIRST_RUN=true
    echo "Initializing node home for chain '$CHAIN_ID' (moniker: $MONIKER)..."
    qorechaind init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR"
fi

# 2. Install the network genesis. `init` writes a throwaway local genesis, so on
#    the first run we replace it with the real network genesis. Joining requires
#    the network's own genesis — we never start from the local one.
if [ "$FIRST_RUN" = "true" ]; then
    if [ -n "${GENESIS_URL:-}" ]; then
        echo "Downloading network genesis from $GENESIS_URL"
        curl -fsSL "$GENESIS_URL" -o "$GENESIS_FILE"
    elif [ -f "/genesis/genesis.json" ]; then
        echo "Using mounted network genesis at /genesis/genesis.json"
        cp /genesis/genesis.json "$GENESIS_FILE"
    else
        echo "ERROR: no network genesis available."
        echo "Set GENESIS_URL=<url>, or mount the network genesis at /genesis/genesis.json."
        exit 1
    fi
    qorechaind genesis validate --home "$HOME_DIR" || {
        echo "ERROR: the provided genesis failed validation."; exit 1;
    }
fi

# 3. Peers. Applied on every start so operators can update them via env without
#    rebuilding. Format: "<node_id>@<host>:<port>,<node_id>@<host>:<port>".
if [ -n "${SEEDS:-}" ]; then
    sed -i "s|^seeds = .*|seeds = \"$SEEDS\"|" "$CONFIG_DIR/config.toml"
fi
if [ -n "${PERSISTENT_PEERS:-}" ]; then
    sed -i "s|^persistent_peers = .*|persistent_peers = \"$PERSISTENT_PEERS\"|" "$CONFIG_DIR/config.toml"
fi

if [ -z "${SEEDS:-}" ] && [ -z "${PERSISTENT_PEERS:-}" ]; then
    echo "WARNING: no SEEDS or PERSISTENT_PEERS set — the node has no one to dial."
    echo "         Set SEEDS and/or PERSISTENT_PEERS so the node can discover the network."
fi

echo "Starting QoreChain full node ($CHAIN_ID)..."
exec qorechaind start \
    --home "$HOME_DIR" \
    --rpc.laddr "tcp://0.0.0.0:26657" \
    --grpc.address "0.0.0.0:9090" \
    --api.enable true \
    --api.address "tcp://0.0.0.0:1317" \
    --api.swagger "${ENABLE_SWAGGER:-false}" \
    --p2p.laddr "tcp://0.0.0.0:26656" \
    --minimum-gas-prices "${MIN_GAS_PRICE:-0.001uqor}"
