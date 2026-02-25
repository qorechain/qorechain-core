package xqore

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/xqore/types"
)

// XQOREKeeper is the interface for the x/xqore module's keeper.
// Any implementation also satisfies rlconsensus.TokenomicsKeeper via GetXQOREBalance,
// replacing NilTokenomicsKeeper with real balance lookups for QDRW governance.
type XQOREKeeper interface {
	Logger() log.Logger

	GetParams(ctx sdk.Context) types.Params
	SetParams(ctx sdk.Context, params types.Params) error

	// GetXQOREBalance returns the xQORE balance for an address.
	// Satisfies rlconsensus.TokenomicsKeeper interface.
	GetXQOREBalance(ctx sdk.Context, addr sdk.AccAddress) math.Int

	// Position management
	Lock(ctx sdk.Context, owner sdk.AccAddress, amount math.Int) error
	Unlock(ctx sdk.Context, owner sdk.AccAddress, amount math.Int) error
	GetPosition(ctx sdk.Context, owner sdk.AccAddress) (types.XQOREPosition, bool)
	GetAllPositions(ctx sdk.Context) []types.XQOREPosition

	// Totals
	GetTotalLocked(ctx sdk.Context) math.Int
	GetTotalXQORESupply(ctx sdk.Context) math.Int

	// Governance multiplier
	GetGovernanceMultiplier(ctx sdk.Context, addr sdk.AccAddress) math.LegacyDec

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}
