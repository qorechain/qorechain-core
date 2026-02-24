//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// QueryServer implements query handlers for the SVM module.
type QueryServer struct {
	keeper *Keeper
}

// NewQueryServer creates a new SVM query server.
func NewQueryServer(k *Keeper) *QueryServer {
	return &QueryServer{keeper: k}
}

// QueryAccount returns the SVM account at the given 32-byte address.
func (q *QueryServer) QueryAccount(ctx sdk.Context, addr [32]byte) (*types.SVMAccount, error) {
	return q.keeper.GetAccount(ctx, addr)
}

// QueryProgram returns the program metadata for the given program address.
func (q *QueryServer) QueryProgram(ctx sdk.Context, addr [32]byte) (*types.ProgramMeta, error) {
	return q.keeper.GetProgramMeta(ctx, addr)
}

// QueryParams returns the current SVM module parameters.
func (q *QueryServer) QueryParams(ctx sdk.Context) types.Params {
	return q.keeper.GetParams(ctx)
}

// QueryMinimumBalance returns the minimum lamports for rent exemption
// given a data length.
func (q *QueryServer) QueryMinimumBalance(dataLen uint64) uint64 {
	return q.keeper.GetMinimumBalance(dataLen)
}

// QueryCurrentSlot returns the current SVM slot.
func (q *QueryServer) QueryCurrentSlot(ctx sdk.Context) uint64 {
	return q.keeper.GetCurrentSlot(ctx)
}

// QueryAllAccounts returns all SVM accounts (use with caution on large state).
func (q *QueryServer) QueryAllAccounts(ctx sdk.Context) []types.SVMAccount {
	return q.keeper.GetAllAccounts(ctx)
}

// QueryAllPrograms returns all program metadata entries.
func (q *QueryServer) QueryAllPrograms(ctx sdk.Context) []types.ProgramMeta {
	return q.keeper.GetAllProgramMetas(ctx)
}
