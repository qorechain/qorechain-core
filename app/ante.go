package app

import (
	"errors"

	corestoretypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	circuitante "cosmossdk.io/x/circuit/ante"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sdkvesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"

	cosmosevmante "github.com/cosmos/evm/ante"
	cosmosante "github.com/cosmos/evm/ante/cosmos"
	evmante "github.com/cosmos/evm/ante/evm"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	ibcante "github.com/cosmos/ibc-go/v10/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"

	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	feemarketkeeper "github.com/cosmos/evm/x/feemarket/keeper"

	aimod "github.com/qorechain/qorechain-core/x/ai"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	svmmod "github.com/qorechain/qorechain-core/x/svm"
)

// HandlerOptions are the options required for constructing the QoreChain AnteHandler.
type HandlerOptions struct {
	ante.HandlerOptions

	// QoreChain keepers
	CircuitKeeper circuitante.CircuitBreaker
	PQCKeeper     pqcmod.PQCKeeper
	PQCClient     pqcmod.PQCClient
	AIKeeper      aimod.AIKeeper
	SVMKeeper     svmmod.SVMKeeper

	// EVM keepers — the concrete AccountKeeper is needed because the EVM ante
	// interfaces require GetSequence which the SDK ante.AccountKeeper interface lacks.
	EVMAccountKeeper authkeeper.AccountKeeper
	FeeMarketKeeper  feemarketkeeper.Keeper
	EvmKeeper        *evmkeeper.Keeper
	IBCKeeper        *ibckeeper.Keeper
	MaxTxGasWanted   uint64

	// CosmWasm
	WasmKeeper            *wasmkeeper.Keeper
	WasmConfig            *wasmtypes.NodeConfig
	TXCounterStoreService corestoretypes.KVStoreService
}

// NewAnteHandler returns an AnteHandler with dual routing:
//   - EVM path: for transactions with ExtensionOptionsEthereumTx
//   - QoreChain SDK path: for standard QoreChain SDK transactions
//
// The QoreChain SDK path includes PQC verification and AI anomaly detection.
// The EVM path uses the EVMMonoDecorator which handles all EVM-specific checks.
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
	if options.EvmKeeper == nil {
		return nil, errors.New("evm keeper is required for ante builder")
	}
	if options.IBCKeeper == nil {
		return nil, errors.New("ibc keeper is required for ante builder")
	}

	// Return the dual-routing ante handler function.
	return func(ctx sdk.Context, tx sdk.Tx, sim bool) (sdk.Context, error) {
		var anteHandler sdk.AnteHandler

		// Check for EVM extension options to route appropriately.
		txWithExtensions, ok := tx.(ante.HasExtensionOptionsTx)
		if ok {
			opts := txWithExtensions.GetExtensionOptions()
			if len(opts) > 0 {
				switch typeURL := opts[0].GetTypeUrl(); typeURL {
				case "/cosmos.evm.vm.v1.ExtensionOptionsEthereumTx":
					// EVM transaction — route to mono decorator
					anteHandler = newMonoEVMAnteHandler(options)
				case "/cosmos.evm.types.v1.ExtensionOptionDynamicFeeTx":
					// QoreChain SDK tx with dynamic fee — route to QoreChain SDK path
					anteHandler = newCosmosAnteHandler(options)
				default:
					return ctx, errorsmod.Wrapf(
						errortypes.ErrUnknownExtensionOptions,
						"rejecting tx with unsupported extension option: %s", typeURL,
					)
				}
				return anteHandler(ctx, tx, sim)
			}
		}

		// Standard QoreChain SDK transaction — route to QoreChain SDK path.
		switch tx.(type) {
		case sdk.Tx:
			anteHandler = newCosmosAnteHandler(options)
		default:
			return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid transaction type: %T", tx)
		}

		return anteHandler(ctx, tx, sim)
	}, nil
}

// newMonoEVMAnteHandler returns an AnteHandler for Ethereum transactions.
// Uses the EVMMonoDecorator which handles all EVM-specific pre-checks in one pass.
func newMonoEVMAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmante.NewEVMMonoDecorator(
			options.EVMAccountKeeper,
			options.FeeMarketKeeper,
			options.EvmKeeper,
			options.MaxTxGasWanted,
		),
	)
}

// newCosmosAnteHandler returns an AnteHandler for standard QoreChain SDK transactions.
// This path includes:
//   - EVM message rejection (MsgEthereumTx must use extension options)
//   - AuthZ limiter (prevents EVM msgs in authz)
//   - PQC signature verification (QoreChain custom)
//   - AI anomaly detection (QoreChain custom)
//   - Standard SDK decorators
//   - IBC redundant relay check
//   - EVM gas wanted tracking
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// Reject MsgEthereumTx on the QoreChain SDK path — must use ExtensionOptionsEthereumTx
		cosmosante.NewRejectMessagesDecorator(),
		// Prevent MsgEthereumTx and MsgCreateVestingAccount from being used in authz
		cosmosante.NewAuthzLimiterDecorator(
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
			sdk.MsgTypeURL(&sdkvesting.MsgCreateVestingAccount{}),
		),
		ante.NewSetUpContextDecorator(),
		// CosmWasm decorators — limit simulation gas and track tx position
		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
		wasmkeeper.NewGasRegisterDecorator(options.WasmKeeper.GetGasRegister()),
		wasmkeeper.NewTxContractsDecorator(),
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
		// PQC signature verification — runs before standard sig verify
		NewPQCVerifyDecorator(options.PQCKeeper, options.PQCClient),
		// AI anomaly check — runs after PQC, before standard decorators
		NewAIAnomalyDecorator(options.AIKeeper),
		// SVM compute budget check — validates SVM messages are within params
		NewSVMComputeBudgetDecorator(options.SVMKeeper),
		// SVM fee deduction — placeholder for future compute-unit fee logic
		NewSVMDeductFeeDecorator(options.SVMKeeper),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		// Use EVM fee market min gas price instead of standard min gas price
		cosmosante.NewMinGasPriceDecorator(options.FeeMarketKeeper, options.EvmKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TxFeeChecker),
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		// IBC redundant relay check — prevents relayers from wasting fees
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		// Track cumulative gas wanted for EIP-1559 base fee calculation
		evmante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper),
	)
}

// sigVerificationGasConsumerWithPQC is the signature gas consumer that handles
// both standard key types and PQC keys. It delegates to the EVM gas consumer
// for standard types (eth_secp256k1, multisig).
// PQC keys are verified separately by the PQCVerifyDecorator before this point,
// and the PQC decorator handles its own gas consumption.
func sigVerificationGasConsumerWithPQC(
	meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params,
) error {
	return cosmosevmante.SigVerificationGasConsumer(meter, sig, params)
}
