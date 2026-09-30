package types

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
)

// PoolReader is what fee conversion reads from x/amm.
type PoolReader interface {
	GetPool(ctx sdk.Context, id uint64) (ammtypes.Pool, bool)
}

// ConversionBankKeeper is what fee conversion needs from x/bank.
type ConversionBankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// ConversionAMM is what the float refill needs from x/amm.
type ConversionAMM interface {
	PoolReader
	SwapExactIn(ctx sdk.Context, msg ammtypes.MsgSwapExactIn) (math.Int, error)
}

// ConversionDeps is what the float refill and the float query need from
// other modules.
type ConversionDeps struct {
	Bank ConversionBankKeeper
	AMM  ConversionAMM
}
