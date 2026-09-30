package app

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
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
// every node-local execution mode, and in block execution once the v3.2.2
// protocol feature FeatureMultisigSignatureBound is on.
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
// From v3.2.2 the linked SDK is v0.53.8, whose SetPubKeyDecorator applies the
// same bound itself (flattenSignaturesAtDepth) in every mode, including block
// execution below the plan height. That refusal comes after DeductFee and the
// size gas, so it reports more gas used than this guard's. This guard is what
// keeps the refusal ahead of every decorator that could touch the signature,
// and in block execution it takes over from the plan height.
//
// NODE-LOCAL MODES (since v3.2.1, unconditional). CheckTx, ReCheckTx, simulate
// (gRPC, REST and abci_query /app/simulate all arrive with simulate set),
// PrepareProposal and ProcessProposal. None of these modes executes during replay
// or changes stored state: simulate and CheckTx never commit, and a
// ProcessProposal refusal is this validator's vote against a proposal, not a state
// transition. So all of them can be rolled out one node at a time. Refusing in
// ProcessProposal is what stops a malicious proposer from putting such a
// transaction straight into a block and exhausting every validator's memory;
// honest proposers never include one, because every mempool already refuses it.
//
// BLOCK EXECUTION (FinalizeBlock, and InitChain's gentx delivery) enforces the
// same bound only when FeatureMultisigSignatureBound is enabled
// (QSR-2026-0035, QSR-2026-0164). A DeliverTx refusal changes which transactions
// are valid, so it waits for the v3.2.2 plan height: below it the switch is off
// and the decorator passes block execution straight through, exactly as before,
// so history replays byte-identically. The switch is read off the gas meter, so
// consulting it changes no gas either. The release handler turns it on in
// PreBlock of the plan height, before any transaction of that block runs, so
// there is no one-block gap. A public build's PQC keeper stub always answers
// "off", so only the full validator build enforces this leg.
//
// This runs BEFORE the SDK's SetPubKeyDecorator, which is the thing that would
// expand the signature, and before DeductFee, so a refused transaction pays no
// fee and emits nothing. A guard that ran after SetPubKey would be measuring a
// node that had already run out of memory.
type MultisigDepthGuardDecorator struct {
	// blockBoundEnabled reports whether the bound also applies in block
	// execution. Nil means never, which is the pre-v3.2.2 behaviour.
	blockBoundEnabled func(sdk.Context) bool
}

// NewMultisigDepthGuardDecorator builds the guard. blockBoundEnabled answers
// whether FeatureMultisigSignatureBound is on for ctx; it is consulted only in
// block execution. Pass nil to keep block execution unchecked.
func NewMultisigDepthGuardDecorator(blockBoundEnabled func(sdk.Context) bool) MultisigDepthGuardDecorator {
	return MultisigDepthGuardDecorator{blockBoundEnabled: blockBoundEnabled}
}

// multisigBoundFeature is the block-execution switch read from the PQC
// keeper's protocol-feature record.
func multisigBoundFeature(k interface {
	ProtocolFeatureEnabled(ctx sdk.Context, name string) bool
}) func(sdk.Context) bool {
	if k == nil {
		return nil
	}
	return func(ctx sdk.Context) bool {
		return k.ProtocolFeatureEnabled(ctx, pqctypes.FeatureMultisigSignatureBound)
	}
}

// enforces reports whether the bound applies to this transaction.
func (d MultisigDepthGuardDecorator) enforces(ctx sdk.Context, simulate bool) bool {
	if enforcesNodeLocalPolicy(ctx, simulate) {
		return true
	}
	return d.blockBoundEnabled != nil && d.blockBoundEnabled(ctx)
}

func (d MultisigDepthGuardDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	if !d.enforces(ctx, simulate) {
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
// voting. It is false for block execution, where the bound depends on
// FeatureMultisigSignatureBound instead (see MultisigDepthGuardDecorator).
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
