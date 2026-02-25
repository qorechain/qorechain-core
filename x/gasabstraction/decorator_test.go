package gasabstraction

import (
	"testing"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// mockGasAbstractionKeeper is a minimal mock implementing GasAbstractionKeeper for testing.
type mockGasAbstractionKeeper struct {
	enabled bool
	config  types.GasAbstractionConfig
}

func (m mockGasAbstractionKeeper) Logger() log.Logger {
	return log.NewNopLogger()
}

func (m mockGasAbstractionKeeper) GetConfig(_ sdk.Context) types.GasAbstractionConfig {
	return m.config
}

func (m mockGasAbstractionKeeper) SetConfig(_ sdk.Context, _ types.GasAbstractionConfig) error {
	return nil
}

func (m mockGasAbstractionKeeper) IsEnabled(_ sdk.Context) bool {
	return m.enabled
}

func (m mockGasAbstractionKeeper) GetAcceptedTokens(_ sdk.Context) []types.AcceptedFeeToken {
	return m.config.AcceptedTokens
}

func (m mockGasAbstractionKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}

func (m mockGasAbstractionKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}

func TestGasAbstractionDecoratorDisabled(t *testing.T) {
	// Disabled module should pass through without inspecting fees.
	keeper := mockGasAbstractionKeeper{
		enabled: false,
		config:  types.DefaultGasAbstractionConfig(),
	}
	decorator := NewGasAbstractionDecorator(keeper)

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
		t.Error("expected next handler to be called when gas abstraction is disabled")
	}
}

func TestGasAbstractionDecoratorNativeDenom(t *testing.T) {
	// Enabled module with a nil tx (non-FeeTx) should pass through.
	keeper := mockGasAbstractionKeeper{
		enabled: true,
		config:  types.DefaultGasAbstractionConfig(),
	}
	decorator := NewGasAbstractionDecorator(keeper)

	nextCalled := false
	nextHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	// A nil tx does not implement sdk.FeeTx, so it falls through to next handler.
	ctx := sdk.Context{}
	_, err := decorator.AnteHandle(ctx, nil, false, nextHandler)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !nextCalled {
		t.Error("expected next handler to be called when tx is not a FeeTx")
	}
}
