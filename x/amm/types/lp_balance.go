package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// LPBalance is a single (pool_id, holder) → amount tuple. The AMM tracks
// LP balances directly in its own KV space rather than minting native bank
// tokens for them — this avoids spam in the bank module's denom set and
// gives the AMM keeper full control of LP issuance/burning.
//
// Genesis import/export uses []LPBalance flattened across all pools.
type LPBalance struct {
	PoolID uint64   `json:"pool_id"`
	Holder string   `json:"holder"` // bech32
	Amount math.Int `json:"amount"`
}

// Validate enforces non-empty fields and a non-negative amount.
func (b LPBalance) Validate() error {
	if b.PoolID == 0 {
		return fmt.Errorf("pool_id must be > 0")
	}
	if _, err := sdk.AccAddressFromBech32(b.Holder); err != nil {
		return fmt.Errorf("invalid holder address %q: %w", b.Holder, err)
	}
	if b.Amount.IsNil() || b.Amount.IsNegative() {
		return fmt.Errorf("amount must be non-negative")
	}
	return nil
}
