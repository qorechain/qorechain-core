package app

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
)

// The multisig signature bound, identical to the one cosmos-sdk v0.53.8 enforces
// in flattenSignaturesAtDepth(data, 0, 2, 32): depth counts from 0 at the top
// signature, no node may sit deeper than MaxMultisigNestingDepth, and no multisig
// node may carry more than MaxMultisigBreadth members. That admits a multisig of
// multisigs of single keys and nothing deeper. Matching upstream exactly means the
// eventual SDK bump changes nothing for clients on this path.
const (
	MaxMultisigNestingDepth = 2
	MaxMultisigBreadth      = 32
)

// MultisigDepthGuardDecorator refuses multisig signatures beyond the bound in
// every execution mode except block execution.
//
// THE DEFECT IT COVERS, which is upstream and not ours. The SDK's signatureDataToBz
// (v0.53.6) recurses into nested multisig data, collects every descendant's bytes
// into one slice, and then marshals the accumulated set and appends that aggregate
// to the same slice. The output roughly doubles per level, so its size is
// exponential in nesting depth. It is called from SetPubKeyDecorator purely to
// build events, which is to say BEFORE any signature is verified: the attacker
// needs no valid signature, and the signature data comes from mode_info, so an
// ordinary single-key account can carry an arbitrarily deep multisig.
//
// WHERE IT RUNS. CheckTx, ReCheckTx, simulate (gRPC, REST and abci_query
// /app/simulate all arrive with simulate set), PrepareProposal and ProcessProposal.
// None of these modes executes during replay or changes stored state: simulate
// and CheckTx never commit, and a ProcessProposal refusal is this validator's vote
// against a proposal, not a state transition. So all of them can be rolled out one
// node at a time. Refusing in ProcessProposal is what stops a malicious proposer
// from putting such a transaction straight into a block and exhausting every
// validator's memory; honest proposers never include one, because every mempool
// already refuses it.
//
// Block execution (FinalizeBlock, and InitChain's gentx delivery) is left
// untouched: a DeliverTx refusal would change which transactions are valid and
// needs a coordinated height. That leg ships separately behind a protocol feature.
//
// This runs BEFORE the SDK's SetPubKeyDecorator, which is the thing that would
// expand the signature. A guard that ran after it would be measuring a node that
// had already run out of memory.
type MultisigDepthGuardDecorator struct{}

func NewMultisigDepthGuardDecorator() MultisigDepthGuardDecorator {
	return MultisigDepthGuardDecorator{}
}

func (d MultisigDepthGuardDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	if !enforcesNodeLocalPolicy(ctx, simulate) {
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
		if err := checkSignatureBound(sig.Data, 0); err != nil {
			return ctx, errorsmod.Wrapf(errortypes.ErrTooManySignatures,
				"signature %d: %v", i, err)
		}
	}

	return next(ctx, tx, simulate)
}

// enforcesNodeLocalPolicy reports whether ctx is one of the execution modes in
// which a node may apply its own admission policy: the mempool (CheckTx and
// ReCheckTx, both on the check state), simulation, and proposal building and
// voting. It is false for block execution.
//
// The mode is read from IsCheckTx and the simulate flag as well as ExecMode,
// because baseapp only stamps ExecMode on some contexts: the check state carries
// the zero value (ExecModeCheck) with IsCheckTx set, and InitChain's context also
// carries the zero value but with IsCheckTx clear. Reading ExecMode alone would
// therefore mistake genesis transaction delivery for CheckTx.
func enforcesNodeLocalPolicy(ctx sdk.Context, simulate bool) bool {
	if ctx.IsCheckTx() || simulate {
		return true
	}
	switch ctx.ExecMode() {
	case sdk.ExecModeSimulate, sdk.ExecModePrepareProposal, sdk.ExecModeProcessProposal:
		return true
	}
	return false
}

// checkSignatureBound is flattenSignaturesAtDepth from cosmos-sdk v0.53.8 with
// the byte building taken out: the same checks, in the same order, per node, and
// it stops at the first violation. Keeping the upstream control flow (depth is
// judged before type) is what makes the verdict identical, including the edge
// cases: an empty multisig at the depth limit is accepted, because no member ever
// reaches the next level, while a nil or unknown signature is refused at any depth.
//
// It builds no bytes and never descends past the bound, so a deliberately enormous
// tree is refused after visiting at most about a thousand nodes (32 members of 32
// members), however large the rest of it is.
func checkSignatureBound(data signing.SignatureData, depth int) error {
	switch {
	case data == nil:
		return fmt.Errorf("SignatureData is required")
	case depth > MaxMultisigNestingDepth:
		return fmt.Errorf("max depth of %d reached", MaxMultisigNestingDepth)
	}

	if _, ok := data.(*signing.SingleSignatureData); ok {
		return nil
	}

	multi, ok := data.(*signing.MultiSignatureData)
	switch {
	case !ok:
		return fmt.Errorf("unexpected signature data type %T", data)
	case len(multi.Signatures) > MaxMultisigBreadth:
		return fmt.Errorf("max breadth of %d reached", MaxMultisigBreadth)
	}

	for _, member := range multi.Signatures {
		if err := checkSignatureBound(member, depth+1); err != nil {
			return err
		}
	}
	return nil
}
