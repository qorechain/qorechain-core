package app

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

// MaxMultisigNestingDepth bounds how deeply a multisig signature may nest.
//
// Three is generous for anything legitimate: a multisig of multisigs of multisigs
// is already an unusual governance arrangement, and nothing in this chain's own
// operations goes past two.
const MaxMultisigNestingDepth = 3

// MultisigDepthGuardDecorator refuses deeply nested multisig signatures in CheckTx.
//
// THE DEFECT IT COVERS, which is upstream and not ours. The SDK's signatureDataToBz
// recurses into nested multisig data, collects every descendant's bytes into one
// slice, and then marshals the accumulated set and appends that aggregate to the
// same slice. The output roughly doubles per level, so its size is exponential in
// nesting depth. It is called from SetPubKeyDecorator purely to build events, which
// is to say BEFORE any signature is verified: the attacker needs no valid signature
// and no funded account. One small transaction, broadcast to any public RPC, can
// exhaust a node's memory. Our endpoints accept broadcasts.
//
// WHY CheckTx ONLY, and why that is enough here. A DeliverTx rejection would change
// which transactions are valid, making this consensus-breaking and requiring a
// coordinated height. CheckTx is block-production policy: it can be rolled out one
// node at a time, and a block that somehow already contains such a transaction still
// executes identically everywhere. The attack is a memory exhaustion of the node
// that ACCEPTS the transaction, and it never needs to reach a block at all, so
// refusing it at the mempool is refusing it where the damage happens.
//
// This runs BEFORE the SDK's SetPubKeyDecorator, which is the thing that would
// expand the signature. A guard that ran after it would be measuring a node that had
// already run out of memory.
type MultisigDepthGuardDecorator struct{}

func NewMultisigDepthGuardDecorator() MultisigDepthGuardDecorator {
	return MultisigDepthGuardDecorator{}
}

func (d MultisigDepthGuardDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	// Mempool only, and simulation is left alone so tooling can still estimate gas.
	if !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}

	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return next(ctx, tx, simulate)
	}
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		// Malformed signatures are the signature decorators' business to report.
		// Refusing here would give a worse error for an unrelated problem.
		return next(ctx, tx, simulate)
	}

	for i, sig := range sigs {
		if depth := multisigDepth(sig.Data, 0); depth > MaxMultisigNestingDepth {
			return ctx, errorsmod.Wrapf(errortypes.ErrTooManySignatures,
				"signature %d nests multisig %d levels deep, more than the %d this node "+
					"will admit; expanding it costs memory exponential in depth and "+
					"happens before any signature is verified",
				i, depth, MaxMultisigNestingDepth)
		}
	}

	return next(ctx, tx, simulate)
}

// multisigDepth reports how deeply signature data nests.
//
// It stops descending as soon as the bound is passed. Walking the whole tree to
// report an exact depth would do the very work this guard exists to avoid: a
// deliberately enormous tree would exhaust memory inside the check meant to prevent
// exhausting memory.
func multisigDepth(data signing.SignatureData, depth int) int {
	multi, ok := data.(*signing.MultiSignatureData)
	if !ok {
		return depth
	}
	depth++
	if depth > MaxMultisigNestingDepth {
		return depth
	}
	deepest := depth
	for _, child := range multi.Signatures {
		if d := multisigDepth(child, depth); d > deepest {
			deepest = d
			if deepest > MaxMultisigNestingDepth {
				return deepest
			}
		}
	}
	return deepest
}
