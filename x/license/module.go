package license

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/qorechain/qorechain-core/x/license/client/cli"
	"github.com/qorechain/qorechain-core/x/license/types"
)

var (
	_ module.AppModuleBasic   = AppModuleBasic{}
	_ module.HasGenesis       = AppModule{}
	_ module.HasServices      = AppModule{}
	_ appmodule.AppModule     = AppModule{}
	_ appmodule.HasEndBlocker = AppModule{}
)

// RegisterServices registers the license Msg service so grant/revoke/suspend/
// resume transactions route to the keeper.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), NewMsgServer(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), NewQueryServer(am.keeper))
}

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string                                  { return types.ModuleName }
func (AppModuleBasic) RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {}
func (AppModuleBasic) RegisterInterfaces(reg codectypes.InterfaceRegistry) {
	reg.RegisterImplementations((*sdk.Msg)(nil),
		&types.MsgGrantLicense{},
		&types.MsgRevokeLicense{},
		&types.MsgSuspendLicense{},
		&types.MsgResumeLicense{},
	)
}
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

func (AppModuleBasic) DefaultGenesis(_ codec.JSONCodec) json.RawMessage {
	gs := types.DefaultGenesisState()
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(err)
	}
	return bz
}

func (AppModuleBasic) ValidateGenesis(_ codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var gs types.GenesisState
	if err := json.Unmarshal(bz, &gs); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return gs.Validate()
}

func (AppModuleBasic) GetTxCmd() *cobra.Command    { return cli.GetTxCmd() }
func (AppModuleBasic) GetQueryCmd() *cobra.Command { return cli.GetQueryCmd() }

type AppModule struct {
	AppModuleBasic
	keeper LicenseKeeper
}

func NewAppModule(k LicenseKeeper) AppModule {
	return AppModule{AppModuleBasic: AppModuleBasic{}, keeper: k}
}

func (AppModule) IsOnePerModuleType()      {}
func (AppModule) IsAppModule()             {}
func (AppModule) ConsensusVersion() uint64 { return 1 }

func (am AppModule) InitGenesis(ctx sdk.Context, _ codec.JSONCodec, data json.RawMessage) {
	var gs types.GenesisState
	if err := json.Unmarshal(data, &gs); err != nil {
		panic(fmt.Sprintf("failed to unmarshal license genesis state: %v", err))
	}
	am.keeper.InitGenesis(ctx, gs)
}

func (am AppModule) ExportGenesis(ctx sdk.Context, _ codec.JSONCodec) json.RawMessage {
	gs := am.keeper.ExportGenesis(ctx)
	bz, err := json.Marshal(gs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal license genesis state: %v", err))
	}
	return bz
}

func (am AppModule) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return am.keeper.EndBlocker(sdkCtx)
}
