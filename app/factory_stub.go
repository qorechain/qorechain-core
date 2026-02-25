//go:build !proprietary

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
	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
	svmmod "github.com/qorechain/qorechain-core/x/svm"
)

func init() {
	NewPQCClient = func() pqcmod.PQCClient {
		return pqcmod.NewStubPQCClient()
	}
	NewPQCKeeper = func(_ codec.Codec, _ storetypes.StoreKey, _ pqcmod.PQCClient, logger log.Logger) pqcmod.PQCKeeper {
		return pqcmod.NewStubKeeper(logger)
	}
	NewPQCAppModule = func(keeper pqcmod.PQCKeeper) module.AppModule {
		return pqcmod.NewAppModule(keeper)
	}
	NewPQCModuleBasic = func() module.AppModuleBasic {
		return pqcmod.AppModuleBasic{}
	}
	NewPQCVerifyDecorator = func(keeper pqcmod.PQCKeeper, client pqcmod.PQCClient) sdk.AnteDecorator {
		return pqcmod.NewPQCVerifyDecorator(keeper, client)
	}

	NewAIKeeper = func(_ codec.Codec, _ storetypes.StoreKey, logger log.Logger) aimod.AIKeeper {
		return aimod.NewStubKeeper(logger)
	}
	NewAIAppModule = func(keeper aimod.AIKeeper) module.AppModule {
		return aimod.NewAppModule(keeper)
	}
	NewAIModuleBasic = func() module.AppModuleBasic {
		return aimod.AppModuleBasic{}
	}
	NewAIAnomalyDecorator = func(keeper aimod.AIKeeper) sdk.AnteDecorator {
		return aimod.NewAIAnomalyDecorator(keeper)
	}

	NewBridgeKeeper = func(_ codec.Codec, _ storetypes.StoreKey, _ pqcmod.PQCKeeper, logger log.Logger) bridgemod.BridgeKeeper {
		return bridgemod.NewStubKeeper(logger)
	}
	NewBridgeAppModule = func(keeper bridgemod.BridgeKeeper) module.AppModule {
		return bridgemod.NewAppModule(keeper)
	}
	NewBridgeModuleBasic = func() module.AppModuleBasic {
		return bridgemod.AppModuleBasic{}
	}

	NewCrossVMKeeper = func(_ codec.Codec, _ storetypes.StoreKey, _ *evmkeeper.Keeper, _ *wasmkeeper.Keeper, logger log.Logger) crossvmmod.CrossVMKeeper {
		return crossvmmod.NewStubKeeper(logger)
	}
	NewCrossVMAppModule = func(keeper crossvmmod.CrossVMKeeper) module.AppModule {
		return crossvmmod.NewAppModule(keeper)
	}
	NewCrossVMModuleBasic = func() module.AppModuleBasic {
		return crossvmmod.AppModuleBasic{}
	}

	NewMultilayerKeeper = func(_ codec.Codec, _ storetypes.StoreKey, logger log.Logger) multilayermod.MultilayerKeeper {
		return multilayermod.NewStubKeeper(logger)
	}
	NewMultilayerAppModule = func(keeper multilayermod.MultilayerKeeper) module.AppModule {
		return multilayermod.NewAppModule(keeper)
	}
	NewMultilayerModuleBasic = func() module.AppModuleBasic {
		return multilayermod.AppModuleBasic{}
	}

	NewSVMKeeper = func(_ codec.Codec, _ storetypes.StoreKey, _ pqcmod.PQCKeeper,
		_ aimod.AIKeeper, _ crossvmmod.CrossVMKeeper, logger log.Logger) svmmod.SVMKeeper {
		return svmmod.NewStubKeeper(logger)
	}
	NewSVMAppModule = func(keeper svmmod.SVMKeeper) module.AppModule {
		return svmmod.NewAppModule(keeper)
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

	NewRLConsensusKeeper = func(_ codec.Codec, _ storetypes.StoreKey, logger log.Logger) rlconsensusmod.RLConsensusKeeper {
		return rlconsensusmod.NewStubKeeper(logger)
	}
	NewRLConsensusAppModule = func(keeper rlconsensusmod.RLConsensusKeeper) module.AppModule {
		return rlconsensusmod.NewAppModule(keeper)
	}
	NewRLConsensusModuleBasic = func() module.AppModuleBasic {
		return rlconsensusmod.AppModuleBasic{}
	}
}
