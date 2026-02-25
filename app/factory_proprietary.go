//go:build proprietary

package app

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	evmkeeper "github.com/cosmos/evm/x/vm/keeper"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	aimod "github.com/qorechain/qorechain-core/x/ai"
	bridgemod "github.com/qorechain/qorechain-core/x/bridge"
	burnmod "github.com/qorechain/qorechain-core/x/burn"
	crossvmmod "github.com/qorechain/qorechain-core/x/crossvm"
	inflationmod "github.com/qorechain/qorechain-core/x/inflation"
	multilayermod "github.com/qorechain/qorechain-core/x/multilayer"
	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
	svmmod "github.com/qorechain/qorechain-core/x/svm"
	xqoremod "github.com/qorechain/qorechain-core/x/xqore"
)

func init() {
	// PQC factories — use real FFI-backed implementations
	NewPQCClient = func() pqcmod.PQCClient {
		return pqcmod.RealNewPQCClient()
	}
	NewPQCKeeper = func(cdc codec.Codec, storeKey storetypes.StoreKey, client pqcmod.PQCClient, logger log.Logger) pqcmod.PQCKeeper {
		return pqcmod.RealNewPQCKeeper(cdc, storeKey, client, logger)
	}
	NewPQCAppModule = func(keeper pqcmod.PQCKeeper) module.AppModule {
		return pqcmod.RealNewAppModule(keeper)
	}
	NewPQCModuleBasic = func() module.AppModuleBasic {
		return pqcmod.AppModuleBasic{}
	}
	NewPQCVerifyDecorator = func(keeper pqcmod.PQCKeeper, client pqcmod.PQCClient) sdk.AnteDecorator {
		return pqcmod.RealNewPQCVerifyDecorator(keeper, client)
	}
	NewPQCHybridVerifyDecorator = func(keeper pqcmod.PQCKeeper, client pqcmod.PQCClient) sdk.AnteDecorator {
		return pqcmod.RealNewPQCHybridVerifyDecorator(keeper, client)
	}

	// AI factories — use real heuristic engine implementations
	NewAIKeeper = func(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) aimod.AIKeeper {
		return aimod.RealNewAIKeeper(cdc, storeKey, logger)
	}
	NewAIAppModule = func(keeper aimod.AIKeeper) module.AppModule {
		return aimod.RealNewAppModule(keeper)
	}
	NewAIModuleBasic = func() module.AppModuleBasic {
		return aimod.AppModuleBasic{}
	}
	NewAIAnomalyDecorator = func(keeper aimod.AIKeeper) sdk.AnteDecorator {
		return aimod.RealNewAIAnomalyDecorator(keeper)
	}

	// Bridge factories — use real multi-protocol bridge implementations
	NewBridgeKeeper = func(cdc codec.Codec, storeKey storetypes.StoreKey, pqcKeeper pqcmod.PQCKeeper, logger log.Logger) bridgemod.BridgeKeeper {
		return bridgemod.RealNewBridgeKeeper(cdc, storeKey, pqcKeeper, logger)
	}
	NewBridgeAppModule = func(keeper bridgemod.BridgeKeeper) module.AppModule {
		return bridgemod.RealNewAppModule(keeper)
	}
	NewBridgeModuleBasic = func() module.AppModuleBasic {
		return bridgemod.AppModuleBasic{}
	}

	// CrossVM factories — use real cross-VM bridge implementations
	NewCrossVMKeeper = func(cdc codec.Codec, storeKey storetypes.StoreKey, evmKeeper *evmkeeper.Keeper, wasmKeeper *wasmkeeper.Keeper, logger log.Logger) crossvmmod.CrossVMKeeper {
		return crossvmmod.RealNewCrossVMKeeper(cdc, storeKey, evmKeeper, wasmKeeper, logger)
	}
	NewCrossVMAppModule = func(keeper crossvmmod.CrossVMKeeper) module.AppModule {
		return crossvmmod.RealNewAppModule(keeper)
	}
	NewCrossVMModuleBasic = func() module.AppModuleBasic {
		return crossvmmod.AppModuleBasic{}
	}

	// Multilayer factories — use real multi-layer architecture implementations
	NewMultilayerKeeper = func(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) multilayermod.MultilayerKeeper {
		return multilayermod.RealNewMultilayerKeeper(cdc, storeKey, logger)
	}
	NewMultilayerAppModule = func(keeper multilayermod.MultilayerKeeper) module.AppModule {
		return multilayermod.RealNewAppModule(keeper)
	}
	NewMultilayerModuleBasic = func() module.AppModuleBasic {
		return multilayermod.RealNewModuleBasic()
	}

	// SVM factories — use real BPF executor-backed implementations
	NewSVMKeeper = func(cdc codec.Codec, storeKey storetypes.StoreKey,
		pqcKeeper pqcmod.PQCKeeper, aiKeeper aimod.AIKeeper,
		crossvmKeeper crossvmmod.CrossVMKeeper,
		logger log.Logger) svmmod.SVMKeeper {
		return svmmod.RealNewSVMKeeper(cdc, storeKey, pqcKeeper, aiKeeper, crossvmKeeper, logger)
	}
	NewSVMAppModule = func(keeper svmmod.SVMKeeper) module.AppModule {
		return svmmod.RealNewAppModule(keeper)
	}
	NewSVMModuleBasic = func() module.AppModuleBasic {
		return svmmod.AppModuleBasic{}
	}
	NewSVMComputeBudgetDecorator = func(keeper svmmod.SVMKeeper) sdk.AnteDecorator {
		return svmmod.NewSVMComputeBudgetDecorator(keeper)
	}
	NewSVMDeductFeeDecorator = func(keeper svmmod.SVMKeeper) sdk.AnteDecorator {
		return svmmod.NewSVMDeductFeeDecorator(keeper)
	}

	// RL Consensus factories — use real PPO-based implementations
	NewRLConsensusKeeper = func(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) rlconsensusmod.RLConsensusKeeper {
		return rlconsensusmod.RealNewRLConsensusKeeper(cdc, storeKey, logger)
	}
	NewRLConsensusAppModule = func(keeper rlconsensusmod.RLConsensusKeeper) module.AppModule {
		return rlconsensusmod.RealNewAppModule(keeper)
	}
	NewRLConsensusModuleBasic = func() module.AppModuleBasic {
		return rlconsensusmod.AppModuleBasic{}
	}

	// Burn factories — use real burn keeper
	NewBurnKeeper = func(cdc codec.Codec, storeKey storetypes.StoreKey, bk bankkeeper.BaseKeeper, logger log.Logger) burnmod.BurnKeeper {
		return burnmod.RealNewBurnKeeper(cdc, storeKey, bk, logger)
	}
	NewBurnAppModule = func(keeper burnmod.BurnKeeper) module.AppModule {
		return burnmod.RealNewAppModule(keeper)
	}
	NewBurnModuleBasic = func() module.AppModuleBasic {
		return burnmod.AppModuleBasic{}
	}

	// xQORE factories — use real xQORE keeper
	NewXQOREKeeper = func(cdc codec.Codec, storeKey storetypes.StoreKey, bk bankkeeper.BaseKeeper, logger log.Logger) xqoremod.XQOREKeeper {
		return xqoremod.RealNewXQOREKeeper(cdc, storeKey, bk, logger)
	}
	NewXQOREAppModule = func(keeper xqoremod.XQOREKeeper) module.AppModule {
		return xqoremod.RealNewAppModule(keeper)
	}
	NewXQOREModuleBasic = func() module.AppModuleBasic {
		return xqoremod.AppModuleBasic{}
	}

	// Inflation factories — use real inflation keeper
	NewInflationKeeper = func(cdc codec.Codec, storeKey storetypes.StoreKey, bk bankkeeper.BaseKeeper, logger log.Logger) inflationmod.InflationKeeper {
		return inflationmod.RealNewInflationKeeper(cdc, storeKey, bk, logger)
	}
	NewInflationAppModule = func(keeper inflationmod.InflationKeeper) module.AppModule {
		return inflationmod.RealNewAppModule(keeper)
	}
	NewInflationModuleBasic = func() module.AppModuleBasic {
		return inflationmod.AppModuleBasic{}
	}
}
