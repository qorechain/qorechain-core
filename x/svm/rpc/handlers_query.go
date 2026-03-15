//go:build proprietary

package rpc

import (
	"encoding/base64"
	"fmt"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// handleGetProgramAccounts returns all accounts owned by a given program.
//
// params[0]: base58 program ID
// params[1]: optional config object with filters (e.g. {filters: [{dataSize: N}]})
func (s *Server) handleGetProgramAccounts(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 1 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected program address as first parameter"}
	}

	pidStr, ok := params[0].(string)
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "program address must be a string"}
	}

	programID, err := types.Base58Decode(pidStr)
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid Base58 address: %v", err)}
	}

	// Parse optional dataSize filter.
	var dataSizeFilter int64 = -1
	if len(params) > 1 {
		if cfg, ok := params[1].(map[string]interface{}); ok {
			if filters, ok := cfg["filters"].([]interface{}); ok {
				for _, f := range filters {
					if fm, ok := f.(map[string]interface{}); ok {
						if ds, ok := fm["dataSize"].(float64); ok {
							dataSizeFilter = int64(ds)
						}
					}
				}
			}
		}
	}

	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	var results []ProgramAccountResult

	s.svmKeeper.IterateAccounts(ctx, func(acct types.SVMAccount) bool {
		if acct.Owner != programID {
			return false
		}

		// Apply dataSize filter if specified.
		if dataSizeFilter >= 0 && int64(acct.DataLen) != dataSizeFilter {
			return false
		}

		results = append(results, ProgramAccountResult{
			Pubkey: types.Base58Encode(acct.Address),
			Account: &AccountInfo{
				Data:       []string{base64.StdEncoding.EncodeToString(acct.Data), "base64"},
				Executable: acct.Executable,
				Lamports:   acct.Lamports,
				Owner:      types.Base58Encode(acct.Owner),
				RentEpoch:  acct.RentEpoch,
			},
		})
		return false
	})

	if results == nil {
		results = []ProgramAccountResult{}
	}

	return results, nil
}

// handleGetMultipleAccounts returns account info for multiple addresses in a
// single batch request.
//
// params[0]: array of base58 addresses (max 100)
func (s *Server) handleGetMultipleAccounts(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 1 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected array of addresses as first parameter"}
	}

	addrList, ok := params[0].([]interface{})
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "first parameter must be an array of addresses"}
	}

	if len(addrList) > 100 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "maximum 100 addresses per request"}
	}

	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	slot := s.svmKeeper.GetCurrentSlot(ctx)

	values := make([]*AccountInfo, len(addrList))
	for i, item := range addrList {
		addrStr, ok := item.(string)
		if !ok {
			continue
		}

		addr, err := types.Base58Decode(addrStr)
		if err != nil {
			continue
		}

		acct, err := s.svmKeeper.GetAccount(ctx, addr)
		if err != nil {
			// null for missing accounts
			continue
		}

		values[i] = &AccountInfo{
			Data:       []string{base64.StdEncoding.EncodeToString(acct.Data), "base64"},
			Executable: acct.Executable,
			Lamports:   acct.Lamports,
			Owner:      types.Base58Encode(acct.Owner),
			RentEpoch:  acct.RentEpoch,
		}
	}

	return map[string]interface{}{
		"context": ContextResult{Slot: slot},
		"value":   values,
	}, nil
}

// handleGetSignaturesForAddress is a stub that returns an empty array.
// Full implementation requires indexer integration.
//
// params[0]: base58 address
func (s *Server) handleGetSignaturesForAddress(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 1 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected address as first parameter"}
	}

	// Validate the address format.
	addrStr, ok := params[0].(string)
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "address must be a string"}
	}
	if _, err := types.Base58Decode(addrStr); err != nil {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid Base58 address: %v", err)}
	}

	return []interface{}{}, nil
}

// handleGetTransaction is a stub that returns null.
// Full implementation requires indexer integration.
//
// params[0]: transaction signature
func (s *Server) handleGetTransaction(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 1 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected transaction signature as first parameter"}
	}

	return nil, nil
}
