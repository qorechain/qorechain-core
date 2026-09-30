package app

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"

	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// upgradeV3_2_2EVMMisc is this area's one-time state work at the v3.2.2 plan
// height: every module account in the permission table is made to exist, as a
// module account, before FeatureModuleAccountGuard turns on (QSR-2026-0304).
func (app *QoreChainApp) upgradeV3_2_2EVMMisc(ctx sdk.Context) error {
	return materialiseModuleAccounts(ctx, app.AccountKeeper, app.EVMKeeper, moduleAccountNames(moduleAccPerms))
}

// Events emitted per module account the upgrade touches.
const (
	EventTypeModuleAccountMaterialised = "module_account_materialised"
	EventTypeModuleAccountRepaired     = "module_account_repaired"
)

// moduleAccountStore is the part of the auth keeper the upgrade step uses.
type moduleAccountStore interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	GetModuleAddressAndPermissions(moduleName string) (sdk.AccAddress, []string)
}

// evmCodeHashStore is the part of the EVM keeper the upgrade step uses.
type evmCodeHashStore interface {
	DeleteCodeHash(ctx sdk.Context, addr common.Address)
}

// moduleAccountNames lists the module accounts in permission-table order, the
// order the upgrade step visits them in (deterministic on every node).
func moduleAccountNames(perms []*authmodulev1.ModuleAccountPermission) []string {
	out := make([]string, 0, len(perms))
	for _, p := range perms {
		out = append(out, p.GetAccount())
	}
	return out
}

// materialiseModuleAccounts makes every module account in the table exist as
// a ModuleAccount.
//
// The SDK creates a module account lazily, the first time its module asks for
// it; until then nothing is stored at the module address. On mainnet and the
// testnet twelve of them had never been asked for (transfer, wasm, xqore,
// bridge, multilayer, babylon, abstractaccount, fairblock, gasabstraction,
// rdk, amm, nft). Anything that stored an ordinary account at one of those
// addresses first, which the EVM did for a zero-value touch, turned every
// later mint, burn or escrow of that module into a panic ("account is not a
// module account"). Creating them all here removes the precondition for every
// entry point at once; the EVM-side guard keeps it removed for modules added
// later.
//
// Three states per address:
//   - nothing stored: the module account is created (one account number each,
//     in table order);
//   - a module account: left alone;
//   - an ordinary account planted there: converted in place into the module
//     account, keeping its account number. Bank balances are keyed by
//     address, so any coins stay where they are, now held by the module. An
//     EVM code hash at the address is removed. A planted account cannot carry
//     a public key (no key hashes to a module address); if one does, the
//     upgrade stops rather than guess.
//
// Idempotent: a second run finds only module accounts and changes nothing.
func materialiseModuleAccounts(ctx sdk.Context, ak moduleAccountStore, evm evmCodeHashStore, names []string) error {
	for _, name := range names {
		addr, modulePerms := ak.GetModuleAddressAndPermissions(name)
		if addr == nil {
			return fmt.Errorf("module account %q is in the permission table but unknown to the auth keeper", name)
		}
		if !addr.Equals(authtypes.NewModuleAddress(name)) {
			return fmt.Errorf("module account %q resolves to %s, not its derived address", name, addr)
		}

		existing := ak.GetAccount(ctx, addr)
		switch acc := existing.(type) {
		case nil:
			macc := ak.GetModuleAccount(ctx, name)
			if macc == nil {
				return fmt.Errorf("module account %q could not be created", name)
			}
			ctx.EventManager().EmitEvent(sdk.NewEvent(EventTypeModuleAccountMaterialised,
				sdk.NewAttribute("module", name),
				sdk.NewAttribute("address", addr.String()),
				sdk.NewAttribute("account_number", strconv.FormatUint(macc.GetAccountNumber(), 10)),
			))
		case sdk.ModuleAccountI:
			if acc.GetName() != name {
				return fmt.Errorf("module account at %s is named %q, want %q", addr, acc.GetName(), name)
			}
		default:
			if acc.GetPubKey() != nil {
				return fmt.Errorf("account at module address %s (%s) carries a public key; refusing to convert it", addr, name)
			}
			macc := authtypes.NewModuleAccount(
				authtypes.NewBaseAccount(addr, nil, acc.GetAccountNumber(), 0), name, modulePerms...)
			ak.SetAccount(ctx, macc)
			evm.DeleteCodeHash(ctx, common.BytesToAddress(addr))
			ctx.EventManager().EmitEvent(sdk.NewEvent(EventTypeModuleAccountRepaired,
				sdk.NewAttribute("module", name),
				sdk.NewAttribute("address", addr.String()),
				sdk.NewAttribute("account_number", strconv.FormatUint(acc.GetAccountNumber(), 10)),
				sdk.NewAttribute("previous_type", fmt.Sprintf("%T", acc)),
				sdk.NewAttribute("previous_sequence", strconv.FormatUint(acc.GetSequence(), 10)),
			))
		}
	}
	return nil
}
