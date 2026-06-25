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
	antetypes "github.com/cosmos/evm/ante/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	ibcante "github.com/cosmos/ibc-go/v10/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"

	feemarketkeeper "github.com/cosmos/evm/x/feemarket/keeper"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"

	aimod "github.com/qorechain/qorechain-core/x/ai"
	burnmod "github.com/qorechain/qorechain-core/x/burn"
	licensemod "github.com/qorechain/qorechain-core/x/license"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
	fairblockmod "github.com/qorechain/qorechain-core/x/fairblock"
	gasabstractionmod "github.com/qorechain/qorechain-core/x/gasabstraction"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	svmmod "github.com/qorechain/qorechain-core/x/svm"
)

// HandlerOptions are the options required for constructing the QoreChain AnteHandler.
type HandlerOptions struct {
	ante.HandlerOptions

	// QoreChain keepers
	CircuitKeeper        circuitante.CircuitBreaker
	PQCKeeper            pqcmod.PQCKeeper
	PQCClient            pqcmod.PQCClient
	AIKeeper             aimod.AIKeeper
	SVMKeeper            svmmod.SVMKeeper
	SVMBankKeeper        svmmod.SVMBankKeeper
	FairBlockKeeper      fairblockmod.FairBlockKeeper
	GasAbstractionKeeper gasabstractionmod.GasAbstractionKeeper
	BurnKeeper           burnmod.BurnKeeper
	LicenseKeeper        licensemod.LicenseKeeper

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
//   - Cosmos SDK path: for standard Cosmos SDK transactions
//
// The Cosmos SDK path includes PQC verification and AI anomaly detection.
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

	// Pre-build both handler chains once to avoid per-TX allocation overhead.
	// The decorators hold pointers to these param structs; we refresh the
	// pointed-to values from keeper state at the start of every AnteHandle
	// invocation so that governance-updated params take effect immediately
	// instead of being stale defaults for the lifetime of the node (ME-05).
	evmParams := evmtypes.DefaultParams()
	fmParams := feemarkettypes.DefaultParams()
	evmHandler := newMonoEVMAnteHandler(options, &evmParams, &fmParams)
	cosmosHandler := newCosmosAnteHandler(options, &fmParams)

	// Return the dual-routing ante handler function.
	return func(ctx sdk.Context, tx sdk.Tx, sim bool) (sdk.Context, error) {
		// Refresh cached params from on-chain state so decorators always
		// operate on live values rather than stale construction-time defaults.
		evmParams = options.EvmKeeper.GetParams(ctx)
		fmParams = options.FeeMarketKeeper.GetParams(ctx)
		// Check for EVM extension options to route appropriately.
		txWithExtensions, ok := tx.(ante.HasExtensionOptionsTx)
		if ok {
			opts := txWithExtensions.GetExtensionOptions()
			if len(opts) > 0 {
				switch typeURL := opts[0].GetTypeUrl(); typeURL {
				case "/cosmos.evm.vm.v1.ExtensionOptionsEthereumTx":
					// EVM transaction — route to mono decorator
					return evmHandler(ctx, tx, sim)
				case "/cosmos.evm.ante.v1.ExtensionOptionDynamicFeeTx":
					// Cosmos SDK tx with dynamic fee — route to Cosmos SDK path
					return cosmosHandler(ctx, tx, sim)
				case pqctypes.HybridSigTypeURL:
					// Cosmos SDK tx carrying a PQC hybrid signature — route to the
					// Cosmos SDK path (the hybrid verify decorator runs there).
					return cosmosHandler(ctx, tx, sim)
				default:
					return ctx, errorsmod.Wrapf(
						errortypes.ErrUnknownExtensionOptions,
						"rejecting tx with unsupported extension option: %s", typeURL,
					)
				}
			}
		}

		// Standard Cosmos SDK transaction — route to Cosmos SDK path.
		switch tx.(type) {
		case sdk.Tx:
			return cosmosHandler(ctx, tx, sim)
		default:
			return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid transaction type: %T", tx)
		}
	}, nil
}

