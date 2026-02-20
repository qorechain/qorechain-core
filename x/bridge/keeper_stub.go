//go:build !proprietary

package bridge

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// StubKeeper is a no-op implementation of BridgeKeeper for public builds.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper creates a new stub bridge keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{
		logger: logger.With("module", types.ModuleName),
	}
}

func (k *StubKeeper) Logger() log.Logger { return k.logger }
func (k *StubKeeper) GetConfig(_ sdk.Context) types.BridgeConfig {
	return types.DefaultBridgeConfig()
}
func (k *StubKeeper) SetConfig(_ sdk.Context, _ types.BridgeConfig) error       { return nil }
func (k *StubKeeper) GetChainConfig(_ sdk.Context, _ string) (types.ChainConfig, bool) {
	return types.ChainConfig{}, false
}
func (k *StubKeeper) SetChainConfig(_ sdk.Context, _ types.ChainConfig) error    { return nil }
func (k *StubKeeper) GetAllChainConfigs(_ sdk.Context) []types.ChainConfig       { return nil }
func (k *StubKeeper) GetBridgeValidator(_ sdk.Context, _ string) (types.BridgeValidator, bool) {
	return types.BridgeValidator{}, false
}
func (k *StubKeeper) SetBridgeValidator(_ sdk.Context, _ types.BridgeValidator) error { return nil }
func (k *StubKeeper) GetAllBridgeValidators(_ sdk.Context) []types.BridgeValidator    { return nil }
func (k *StubKeeper) GetActiveValidatorsForChain(_ sdk.Context, _ string) []types.BridgeValidator {
	return nil
}
func (k *StubKeeper) GetOperation(_ sdk.Context, _ string) (types.BridgeOperation, bool) {
	return types.BridgeOperation{}, false
}
func (k *StubKeeper) SetOperation(_ sdk.Context, _ types.BridgeOperation) error { return nil }
func (k *StubKeeper) GetAllOperations(_ sdk.Context) []types.BridgeOperation    { return nil }
func (k *StubKeeper) NextOperationID(_ sdk.Context) string                       { return "op-0" }
func (k *StubKeeper) GetLockedAmount(_ sdk.Context, _, _ string) types.LockedAmount {
	return types.LockedAmount{}
}
func (k *StubKeeper) SetLockedAmount(_ sdk.Context, _ types.LockedAmount) error { return nil }
func (k *StubKeeper) GetAllLockedAmounts(_ sdk.Context) []types.LockedAmount    { return nil }
func (k *StubKeeper) GetCircuitBreaker(_ sdk.Context, _ string) types.CircuitBreakerState {
	return types.CircuitBreakerState{}
}
func (k *StubKeeper) SetCircuitBreaker(_ sdk.Context, _ types.CircuitBreakerState) error { return nil }
func (k *StubKeeper) GetAllCircuitBreakers(_ sdk.Context) []types.CircuitBreakerState    { return nil }
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState)                    {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
