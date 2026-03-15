//go:build !proprietary

package rpc

func (s *Server) handleRequestAirdrop(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}
