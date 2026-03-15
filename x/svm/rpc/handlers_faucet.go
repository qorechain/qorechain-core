//go:build proprietary

package rpc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// maxAirdropLamports caps the airdrop at 2 SOL equivalent.
const maxAirdropLamports = 2_000_000_000

// handleRequestAirdrop mints lamports to an SVM account for testnet use.
//
// params[0]: base58 recipient address
// params[1]: lamports to airdrop (number)
func (s *Server) handleRequestAirdrop(params []interface{}) (interface{}, *RPCError) {
	if len(params) < 2 {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "expected address and lamports amount"}
	}

	addrStr, ok := params[0].(string)
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "address must be a string"}
	}

	addr, err := types.Base58Decode(addrStr)
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: fmt.Sprintf("invalid Base58 address: %v", err)}
	}

	lamportsF, ok := params[1].(float64)
	if !ok {
		return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "lamports must be a number"}
	}
	lamports := uint64(lamportsF)

	if lamports > maxAirdropLamports {
		return nil, &RPCError{
			Code:    ErrCodeInvalidParams,
			Message: fmt.Sprintf("requested %d lamports exceeds maximum airdrop of %d", lamports, maxAirdropLamports),
		}
	}

	ctx, err := s.getQueryContext()
	if err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: err.Error()}
	}

	// Look up or create the account.
	acct, err := s.svmKeeper.GetAccount(ctx, addr)
	if err != nil {
		// Account does not exist; create a new one owned by the System Program.
		acct = &types.SVMAccount{
			Address:    addr,
			Lamports:   0,
			DataLen:    0,
			Data:       []byte{},
			Owner:      types.SystemProgramAddress,
			Executable: false,
			RentEpoch:  0,
		}
	}

	acct.Lamports += lamports

	if err := s.svmKeeper.SetAccount(ctx, acct); err != nil {
		return nil, &RPCError{Code: ErrCodeInternal, Message: fmt.Sprintf("failed to update account: %v", err)}
	}

	// Generate a deterministic signature for the airdrop transaction.
	slot := s.svmKeeper.GetCurrentSlot(ctx)
	sig := computeAirdropSignature(addr, lamports, slot)

	return SendTransactionResult(sig), nil
}

// computeAirdropSignature generates a deterministic hex signature for an airdrop.
func computeAirdropSignature(addr [32]byte, lamports uint64, slot uint64) string {
	h := sha256.New()
	h.Write(addr[:])
	buf := make([]byte, 8)
	buf[0] = byte(lamports)
	buf[1] = byte(lamports >> 8)
	buf[2] = byte(lamports >> 16)
	buf[3] = byte(lamports >> 24)
	buf[4] = byte(lamports >> 32)
	buf[5] = byte(lamports >> 40)
	buf[6] = byte(lamports >> 48)
	buf[7] = byte(lamports >> 56)
	h.Write(buf)
	buf[0] = byte(slot)
	buf[1] = byte(slot >> 8)
	buf[2] = byte(slot >> 16)
	buf[3] = byte(slot >> 24)
	buf[4] = byte(slot >> 32)
	buf[5] = byte(slot >> 40)
	buf[6] = byte(slot >> 48)
	buf[7] = byte(slot >> 56)
	h.Write(buf)
	return hex.EncodeToString(h.Sum(nil))
}
