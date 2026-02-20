//go:build !proprietary

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
}
