//go:build proprietary

package rpc

import (
	"encoding/base64"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// ctxProvider is set by the app to provide the current query context.
// This avoids the server needing to know about the app's internals.
var ctxProvider func() sdk.Context

// SetContextProvider sets the function that provides the current query context.
func SetContextProvider(fn func() sdk.Context) {
	ctxProvider = fn
}

func (s *Server) getQueryContext() (sdk.Context, error) {
	if ctxProvider == nil {
		return sdk.Context{}, fmt.Errorf("context provider not set")
	}
	return ctxProvider(), nil
}

func (s *Server) handleGetAccountInfo(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 1 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected account address as first parameter"}
	}

	addrStr, ok := params[0].(string)
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "account address must be a string"}
	}

	addr, err := types.Base58Decode(addrStr)
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid Base58 address: %v", err)}
	}

	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	account, err := s.svmKeeper.GetAccount(ctx, addr)
	if err != nil {
		// Account not found -- return null value (Solana convention)
		return GetAccountInfoResult{
			Context: ContextResult{Slot: 0},
			Value:   nil,
		}, nil
	}

	return GetAccountInfoResult{
		Context: ContextResult{Slot: 0},
		Value: &AccountInfo{
			Data:       []string{base64.StdEncoding.EncodeToString(account.Data), "base64"},
			Executable: account.Executable,
			Lamports:   account.Lamports,
			Owner:      types.Base58Encode(account.Owner),
			RentEpoch:  account.RentEpoch,
		},
	}, nil
}

func (s *Server) handleGetBalance(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 1 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected account address"}
	}

	addrStr, ok := params[0].(string)
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "account address must be a string"}
	}

	addr, err := types.Base58Decode(addrStr)
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid Base58 address: %v", err)}
	}

	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	account, err := s.svmKeeper.GetAccount(ctx, addr)
	if err != nil {
		return GetBalanceResult{
			Context: ContextResult{Slot: 0},
			Value:   0,
		}, nil
	}

	return GetBalanceResult{
		Context: ContextResult{Slot: 0},
		Value:   account.Lamports,
	}, nil
}

func (s *Server) handleGetSlot(_ []interface{}) (interface{}, *RPCError) {
	// Return 0 for now; will be connected to block height later
	return uint64(0), nil
}

func (s *Server) handleGetMinimumBalance(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 1 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected data length as first parameter"}
	}

	// JSON numbers are float64
	dataLenF, ok := params[0].(float64)
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "data length must be a number"}
	}
	dataLen := uint64(dataLenF)

	minBalance := s.svmKeeper.GetMinimumBalance(dataLen)
	return minBalance, nil
}

func (s *Server) handleGetVersion(_ []interface{}) (interface{}, *RPCError) {
	return map[string]interface{}{
		"solana-core": "1.18.0-qorechain",
		"feature-set": uint64(0),
	}, nil
}

func (s *Server) handleGetHealth(_ []interface{}) (interface{}, *RPCError) {
	return "ok", nil
}
