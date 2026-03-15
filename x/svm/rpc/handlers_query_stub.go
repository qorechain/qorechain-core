//go:build !proprietary

package rpc

func (s *Server) handleGetProgramAccounts(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}

func (s *Server) handleGetMultipleAccounts(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}

func (s *Server) handleGetSignaturesForAddress(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}

func (s *Server) handleGetTransaction(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}
