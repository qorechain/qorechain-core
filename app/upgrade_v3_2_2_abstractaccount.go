package app

import sdk "github.com/cosmos/cosmos-sdk/types"

// upgradeV3_2_2AbstractAccount is this area's one-time state work at the v3.2.2 plan
// height. The abstract-account spending policy (FeatureAASpendingPolicy) needs
// none:
//   - SpendingRule gains allowed_evm_calls and denom_limits as optional fields,
//     and stored accounts are JSON with those fields omitted when empty, so every
//     existing record already has its v3.2.2 form;
//   - per-denom spend buckets live under a new prefix that starts empty;
//   - the module's enabled flag is left as it is. Enabling abstract accounts is a
//     separate decision, taken after this release is live.
func (app *QoreChainApp) upgradeV3_2_2AbstractAccount(_ sdk.Context) error { return nil }
