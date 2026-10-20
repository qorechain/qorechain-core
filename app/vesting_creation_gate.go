package app

import (
	"bytes"
	"errors"

	"cosmossdk.io/collections"
	circuittypes "cosmossdk.io/x/circuit/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// QSR-2026-0402. The stock vesting handlers let any funded sender turn any
// address that does not exist yet into a vesting or permanently locked account,
// with a schedule the sender picks. The account type is permanent: the owner's
// later delegations cost more gas, and the vesting account the owner was meant
// to receive there can never be created (code 18, "already exists"). The
// authz-nested form has been refused since QSR-2026-0195; the direct form, and
// the forms dispatched by x/group or a contract, reach the stock handler.
//
// From the vesting-create-self-only switch the vesting message server (patched
// SDK, deps/patches/cosmos-sdk/0004) asks vestingCreationGate first, and it
// admits a creation only when:
//   - the sender creates the account for itself (to == from), or
//   - the sender is the governance module account, or
//   - the sender holds an x/circuit permission over that message type
//     (super admin, all messages, or some messages naming it). Such an account
//     can already open and close the message for everyone, so letting it create
//     accounts for others grants nothing it could not reach already. This is
//     how the allocation escrows deliver vesting schedules to buyers: a circuit
//     super admin grants an escrow LEVEL_SOME_MSGS over
//     MsgCreatePeriodicVestingAccount, and revokes it after the delivery.
//   - the sender is one of the mainnet investor allocation escrows below, which
//     deliver vesting schedules to buyers as part of their purpose.

// investorAllocationEscrows are the qorechain-vladi genesis escrows that create
// vesting accounts for investors (seed, private, strategic, public sale). They
// are mainnet addresses; on any other chain the list matches nothing.
var investorAllocationEscrows = mustDecodeEscrows(
	"qor1ue0ujtz56wy2z2paje3sxyhjx445we8yys7x8v", // seed
	"qor12nujzu7yf4e30yyq8ndax4e3hde2lulqsh8ad4", // private
	"qor1w225478z08ek8z3udlvv97fy2atfnmvrctqsrg", // strategic
	"qor1us7vg99vcyqep8jf8xx4fmt0fvt7e05sa07d34", // public sale
)

// mustDecodeEscrows keys the escrows by their address bytes, so the check does
// not depend on the process-wide bech32 prefix configuration.
func mustDecodeEscrows(addrs ...string) map[string]bool {
	out := make(map[string]bool, len(addrs))
	for _, a := range addrs {
		_, bz, err := bech32.DecodeAndConvert(a)
		if err != nil {
			panic(err)
		}
		out[string(bz)] = true
	}
	return out
}

// circuitPermissionReader returns the circuit-breaker permissions stored for an
// address, and whether there are any.
type circuitPermissionReader func(ctx sdk.Context, addr []byte) (circuittypes.Permissions, bool, error)

// checkVestingCreation applies the rule above. It reads no state unless the
// creation names somebody else.
func checkVestingCreation(ctx sdk.Context, perms circuitPermissionReader, msg sdk.Msg, from, to sdk.AccAddress) error {
	if bytes.Equal(from, to) {
		return nil
	}
	if bytes.Equal(from, authtypes.NewModuleAddress(govtypes.ModuleName)) {
		return nil
	}
	if investorAllocationEscrows[string(from)] {
		return nil
	}
	typeURL := sdk.MsgTypeURL(msg)
	if perms != nil {
		p, found, err := perms(ctx, from)
		if err != nil {
			return err
		}
		if found && circuitPermissionCovers(p, typeURL) {
			return nil
		}
	}
	return sdkerrors.ErrUnauthorized.Wrapf(
		"%s may only create an account for its own address unless it is governance or holds a circuit permission over %s",
		sdk.AccAddress(from).String(), typeURL)
}

// circuitPermissionCovers reports whether p lets its holder trip or reset the
// circuit for typeURL, with the same levels x/circuit itself honours.
func circuitPermissionCovers(p circuittypes.Permissions, typeURL string) bool {
	switch p.Level {
	case circuittypes.Permissions_LEVEL_SUPER_ADMIN, circuittypes.Permissions_LEVEL_ALL_MSGS:
		return true
	case circuittypes.Permissions_LEVEL_SOME_MSGS:
		for _, u := range p.LimitTypeUrls {
			if u == typeURL {
				return true
			}
		}
	}
	return false
}

// vestingCreationGate is wired into the patched vesting message server by the
// full build. Below the switch it answers nil without reading state.
func (app *QoreChainApp) vestingCreationGate(ctx sdk.Context, msg sdk.Msg, from, to sdk.AccAddress) error {
	if app.PQCKeeper == nil || !app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureVestingCreateSelfOnly) {
		return nil
	}
	return checkVestingCreation(ctx, app.circuitPermissions, msg, from, to)
}

func (app *QoreChainApp) circuitPermissions(ctx sdk.Context, addr []byte) (circuittypes.Permissions, bool, error) {
	p, err := app.CircuitKeeper.Permissions.Get(ctx, addr)
	if errors.Is(err, collections.ErrNotFound) {
		return circuittypes.Permissions{}, false, nil
	}
	if err != nil {
		return circuittypes.Permissions{}, false, err
	}
	return p, true, nil
}
