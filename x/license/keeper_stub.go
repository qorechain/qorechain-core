//go:build !full

package license

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/license/types"
)

// StubKeeper is a no-op implementation for public builds.
type StubKeeper struct {
	logger    log.Logger
	authority string
}

// NewStubKeeper creates a new stub license keeper.
func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{logger: logger}
}

func (k *StubKeeper) Logger() log.Logger { return k.logger }

func (k *StubKeeper) GrantLicense(_ sdk.Context, _ string, _ types.License) error {
	return nil
}

func (k *StubKeeper) RevokeLicense(_ sdk.Context, _, _, _ string) error {
	return nil
}

func (k *StubKeeper) SuspendLicense(_ sdk.Context, _, _ string) error {
	return nil
}

func (k *StubKeeper) ResumeLicense(_ sdk.Context, _, _ string) error {
	return nil
}

func (k *StubKeeper) GetLicense(_ sdk.Context, _, _ string) (types.License, error) {
	return types.License{}, types.ErrLicenseNotFound
}

func (k *StubKeeper) GetLicenses(_ sdk.Context, _ string) ([]types.License, error) {
	return nil, nil
}

func (k *StubKeeper) GetLicenseHolders(_ sdk.Context, _ string) ([]types.License, error) {
	return nil, nil
}

func (k *StubKeeper) HasActiveLicense(_ sdk.Context, _, _ string) bool {
	return false
}

func (k *StubKeeper) GetAuthority() string { return k.authority }

func (k *StubKeeper) InitGenesis(_ sdk.Context, gs types.GenesisState) {
	k.authority = gs.Authority
}

func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesisState()
}

func (k *StubKeeper) EndBlocker(_ sdk.Context) error { return nil }
