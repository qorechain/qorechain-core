//go:build !proprietary

package multilayer

import (
	"fmt"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/multilayer/types"
)

// StubKeeper implements MultilayerKeeper with no-op implementations
// for public (non-proprietary) builds.
// The multi-layer architecture requires the proprietary build for full functionality.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper creates a new stub multilayer keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) RegisterSidechain(_ sdk.Context, _ *types.MsgRegisterSidechain) (*types.MsgRegisterSidechainResponse, error) {
	return nil, fmt.Errorf("multi-layer architecture requires proprietary build")
}

func (k *StubKeeper) RegisterPaychain(_ sdk.Context, _ *types.MsgRegisterPaychain) (*types.MsgRegisterPaychainResponse, error) {
	return nil, fmt.Errorf("multi-layer architecture requires proprietary build")
}

func (k *StubKeeper) GetLayer(_ sdk.Context, _ string) (*types.LayerConfig, error) {
	return nil, nil
}

func (k *StubKeeper) GetAllLayers(_ sdk.Context) ([]*types.LayerConfig, error) {
	return nil, nil
}

func (k *StubKeeper) GetLayersByType(_ sdk.Context, _ types.LayerType) ([]*types.LayerConfig, error) {
	return nil, nil
}

func (k *StubKeeper) UpdateLayerStatus(_ sdk.Context, _ string, _ types.LayerStatus, _ string) error {
	return fmt.Errorf("multi-layer architecture requires proprietary build")
}

func (k *StubKeeper) AnchorState(_ sdk.Context, _ *types.MsgAnchorState) (*types.MsgAnchorStateResponse, error) {
	return nil, fmt.Errorf("multi-layer architecture requires proprietary build")
}

func (k *StubKeeper) GetLatestAnchor(_ sdk.Context, _ string) (*types.StateAnchor, error) {
	return nil, nil
}

func (k *StubKeeper) GetAnchors(_ sdk.Context, _ string) ([]*types.StateAnchor, error) {
	return nil, nil
}

func (k *StubKeeper) ChallengeAnchor(_ sdk.Context, _ *types.MsgChallengeAnchor) (*types.MsgChallengeAnchorResponse, error) {
	return nil, fmt.Errorf("multi-layer architecture requires proprietary build")
}

func (k *StubKeeper) RouteTransaction(_ sdk.Context, _ *types.MsgRouteTransaction) (*types.MsgRouteTransactionResponse, error) {
	// In stub mode, all transactions stay on main chain
	return &types.MsgRouteTransactionResponse{
		Decision: &types.RoutingDecision{
			SelectedLayer: "main",
			Reason:        "multi-layer routing requires proprietary build; defaulting to main chain",
		},
	}, nil
}

func (k *StubKeeper) SimulateRoute(_ sdk.Context, _ []byte, _ uint64, _ string) (*types.RoutingDecision, error) {
	return &types.RoutingDecision{
		SelectedLayer: "main",
		Reason:        "simulation requires proprietary build",
	}, nil
}

func (k *StubKeeper) GetRoutingStats(_ sdk.Context) (*types.QueryRoutingStatsResponse, error) {
	return &types.QueryRoutingStatsResponse{}, nil
}

func (k *StubKeeper) CalculateCrossLayerFee(_ sdk.Context, _ []string, _ uint64) (sdk.Coins, error) {
	return nil, nil
}

func (k *StubKeeper) GetParams(_ sdk.Context) types.Params {
	return types.DefaultParams()
}

func (k *StubKeeper) SetParams(_ sdk.Context, _ types.Params) error {
	return nil
}

func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}

func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
