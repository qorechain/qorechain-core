package gasabstraction

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// PoolReader is an alias of types.PoolReader.
type PoolReader = types.PoolReader

// ResolveRate is types.ResolveRate.
func ResolveRate(ctx sdk.Context, cfg types.GasAbstractionConfig, token types.AcceptedFeeToken, pools PoolReader) (math.LegacyDec, error) {
	return types.ResolveRate(ctx, cfg, token, pools)
}

// NativeValue is types.NativeValue.
func NativeValue(amount math.Int, rate math.LegacyDec, premiumBps uint32) math.Int {
	return types.NativeValue(amount, rate, premiumBps)
}
