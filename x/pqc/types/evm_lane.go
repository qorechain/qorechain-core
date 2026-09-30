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
