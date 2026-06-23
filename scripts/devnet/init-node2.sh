#!/usr/bin/env bash
# Init script for devnet node-2 (the joining peer validator).
# Mounted by docker-compose.devnet.yml as /scripts/init-node.sh.
#
# Responsibilities:
#   1. Initialise the home dir if it doesn't exist
#   2. Pull node-1's genesis.json over the network
#   3. Configure node-1 as the persistent peer
#   4. Start the node — once it catches up to height >0 and can produce
#      a validator-tx, the operator can self-delegate from a separate
#      account to add it as a validator (devnet flow)

set -euo pipefail

HOME_DIR="${HOME_DIR:-/home/qorechaind/.qorechaind}"
CHAIN_ID="${CHAIN_ID:-qorechain-devnet}"
MONIKER="${MONIKER:-devnet-node-2}"
SEED_RPC="${SEED_RPC:-http://node-1:26657}"

if [ ! -f "$HOME_DIR/config/genesis.json" ]; then
  echo "[init-node2] First boot — initialising home dir at $HOME_DIR"
  qorechaind init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR"

  # Wait for node-1 to be reachable
  echo "[init-node2] Waiting for $SEED_RPC to be reachable..."
  for i in $(seq 1 60); do
    if curl -fsS "$SEED_RPC/status" >/dev/null 2>&1; then
      break
    fi
    sleep 2
  done

  # Fetch genesis from node-1. The /genesis endpoint is capped (~20MB); assemble
  # it from genesis_chunked so this works regardless of genesis size. jq (not
  # python3) is the JSON tool present in the runtime image.
  echo "[init-node2] Fetching genesis from $SEED_RPC"
  TOTAL=$(curl -fsS "$SEED_RPC/genesis_chunked?chunk=0" | jq -r '.result.total')
  : > "$HOME_DIR/config/genesis.b64"
  for c in $(seq 0 $((TOTAL - 1))); do
    curl -fsS "$SEED_RPC/genesis_chunked?chunk=$c" | jq -r '.result.data' >> "$HOME_DIR/config/genesis.b64"
  done
  base64 -d "$HOME_DIR/config/genesis.b64" > "$HOME_DIR/config/genesis.json"
  rm -f "$HOME_DIR/config/genesis.b64"

  # Discover node-1's node ID for persistent_peers
  NODE_1_ID=$(curl -fsS "$SEED_RPC/status" | jq -r '.result.node_info.id')

  PEERS="${NODE_1_ID}@node-1:26656"
  sed -i "s|persistent_peers = \"\"|persistent_peers = \"$PEERS\"|" \
    "$HOME_DIR/config/config.toml"

  # Match node-1's faster block times
  sed -i 's/timeout_commit = "5s"/timeout_commit = "1s"/' "$HOME_DIR/config/config.toml"
  sed -i 's/timeout_propose = "3s"/timeout_propose = "1s"/' "$HOME_DIR/config/config.toml"

  # Allow inbound peers
  sed -i 's/laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/' \
    "$HOME_DIR/config/config.toml"

  # Expose the REST API on all interfaces (the API address has no start flag).
  sed -i 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' \
    "$HOME_DIR/config/app.toml"
fi

echo "[init-node2] Starting qorechaind"
exec qorechaind start --home "$HOME_DIR" \
  --api.enable \
  --grpc.enable --grpc.address "0.0.0.0:9090" \
  --json-rpc.enable --json-rpc.address "0.0.0.0:8545" --json-rpc.ws-address "0.0.0.0:8546"
