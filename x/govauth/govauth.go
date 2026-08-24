// Package govauth centralises the check that an administrative message really
// came from governance.
//
// A message declaring `option (cosmos.msg.v1.signer) = "authority"` is signed by
// whatever address sits in its own authority field. The field therefore proves
// nothing on its own: an attacker writes their own address there and signs the
// transaction themselves. Logging the value into an event does not gate it
// either. Every handler for such a message must call Assert.
package govauth

import (
	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// Address returns the governance module account: the only legitimate caller of
// an authority-signed message. Derived rather than injected, so adding the check
// to a module needs no constructor or app-wiring change.
func Address() string {
	return authtypes.NewModuleAddress(govtypes.ModuleName).String()
}

// Assert fails unless authority is the governance module account.
func Assert(authority string) error {
	if want := Address(); authority != want {
		return errorsmod.Wrapf(
			sdkerrors.ErrUnauthorized,
			"message must come from governance: expected authority %s, got %q",
			want, authority,
		)
	}
	return nil
}
