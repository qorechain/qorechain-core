package app

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	pqcmod "github.com/qorechain/qorechain-core/x/pqc"
	aimod "github.com/qorechain/qorechain-core/x/ai"
	bridgemod "github.com/qorechain/qorechain-core/x/bridge"
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
)
