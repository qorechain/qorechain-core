package app

import (
	"errors"

	circuitante "cosmossdk.io/x/circuit/ante"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	aimod "github.com/qorechain/qorechain-core/x/ai"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
)

// HandlerOptions are the options required for constructing the QoreChain AnteHandler.
type HandlerOptions struct {
	ante.HandlerOptions
	CircuitKeeper circuitante.CircuitBreaker
	PQCKeeper     pqcmod.PQCKeeper
	PQCClient     pqcmod.PQCClient
	AIKeeper      aimod.AIKeeper
}

// NewAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer.
//
// AnteHandler chain order (per architecture spec):
//  1. PQC verify (x/pqc)
//  2. AI anomaly check (x/ai)
//  3. Standard decorators
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errors.New("account keeper is required for ante builder")
	}

	if options.BankKeeper == nil {
		return nil, errors.New("bank keeper is required for ante builder")
	}

	if options.SignModeHandler == nil {
		return nil, errors.New("sign mode handler is required for ante builder")
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(),
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
		// PQC signature verification — runs before standard sig verify
		NewPQCVerifyDecorator(options.PQCKeeper, options.PQCClient),
		// AI anomaly check — runs after PQC, before standard decorators
		NewAIAnomalyDecorator(options.AIKeeper),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler, options.SigVerifyOptions...),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
