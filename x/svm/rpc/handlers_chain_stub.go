//go:build !proprietary

package rpc

func (s *Server) handleGetBlockHeight(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}

func (s *Server) handleGetRecentBlockhash(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}

func (s *Server) handleGetLatestBlockhash(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}

func (s *Server) handleGetFeeForMessage(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}

func (s *Server) handleIsBlockhashValid(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}
