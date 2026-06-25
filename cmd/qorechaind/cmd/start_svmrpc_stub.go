//go:build !full

package cmd

import (
	"cosmossdk.io/log"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"

	"github.com/qorechain/qorechain-core/app"
)

// startSVMRPCIfEnabled is a no-op in the community build: the Solana-compatible
// JSON-RPC server (x/svm/rpc) ships only in the full build. The full-build
// implementation is overlaid from qorechain-proprietary.
func startSVMRPCIfEnabled(_ log.Logger, _ *app.QoreChainApp, _ servertypes.AppOptions) {}
