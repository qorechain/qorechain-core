package license

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/license/types"
)

// LicenseKeeper defines the interface for the license module keeper.
type LicenseKeeper interface {
	Logger() log.Logger

	// License management
	GrantLicense(ctx sdk.Context, caller string, license types.License) error
	RevokeLicense(ctx sdk.Context, caller string, grantee, featureID string) error
	SuspendLicense(ctx sdk.Context, grantee, featureID string) error
	ResumeLicense(ctx sdk.Context, grantee, featureID string) error

	// Queries
	GetLicense(ctx sdk.Context, grantee, featureID string) (types.License, error)
	GetLicenses(ctx sdk.Context, grantee string) ([]types.License, error)
	GetLicenseHolders(ctx sdk.Context, featureID string) ([]types.License, error)
	HasActiveLicense(ctx sdk.Context, grantee, featureID string) bool

	// Authority
	GetAuthority() string

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState

	// EndBlocker — expire licenses
	EndBlocker(ctx sdk.Context) error
}
