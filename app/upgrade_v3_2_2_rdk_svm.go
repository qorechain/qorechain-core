package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	multilayertypes "github.com/qorechain/qorechain-core/x/multilayer/types"
)

// upgradeV3_2_2RDKSVM is this area's one-time state work at the v3.2.2 plan
// height. It changes no state.
//
// FeatureRDKIntegrity restricts NEW rollup and layer ids to the id grammar;
// ids already in state stay valid and keep working, and the keeper loops that
// used to bleed across ids now compare the decoded id, so an old id needs no
// rewrite. Both networks had no rollups and no layers when this release was
// cut; on any other chain an existing id outside the grammar is only logged
// here, for its operator. FeatureSVMBase58Canonical has no stored state.
func (app *QoreChainApp) upgradeV3_2_2RDKSVM(ctx sdk.Context) error {
	logger := ctx.Logger().With("upgrade", "v3.2.2", "area", "rdk-svm")
	if rollups, err := app.RDKKeeper.ListRollups(ctx); err == nil {
		for _, r := range rollups {
			if msg := multilayertypes.IDGrammarError(r.RollupID); msg != "" {
				logger.Info("existing rollup id is outside the v3.2.2 id grammar; it stays valid",
					"rollup_id", r.RollupID, "reason", msg)
			}
		}
	}
	if layers, err := app.MultilayerKeeper.GetAllLayers(ctx); err == nil {
		for _, l := range layers {
			if msg := multilayertypes.IDGrammarError(l.LayerID); msg != "" {
				logger.Info("existing layer id is outside the v3.2.2 id grammar; it stays valid",
					"layer_id", l.LayerID, "reason", msg)
			}
		}
	}
	return nil
}
