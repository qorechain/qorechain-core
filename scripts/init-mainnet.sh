#!/bin/bash
set -euo pipefail

# QoreChain Mainnet (qorechain-vladi) genesis initialization.
#
# This is the genesis-coordinator step: it builds the genesis file with the
# canonical QOR tokenomics (4.5B total supply) and production consensus/economic
# parameters. Each of the 6 validators runs `gentx` separately and submits their
# gentx; the coordinator collects them (see VALIDATOR GENTX section).
#
# Unlike the testnet, mainnet has NO faucet and NO unauthenticated state-mutating
# RPC (QORE_SVM_RPC_ALLOW_WRITES is left unset).
#
# Allocation wallet addresses MUST be provided via environment variables — they
# are foundation-controlled (ideally multisig). The script refuses to run with
# placeholder addresses.

CHAIN_ID="${CHAIN_ID:-qorechain-vladi}"
MONIKER="${MONIKER:-qorechain-mainnet-coordinator}"
HOME_DIR="${HOME_DIR:-/home/qorechaind/.qorechaind-mainnet}"
DENOM="uqor"
KEYRING="${KEYRING:-file}"            # mainnet: NEVER "test"
export QORE_EVM_CHAIN_ID="${QORE_EVM_CHAIN_ID:-9801}"  # mainnet EIP-155 id

# 1 QOR = 1_000_000 uqor (6 decimals).
qor() { echo "${1}000000"; }

# --- Canonical tokenomics: total supply 4,500,000,000 QOR -------------------
# Each bucket is a foundation-controlled address supplied via env. Vesting
# (cliffs/linear release per the tokenomics) is applied per bucket where noted;
# the coordinator attaches vesting accounts before collect-gentxs.
declare -A ALLOC=(
  [ECOSYSTEM]=1545000000      # Ecosystem & Protocol      34.33%
  [COMMUNITY]=735000000       # Community Distribution     16.33%
  [TEAM]=720000000            # Team & Advisors            16%   (vested)
  [TREASURY]=560000000        # Treasury & Operations      12.44%
  [INVESTORS]=410000000       # Investors                   9.11% (vested)
  [MARKETING]=270000000       # Marketing & Partnerships    6%
  [PROGRAMS]=135000000        # Community Programs          3%
  [RESERVES]=125000000        # Reserves & Burns            2.78% (incl. 80M genesis burn)
)
# Sum check: 1545+735+720+560+410+270+135+125 = 4500 (M QOR).

GENESIS_BURN_QOR=80000000     # 80M QOR burned at genesis (from RESERVES bucket)

echo "=== QoreChain MAINNET Initialization ($CHAIN_ID) ==="
echo "EVM chain ID: $QORE_EVM_CHAIN_ID | Keyring: $KEYRING | Home: $HOME_DIR"

# --- Validate allocation addresses are real (no placeholders) ----------------
require_addr() {
  local name="$1" addr="${2:-}"
  if [ -z "$addr" ] || [[ "$addr" != qor1* ]]; then
    echo "ERROR: address for $name is missing or not a qor1... address." >&2
    echo "       Set ${name}_ADDR to a foundation-controlled (multisig) address." >&2
    exit 1
  fi
}
require_addr ECOSYSTEM "${ECOSYSTEM_ADDR:-}"
require_addr COMMUNITY "${COMMUNITY_ADDR:-}"
require_addr TEAM "${TEAM_ADDR:-}"
require_addr TREASURY "${TREASURY_ADDR:-}"
require_addr INVESTORS "${INVESTORS_ADDR:-}"
require_addr MARKETING "${MARKETING_ADDR:-}"
require_addr PROGRAMS "${PROGRAMS_ADDR:-}"
require_addr RESERVES "${RESERVES_ADDR:-}"

# Step 1: init the node.
qorechaind init "$MONIKER" --chain-id "$CHAIN_ID" --home "$HOME_DIR" 2>&1 | grep -v "already exists" || true

# Step 2: fund the allocation buckets in genesis.
add_alloc() {
  local name="$1" addr="$2" amount_qor="$3"
  qorechaind genesis add-genesis-account "$addr" "$(qor "$amount_qor")${DENOM}" --home "$HOME_DIR"
  echo "  $name: $(printf "%'d" "$amount_qor") QOR -> $addr"
}
add_alloc ECOSYSTEM "$ECOSYSTEM_ADDR" "${ALLOC[ECOSYSTEM]}"
add_alloc COMMUNITY "$COMMUNITY_ADDR" "${ALLOC[COMMUNITY]}"
add_alloc TREASURY  "$TREASURY_ADDR"  "${ALLOC[TREASURY]}"
add_alloc MARKETING "$MARKETING_ADDR" "${ALLOC[MARKETING]}"
add_alloc PROGRAMS  "$PROGRAMS_ADDR"  "${ALLOC[PROGRAMS]}"
add_alloc RESERVES  "$RESERVES_ADDR"  "${ALLOC[RESERVES]}"

