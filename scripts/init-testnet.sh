#!/bin/bash
set -euo pipefail

# QoreChain Testnet Initialization Script
# Creates genesis, accounts, and validator configuration

CHAIN_ID="${CHAIN_ID:-qorechain-testnet-1}"
MONIKER="${MONIKER:-qorechain-validator-1}"
HOME_DIR="${HOME_DIR:-/root/.qorechaind}"
DENOM="uqor"
KEYRING="test"

echo "=== QoreChain Testnet Initialization ==="
echo "Chain ID:  $CHAIN_ID"
echo "Moniker:   $MONIKER"
echo "Home:      $HOME_DIR"

# Step 1: Initialize the node
qorechaind init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR" 2>/dev/null

# Step 2: Create validator key
qorechaind keys add validator --keyring-backend "$KEYRING" --home "$HOME_DIR" 2>/dev/null
VALIDATOR_ADDR=$(qorechaind keys show validator -a --keyring-backend "$KEYRING" --home "$HOME_DIR")
echo "Validator address: $VALIDATOR_ADDR"

# Step 3: Create faucet key
qorechaind keys add faucet --keyring-backend "$KEYRING" --home "$HOME_DIR" 2>/dev/null
FAUCET_ADDR=$(qorechaind keys show faucet -a --keyring-backend "$KEYRING" --home "$HOME_DIR")
echo "Faucet address:    $FAUCET_ADDR"

# Step 4: Fund accounts in genesis
# Validator: 100M QOR = 100_000_000_000_000 uqor
qorechaind genesis add-genesis-account "$VALIDATOR_ADDR" "100000000000000${DENOM}" --keyring-backend "$KEYRING" --home "$HOME_DIR"

# Faucet: 1B QOR = 1_000_000_000_000_000 uqor (for testnet distribution)
qorechaind genesis add-genesis-account "$FAUCET_ADDR" "1000000000000000${DENOM}" --keyring-backend "$KEYRING" --home "$HOME_DIR"

# Step 5: Create gentx for validator
qorechaind genesis gentx validator "10000000000000${DENOM}" \
    --chain-id "$CHAIN_ID" \
    --moniker "$MONIKER" \
    --commission-rate "0.10" \
    --commission-max-rate "0.20" \
    --commission-max-change-rate "0.01" \
    --min-self-delegation "1" \
    --keyring-backend "$KEYRING" \
    --home "$HOME_DIR"

# Step 6: Collect gentxs
qorechaind genesis collect-gentxs --home "$HOME_DIR"

# Step 7: Customize genesis parameters
GENESIS="$HOME_DIR/config/genesis.json"

# Set bond denom
jq '.app_state.staking.params.bond_denom = "uqor"' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"
jq '.app_state.staking.params.unbonding_time = "600s"' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"
jq '.app_state.staking.params.min_commission_rate = "0.050000000000000000"' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"

# Set mint denom
jq '.app_state.mint.minter.inflation = "0.130000000000000000"' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"

# Set gov deposit denom and voting period
jq '.app_state.gov.params.min_deposit[0].denom = "uqor"' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"
jq '.app_state.gov.params.min_deposit[0].amount = "10000000"' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"
jq '.app_state.gov.params.voting_period = "600s"' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"

# Set crisis fee denom
jq '.app_state.bank.denom_metadata = [
  {
    "description": "The native staking token of QoreChain",
    "denom_units": [
      {"denom": "uqor", "exponent": 0, "aliases": ["microqor"]},
      {"denom": "mqor", "exponent": 3, "aliases": ["milliqor"]},
      {"denom": "qor",  "exponent": 6, "aliases": ["QOR"]}
    ],
    "base": "uqor",
    "display": "qor",
    "name": "QOR",
    "symbol": "QOR"
  }
]' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"

# Increase block size for PQC signatures (~4.6KB each)
jq '.consensus.params.block.max_bytes = "4194304"' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"
jq '.consensus.params.block.max_gas = "100000000"' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"

# Step 8: Configure CometBFT
CONFIG="$HOME_DIR/config/config.toml"
sed -i 's/timeout_commit = .*/timeout_commit = "5s"/' "$CONFIG" 2>/dev/null || \
    sed -i '' 's/timeout_commit = .*/timeout_commit = "5s"/' "$CONFIG"

# Enable Prometheus metrics
sed -i 's/prometheus = false/prometheus = true/' "$CONFIG" 2>/dev/null || \
    sed -i '' 's/prometheus = false/prometheus = true/' "$CONFIG"

# Step 9: Validate genesis
qorechaind genesis validate --home "$HOME_DIR"

echo ""
echo "=== QoreChain Testnet Initialized ==="
echo "Chain ID:  $CHAIN_ID"
echo "Validator: $VALIDATOR_ADDR"
echo "Faucet:    $FAUCET_ADDR"
echo "Genesis:   $GENESIS"
echo ""
echo "Start with: qorechaind start --home $HOME_DIR"
