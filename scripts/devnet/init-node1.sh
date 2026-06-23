#!/usr/bin/env bash
# Init script for devnet node-1 (the genesis-creating validator).
# Mounted by docker-compose.devnet.yml as /scripts/init-node.sh.
#
# Responsibilities:
#   1. Initialise the home dir if it doesn't exist
#   2. Create a single funded validator account
#   3. Generate genesis with this validator pre-staked
#   4. Start the node with peer discovery enabled

set -euo pipefail

HOME_DIR="${HOME_DIR:-/home/qorechaind/.qorechaind}"
CHAIN_ID="${CHAIN_ID:-qorechain-devnet}"
MONIKER="${MONIKER:-devnet-node-1}"
KEYRING="test"

if [ ! -f "$HOME_DIR/config/genesis.json" ]; then
  echo "[init-node1] First boot — initialising home dir at $HOME_DIR"
  qorechaind init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR"

  # Create the validator key
  qorechaind keys add validator --keyring-backend "$KEYRING" --home "$HOME_DIR"

  # Fund the validator with 100,000,000 QOR (uqor)
  ADDR=$(qorechaind keys show validator -a --keyring-backend "$KEYRING" --home "$HOME_DIR")
  qorechaind genesis add-genesis-account "$ADDR" 100000000000000uqor --home "$HOME_DIR"

  # Self-delegate 50,000,000 QOR to bootstrap consensus
  qorechaind genesis gentx validator 50000000000000uqor \
    --chain-id "$CHAIN_ID" \
    --keyring-backend "$KEYRING" \
    --home "$HOME_DIR"

  qorechaind genesis collect-gentxs --home "$HOME_DIR"

  # Lower block times for the devnet (1s vs 5s default)
  sed -i 's/timeout_commit = "5s"/timeout_commit = "1s"/' "$HOME_DIR/config/config.toml"
  sed -i 's/timeout_propose = "3s"/timeout_propose = "1s"/' "$HOME_DIR/config/config.toml"

  # Allow inbound peers from node-2
  sed -i 's/laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/' \
    "$HOME_DIR/config/config.toml"

  # Expose the REST API on all interfaces (the API address has no start flag).
  sed -i 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' \
    "$HOME_DIR/config/app.toml"
fi

echo "[init-node1] Starting qorechaind"
exec qorechaind start --home "$HOME_DIR" \
  --api.enable \
  --grpc.enable --grpc.address "0.0.0.0:9090" \
  --json-rpc.enable --json-rpc.address "0.0.0.0:8545" --json-rpc.ws-address "0.0.0.0:8546"
