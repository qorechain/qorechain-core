//go:build !proprietary

package rpc

func (s *Server) handleGetTokenAccountsByOwner(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}

func (s *Server) handleGetTokenAccountsByDelegate(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}
