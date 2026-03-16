//go:build proprietary

package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// isNativeProgram returns true if the given program ID matches one of the
// built-in native programs (System, SPL Token, ATA, Memo).
func isNativeProgram(id [32]byte) bool {
	return id == types.SystemProgramAddress ||
		id == types.SPLTokenAddress ||
		id == types.ATAAddress ||
		id == types.MemoAddress
}

// ExecuteProgram orchestrates the execution of an SVM program instruction.
//
// Native programs (System, Token, ATA, Memo) are routed to ExecuteNative
// which handles them without BPF interpretation. User-deployed programs are
// executed via ExecuteV2 with full Solana-compatible account serialization.
//
// Steps:
//  1. Load all referenced accounts from the KVStore.
//  2. Route to ExecuteNative or ExecuteV2 based on the program ID.
//  3. Write back any modified accounts that were declared writable.
//  4. Return the execution result with logs and compute units used.
func (k *Keeper) ExecuteProgram(
	ctx sdk.Context,
	programID [32]byte,
	instruction []byte,
	accounts []types.AccountMeta,
	signers [][32]byte,
) (*types.ExecutionResult, error) {
	params := k.GetParams(ctx)
	if !params.Enabled {
		return nil, types.ErrSVMDisabled
	}

	if k.executor == nil {
		return nil, types.ErrSVMDisabled.Wrap("BPF executor not initialized")
	}

	// Load account snapshots for the executor.
	svmAccounts := make([]types.SVMAccount, len(accounts))
	for i, meta := range accounts {
		acc, err := k.GetAccount(ctx, meta.Address)
		if err != nil {
			// Create a zero-lamport account if it does not exist and is not a signer.
			if !meta.IsSigner {
				svmAccounts[i] = types.SVMAccount{
					Address:  meta.Address,
					Lamports: 0,
					DataLen:  0,
					Data:     []byte{},
					Owner:    types.SystemProgramAddress,
				}
				continue
			}
			return nil, types.ErrAccountNotFound.Wrapf("signer account %s not found",
				types.Base58Encode(meta.Address))
		}
		svmAccounts[i] = *acc
	}

	var result *types.ExecutionResult
	var err error

	blockTime := ctx.BlockTime().Unix()

	if isNativeProgram(programID) {
		// Native program execution — no bytecode needed.
		result, err = k.executor.ExecuteNative(programID, svmAccounts, accounts, instruction, blockTime)
	} else {
		// BPF program execution — verify the program account and load bytecode.
		progAcc, lookupErr := k.GetAccount(ctx, programID)
		if lookupErr != nil {
			return nil, types.ErrProgramNotFound.Wrapf("program %s: %v",
				types.Base58Encode(programID), lookupErr)
		}
		if !progAcc.Executable {
			return nil, types.ErrProgramNotExecutable.Wrapf("account %s is not executable",
				types.Base58Encode(programID))
		}

		bytecode, loadErr := k.LoadProgramBytecode(ctx, programID)
		if loadErr != nil {
			return nil, fmt.Errorf("failed to load bytecode for %s: %w",
				types.Base58Encode(programID), loadErr)
		}

		result, err = k.executor.ExecuteV2(bytecode, svmAccounts, accounts, instruction,
			programID, params.ComputeBudgetMax, blockTime)
	}

	if err != nil {
		return nil, fmt.Errorf("SVM execution error: %w", err)
	}

	// If the executor returned modified accounts, write them back to the store.
	if result.Success && len(result.ModifiedAccounts) > 0 {
		for i := range result.ModifiedAccounts {
			modified := &result.ModifiedAccounts[i]

			// Only write back accounts that were declared writable.
			writable := false
			for _, meta := range accounts {
				if meta.Address == modified.Address && meta.IsWritable {
					writable = true
					break
				}
			}
			if !writable {
				continue
			}

			if err := k.SetAccount(ctx, modified); err != nil {
				k.logger.Error("failed to persist modified account",
					"address", types.Base58Encode(modified.Address),
					"error", err,
				)
			}
		}
	}

	// Emit execution event.
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"svm_execute_program",
		sdk.NewAttribute("program_id", types.Base58Encode(programID)),
		sdk.NewAttribute("success", fmt.Sprintf("%t", result.Success)),
		sdk.NewAttribute("compute_units", fmt.Sprintf("%d", result.ComputeUnitsUsed)),
	))

	k.logger.Info("program executed",
		"program", types.Base58Encode(programID),
		"success", result.Success,
		"cu_used", result.ComputeUnitsUsed,
	)

	return result, nil
}
