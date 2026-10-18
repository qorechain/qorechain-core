package stakingtier

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/stakingtier/types"
)

// StakingTierKeeper is the interface for the x/stakingtier module's keeper.
type StakingTierKeeper interface {
	Logger() log.Logger

	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error
	GetPool(ctx sdk.Context) types.Pool

	// CheckStakingMsg refuses a MsgUndelegate or MsgBeginRedelegate that would
	// take locked stake out of a delegation. The staking message server calls
	// it for every such message, whatever dispatched it.
	CheckStakingMsg(ctx sdk.Context, msg sdk.Msg) error

	// SettleEpochBonus is called by x/inflation once per epoch, after the base
	// emission. It computes the bonus owed since the previous settlement,
	// spends the module's surplus first and mints the rest through mint, at
	// most mintable when capped, and credits every bonus (pro rata when
	// funding falls short). The settled period is at most maxPeriodSeconds
	// back from the block time, so a pause in emission is not paid in one go
	// afterwards. It returns what was minted, which x/inflation counts
	// against its cumulative cap.
	SettleEpochBonus(ctx sdk.Context, baseEmission, mintable math.Int, capped bool, maxPeriodSeconds int64,
		mint func(ctx sdk.Context, amount math.Int) error) (math.Int, error)

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
