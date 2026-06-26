#!/bin/bash
set -euo pipefail

# QoreChain Testnet Initialization Script
# Creates genesis, accounts, and validator configuration

CHAIN_ID="${CHAIN_ID:-qorechain-diana}"
MONIKER="${MONIKER:-qorechain-validator-1}"
HOME_DIR="${HOME_DIR:-/home/qorechaind/.qorechaind}"
DENOM="uqor"
KEYRING="test"
#ni13
echo "=== QoreChain Testnet Initialization ==="
echo "Chain ID:  $CHAIN_ID"
echo "Moniker:   $MONIKER"
echo "Home:      $HOME_DIR"

# Step 1: Initialize the node
qorechaind init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR" 2>&1 | grep -v "already exists" || true

# Step 2: Create validator key
qorechaind keys add validator --keyring-backend "$KEYRING" --home "$HOME_DIR" 2>&1 | grep -v "already exists" || true
VALIDATOR_ADDR=$(qorechaind keys show validator -a --keyring-backend "$KEYRING" --home "$HOME_DIR")
echo "Validator address: $VALIDATOR_ADDR"

# Step 3: Create faucet key
qorechaind keys add faucet --keyring-backend "$KEYRING" --home "$HOME_DIR" 2>&1 | grep -v "already exists" || true
FAUCET_ADDR=$(qorechaind keys show faucet -a --keyring-backend "$KEYRING" --home "$HOME_DIR")
echo "Faucet address:    $FAUCET_ADDR"

# Step 3b: License authority + bridge admin.
# Both default to the validator (genesis account 0) so a fresh deploy has
# licensing + bridge administration enabled with NO hand-edit and NO governance.
# Override by exporting LICENSE_AUTHORITY / BRIDGE_ADMIN (bech32 qor1... addrs).
LICENSE_AUTHORITY="${LICENSE_AUTHORITY:-$VALIDATOR_ADDR}"
BRIDGE_ADMIN="${BRIDGE_ADMIN:-$VALIDATOR_ADDR}"
echo "License authority: $LICENSE_AUTHORITY"
echo "Bridge admin:      $BRIDGE_ADMIN"

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
    --min-self-delegation "100000000000" \
    --keyring-backend "$KEYRING" \
    --home "$HOME_DIR"

# Step 6: Collect gentxs
qorechaind genesis collect-gentxs --home "$HOME_DIR"

# Step 7: Customize genesis parameters
GENESIS="$HOME_DIR/config/genesis.json"

jq '
  .app_state.staking.params.bond_denom = "uqor" |
  .app_state.staking.params.unbonding_time = "600s" |
  .app_state.staking.params.min_commission_rate = "0.050000000000000000" |
  .app_state.staking.params.max_validators = 150 |
  .app_state.mint.minter.inflation = "0.130000000000000000" |
  .app_state.gov.params.min_deposit[0].denom = "uqor" |
  .app_state.gov.params.min_deposit[0].amount = "10000000000" |
  .app_state.gov.params.expedited_min_deposit[0].denom = "uqor" |
  .app_state.gov.params.expedited_min_deposit[0].amount = "20000000000" |
  .app_state.gov.params.quorum = "0.100000000000000000" |
  .app_state.gov.params.voting_period = "600s" |
  .app_state.gov.params.expedited_voting_period = "300s" |
  .app_state.bank.denom_metadata = [
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
  ] |
  .consensus.params.block.max_bytes = "4194304" |
  .consensus.params.block.max_gas = "100000000"
' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"

# Step 7b: Bake license authority, bridge admin, and a seed qcb_bridge grant so
# licensing + bridge administration work immediately post-deploy with no
# hand-edit and no governance proposal. Idempotent: the grant is rebuilt by
# grantee/feature_id key each run rather than appended.
NOW_TS=$(date +%s)
jq \
  --arg lic_auth "$LICENSE_AUTHORITY" \
  --arg br_admin "$BRIDGE_ADMIN" \
  --argjson now "$NOW_TS" '
  # 1) license module: store-backed grant authority.
  .app_state.license.authority = $lic_auth |
  # 2) bridge module: admin authorized to activate chains without governance.
  .app_state.bridge.config.bridge_admin = $br_admin |
  # 3) seed an umbrella qcb_bridge grant to the bridge admin (idempotent:
  #    drop any existing qcb_bridge grant for this grantee, then add one).
  .app_state.license.licenses = (
    ((.app_state.license.licenses // [])
      | map(select(.grantee != $br_admin or .feature_id != "qcb_bridge")))
    + [{
        "grantee":    $br_admin,
        "feature_id": "qcb_bridge",
        "expires_at": 0,
        "granted_at": $now,
        "granted_by": $lic_auth,
        "suspended":  false,
        "metadata":   "genesis-seeded bridge admin umbrella grant"
      }]
  )
' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"

# Step 8: Configure Consensus Engine Engine
CONFIG="$HOME_DIR/config/config.toml"
# Anchor to start-of-line so this does NOT also rewrite `skip_timeout_commit`
# (which ends in "timeout_commit") into a non-bool value and break config parsing.
sed -i 's/^timeout_commit = .*/timeout_commit = "5s"/' "$CONFIG" 2>/dev/null || \
    sed -i '' 's/^timeout_commit = .*/timeout_commit = "5s"/' "$CONFIG"

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
