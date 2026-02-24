//go:build !proprietary

package rpc

import "cosmossdk.io/log"

// Server is a stub for the public build.
type Server struct{}

// NewServer returns a no-op server in the public build.
func NewServer(_ string, _ interface{}, _ log.Logger) *Server {
	return &Server{}
}

// Start is a no-op in the public build.
func (s *Server) Start() error { return nil }

// Stop is a no-op in the public build.
func (s *Server) Stop() error { return nil }

// SetContextProvider is a no-op in the public build.
func SetContextProvider(_ interface{}) {}
