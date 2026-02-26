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

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	abstractaccountmod "github.com/qorechain/qorechain-core/x/abstractaccount"
	aimod "github.com/qorechain/qorechain-core/x/ai"
	babylonmod "github.com/qorechain/qorechain-core/x/babylon"
	bridgemod "github.com/qorechain/qorechain-core/x/bridge"
	burnmod "github.com/qorechain/qorechain-core/x/burn"
	crossvmmod "github.com/qorechain/qorechain-core/x/crossvm"
	fairblockmod "github.com/qorechain/qorechain-core/x/fairblock"
	gasabstractionmod "github.com/qorechain/qorechain-core/x/gasabstraction"
	rdkmod "github.com/qorechain/qorechain-core/x/rdk"
	inflationmod "github.com/qorechain/qorechain-core/x/inflation"
	multilayermod "github.com/qorechain/qorechain-core/x/multilayer"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
	svmmod "github.com/qorechain/qorechain-core/x/svm"
	xqoremod "github.com/qorechain/qorechain-core/x/xqore"
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
	NewPQCHybridVerifyDecorator = func(keeper pqcmod.PQCKeeper, client pqcmod.PQCClient) sdk.AnteDecorator {
		return pqcmod.NewPQCHybridVerifyDecorator(keeper, client)
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

	NewBridgeKeeper = func(_ codec.Codec, _ storetypes.StoreKey, _ pqcmod.PQCKeeper, _ burnmod.BurnKeeper, logger log.Logger) bridgemod.BridgeKeeper {
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

	NewBurnKeeper = func(_ codec.Codec, _ storetypes.StoreKey, _ bankkeeper.BaseKeeper, logger log.Logger) burnmod.BurnKeeper {
		return burnmod.NewStubKeeper(logger)
	}
	NewBurnAppModule = func(keeper burnmod.BurnKeeper) module.AppModule {
		return burnmod.NewAppModule(keeper)
	}
	NewBurnModuleBasic = func() module.AppModuleBasic {
		return burnmod.AppModuleBasic{}
	}

	NewXQOREKeeper = func(_ codec.Codec, _ storetypes.StoreKey, _ bankkeeper.BaseKeeper, logger log.Logger) xqoremod.XQOREKeeper {
		return xqoremod.NewStubKeeper(logger)
	}
	NewXQOREAppModule = func(keeper xqoremod.XQOREKeeper) module.AppModule {
		return xqoremod.NewAppModule(keeper)
	}
	NewXQOREModuleBasic = func() module.AppModuleBasic {
		return xqoremod.AppModuleBasic{}
	}

	NewInflationKeeper = func(_ codec.Codec, _ storetypes.StoreKey, _ bankkeeper.BaseKeeper, logger log.Logger) inflationmod.InflationKeeper {
		return inflationmod.NewStubKeeper(logger)
	}
	NewInflationAppModule = func(keeper inflationmod.InflationKeeper) module.AppModule {
		return inflationmod.NewAppModule(keeper)
	}
	NewInflationModuleBasic = func() module.AppModuleBasic {
		return inflationmod.AppModuleBasic{}
	}

	// Babylon — stub factories
	NewBabylonKeeper = func(_ codec.Codec, _ storetypes.StoreKey, logger log.Logger) babylonmod.BabylonKeeper {
		return babylonmod.NewStubKeeper(logger)
	}
	NewBabylonAppModule = func(keeper babylonmod.BabylonKeeper) module.AppModule {
		return babylonmod.NewAppModule(keeper)
	}
	NewBabylonModuleBasic = func() module.AppModuleBasic {
		return babylonmod.AppModuleBasic{}
	}

	// AbstractAccount — stub factories
	NewAbstractAccountKeeper = func(_ codec.Codec, _ storetypes.StoreKey, logger log.Logger) abstractaccountmod.AbstractAccountKeeper {
		return abstractaccountmod.NewStubKeeper(logger)
	}
	NewAbstractAccountAppModule = func(keeper abstractaccountmod.AbstractAccountKeeper) module.AppModule {
		return abstractaccountmod.NewAppModule(keeper)
	}
	NewAbstractAccountModuleBasic = func() module.AppModuleBasic {
		return abstractaccountmod.AppModuleBasic{}
	}

	// FairBlock — stub factories
	NewFairBlockKeeper = func(_ codec.Codec, _ storetypes.StoreKey, logger log.Logger) fairblockmod.FairBlockKeeper {
		return fairblockmod.NewStubKeeper(logger)
	}
	NewFairBlockAppModule = func(keeper fairblockmod.FairBlockKeeper) module.AppModule {
		return fairblockmod.NewAppModule(keeper)
	}
	NewFairBlockModuleBasic = func() module.AppModuleBasic {
		return fairblockmod.AppModuleBasic{}
	}
	NewFairBlockDecorator = func(keeper fairblockmod.FairBlockKeeper) sdk.AnteDecorator {
		return fairblockmod.NewFairBlockDecorator(keeper)
	}

	// GasAbstraction — stub factories
	NewGasAbstractionKeeper = func(_ codec.Codec, _ storetypes.StoreKey, logger log.Logger) gasabstractionmod.GasAbstractionKeeper {
		return gasabstractionmod.NewStubKeeper(logger)
	}
	NewGasAbstractionAppModule = func(keeper gasabstractionmod.GasAbstractionKeeper) module.AppModule {
		return gasabstractionmod.NewAppModule(keeper)
	}
	NewGasAbstractionModuleBasic = func() module.AppModuleBasic {
		return gasabstractionmod.AppModuleBasic{}
	}
	NewGasAbstractionDecorator = func(keeper gasabstractionmod.GasAbstractionKeeper) sdk.AnteDecorator {
		return gasabstractionmod.NewGasAbstractionDecorator(keeper)
	}

	// RDK — stub factories
	NewRDKKeeper = func(_ codec.Codec, _ storetypes.StoreKey, _ burnmod.BurnKeeper, _ multilayermod.MultilayerKeeper, _ rlconsensusmod.RLConsensusKeeper, _ bankkeeper.BaseKeeper, logger log.Logger) rdkmod.RDKKeeper {
		return rdkmod.NewStubKeeper(logger)
	}
	NewRDKAppModule = func(keeper rdkmod.RDKKeeper) module.AppModule {
		return rdkmod.NewAppModule(keeper)
	}
	NewRDKModuleBasic = func() module.AppModuleBasic {
		return rdkmod.AppModuleBasic{}
	}
}
