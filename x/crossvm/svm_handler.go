//go:build proprietary

package crossvm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/crossvm/keeper"
)

// SVMCallFunc is the callback signature for routing cross-VM calls to the SVM runtime.
type SVMCallFunc func(ctx sdk.Context, targetContract string, payload []byte, sender string) ([]byte, error)

// SetSVMCallHandler registers the SVM call handler for cross-VM routing.
// Must be called during app initialization after both SVM and CrossVM keepers are created.
func SetSVMCallHandler(fn SVMCallFunc) {
	keeper.SetSVMCallHandler(keeper.SVMCallFunc(fn))
}
