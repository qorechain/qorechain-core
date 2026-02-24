//go:build proprietary

package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// ExecuteProgram orchestrates the execution of an SVM program instruction.
//
// Steps:
//  1. Load the program bytecode from the data account.
//  2. Load all referenced accounts from the KVStore.
//  3. Call the BPF executor (FFI bridge).
//  4. Write back any modified accounts.
//  5. Return the execution result with logs and compute units used.
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

	// Verify the program account exists and is executable.
	progAcc, err := k.GetAccount(ctx, programID)
	if err != nil {
		return nil, types.ErrProgramNotFound.Wrapf("program %s: %v",
			types.Base58Encode(programID), err)
	}
	if !progAcc.Executable {
		return nil, types.ErrProgramNotExecutable.Wrapf("account %s is not executable",
			types.Base58Encode(programID))
	}

	// Load the BPF bytecode.
	bytecode, err := k.LoadProgramBytecode(ctx, programID)
	if err != nil {
		return nil, fmt.Errorf("failed to load bytecode for %s: %w",
			types.Base58Encode(programID), err)
	}

	// Load account snapshots for the executor.
	svmAccounts := make([]types.SVMAccount, len(accounts))
	for i, meta := range accounts {
		acc, err := k.GetAccount(ctx, meta.Address)
		if err != nil {
			// Create a zero-lamport account if it does not exist and is not a signer.
			if !meta.IsSigner {
				svmAccounts[i] = types.SVMAccount{
					Address:    meta.Address,
					Lamports:   0,
					DataLen:    0,
					Data:       []byte{},
					Owner:      types.SystemProgramAddress,
					Executable: false,
					RentEpoch:  0,
				}
				continue
			}
			return nil, types.ErrAccountNotFound.Wrapf("signer account %s not found",
				types.Base58Encode(meta.Address))
		}
		svmAccounts[i] = *acc
	}

	// Execute the BPF program.
	result, err := k.executor.Execute(bytecode, instruction, svmAccounts, params.ComputeBudgetMax)
	if err != nil {
		return nil, fmt.Errorf("BPF execution error: %w", err)
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
