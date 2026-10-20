package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	licensemod "github.com/qorechain/qorechain-core/x/license"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// wireModuleGates connects the v3.2.2 module fixes that live in this
// repository's own packages to the protocol-feature record, in both builds.
// (The fixes in the full build's module keepers are wired by
// wireDependencyGates.) Each switch answers per block from state, so every
// node flips at the plan height.
func (app *QoreChainApp) wireModuleGates() {
	GovProposerRewardCapEnabled = func(ctx sdk.Context) bool {
		return app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureGovProposerRewardCap)
	}
	licensemod.AuthorityUpdateEnabled = func(ctx sdk.Context) bool {
		return app.PQCKeeper.ProtocolFeatureEnabled(ctx, pqctypes.FeatureLicenseAuthorityUpdate)
	}
}
