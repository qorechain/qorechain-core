package amm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	ammcli "github.com/qorechain/qorechain-core/x/amm/client/cli"
	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
)

// ConsensusVersion of the AMM module.
const ConsensusVersion uint64 = 1

// AppModuleBasic implements module.AppModuleBasic.
type AppModuleBasic struct{}

func (AppModuleBasic) Name() string { return ammtypes.ModuleName }

func (AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}

func (AppModuleBasic) RegisterInterfaces(_ cdctypes.InterfaceRegistry) {
	// Proto-derived registrations land here once proto stubs are generated;
	// keep empty so the chain still boots from genesis with the JSON wire
	// types defined in x/amm/types.
}

func (AppModuleBasic) DefaultGenesis(_ codec.JSONCodec) json.RawMessage {
	bz, _ := json.Marshal(ammtypes.DefaultGenesisState())
	return bz
}

func (AppModuleBasic) ValidateGenesis(_ codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs ammtypes.GenesisState
	if err := json.Unmarshal(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", ammtypes.ModuleName, err)
	}
	return gs.Validate()
}

func (AppModuleBasic) RegisterGRPCGatewayRoutes(_ client.Context, _ *runtime.ServeMux) {}

func (AppModuleBasic) GetTxCmd() *cobra.Command    { return ammcli.GetTxCmd() }
func (AppModuleBasic) GetQueryCmd() *cobra.Command { return ammcli.GetQueryCmd() }

// AppModule implements module.AppModule (and the appmodule.AppModule shim).
type AppModule struct {
	AppModuleBasic
	keeper AMMKeeper
}

// NewAppModule creates an AMM AppModule wrapping the given keeper.
func NewAppModule(k AMMKeeper) AppModule {
	return AppModule{keeper: k}
}

// IsAppModule and IsOnePerModuleType are appmodule.AppModule markers.
func (AppModule) IsAppModule()        {}
func (AppModule) IsOnePerModuleType() {}

func (am AppModule) Name() string                                    { return ammtypes.ModuleName }
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry)      {}
func (am AppModule) QuerierRoute() string                            { return ammtypes.QuerierRoute }
func (am AppModule) RegisterServices(_ module.Configurator)          {}
func (am AppModule) ConsensusVersion() uint64                        { return ConsensusVersion }

// InitGenesis loads the AMM module state from genesis JSON.
func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, data json.RawMessage) {
	var gs ammtypes.GenesisState
	if err := json.Unmarshal(data, &gs); err != nil {
		panic(fmt.Sprintf("failed to unmarshal amm genesis state: %v", err))
	}
	if am.keeper != nil {
		am.keeper.InitGenesis(ctx, gs)
	}
}

// ExportGenesis serializes the AMM module state for genesis export.
func (am AppModule) ExportGenesis(ctx sdk.Context, _ codec.JSONCodec) json.RawMessage {
	var gs *ammtypes.GenesisState
	if am.keeper == nil {
		gs = ammtypes.DefaultGenesisState()
	} else {
		gs = am.keeper.ExportGenesis(ctx)
	}
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal amm genesis state: %v", err))
	}
	return bz
}

// EndBlock recomputes weighted-average prices for active pools.
// Implementation lives in the keeper to keep AppModule free of business logic.
func (am AppModule) EndBlock(goCtx context.Context) error {
	if am.keeper == nil {
		return nil
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	return am.keeper.EndBlock(ctx)
}

// HasGenesis interface guard.
var _ module.HasGenesis = AppModule{}

// HasABCIEndBlock interface guard.
var _ appmodule.HasEndBlocker = AppModule{}
