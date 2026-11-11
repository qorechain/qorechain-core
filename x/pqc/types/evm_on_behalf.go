package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// EVM runs made on an account's behalf (FeatureCrossVMOnBehalfEVMWindow).
//
// A module that runs the EVM as an account it did not receive a signature
// from (x/crossvm executing a MsgCrossVMCall sent under an authz grant) marks
// the context of that run. The EVM keeper then settles the run's sender as
// the sender of an EVM-lane transaction instead of taking it as authorised by
// the module, and the run carries the EVM-lane mark, so the standard
// precompiles and the cross-VM bridge treat the account as the window-bound
// account it is. The mark lives only on the context of that run and is never
// stored.

type evmSenderSettledKey struct{}

// WithEVMSenderSettled marks ctx as an EVM run on an account's behalf whose
// sender is settled against its own window, and as an EVM-lane run.
func WithEVMSenderSettled(ctx sdk.Context) sdk.Context {
	return WithEVMLaneAnte(ctx.WithValue(evmSenderSettledKey{}, true))
}

// EVMSenderSettled reports whether ctx carries the mark.
func EVMSenderSettled(ctx sdk.Context) bool {
	if ctx.Context() == nil {
		return false
	}
	ok, _ := ctx.Value(evmSenderSettledKey{}).(bool)
	return ok
}

// EVMWindowUsedSourceCrossVM is the AttributeKeySource of the window charge
// a cross-VM call into the EVM made on the account's behalf uses (one window
// transaction, when the call succeeds).
const EVMWindowUsedSourceCrossVM = "crossvm"
