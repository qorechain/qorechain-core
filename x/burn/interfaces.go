package burn

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/burn/types"
)

// BurnKeeper is the interface for the x/burn module's keeper.
// DistributionLiability reports what the distribution module owes, so the burn
// module can tell that apart from what merely sits in its balance. Implemented
// by the application over the SDK's distribution keeper.
type DistributionLiability interface {
	// TotalOwed is the v3.2.0 formula, which counts unwithdrawn commission
	// twice (QSR-2026-0341). Kept unchanged for replay of that handler.
	TotalOwed(ctx sdk.Context) math.Int
	// ClaimsOwed is outstanding rewards plus the community pool only.
	ClaimsOwed(ctx sdk.Context) math.Int
}

type BurnKeeper interface {
	Logger() log.Logger

	// Stranded staker share: recorded once by the release upgrade handler,
	// released only by a governance vote.
	EnsureStrandedSnapshot(ctx sdk.Context) types.StrandedRewards
	GetStrandedRewards(ctx sdk.Context) types.StrandedRewards
	ReleaseStranded(ctx sdk.Context, amount math.Int, destination string) error

	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error

	// Core burn operations
	BurnFromSource(ctx sdk.Context, source types.BurnSource, amount math.Int, txHash string) error
	GetTotalBurned(ctx sdk.Context) math.Int
	GetBurnStats(ctx sdk.Context) types.BurnStats
	GetBurnRecords(ctx sdk.Context, limit int) []types.BurnRecord

	// Fee distribution (EndBlocker)
	DistributeFees(ctx sdk.Context) error

	// Failed-transaction fees (failed_tx burn channel). RecordTxFee adds a
	// delivered transaction's fee to the block's accumulator and reports
	// whether it did (false when the channel is not enabled); ReleaseTxFee
	// takes it back out once the transaction's messages succeeded.
	RecordTxFee(ctx sdk.Context, fee math.Int) bool
	ReleaseTxFee(ctx sdk.Context, fee math.Int)

	// Milestone burn
	IncrementBlockTxCount(ctx sdk.Context)
	CheckMilestoneBurn(ctx sdk.Context, blockTxCount uint64) error
	GetMilestoneState(ctx sdk.Context) types.MilestoneState

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
