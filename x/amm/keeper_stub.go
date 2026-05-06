//go:build !full

package amm

import (
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
)

// StubKeeper is the no-op AMM keeper used by community-edition builds.
// Every state-mutating method returns ErrNotImplemented; read methods
// return the type's zero value or DefaultParams. The chain still boots
// and runs — the AMM module simply exposes no functionality.
//
// Genesis import/export round-trips DefaultGenesisState() so a chain
// built without -tags full can still initialize and export an empty
// AMM section.
type StubKeeper struct {
	logger log.Logger
}

// NewStubKeeper constructs a no-op AMM keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{logger: logger}
}

// compile-time assertion that StubKeeper satisfies AMMKeeper.
var _ AMMKeeper = (*StubKeeper)(nil)

func (k *StubKeeper) Logger() log.Logger { return k.logger }

func (k *StubKeeper) GetParams(_ sdk.Context) ammtypes.Params {
	return ammtypes.DefaultParams()
}

func (k *StubKeeper) SetParams(_ sdk.Context, _ ammtypes.Params) error {
	return ammtypes.ErrNotImplemented
}

func (k *StubKeeper) CreatePool(_ sdk.Context, _ ammtypes.MsgCreatePool) (ammtypes.Pool, math.Int, error) {
	return ammtypes.Pool{}, math.ZeroInt(), ammtypes.ErrNotImplemented
}

func (k *StubKeeper) GetPool(_ sdk.Context, _ uint64) (ammtypes.Pool, bool) {
	return ammtypes.Pool{}, false
}

func (k *StubKeeper) GetPoolByDenoms(_ sdk.Context, _, _ string) (ammtypes.Pool, bool) {
	return ammtypes.Pool{}, false
}

func (k *StubKeeper) GetAllPools(_ sdk.Context) []ammtypes.Pool { return nil }

func (k *StubKeeper) PausePool(_ sdk.Context, _ ammtypes.MsgPausePool) error {
	return ammtypes.ErrNotImplemented
}

func (k *StubKeeper) ResumePool(_ sdk.Context, _ ammtypes.MsgResumePool) error {
	return ammtypes.ErrNotImplemented
}

func (k *StubKeeper) AddLiquidity(_ sdk.Context, _ ammtypes.MsgAddLiquidity) (math.Int, error) {
	return math.ZeroInt(), ammtypes.ErrNotImplemented
}

func (k *StubKeeper) RemoveLiquidity(_ sdk.Context, _ ammtypes.MsgRemoveLiquidity) (sdk.Coin, sdk.Coin, error) {
	return sdk.Coin{}, sdk.Coin{}, ammtypes.ErrNotImplemented
}

func (k *StubKeeper) SwapExactIn(_ sdk.Context, _ ammtypes.MsgSwapExactIn) (math.Int, error) {
	return math.ZeroInt(), ammtypes.ErrNotImplemented
}

func (k *StubKeeper) SwapExactOut(_ sdk.Context, _ ammtypes.MsgSwapExactOut) (math.Int, error) {
	return math.ZeroInt(), ammtypes.ErrNotImplemented
}

func (k *StubKeeper) GetLPBalance(_ sdk.Context, _ uint64, _ sdk.AccAddress) math.Int {
	return math.ZeroInt()
}

func (k *StubKeeper) GetLPSupply(_ sdk.Context, _ uint64) math.Int {
	return math.ZeroInt()
}

func (k *StubKeeper) QuoteExactIn(_ sdk.Context, _ uint64, _ string, _ math.Int) (math.Int, math.Int, error) {
	return math.ZeroInt(), math.ZeroInt(), ammtypes.ErrNotImplemented
}

func (k *StubKeeper) QuoteExactOut(_ sdk.Context, _ uint64, _ string, _ math.Int) (math.Int, math.Int, error) {
	return math.ZeroInt(), math.ZeroInt(), ammtypes.ErrNotImplemented
}

func (k *StubKeeper) SwapFromEVM(_ sdk.Context, _ sdk.AccAddress, _ string, _ math.Int, _ string, _ math.Int) (math.Int, error) {
	return math.ZeroInt(), ammtypes.ErrNotImplemented
}

func (k *StubKeeper) SuggestRoute(_ sdk.Context, _, _ string, _ math.Int) ammtypes.RouteSuggestion {
	return ammtypes.RouteSuggestion{}
}

func (k *StubKeeper) EndBlock(_ sdk.Context) error { return nil }

func (k *StubKeeper) InitGenesis(_ sdk.Context, _ ammtypes.GenesisState) {}

func (k *StubKeeper) ExportGenesis(_ sdk.Context) *ammtypes.GenesisState {
	return ammtypes.DefaultGenesisState()
}
