package app

import (
	"context"
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
)

// upgradeV3_2_3StakingVesting runs the one-time state steps of the v3.2.3
// staking and vesting area at the plan height:
//   - module accounts that never mint or burn lose those permissions
//     (QSR-2026-0532);
//   - the IBC client allow list drops the "*" wildcard, which admitted the
//     09-localhost loop (QSR-2026-0346 backstop).
func (app *QoreChainApp) upgradeV3_2_3StakingVesting(ctx sdk.Context) error {
	if err := restrictModuleAccountPermissions(ctx, app.AccountKeeper); err != nil {
		return err
	}
	if app.IBCKeeper != nil && app.IBCKeeper.ClientKeeper != nil {
		pinIBCAllowedClients(ctx, app.IBCKeeper.ClientKeeper)
	}
	return nil
}

// ---- QSR-2026-0532: module account least privilege --------------------------

// leastPrivilegeModulePerms are the permissions these module accounts keep from
// v3.2.3. None of the three ever calls MintCoins (amm and xqore only burn: the
// pool creation fee and the xQORE exit penalty), and rdk neither mints nor
// burns.
//
// x/auth keeps the permissions an account was created with, and bank checks
// the stored account, so this is applied to the stored accounts. The permission
// table in app_config.go still carries the old grants on purpose: it is what a
// LAZILY CREATED module account gets, and on mainnet none of the three exists
// yet. Should one be created below the plan height (an AMM pool, say), a node
// replaying that block on this binary must create it exactly as the previous
// binary did. The step below creates any missing account, so from the plan
// height the table is never consulted for these three again; drop the entries
// from the table in a release whose replay range starts after this height.
var leastPrivilegeModulePerms = []struct {
	name  string
	perms []string
}{
	{"amm", []string{authtypes.Burner}},
	{"xqore", []string{authtypes.Burner}},
	{"rdk", nil},
}

type leastPrivilegeAccountStore interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	SetModuleAccount(ctx context.Context, macc sdk.ModuleAccountI)
}

// restrictModuleAccountPermissions creates each listed module account if it is
// missing and sets its stored permissions to the reduced set. An address that
// holds some other kind of account is logged and left alone: GetModuleAccount
// would panic on it and halt the chain at the plan height.
func restrictModuleAccountPermissions(ctx sdk.Context, ak leastPrivilegeAccountStore) error {
	for _, e := range leastPrivilegeModulePerms {
		addr := authtypes.NewModuleAddress(e.name)
		if acc := ak.GetAccount(ctx, addr); acc != nil {
			if _, ok := acc.(sdk.ModuleAccountI); !ok {
				ctx.Logger().Error("v3.2.3 module account least privilege: address holds a non-module account, left unchanged",
					"module", e.name, "address", addr.String())
				continue
			}
		}
		macc, ok := ak.GetModuleAccount(ctx, e.name).(*authtypes.ModuleAccount)
		if !ok || macc == nil {
			ctx.Logger().Error("v3.2.3 module account least privilege: unexpected account type, left unchanged", "module", e.name)
			continue
		}
		if slices.Equal(macc.Permissions, e.perms) || (len(macc.Permissions) == 0 && len(e.perms) == 0) {
			continue
		}
		old := macc.Permissions
		updated := *macc
		updated.Permissions = slices.Clone(e.perms)
		ak.SetModuleAccount(ctx, &updated)
		ctx.Logger().Info("v3.2.3 module account least privilege", "module", e.name,
			"old_permissions", old, "new_permissions", updated.Permissions)
	}
	return nil
}

// ---- QSR-2026-0346: no 09-localhost -----------------------------------------

type ibcClientParamStore interface {
	GetParams(ctx sdk.Context) clienttypes.Params
	SetParams(ctx sdk.Context, params clienttypes.Params)
}

// pinIBCAllowedClients replaces the ibc-go default allow list ["*"] with
// ["07-tendermint"]. With the wildcard, the 09-localhost client is allowed and
// anyone can open channels over the pre-opened localhost connection and create
// token pairs. Governance is asked to set the same value before the upgrade; a
// list governance has already narrowed or chosen is left as it is.
func pinIBCAllowedClients(ctx sdk.Context, ck ibcClientParamStore) {
	p := ck.GetParams(ctx)
	if len(p.AllowedClients) != 1 || p.AllowedClients[0] != clienttypes.AllowAllClients {
		return
	}
	p.AllowedClients = []string{ibctm.ModuleName}
	ck.SetParams(ctx, p)
	ctx.Logger().Info("v3.2.3 IBC allowed clients pinned", "allowed_clients", p.AllowedClients)
}
