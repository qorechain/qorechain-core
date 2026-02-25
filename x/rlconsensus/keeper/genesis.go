//go:build proprietary

package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// InitGenesis initializes the rlconsensus module state from a genesis state.
func (k *Keeper) InitGenesis(ctx sdk.Context, gs types.GenesisState) {
	if err := k.SetParams(ctx, gs.Params); err != nil {
		panic(fmt.Sprintf("failed to set rlconsensus params: %v", err))
	}

	if err := k.SetAgentStatus(ctx, gs.AgentStatus); err != nil {
		panic(fmt.Sprintf("failed to set rlconsensus agent status: %v", err))
	}

	if gs.PolicyWeights != nil {
		if err := k.SetPolicyWeights(ctx, gs.PolicyWeights); err != nil {
			panic(fmt.Sprintf("failed to set rlconsensus policy weights: %v", err))
		}
		k.agent = NewPPOAgent(gs.PolicyWeights)
	}

	// Set default applied params based on genesis params
	defaultApplied := AppliedConsensusParams{
		BlockTimeMs:    gs.Params.DefaultBlockTimeMs,
		GasLimit:       0,
		GasPriceFloor:  gs.Params.DefaultBaseGasPrice,
		PoolWeightRPoS: "0.400000000000000000",
		PoolWeightDPoS: "0.350000000000000000",
	}
	if err := k.SetAppliedParams(ctx, defaultApplied); err != nil {
		panic(fmt.Sprintf("failed to set rlconsensus applied params: %v", err))
	}
}

// ExportGenesis exports the current rlconsensus module state as a genesis state.
func (k *Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	pw, _ := k.GetPolicyWeights(ctx)

	return &types.GenesisState{
		Params:        k.GetParams(ctx),
		AgentStatus:   k.GetAgentStatus(ctx),
		PolicyWeights: pw,
	}
}
