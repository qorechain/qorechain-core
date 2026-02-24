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

func TestStubAnteDecoratorPassesThrough(t *testing.T) {
	decorator := NewSVMAnteDecorator(nil)
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
