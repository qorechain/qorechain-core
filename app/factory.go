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
	ammmod "github.com/qorechain/qorechain-core/x/amm"
	babylonmod "github.com/qorechain/qorechain-core/x/babylon"
	bridgemod "github.com/qorechain/qorechain-core/x/bridge"
	burnmod "github.com/qorechain/qorechain-core/x/burn"
	crossvmmod "github.com/qorechain/qorechain-core/x/crossvm"
	fairblockmod "github.com/qorechain/qorechain-core/x/fairblock"
	gasabstractionmod "github.com/qorechain/qorechain-core/x/gasabstraction"
	inflationmod "github.com/qorechain/qorechain-core/x/inflation"
	licensemod "github.com/qorechain/qorechain-core/x/license"
	lightnodemod "github.com/qorechain/qorechain-core/x/lightnode"
	multilayermod "github.com/qorechain/qorechain-core/x/multilayer"
	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	rdkmod "github.com/qorechain/qorechain-core/x/rdk"
	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
	svmmod "github.com/qorechain/qorechain-core/x/svm"
	xqoremod "github.com/qorechain/qorechain-core/x/xqore"
)

// Module factory function variables.
// In public builds (!full), these are set to stub factories by factory_stub.go.
// In full builds, these are overridden by register.go files in each module.
var (
	// PQC module factories
	NewPQCClient                func() pqcmod.PQCClient
	NewPQCKeeper                func(cdc codec.Codec, storeKey storetypes.StoreKey, client pqcmod.PQCClient, logger log.Logger) pqcmod.PQCKeeper
	NewPQCAppModule             func(keeper pqcmod.PQCKeeper) module.AppModule
	NewPQCModuleBasic           func() module.AppModuleBasic
	NewPQCVerifyDecorator       func(keeper pqcmod.PQCKeeper, client pqcmod.PQCClient) sdk.AnteDecorator
	NewPQCHybridVerifyDecorator func(keeper pqcmod.PQCKeeper, client pqcmod.PQCClient) sdk.AnteDecorator
	NewPQCReplayGuardDecorator  func() sdk.AnteDecorator

	// AI module factories
	NewAIKeeper           func(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) aimod.AIKeeper
	NewAIAppModule        func(keeper aimod.AIKeeper) module.AppModule
	NewAIModuleBasic      func() module.AppModuleBasic
	NewAIAnomalyDecorator func(keeper aimod.AIKeeper) sdk.AnteDecorator

	// Bridge module factories
	NewBridgeKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, pqcKeeper pqcmod.PQCKeeper, burnKeeper burnmod.BurnKeeper, logger log.Logger, authority string) bridgemod.BridgeKeeper
	NewBridgeAppModule   func(keeper bridgemod.BridgeKeeper) module.AppModule
	NewBridgeModuleBasic func() module.AppModuleBasic

	// CrossVM module factories
	NewCrossVMKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, evmKeeper *evmkeeper.Keeper, wasmKeeper *wasmkeeper.Keeper, logger log.Logger) crossvmmod.CrossVMKeeper
	NewCrossVMAppModule   func(keeper crossvmmod.CrossVMKeeper) module.AppModule
	NewCrossVMModuleBasic func() module.AppModuleBasic

	// Multilayer module factories
	NewMultilayerKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) multilayermod.MultilayerKeeper
	NewMultilayerAppModule   func(keeper multilayermod.MultilayerKeeper) module.AppModule
	NewMultilayerModuleBasic func() module.AppModuleBasic

	// SVM module factories
	NewSVMKeeper func(cdc codec.Codec, storeKey storetypes.StoreKey,
		pqcKeeper pqcmod.PQCKeeper, aiKeeper aimod.AIKeeper,
		crossvmKeeper crossvmmod.CrossVMKeeper,
		logger log.Logger) svmmod.SVMKeeper
	NewSVMAppModule              func(keeper svmmod.SVMKeeper) module.AppModule
	NewSVMModuleBasic            func() module.AppModuleBasic
	NewSVMComputeBudgetDecorator func(keeper svmmod.SVMKeeper) sdk.AnteDecorator
	NewSVMDeductFeeDecorator     func(keeper svmmod.SVMKeeper, bankKeeper svmmod.SVMBankKeeper) sdk.AnteDecorator

	// RL Consensus module factories
	NewRLConsensusKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) rlconsensusmod.RLConsensusKeeper
	NewRLConsensusAppModule   func(keeper rlconsensusmod.RLConsensusKeeper) module.AppModule
	NewRLConsensusModuleBasic func() module.AppModuleBasic

	// Burn module factories
	NewBurnKeeper           func(cdc codec.Codec, storeKey storetypes.StoreKey, bankKeeper bankkeeper.BaseKeeper, logger log.Logger) burnmod.BurnKeeper
	NewBurnAppModule        func(keeper burnmod.BurnKeeper) module.AppModule
	NewBurnModuleBasic      func() module.AppModuleBasic
	NewBurnTxCountDecorator func(keeper burnmod.BurnKeeper) sdk.AnteDecorator

	// xQORE module factories
	NewXQOREKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, bankKeeper bankkeeper.BaseKeeper, logger log.Logger) xqoremod.XQOREKeeper
	NewXQOREAppModule   func(keeper xqoremod.XQOREKeeper) module.AppModule
	NewXQOREModuleBasic func() module.AppModuleBasic

	// Inflation module factories
	NewInflationKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, bankKeeper bankkeeper.BaseKeeper, logger log.Logger) inflationmod.InflationKeeper
	NewInflationAppModule   func(keeper inflationmod.InflationKeeper) module.AppModule
	NewInflationModuleBasic func() module.AppModuleBasic

	// Babylon module factories (v1.2.0 — BTC restaking)
	NewBabylonKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) babylonmod.BabylonKeeper
	NewBabylonAppModule   func(keeper babylonmod.BabylonKeeper) module.AppModule
	NewBabylonModuleBasic func() module.AppModuleBasic

	// AbstractAccount module factories (v1.2.0 — account abstraction)
	NewAbstractAccountKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) abstractaccountmod.AbstractAccountKeeper
	NewAbstractAccountAppModule   func(keeper abstractaccountmod.AbstractAccountKeeper) module.AppModule
	NewAbstractAccountModuleBasic func() module.AppModuleBasic

	// FairBlock module factories (v1.2.0 — threshold IBE)
	NewFairBlockKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) fairblockmod.FairBlockKeeper
	NewFairBlockAppModule   func(keeper fairblockmod.FairBlockKeeper) module.AppModule
	NewFairBlockModuleBasic func() module.AppModuleBasic
	NewFairBlockDecorator   func(keeper fairblockmod.FairBlockKeeper) sdk.AnteDecorator

	// GasAbstraction module factories (v1.2.0 — IBC token fees)
	NewGasAbstractionKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) gasabstractionmod.GasAbstractionKeeper
	NewGasAbstractionAppModule   func(keeper gasabstractionmod.GasAbstractionKeeper) module.AppModule
	NewGasAbstractionModuleBasic func() module.AppModuleBasic
	NewGasAbstractionDecorator   func(keeper gasabstractionmod.GasAbstractionKeeper) sdk.AnteDecorator

	// RDK module factories (v1.3.0 — Rollup Development Kit)
	NewRDKKeeper func(
		cdc codec.Codec,
		storeKey storetypes.StoreKey,
		burnKeeper burnmod.BurnKeeper,
		multilayerKeeper multilayermod.MultilayerKeeper,
		rlKeeper rlconsensusmod.RLConsensusKeeper,
		bankKeeper bankkeeper.BaseKeeper,
		logger log.Logger,
	) rdkmod.RDKKeeper
	NewRDKAppModule   func(keeper rdkmod.RDKKeeper) module.AppModule
	NewRDKModuleBasic func() module.AppModuleBasic

	// LightNode module factories (v1.15.0 — light node registration + rewards)
	NewLightNodeKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, bankKeeper bankkeeper.BaseKeeper, logger log.Logger) lightnodemod.LightNodeKeeper
	NewLightNodeAppModule   func(keeper lightnodemod.LightNodeKeeper) module.AppModule
	NewLightNodeModuleBasic func() module.AppModuleBasic

	// License module factories (v1.4.0 — validator bridge & multi-chain)
	NewLicenseKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, authority string, logger log.Logger) licensemod.LicenseKeeper
	NewLicenseAppModule   func(keeper licensemod.LicenseKeeper) module.AppModule
	NewLicenseModuleBasic func() module.AppModuleBasic

	// AMM module factories (v3.0.0 — native automated market maker)
	NewAMMKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, bankKeeper bankkeeper.BaseKeeper, logger log.Logger) ammmod.AMMKeeper
	NewAMMAppModule   func(keeper ammmod.AMMKeeper) module.AppModule
	NewAMMModuleBasic func() module.AppModuleBasic
)
