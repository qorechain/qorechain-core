//go:build proprietary

package rpc

import (
	"encoding/hex"
	"fmt"
)

// handleGetBlockHeight returns the current block height.
func (s *Server) handleGetBlockHeight(_ []interface{}) (interface{}, *RPCError) {
	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}
	return uint64(ctx.BlockHeight()), nil
}

// handleGetRecentBlockhash returns the most recent blockhash and fee calculator.
// This is the deprecated Solana method; getLatestBlockhash is preferred.
func (s *Server) handleGetRecentBlockhash(_ []interface{}) (interface{}, *RPCError) {
	return s.latestBlockhash()
}

// handleGetLatestBlockhash returns the latest blockhash and its last valid
// block height.
func (s *Server) handleGetLatestBlockhash(_ []interface{}) (interface{}, *RPCError) {
	return s.latestBlockhash()
}

// latestBlockhash is the shared implementation for both getRecentBlockhash and
// getLatestBlockhash.
func (s *Server) latestBlockhash() (interface{}, *RPCError) {
	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	slot := s.svmKeeper.GetCurrentSlot(ctx)
	height := uint64(ctx.BlockHeight())

	// Derive blockhash from the block header's app hash.
	var blockhash string
	header := ctx.BlockHeader()
	if len(header.AppHash) > 0 {
		blockhash = hex.EncodeToString(header.AppHash)
	} else {
		// Fallback: use a deterministic hash from the block height.
		blockhash = fmt.Sprintf("%064x", height)
	}

	return BlockhashResult{
		Context: ContextResult{Slot: slot},
		Value: &BlockhashValue{
			Blockhash:            blockhash,
			LastValidBlockHeight: height + 150, // ~150 blocks validity window
		},
	}, nil
}

// handleGetFeeForMessage returns the fee that the network will charge for a
// given message. For MVP, returns a fixed 5000 lamports (matching the standard
// SVM base fee).
func (s *Server) handleGetFeeForMessage(_ []interface{}) (interface{}, *RPCError) {
	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	slot := s.svmKeeper.GetCurrentSlot(ctx)

	return FeeResult{
		Context: ContextResult{Slot: slot},
		Value:   5000,
	}, nil
}

// handleIsBlockhashValid checks whether a blockhash is still valid for
// transaction submission. For MVP, always returns true.
//
// params[0]: blockhash string
func (s *Server) handleIsBlockhashValid(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 1 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected blockhash as first parameter"}
	}

	_, ok := params[0].(string)
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "blockhash must be a string"}
	}

	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	slot := s.svmKeeper.GetCurrentSlot(ctx)

	return map[string]interface{}{
		"context": ContextResult{Slot: slot},
		"value":   true,
	}, nil
}
