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

	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	aimod "github.com/qorechain/qorechain-core/x/ai"
	bridgemod "github.com/qorechain/qorechain-core/x/bridge"
	crossvmmod "github.com/qorechain/qorechain-core/x/crossvm"
	multilayermod "github.com/qorechain/qorechain-core/x/multilayer"
	svmmod "github.com/qorechain/qorechain-core/x/svm"
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
}
