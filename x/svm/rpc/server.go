//go:build proprietary

package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"cosmossdk.io/log"

	svmmod "github.com/qorechain/qorechain-core/x/svm"
)

// Server is the Solana-compatible JSON-RPC server.
type Server struct {
	httpServer *http.Server
	svmKeeper  svmmod.SVMKeeper
	logger     log.Logger
	mu         sync.Mutex
}

// NewServer creates a new JSON-RPC server.
func NewServer(addr string, svmKeeper svmmod.SVMKeeper, logger log.Logger) *Server {
	s := &Server{
		svmKeeper: svmKeeper,
		logger:    logger.With("module", "svm-rpc"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRPC)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return s
}

// Start begins listening for JSON-RPC requests.
func (s *Server) Start() error {
	s.logger.Info("starting SVM JSON-RPC server", "addr", s.httpServer.Addr)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("SVM JSON-RPC server error", "error", err)
		}
	}()
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	s.logger.Info("stopping SVM JSON-RPC server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// handleRPC processes incoming JSON-RPC requests.
func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		writeRPCError(w, nil, ErrCodeParse, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var req RPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeRPCError(w, nil, ErrCodeParse, "invalid JSON")
		return
	}

	if req.JSONRPC != "2.0" {
		writeRPCError(w, req.ID, ErrCodeInvalidRequest, "jsonrpc must be 2.0")
		return
	}

	result, rpcErr := s.dispatch(req)
	if rpcErr != nil {
		writeRPCResponse(w, RPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   rpcErr,
		})
		return
	}

	writeRPCResponse(w, RPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	})
}

// dispatch routes an RPC request to the appropriate handler.
func (s *Server) dispatch(req RPCRequest) (interface{}, *RPCError) {
	switch req.Method {
	case "getAccountInfo":
		return s.handleGetAccountInfo(req.Params)
	case "getBalance":
		return s.handleGetBalance(req.Params)
	case "getSlot":
		return s.handleGetSlot(req.Params)
	case "getMinimumBalanceForRentExemption":
		return s.handleGetMinimumBalance(req.Params)
	case "getVersion":
		return s.handleGetVersion(req.Params)
	case "getHealth":
		return s.handleGetHealth(req.Params)
	default:
		return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}
}

func writeRPCResponse(w http.ResponseWriter, resp RPCResponse) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeRPCError(w http.ResponseWriter, id interface{}, code int, msg string) {
	writeRPCResponse(w, RPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	})
}