// newMonoEVMAnteHandler returns an AnteHandler for Ethereum transactions.
// Uses the EVMMonoDecorator which handles all EVM-specific pre-checks in one pass.
func newMonoEVMAnteHandler(options HandlerOptions, evmParams *evmtypes.Params, fmParams *feemarkettypes.Params) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmante.NewEVMMonoDecorator(
			options.EVMAccountKeeper,
			options.FeeMarketKeeper,
			options.EvmKeeper,
			options.MaxTxGasWanted,
			evmParams,
			fmParams,
		),
	)
}

// genesisExemptDecorator wraps an ante decorator and skips it during InitChain
// (block height 0). Genesis transactions (gentxs) are delivered fee-free, so
// fee-floor decorators like the min-gas-price check must not run against them.
type genesisExemptDecorator struct{ inner sdk.AnteDecorator }

func (d genesisExemptDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if ctx.BlockHeight() == 0 {
		return next(ctx, tx, simulate)
	}
	return d.inner.AnteHandle(ctx, tx, simulate, next)
}

// newCosmosAnteHandler returns an AnteHandler for standard Cosmos SDK transactions.
// This path includes:
//   - EVM message rejection (MsgEthereumTx must use extension options)
//   - AuthZ limiter (prevents EVM msgs in authz)
//   - PQC signature verification (QoreChain custom)
//   - AI anomaly detection (QoreChain custom)
//   - Standard SDK decorators
//   - IBC redundant relay check
//   - EVM gas wanted tracking
func newCosmosAnteHandler(options HandlerOptions, fmParams *feemarkettypes.Params) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		// Reject MsgEthereumTx on the Cosmos SDK path — must use ExtensionOptionsEthereumTx
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
		// Cheap validators — reject malformed txs before expensive PQC/AI work
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		// Validator licensing gate — MsgCreateValidator requires an active
		// validator_operator license + >=100,000 QOR self-bond (height-0 exempt).
		NewValidatorLicenseDecorator(options.LicenseKeeper),
		// PQC signature verification — runs before standard sig verify
		NewPQCVerifyDecorator(options.PQCKeeper, options.PQCClient),
		// PQC hybrid signature verification — checks TX extension for dual Ed25519 + ML-DSA-87
		NewPQCHybridVerifyDecorator(options.PQCKeeper, options.PQCClient),
		// PQC anti-replay guard — rejects PQC-signed txs with stale/future timestamps
		NewPQCReplayGuardDecorator(),
		// AI anomaly check — runs after PQC, before standard decorators
		NewAIAnomalyDecorator(options.AIKeeper),
		// FairBlock threshold IBE check — passthrough stub in v1.2.0
		NewFairBlockDecorator(options.FairBlockKeeper),
		// SVM compute budget check — validates SVM messages are within params
		NewSVMComputeBudgetDecorator(options.SVMKeeper),
		// SVM fee deduction — charges uqor per SVM execute/deploy message
		NewSVMDeductFeeDecorator(options.SVMKeeper, options.SVMBankKeeper),
		// Use EVM fee market min gas price instead of standard min gas price.
		// Exempt at genesis (height 0): gentxs are delivered fee-free during
		// InitChain, but the min-gas-price floor would otherwise reject them.
		genesisExemptDecorator{cosmosante.NewMinGasPriceDecorator(fmParams)},
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		// Gas abstraction — convert non-native fee denoms for fee deduction
		NewGasAbstractionDecorator(options.GasAbstractionKeeper),
		// Build the fee checker from the live, per-block-refreshed feemarket
		// params (fmParams) rather than options.TxFeeChecker, which is bound to
		// stale DefaultParams (base_fee 1e9) and would ignore the calibrated
		// base_fee for cosmos-path txs.
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, evmante.NewDynamicFeeChecker(fmParams)),
		ante.NewSetPubKeyDecorator(options.AccountKeeper),
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		// Burn module TX counter — counts transactions for milestone burn tracking
		NewBurnTxCountDecorator(options.BurnKeeper),
		// IBC redundant relay check — prevents relayers from wasting fees
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
		// Track cumulative gas wanted for EIP-1559 base fee calculation
		evmante.NewGasWantedDecorator(options.EvmKeeper, options.FeeMarketKeeper, fmParams),
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

// Ensure antetypes import is used for extension option checking.
var _ = antetypes.HasDynamicFeeExtensionOption
