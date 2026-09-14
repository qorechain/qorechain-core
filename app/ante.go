package app

import (
	"errors"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

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
	fairblockmod "github.com/qorechain/qorechain-core/x/fairblock"
	gasabstractionmod "github.com/qorechain/qorechain-core/x/gasabstraction"
	licensemod "github.com/qorechain/qorechain-core/x/license"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
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
	StakingKeeper        *stakingkeeper.Keeper

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

	// PARAMS ARE READ PER INVOCATION, INTO LOCALS, AND NEVER SHARED.
	//
	// This used to pre-build both handler chains once and hand the decorators
	// POINTERS to two param structs that the returned handler then overwrote on
	// every call. The intent was sound, namely that governance-updated params take
	// effect immediately instead of being stale for the process lifetime, but the
	// implementation made those structs shared mutable state.
	//
	// The same ante handler instance serves the SIMULATE path, and simulation is
	// served from the gRPC, REST and JSON-RPC goroutines, which run concurrently
	// with consensus. Our validators run with the API and JSON-RPC enabled, and the
	// public endpoint answers simulation requests. So an unauthenticated
	// eth_estimateGas wrote these structs from a query context while FinalizeBlock
	// decorators read them: an unsynchronised read and write of a multi-field
	// struct, and block execution capable of validating a transaction against
	// fee-market params fetched for a different context. Divergence that depends on
	// timing and on external query traffic is the worst property a consensus bug can
	// have.
	//
	// Locals inside the closure cost an allocation per transaction and remove the
	// sharing entirely. That is the right trade: the previous version was optimising
	// away an allocation at the cost of a consensus hazard.
	return func(ctx sdk.Context, tx sdk.Tx, sim bool) (sdk.Context, error) {
		evmParams := options.EvmKeeper.GetParams(ctx)
		fmParams := options.FeeMarketKeeper.GetParams(ctx)
		evmHandler := newMonoEVMAnteHandler(options, &evmParams, &fmParams)
		cosmosHandler := newCosmosAnteHandler(options, &fmParams)

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
//
// This lane used to contain exactly ONE decorator, the cosmos/evm mono decorator, so
// every QoreChain gate was skipped on it. Two of those absences were not acceptable:
//
//   - THE CIRCUIT BREAKER. It is the chain's emergency brake, the thing that closed
//     the vesting door in minutes without a release. It did not cover the EVM lane at
//     all, so an exploit arriving there could not have been stopped without shipping
//     a binary. An emergency tool with a blind spot on one of three lanes is not an
//     emergency tool.
//
//   - THE POST-QUANTUM REQUIREMENT. The chain states that every account must hold an
//     ML-DSA-87 key before transacting. On this lane that was not enforced, and the
//     consequence was measured, not theoretical: an account that never registered a
//     key moved 1,940 QOR on mainnet. Since a post-quantum signature cannot travel
//     with an Ethereum transaction, the authorisation travels ahead of it as a
//     bounded window opened on the Cosmos lane. See x/pqc/ante_evm_window.go.
//
// ORDER. The window decorator runs AFTER the mono decorator, because the mono
// decorator is what verifies the Ethereum signature and establishes the sender.
// Checking a window against an unverified sender would let anyone consume someone
// else's authorisation. The circuit breaker runs FIRST, because a message type that
// has been switched off should cost nothing to reject.
func newMonoEVMAnteHandler(options HandlerOptions, evmParams *evmtypes.Params, fmParams *feemarkettypes.Params) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
		evmante.NewEVMMonoDecorator(
			options.EVMAccountKeeper,
			options.FeeMarketKeeper,
			options.EvmKeeper,
			options.MaxTxGasWanted,
			evmParams,
			fmParams,
		),
		pqcmod.NewEVMWindowDecorator(options.PQCKeeper),
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
		// Mempool-only guard: refuse CosmWasm upload/instantiate into this
		// node's mempool. CheckTx only, so block validity is unchanged and
		// this can be rolled out per-node. Remove once governance has set
		// code_upload_access to Nobody.
		NewWasmMempoolGuardDecorator(),
		// Runs here, near the front, because the decorator that expands a multisig
		// signature into event bytes sits much further down (SetPubKey) and the
		// expansion is exponential in nesting depth. A guard placed after it would
		// be checking a node that had already run out of memory.
		NewMultisigDepthGuardDecorator(),
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
		NewValidatorLicenseDecorator(options.LicenseKeeper, options.StakingKeeper),
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
