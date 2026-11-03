package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// multisigExtensionEnabled reports whether the post-quantum multisig
// extension (v3.2.7, FeaturePQCMultisig) is accepted. The record is read off
// the gas meter by the keeper. A public build's PQC keeper stub always
// answers no.
func multisigExtensionEnabled(ctx sdk.Context, k pqcmod.PQCKeeper) bool {
	return k != nil && k.ProtocolFeatureEnabled(ctx, pqctypes.FeaturePQCMultisig)
}

// multisigExtensionGateDecorator sits directly in front of the SDK's
// ExtensionOptionsDecorator. The application's extension checker admits the
// multisig extension, because the checker has no context and so cannot read
// the switch; below FeaturePQCMultisig this gate refuses a transaction that
// carries the extension anywhere among its options, with exactly the error
// the checker returned for it before (ErrUnknownExtensionOptions, unwrapped)
// and at the same point of the chain, so nothing a refused transaction is
// charged or told differs. From the switch it passes everything through.
type multisigExtensionGateDecorator struct {
	keeper pqcmod.PQCKeeper
}

func (d multisigExtensionGateDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if multisigExtensionEnabled(ctx, d.keeper) {
		return next(ctx, tx, simulate)
	}
	if extTx, ok := tx.(ante.HasExtensionOptionsTx); ok {
		for _, opt := range extTx.GetExtensionOptions() {
			if opt.GetTypeUrl() == pqctypes.MultisigSigTypeURL {
				return ctx, errortypes.ErrUnknownExtensionOptions
			}
		}
	}
	return next(ctx, tx, simulate)
}
