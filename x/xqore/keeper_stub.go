//go:build !proprietary

package xqore

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	rlconsensusmod "github.com/qorechain/qorechain-core/x/rlconsensus"
	"github.com/qorechain/qorechain-core/x/xqore/types"
)

// Compile-time assertion: StubKeeper satisfies rlconsensus.TokenomicsKeeper,
// replacing NilTokenomicsKeeper with real xQORE balance lookups.
var _ rlconsensusmod.TokenomicsKeeper = (*StubKeeper)(nil)

// StubKeeper is a no-op implementation of XQOREKeeper for public builds.
type StubKeeper struct {
	logger log.Logger
}

func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{logger: logger.With("module", types.ModuleName)}
}

func (k *StubKeeper) Logger() log.Logger                                             { return k.logger }
func (k *StubKeeper) GetParams(_ sdk.Context) types.Params                           { return types.DefaultParams() }
func (k *StubKeeper) SetParams(_ sdk.Context, _ types.Params) error                  { return nil }
func (k *StubKeeper) GetXQOREBalance(_ sdk.Context, _ sdk.AccAddress) math.Int       { return math.ZeroInt() }
func (k *StubKeeper) Lock(_ sdk.Context, _ sdk.AccAddress, _ math.Int) error         { return nil }
func (k *StubKeeper) Unlock(_ sdk.Context, _ sdk.AccAddress, _ math.Int) error       { return nil }
func (k *StubKeeper) GetPosition(_ sdk.Context, _ sdk.AccAddress) (types.XQOREPosition, bool) {
	return types.XQOREPosition{}, false
}
func (k *StubKeeper) GetAllPositions(_ sdk.Context) []types.XQOREPosition { return nil }
func (k *StubKeeper) GetTotalLocked(_ sdk.Context) math.Int               { return math.ZeroInt() }
func (k *StubKeeper) GetTotalXQORESupply(_ sdk.Context) math.Int          { return math.ZeroInt() }
func (k *StubKeeper) GetGovernanceMultiplier(_ sdk.Context, _ sdk.AccAddress) math.LegacyDec {
	return math.LegacyOneDec()
}
func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}
func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}
