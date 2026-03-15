//go:build !proprietary

package rpc

func (s *Server) handleSendTransaction(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}

func (s *Server) handleSimulateTransaction(_ []interface{}) (interface{}, *RPCError) {
	return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "method not available in community build"}
}
