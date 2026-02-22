//go:build proprietary

package multilayer

import (
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/qorechain/qorechain-core/x/multilayer/keeper"
	"github.com/qorechain/qorechain-core/x/multilayer/types"
)

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the MultilayerKeeper interface.
type keeperAdapter struct {
	k keeper.Keeper
}

func (a *keeperAdapter) RegisterSidechain(ctx sdk.Context, msg *types.MsgRegisterSidechain) (*types.MsgRegisterSidechainResponse, error) {
	return a.k.RegisterSidechain(ctx, msg)
}
func (a *keeperAdapter) RegisterPaychain(ctx sdk.Context, msg *types.MsgRegisterPaychain) (*types.MsgRegisterPaychainResponse, error) {
	return a.k.RegisterPaychain(ctx, msg)
}
func (a *keeperAdapter) GetLayer(ctx sdk.Context, layerID string) (*types.LayerConfig, error) {
	return a.k.GetLayer(ctx, layerID)
}
func (a *keeperAdapter) GetAllLayers(ctx sdk.Context) ([]*types.LayerConfig, error) {
	return a.k.GetAllLayers(ctx)
}
func (a *keeperAdapter) GetLayersByType(ctx sdk.Context, layerType types.LayerType) ([]*types.LayerConfig, error) {
	return a.k.GetLayersByType(ctx, layerType)
}
func (a *keeperAdapter) UpdateLayerStatus(ctx sdk.Context, layerID string, status types.LayerStatus, reason string) error {
	return a.k.UpdateLayerStatus(ctx, layerID, status, reason)
}
func (a *keeperAdapter) AnchorState(ctx sdk.Context, msg *types.MsgAnchorState) (*types.MsgAnchorStateResponse, error) {
	return a.k.AnchorState(ctx, msg)
}
func (a *keeperAdapter) GetLatestAnchor(ctx sdk.Context, layerID string) (*types.StateAnchor, error) {
	return a.k.GetLatestAnchor(ctx, layerID)
}
func (a *keeperAdapter) GetAnchors(ctx sdk.Context, layerID string) ([]*types.StateAnchor, error) {
	return a.k.GetAnchors(ctx, layerID)
}
func (a *keeperAdapter) ChallengeAnchor(ctx sdk.Context, msg *types.MsgChallengeAnchor) (*types.MsgChallengeAnchorResponse, error) {
	return a.k.ChallengeAnchor(ctx, msg)
}
func (a *keeperAdapter) RouteTransaction(ctx sdk.Context, msg *types.MsgRouteTransaction) (*types.MsgRouteTransactionResponse, error) {
	return a.k.RouteTransaction(ctx, msg)
}
func (a *keeperAdapter) SimulateRoute(ctx sdk.Context, payload []byte, maxLatency uint64, maxFee string) (*types.RoutingDecision, error) {
	return a.k.SimulateRoute(ctx, payload, maxLatency, maxFee)
}
func (a *keeperAdapter) GetRoutingStats(ctx sdk.Context) (*types.QueryRoutingStatsResponse, error) {
	return a.k.GetRoutingStats(ctx)
}
func (a *keeperAdapter) CalculateCrossLayerFee(ctx sdk.Context, sourceLayers []string, totalGas uint64) (sdk.Coins, error) {
	return a.k.CalculateCrossLayerFee(ctx, sourceLayers, totalGas)
}
func (a *keeperAdapter) GetParams(ctx sdk.Context) types.Params {
	return a.k.GetParams(ctx)
}
func (a *keeperAdapter) SetParams(ctx sdk.Context, params types.Params) error {
	return a.k.SetParams(ctx, params)
}
func (a *keeperAdapter) InitGenesis(ctx sdk.Context, state types.GenesisState) {
	a.k.InitGenesis(ctx, state)
}
func (a *keeperAdapter) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return a.k.ExportGenesis(ctx)
}

// RealNewMultilayerKeeper creates the real multilayer keeper for proprietary builds.
func RealNewMultilayerKeeper(cdc codec.Codec, storeKey storetypes.StoreKey, logger log.Logger) MultilayerKeeper {
	k := keeper.NewKeeper(cdc, storeKey, logger)
	return &keeperAdapter{k: k}
}

// RealNewAppModule creates the real multilayer AppModule for proprietary builds.
func RealNewAppModule(k MultilayerKeeper) module.AppModule {
	return NewAppModule(k)
}

// RealNewModuleBasic creates the real multilayer AppModuleBasic.
func RealNewModuleBasic() module.AppModuleBasic {
	return AppModuleBasic{}
}
