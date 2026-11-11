package types

import sdk "github.com/cosmos/cosmos-sdk/types"

// The EVM-lane mark.
//
// Every check that makes an Ethereum transaction safe to execute runs in the
// EVM lane's ante handler: the Ethereum signature and the sender, the nonce,
// the fee, the post-quantum window and the EIP-7702 consent. The MsgEthereumTx
// handler checks none of them; it recovers the signer from the raw transaction
// and applies it. A MsgEthereumTx that reaches the message router any other
// way (a group proposal, a governance proposal, a contract's dispatched
// message) is executed with none of those checks, as its signer, at no fee,
// with no nonce check, as often as it is dispatched.
//
// The window decorator, which runs on the EVM lane after the signature is
// verified, marks the context it passes on. From the v3.2.2 plan height the
// MsgEthereumTx handler refuses a context without the mark. The context
// returned by the ante handler is the one the transaction's messages execute
// in, so the mark reaches the handler for every EVM-lane transaction and for
// nothing else.

type evmLaneAnteKey struct{}

// WithEVMLaneAnte marks ctx as an EVM-lane transaction whose ante handler ran.
func WithEVMLaneAnte(ctx sdk.Context) sdk.Context {
	return ctx.WithValue(evmLaneAnteKey{}, true)
}

// EVMLaneAnteRan reports whether ctx carries the mark.
func EVMLaneAnteRan(ctx sdk.Context) bool {
	if ctx.Context() == nil {
		return false
	}
	ok, _ := ctx.Value(evmLaneAnteKey{}).(bool)
	return ok
}

// The top-level dispatch budget (FeatureEVMDispatchTopLevelOnly).
//
// The mark above is a context value, so it is inherited by everything that
// runs below an EVM-lane transaction: a CosmWasm contract the transaction
// reaches through the cross-VM bridge sees it too, and a MsgEthereumTx that
// contract dispatched would pass a guard that looked at the mark alone. The
// budget is derived from the transaction itself: the EVM-lane ante handler
// records how many MsgEthereumTx the transaction carries at top level, and
// each execution of the MsgEthereumTx handler takes one. The top-level
// messages execute in order and each takes its unit on entry, before running
// any EVM code, so a MsgEthereumTx dispatched from below one of them finds the
// budget spent. The budget lives only in the transaction's context and is
// never stored, so it cannot outlive the transaction.

type evmLaneBudgetKey struct{}

// evmLaneBudget is shared by every context derived from the ante handler's,
// which is what lets a nested dispatch see that the top level spent it.
type evmLaneBudget struct{ remaining int }

// WithEVMLaneDispatchBudget records that the transaction carries n
// MsgEthereumTx at top level.
func WithEVMLaneDispatchBudget(ctx sdk.Context, n int) sdk.Context {
	return ctx.WithValue(evmLaneBudgetKey{}, &evmLaneBudget{remaining: n})
}

// TakeEVMLaneDispatch takes one unit of the top-level budget and reports
// whether there was one. A context without a budget has none.
func TakeEVMLaneDispatch(ctx sdk.Context) bool {
	if ctx.Context() == nil {
		return false
	}
	b, _ := ctx.Value(evmLaneBudgetKey{}).(*evmLaneBudget)
	if b == nil || b.remaining <= 0 {
		return false
	}
	b.remaining--
	return true
}
