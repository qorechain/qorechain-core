//go:build !proprietary

package svm

import (
	"testing"

	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

func TestStubKeeperGetAccountReturnsDisabled(t *testing.T) {
	keeper := NewStubKeeper(log.NewNopLogger())
	_, err := keeper.GetAccount(sdk.Context{}, [32]byte{1})
	if err == nil {
		t.Fatal("expected ErrSVMDisabled")
	}
}

func TestStubKeeperDeployProgramReturnsDisabled(t *testing.T) {
	keeper := NewStubKeeper(log.NewNopLogger())
	_, err := keeper.DeployProgram(sdk.Context{}, [32]byte{1}, []byte{0x7f})
	if err == nil {
		t.Fatal("expected ErrSVMDisabled")
	}
}

func TestStubComputeBudgetDecoratorPassesThrough(t *testing.T) {
	decorator := NewSVMComputeBudgetDecorator(nil)
	called := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		called = true
		return ctx, nil
	}
	_, err := decorator.AnteHandle(sdk.Context{}, nil, false, next)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
}

func TestStubDeductFeeDecoratorPassesThrough(t *testing.T) {
	decorator := NewSVMDeductFeeDecorator(nil)
	called := false
	next := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		called = true
		return ctx, nil
	}
	_, err := decorator.AnteHandle(sdk.Context{}, nil, false, next)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("next handler was not called")
	}
}

func TestStubKeeperSatisfiesInterface(t *testing.T) {
	keeper := NewStubKeeper(log.NewNopLogger())
	// Compile-time check that StubKeeper implements SVMKeeper
	var _ SVMKeeper = keeper
}

func TestStubKeeperGetMinimumBalance(t *testing.T) {
	keeper := NewStubKeeper(log.NewNopLogger())
	balance := keeper.GetMinimumBalance(0)
	if balance == 0 {
		t.Fatal("minimum balance should be non-zero even for zero data")
	}
	balance1024 := keeper.GetMinimumBalance(1024)
	if balance1024 <= balance {
		t.Fatal("larger data should require larger minimum balance")
	}
}

func TestStubKeeperDefaultParams(t *testing.T) {
	keeper := NewStubKeeper(log.NewNopLogger())
	params := keeper.GetParams(sdk.Context{})
	if params.MaxProgramSize != types.DefaultMaxProgramSize {
		t.Fatalf("expected default MaxProgramSize %d, got %d", types.DefaultMaxProgramSize, params.MaxProgramSize)
	}
}

func TestStubExecutorSatisfiesInterface(t *testing.T) {
	executor := &stubExecutorWrapper{}
	// Compile-time check that ffi.StubExecutor satisfies SVMExecutor
	var _ SVMExecutor = executor
}

// stubExecutorWrapper wraps the ffi.StubExecutor methods for cross-package interface check.
type stubExecutorWrapper struct{}

func (e *stubExecutorWrapper) Execute(_ []byte, _ []byte, _ []types.SVMAccount, _ uint64) (*types.ExecutionResult, error) {
	return nil, types.ErrSVMDisabled
}
func (e *stubExecutorWrapper) ExecuteV2(_ []byte, _ []types.SVMAccount, _ []types.AccountMeta,
	_ []byte, _ [32]byte, _ uint64, _ int64) (*types.ExecutionResult, error) {
	return nil, types.ErrSVMDisabled
}
func (e *stubExecutorWrapper) ExecuteNative(_ [32]byte, _ []types.SVMAccount, _ []types.AccountMeta,
	_ []byte, _ int64) (*types.ExecutionResult, error) {
	return nil, types.ErrSVMDisabled
}
func (e *stubExecutorWrapper) ValidateProgram(_ []byte) error { return types.ErrSVMDisabled }
func (e *stubExecutorWrapper) Close()                         {}
