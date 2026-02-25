package fairblock

import (
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/fairblock/types"
)

// mockFairBlockKeeper is a minimal mock implementing FairBlockKeeper for testing.
type mockFairBlockKeeper struct {
	enabled bool
}

func (m mockFairBlockKeeper) Logger() log.Logger {
	return log.NewNopLogger()
}

func (m mockFairBlockKeeper) GetConfig(_ sdk.Context) types.FairBlockConfig {
	return types.DefaultFairBlockConfig()
}

func (m mockFairBlockKeeper) SetConfig(_ sdk.Context, _ types.FairBlockConfig) error {
	return nil
}

func (m mockFairBlockKeeper) IsEnabled(_ sdk.Context) bool {
	return m.enabled
}

func (m mockFairBlockKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}

func (m mockFairBlockKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}

func TestFairBlockDecoratorPassthrough(t *testing.T) {
	// Create a mock keeper with FairBlock disabled.
	keeper := mockFairBlockKeeper{enabled: false}
	decorator := NewFairBlockDecorator(keeper)

	// Track whether the next handler was called.
	nextCalled := false
	nextHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	ctx := sdk.Context{}
	_, err := decorator.AnteHandle(ctx, nil, false, nextHandler)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !nextCalled {
		t.Error("expected next handler to be called when FairBlock is disabled")
	}
}
