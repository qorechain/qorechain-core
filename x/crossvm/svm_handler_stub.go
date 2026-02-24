//go:build !proprietary

package crossvm

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SVMCallFunc is the callback signature for routing cross-VM calls to the SVM runtime.
type SVMCallFunc func(ctx sdk.Context, targetContract string, payload []byte, sender string) ([]byte, error)

// SetSVMCallHandler is a no-op in the public build.
// SVM cross-VM routing is only available in the proprietary build.
func SetSVMCallHandler(_ SVMCallFunc) {}
