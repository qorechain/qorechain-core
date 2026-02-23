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
)

// Module factory function variables.
// In public builds (!proprietary), these are set to stub factories by factory_stub.go.
// In proprietary builds, these are overridden by register.go files in each module.
var (
	// PQC module factories
	NewPQCClient          func() pqcmod.PQCClient
	NewPQCKeeper          func(cdc codec.Codec, storeKey storetypes.StoreKey, client pqcmod.PQCClient, logger log.Logger) pqcmod.PQCKeeper
	NewPQCAppModule       func(keeper pqcmod.PQCKeeper) module.AppModule
	NewPQCModuleBasic     func() module.AppModuleBasic
	NewPQCVerifyDecorator func(keeper pqcmod.PQCKeeper, client pqcmod.PQCClient) sdk.AnteDecorator

	// AI module factories
	NewAIKeeper          func(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) aimod.AIKeeper
	NewAIAppModule       func(keeper aimod.AIKeeper) module.AppModule
	NewAIModuleBasic     func() module.AppModuleBasic
	NewAIAnomalyDecorator func(keeper aimod.AIKeeper) sdk.AnteDecorator

	// Bridge module factories
	NewBridgeKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, pqcKeeper pqcmod.PQCKeeper, logger log.Logger) bridgemod.BridgeKeeper
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
)
