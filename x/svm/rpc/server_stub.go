//go:build !proprietary

package rpc

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/log"

	svmmod "github.com/qorechain/qorechain-core/x/svm"
)

// Server is a stub for the public build.
type Server struct{}

// NewServer returns a no-op server in the public build.
func NewServer(_ string, _ svmmod.SVMKeeper, _ log.Logger) *Server {
	return &Server{}
}

// Start is a no-op in the public build.
func (s *Server) Start() error { return nil }

// Stop is a no-op in the public build.
func (s *Server) Stop() error { return nil }

// SetContextProvider is a no-op in the public build.
func SetContextProvider(_ func() sdk.Context) {}
