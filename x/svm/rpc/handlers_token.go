//go:build proprietary

package rpc

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// SPL Token account data layout offsets and sizes.
const (
	tokenAccountSize   = 165
	tokenMintOffset    = 0
	tokenOwnerOffset   = 32
	tokenAmountOffset  = 64
	tokenStateOffset   = 108
)

// handleGetTokenAccountsByOwner returns all SPL Token accounts belonging to a
// given wallet owner, optionally filtered by mint.
//
// params[0]: base58 wallet owner address
// params[1]: filter object, one of:
//   - {"mint": "base58"} — filter by token mint
//   - {"programId": "base58"} — filter by token program (must be SPL Token)
// params[2]: optional encoding config
func (s *Server) handleGetTokenAccountsByOwner(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 2 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected owner address and filter object"}
	}

	ownerStr, ok := params[0].(string)
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "owner address must be a string"}
	}

	ownerAddr, err := types.Base58Decode(ownerStr)
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid owner address: %v", err)}
	}

	// Parse the filter object.
	filterMap, ok := params[1].(map[string]interface{})
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "second parameter must be a filter object"}
	}

	var mintFilter [32]byte
	hasMintFilter := false
	if mintStr, ok := filterMap["mint"].(string); ok {
		mf, err := types.Base58Decode(mintStr)
		if err != nil {
			return nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid mint address: %v", err)}
		}
		mintFilter = mf
		hasMintFilter = true
	}

	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	slot := s.svmKeeper.GetCurrentSlot(ctx)

	type tokenAccountEntry struct {
		Pubkey  string      `json:"pubkey"`
		Account interface{} `json:"account"`
	}
	var results []tokenAccountEntry

	s.svmKeeper.IterateAccounts(ctx, func(acct types.SVMAccount) bool {
		// Only look at accounts owned by the SPL Token program.
		if acct.Owner != types.SPLTokenAddress {
			return false
		}

		// Token accounts are exactly 165 bytes.
		if len(acct.Data) < tokenAccountSize {
			return false
		}

		// Extract the wallet owner from the token account data.
		var acctOwner [32]byte
		copy(acctOwner[:], acct.Data[tokenOwnerOffset:tokenOwnerOffset+32])
		if acctOwner != ownerAddr {
			return false
		}

		// Apply mint filter if specified.
		if hasMintFilter {
			var mint [32]byte
			copy(mint[:], acct.Data[tokenMintOffset:tokenMintOffset+32])
			if mint != mintFilter {
				return false
			}
		}

		parsed := parseTokenAccountData(acct.Data)

		results = append(results, tokenAccountEntry{
			Pubkey: types.Base58Encode(acct.Address),
			Account: map[string]interface{}{
				"data": map[string]interface{}{
					"parsed": map[string]interface{}{
						"info": parsed,
						"type": "account",
					},
					"program": "spl-token",
					"space":   tokenAccountSize,
				},
				"executable": acct.Executable,
				"lamports":   acct.Lamports,
				"owner":      types.Base58Encode(acct.Owner),
				"rentEpoch":  acct.RentEpoch,
			},
		})
		return false
	})

	if results == nil {
		results = []tokenAccountEntry{}
	}

	return map[string]interface{}{
		"context": ContextResult{Slot: slot},
		"value":   results,
	}, nil
}