# Vested buckets: Team (16%) and Investors (9.11%) use continuous vesting.
# VESTING_END_UNIX is the cliff/end timestamp; foundation sets per the schedule.
VEST_END="${VESTING_END_UNIX:-}"
if [ -n "$VEST_END" ]; then
  qorechaind genesis add-genesis-account "$TEAM_ADDR" "$(qor "${ALLOC[TEAM]}")${DENOM}" \
    --vesting-amount "$(qor "${ALLOC[TEAM]}")${DENOM}" --vesting-end-time "$VEST_END" --home "$HOME_DIR"
  qorechaind genesis add-genesis-account "$INVESTORS_ADDR" "$(qor "${ALLOC[INVESTORS]}")${DENOM}" \
    --vesting-amount "$(qor "${ALLOC[INVESTORS]}")${DENOM}" --vesting-end-time "$VEST_END" --home "$HOME_DIR"
  echo "  TEAM + INVESTORS funded as vesting accounts (end=$VEST_END)"
else
  echo "WARNING: VESTING_END_UNIX unset — TEAM/INVESTORS funded as liquid. Set it for production." >&2
  add_alloc TEAM      "$TEAM_ADDR"      "${ALLOC[TEAM]}"
  add_alloc INVESTORS "$INVESTORS_ADDR" "${ALLOC[INVESTORS]}"
fi

# Step 3: VALIDATOR GENTX (run on each of the 6 validators, then collect here).
# Each validator: qorechaind genesis gentx <key> <self-bond>uqor --chain-id qorechain-vladi ...
# with --min-self-delegation 100000000000 (100k QOR). Place all gentx json files
# into $HOME_DIR/config/gentx/ before running collect-gentxs.
if [ -d "$HOME_DIR/config/gentx" ] && [ -n "$(ls -A "$HOME_DIR/config/gentx" 2>/dev/null)" ]; then
  qorechaind genesis collect-gentxs --home "$HOME_DIR"
else
  echo "NOTE: no gentx files in $HOME_DIR/config/gentx — collect after validators submit." >&2
fi

# Step 4: production genesis parameters.
GENESIS="$HOME_DIR/config/genesis.json"
jq '
  .app_state.staking.params.bond_denom = "uqor" |
  .app_state.staking.params.unbonding_time = "1814400s" |               # 21 days
  .app_state.staking.params.min_commission_rate = "0.050000000000000000" |
  .app_state.staking.params.max_validators = 100 |
  .app_state.staking.params.min_self_delegation = "100000000000" |       # 100k QOR
  .app_state.mint.minter.inflation = "0.080000000000000000" |            # 8%
  .app_state.mint.params.inflation_max = "0.120000000000000000" |
  .app_state.mint.params.inflation_min = "0.050000000000000000" |
  .app_state.gov.params.min_deposit[0].denom = "uqor" |
  .app_state.gov.params.min_deposit[0].amount = "50000000000" |          # 50k QOR
  .app_state.gov.params.quorum = "0.334000000000000000" |
  .app_state.gov.params.threshold = "0.500000000000000000" |
  .app_state.gov.params.voting_period = "432000s" |                      # 5 days
  .app_state.slashing.params.signed_blocks_window = "10000" |
  .app_state.slashing.params.min_signed_per_window = "0.050000000000000000" |
  .app_state.slashing.params.slash_fraction_double_sign = "0.050000000000000000" |
  .app_state.slashing.params.slash_fraction_downtime = "0.000100000000000000" |
  .app_state.bank.denom_metadata = [
    {
      "description": "The native staking token of QoreChain",
      "denom_units": [
        {"denom": "uqor", "exponent": 0, "aliases": ["microqor"]},
        {"denom": "mqor", "exponent": 3, "aliases": ["milliqor"]},
        {"denom": "qor",  "exponent": 6, "aliases": ["QOR"]}
      ],
      "base": "uqor", "display": "qor", "name": "QOR", "symbol": "QOR"
    }
  ] |
  .consensus.params.block.max_bytes = "4194304" |
  .consensus.params.block.max_gas = "100000000"
' "$GENESIS" > "$GENESIS.tmp" && mv "$GENESIS.tmp" "$GENESIS"

# Step 5: production node config — faster commit, prometheus on.
CONFIG="$HOME_DIR/config/config.toml"
sed -i 's/timeout_commit = .*/timeout_commit = "3s"/' "$CONFIG" 2>/dev/null || \
    sed -i '' 's/timeout_commit = .*/timeout_commit = "3s"/' "$CONFIG"
sed -i 's/prometheus = false/prometheus = true/' "$CONFIG" 2>/dev/null || \
    sed -i '' 's/prometheus = false/prometheus = true/' "$CONFIG"

# Step 6: validate.
qorechaind genesis validate --home "$HOME_DIR"

echo ""
echo "=== QoreChain MAINNET genesis built ==="
echo "Total supply: 4,500,000,000 QOR | Genesis burn target: ${GENESIS_BURN_QOR} QOR (from RESERVES)"
echo "EVM chain ID: $QORE_EVM_CHAIN_ID | unbonding 21d | voting 5d | inflation 8%"
echo "Genesis: $GENESIS"
echo ""
echo "Next: distribute genesis.json to all 6 validators; start with"
echo "  qorechaind start --home $HOME_DIR --chain-id $CHAIN_ID"
