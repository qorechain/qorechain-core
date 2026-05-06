package amm

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
)

// AMMKeeper is the interface contract every AMM keeper implementation
// (community-edition stub or full proprietary) must satisfy.
//
// Methods that mutate consensus state must remain deterministic — see
// the project-level invariants (no time.Now(), no float, sorted map iter).
type AMMKeeper interface {
	Logger() log.Logger

	// Params
	GetParams(ctx sdk.Context) ammtypes.Params
	SetParams(ctx sdk.Context, p ammtypes.Params) error

	// Pool management
	CreatePool(ctx sdk.Context, msg ammtypes.MsgCreatePool) (ammtypes.Pool, math.Int, error)
	GetPool(ctx sdk.Context, id uint64) (ammtypes.Pool, bool)
	GetPoolByDenoms(ctx sdk.Context, denomA, denomB string) (ammtypes.Pool, bool)
	GetAllPools(ctx sdk.Context) []ammtypes.Pool
	PausePool(ctx sdk.Context, msg ammtypes.MsgPausePool) error
	ResumePool(ctx sdk.Context, msg ammtypes.MsgResumePool) error

	// Liquidity
	AddLiquidity(ctx sdk.Context, msg ammtypes.MsgAddLiquidity) (lpMinted math.Int, err error)
	RemoveLiquidity(ctx sdk.Context, msg ammtypes.MsgRemoveLiquidity) (returnedA, returnedB sdk.Coin, err error)

	// Swaps
	SwapExactIn(ctx sdk.Context, msg ammtypes.MsgSwapExactIn) (amountOut math.Int, err error)
	SwapExactOut(ctx sdk.Context, msg ammtypes.MsgSwapExactOut) (amountIn math.Int, err error)

	// LP balances
	GetLPBalance(ctx sdk.Context, poolID uint64, holder sdk.AccAddress) math.Int
	GetLPSupply(ctx sdk.Context, poolID uint64) math.Int

	// Quotes (read-only — usable from queries and from the cross-VM router).
	QuoteExactIn(ctx sdk.Context, poolID uint64, denomIn string, amountIn math.Int) (amountOut math.Int, feePaid math.Int, err error)
	QuoteExactOut(ctx sdk.Context, poolID uint64, denomOut string, amountOut math.Int) (amountIn math.Int, feePaid math.Int, err error)

	// Cross-VM hook — invoked by x/crossvm to route EVM/SVM swap calls into
	// the AMM. The implementation handles the bank movement on `from`'s
	// behalf (it's already authenticated by the caller's authz / precompile
	// gas check).
	SwapFromEVM(ctx sdk.Context, from sdk.AccAddress, denomIn string, amountIn math.Int, denomOut string, minOut math.Int) (amountOut math.Int, err error)

	// AI advisory route hint. May return an empty suggestion if the
	// rlconsensus keeper is not wired or returns no advice.
	SuggestRoute(ctx sdk.Context, denomIn, denomOut string, amount math.Int) ammtypes.RouteSuggestion

	// EndBlocker recomputes weighted average prices for all active pools.
	EndBlock(ctx sdk.Context) error

	// Genesis
	InitGenesis(ctx sdk.Context, gs ammtypes.GenesisState)
	ExportGenesis(ctx sdk.Context) *ammtypes.GenesisState
}

// LPBalance is re-exported for module-level consumers (CLI, queries).
type LPBalance = ammtypes.LPBalance
