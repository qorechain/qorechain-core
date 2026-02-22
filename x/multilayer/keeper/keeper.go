//go:build proprietary

package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/multilayer/types"
)

// Keeper manages the x/multilayer module state for the QoreChain multi-layer architecture.
// It provides layer registry, HCS state anchoring, QCAI transaction routing,
// and cross-layer fee bundling (CLFB).
type Keeper struct {
	cdc      codec.Codec
	storeKey storetypes.StoreKey
	logger   log.Logger
	router   *HeuristicRouter
}

// NewKeeper creates a new multilayer keeper for the QoreChain multi-layer architecture.
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	logger log.Logger,
) Keeper {
	return Keeper{
		cdc:      cdc,
		storeKey: storeKey,
		logger:   logger.With("module", types.ModuleName),
		router:   NewHeuristicRouter(),
	}
}

// Logger returns the module logger.
func (k Keeper) Logger() log.Logger {
	return k.logger
}

// ---- Parameters ----

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams stores the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)
	return nil
}

// ---- Routing Statistics ----

// RoutingStats tracks aggregate QCAI routing statistics
type RoutingStats struct {
	TotalRouted          uint64 `json:"total_routed"`
	RoutedToMain         uint64 `json:"routed_to_main"`
	RoutedToSidechains   uint64 `json:"routed_to_sidechains"`
	RoutedToPaychains    uint64 `json:"routed_to_paychains"`
	TotalGasSavings      uint64 `json:"total_gas_savings"`
	TotalLatencyImprovement uint64 `json:"total_latency_improvement"`
}

// getRoutingStats returns the stored routing statistics.
func (k Keeper) getRoutingStats(ctx sdk.Context) RoutingStats {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.RoutingStatsKeyPrefix)
	if bz == nil {
		return RoutingStats{}
	}
	var stats RoutingStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return RoutingStats{}
	}
	return stats
}

// setRoutingStats stores the routing statistics.
func (k Keeper) setRoutingStats(ctx sdk.Context, stats RoutingStats) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	store.Set(types.RoutingStatsKeyPrefix, bz)
	return nil
}

// GetRoutingStats returns the QCAI routing statistics as a query response.
func (k Keeper) GetRoutingStats(ctx sdk.Context) (*types.QueryRoutingStatsResponse, error) {
	stats := k.getRoutingStats(ctx)
	var avgGas, avgLatency string
	if stats.TotalRouted > 0 {
		avgGas = formatPercent(float64(stats.TotalGasSavings) / float64(stats.TotalRouted))
		avgLatency = formatPercent(float64(stats.TotalLatencyImprovement) / float64(stats.TotalRouted))
	} else {
		avgGas = "0.0"
		avgLatency = "0.0"
	}
	return &types.QueryRoutingStatsResponse{
		TotalRouted:                      stats.TotalRouted,
		RoutedToMain:                     stats.RoutedToMain,
		RoutedToSidechains:               stats.RoutedToSidechains,
		RoutedToPaychains:                stats.RoutedToPaychains,
		AverageGasSavingsPercent:         avgGas,
		AverageLatencyImprovementPercent: avgLatency,
	}, nil
}

func formatPercent(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