// handleGetTokenAccountsByDelegate returns all SPL Token accounts with a given
// delegate authority. Uses a simplified delegate detection from the token account
// layout.
//
// params[0]: base58 delegate address
// params[1]: filter object (same format as getTokenAccountsByOwner)
func (s *Server) handleGetTokenAccountsByDelegate(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 2 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected delegate address and filter object"}
	}

	delegateStr, ok := params[0].(string)
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "delegate address must be a string"}
	}

	delegateAddr, err := types.Base58Decode(delegateStr)
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid delegate address: %v", err)}
	}

	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	slot := s.svmKeeper.GetCurrentSlot(ctx)

	type tokenAccountEntry struct {
		Pubkey  string      `json:"pubkey"`
		Account interface{} `json:"account"`
	}
	var results []tokenAccountEntry

	// SPL Token account delegate field: offset 72, prefixed by a 4-byte COption tag.
	// Tag at offset 72: 1 = Some (delegate present), 0 = None.
	// Delegate address at offset 76, 32 bytes.
	const delegateTagOffset = 72
	const delegateAddrOffset = 76

	s.svmKeeper.IterateAccounts(ctx, func(acct types.SVMAccount) bool {
		if acct.Owner != types.SPLTokenAddress {
			return false
		}
		if len(acct.Data) < tokenAccountSize {
			return false
		}

		// Check if delegate is present (COption tag == 1).
		if len(acct.Data) < delegateAddrOffset+32 {
			return false
		}
		tag := binary.LittleEndian.Uint32(acct.Data[delegateTagOffset : delegateTagOffset+4])
		if tag != 1 {
			return false
		}

		var delegate [32]byte
		copy(delegate[:], acct.Data[delegateAddrOffset:delegateAddrOffset+32])
		if delegate != delegateAddr {
			return false
		}

		parsed := parseTokenAccountData(acct.Data)

		results = append(results, tokenAccountEntry{
			Pubkey: types.Base58Encode(acct.Address),
			Account: map[string]interface{}{
				"data": map[string]interface{}{
					"parsed": map[string]interface{}{
						"info": parsed,
						"type": "account",
					},
					"program": "spl-token",
					"space":   tokenAccountSize,
				},
				"executable": acct.Executable,
				"lamports":   acct.Lamports,
				"owner":      types.Base58Encode(acct.Owner),
				"rentEpoch":  acct.RentEpoch,
			},
		})
		return false
	})

	if results == nil {
		results = []tokenAccountEntry{}
	}

	return map[string]interface{}{
		"context": ContextResult{Slot: slot},
		"value":   results,
	}, nil
}

// parseTokenAccountData extracts fields from the 165-byte SPL Token account layout.
func parseTokenAccountData(data []byte) *TokenAccountInfo {
	var mint, owner [32]byte
	copy(mint[:], data[tokenMintOffset:tokenMintOffset+32])
	copy(owner[:], data[tokenOwnerOffset:tokenOwnerOffset+32])

	amount := binary.LittleEndian.Uint64(data[tokenAmountOffset : tokenAmountOffset+8])

	state := "uninitialized"
	if len(data) > tokenStateOffset {
		switch data[tokenStateOffset] {
		case 1:
			state = "initialized"
		case 2:
			state = "frozen"
		}
	}

	info := &TokenAccountInfo{
		Mint:   types.Base58Encode(mint),
		Owner:  types.Base58Encode(owner),
		Amount: fmt.Sprintf("%d", amount),
		State:  state,
	}

	// Parse delegate (COption at offset 72).
	const delegateTagOff = 72
	const delegateAddrOff = 76
	if len(data) >= delegateAddrOff+32 {
		tag := binary.LittleEndian.Uint32(data[delegateTagOff : delegateTagOff+4])
		if tag == 1 {
			var delegate [32]byte
			copy(delegate[:], data[delegateAddrOff:delegateAddrOff+32])
			info.Delegate = types.Base58Encode(delegate)

			// Delegated amount is at offset 121, 8 bytes.
			if len(data) >= 129 {
				delegatedAmt := binary.LittleEndian.Uint64(data[121:129])
				info.DelegatedAmount = fmt.Sprintf("%d", delegatedAmt)
			}
		}
	}

	// Parse close authority (COption at offset 130).
	const closeTagOff = 130
	const closeAddrOff = 134
	if len(data) >= closeAddrOff+32 {
		tag := binary.LittleEndian.Uint32(data[closeTagOff : closeTagOff+4])
		if tag == 1 {
			var closeAuth [32]byte
			copy(closeAuth[:], data[closeAddrOff:closeAddrOff+32])
			info.CloseAuthority = types.Base58Encode(closeAuth)
		}
	}

	return info
}
