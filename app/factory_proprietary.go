//go:build proprietary

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
}
