package types

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the subset of x/bank used by the AMM module. The full
// build wires this to the Cosmos SDK bank keeper; the public stub uses a
// minimal in-memory implementation in tests.
type BankKeeper interface {
	SendCoinsFromAccountToModule(ctx context.Context, sender sdk.AccAddress, recipient string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, sender string, recipient sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, sender, recipient string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetSupply(ctx context.Context, denom string) sdk.Coin
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

// BurnKeeper is the AMM module's hook into the burn engine: protocol-fee
// shares are routed through the burn keeper using BurnSourceAMM so they
// flow into the standard 37/30/20/10/3 distribution.
//
// The exact signature must match qorechain-core/x/burn/types/BurnSource;
// the AMM module imports the constant directly to avoid a circular
// dependency.
type BurnKeeper interface {
	// CollectAMMProtocolFee receives the protocol portion of a swap fee.
	// Implementation is responsible for invoking the regular burn-engine
	// distribution path; the AMM keeper just hands the coins over.
	CollectAMMProtocolFee(ctx sdk.Context, source string, amt sdk.Coins) error
}

// RLConsensusKeeper provides an OPTIONAL advisory hook. SuggestSwapRoute
// returns a hint that clients (off-chain or smart contracts) may use to
// pick a better route across multiple pools — never used to bind on-chain
// routing decisions, which must remain deterministic.
//
// All implementations MUST be deterministic and side-effect-free in
// consensus paths (the suggestion is computed off pre-existing pool
// state without writing).
type RLConsensusKeeper interface {
	SuggestSwapRoute(ctx sdk.Context, denomIn, denomOut string, amount math.Int) (RouteSuggestion, error)
}

// RouteSuggestion is the advisory output of RLConsensusKeeper.
type RouteSuggestion struct {
	// PoolIDs lists pools to traverse in order; empty list means "no suggestion".
	PoolIDs []uint64

	// EstimatedOut is the predicted amount-out, in denomOut units.
	EstimatedOut math.Int

	// ConfidenceBps is a quality score in basis points (0–10000).
	ConfidenceBps uint32
}
