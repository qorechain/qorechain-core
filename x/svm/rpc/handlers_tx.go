//go:build proprietary

package rpc

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// handleSendTransaction accepts a JSON envelope describing an SVM instruction,
// executes it against the keeper, and returns the transaction signature.
//
// params[0]: JSON object with programId, accounts, and data fields.
func (s *Server) handleSendTransaction(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 1 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected transaction envelope as first parameter"}
	}

	envelope, ok := params[0].(map[string]interface{})
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "transaction must be a JSON object with programId, accounts, data"}
	}

	programID, accounts, ixData, rpcErr := parseTxEnvelope(envelope)
	if rpcErr != nil {
		return nil, rpcErr
	}

	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	// Build signer list from account metas.
	var signers [][32]byte
	for _, acct := range accounts {
		if acct.IsSigner {
			signers = append(signers, acct.Address)
		}
	}

	result, err := s.svmKeeper.ExecuteProgram(ctx, programID, ixData, accounts, signers)
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: fmt.Sprintf("execution failed: %v", err)}
	}

	if !result.Success {
		return nil, &RPCError{Code: ErrCodeInternal, Message: fmt.Sprintf("program error: %s", result.Error)}
	}

	// Generate a deterministic transaction signature from the envelope contents.
	sig := computeTxSignature(programID, ixData, ctx.BlockHeight())

	return SendTransactionResult(sig), nil
}

// handleSimulateTransaction dry-runs an SVM instruction and returns logs and
// account changes without committing state.
//
// params[0]: JSON object with same format as sendTransaction.
func (s *Server) handleSimulateTransaction(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 1 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected transaction envelope as first parameter"}
	}

	envelope, ok := params[0].(map[string]interface{})
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "transaction must be a JSON object with programId, accounts, data"}
	}

	programID, accounts, ixData, rpcErr := parseTxEnvelope(envelope)
	if rpcErr != nil {
		return nil, rpcErr
	}

	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	slot := s.svmKeeper.GetCurrentSlot(ctx)

	// Use a cache-wrapped context for state isolation.
	simCtx, _ := ctx.CacheContext()

	var signers [][32]byte
	for _, acct := range accounts {
		if acct.IsSigner {
			signers = append(signers, acct.Address)
		}
	}

	result, err := s.svmKeeper.ExecuteProgram(simCtx, programID, ixData, accounts, signers)
	if err != nil {
		return SimulateTransactionResult{
			Context: ContextResult{Slot: slot},
			Value: &SimulateValue{
				Err:           err.Error(),
				Logs:          nil,
				Accounts:      nil,
				UnitsConsumed: 0,
			},
		}, nil
	}

	var simErr interface{}
	if !result.Success {
		simErr = result.Error
	}

	return SimulateTransactionResult{
		Context: ContextResult{Slot: slot},
		Value: &SimulateValue{
			Err:           simErr,
			Logs:          result.Logs,
			Accounts:      nil,
			UnitsConsumed: result.ComputeUnitsUsed,
		},
	}, nil
}

// parseTxEnvelope extracts the program ID, account metas, and instruction data
// from a JSON transaction envelope.
func parseTxEnvelope(envelope map[string]interface{}) ([32]byte, []types.AccountMeta, []byte, *RPCError) {
	var programID [32]byte

	// Parse programId.
	pidStr, ok := envelope["programId"].(string)
	if !ok {
		return programID, nil, nil, &RPCError{Code: ErrCodeInvalidParams, Message: "missing or invalid programId"}
	}
	pid, err := types.Base58Decode(pidStr)
	if err != nil {
		return programID, nil, nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid programId: %v", err)}
	}
	programID = pid

	// Parse accounts array.
	var metas []types.AccountMeta
	if acctList, ok := envelope["accounts"].([]interface{}); ok {
		for i, item := range acctList {
			acctMap, ok := item.(map[string]interface{})
			if !ok {
				return programID, nil, nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("account[%d] must be an object", i)}
			}

			pubkeyStr, _ := acctMap["pubkey"].(string)
			addr, err := types.Base58Decode(pubkeyStr)
			if err != nil {
				return programID, nil, nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("account[%d] invalid pubkey: %v", i, err)}
			}

			isSigner, _ := acctMap["isSigner"].(bool)
			isWritable, _ := acctMap["isWritable"].(bool)

			metas = append(metas, types.AccountMeta{
				Address:    addr,
				IsSigner:   isSigner,
				IsWritable: isWritable,
			})
		}
	}

	// Parse instruction data (base64-encoded).
	var ixData []byte
	if dataStr, ok := envelope["data"].(string); ok && dataStr != "" {
		ixData, err = base64.StdEncoding.DecodeString(dataStr)
		if err != nil {
			return programID, nil, nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid base64 data: %v", err)}
		}
	}

	return programID, metas, ixData, nil
}

// computeTxSignature generates a deterministic hex signature for a transaction.
func computeTxSignature(programID [32]byte, data []byte, blockHeight int64) string {
	h := sha256.New()
	h.Write(programID[:])
	h.Write(data)
	buf := make([]byte, 8)
	buf[0] = byte(blockHeight)
	buf[1] = byte(blockHeight >> 8)
	buf[2] = byte(blockHeight >> 16)
	buf[3] = byte(blockHeight >> 24)
	buf[4] = byte(blockHeight >> 32)
	buf[5] = byte(blockHeight >> 40)
	buf[6] = byte(blockHeight >> 48)
	buf[7] = byte(blockHeight >> 56)
	h.Write(buf)
	return hex.EncodeToString(h.Sum(nil))
}
