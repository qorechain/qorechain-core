package app

import (
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	reputationkeeper "github.com/qorechain/qorechain-core/x/reputation/keeper"
)

// ProvideReputationHooks installs the reputation keeper's gov and staking hooks
// through depinject, the only way to add hooks under the v0.53 flow: gov's and
// staking's invokers set the keepers' hooks during Build, once. The reputation
// keeper itself is built manually after Build, so what is supplied is the
// address of the app's field; nothing calls a hook before the app is built.
//
// Every hook is a no-op below the reputation-scoring switch, so blocks before
// it replay unchanged. They exist for reputation scoring v2: governance
// participation feeds ContributionScore, and a removed validator's record is
// deleted (QSR-2026-0030, -0312, -0145).
func ProvideReputationHooks(k *reputationkeeper.Keeper) (govtypes.GovHooksWrapper, stakingtypes.StakingHooksWrapper) {
	h := reputationkeeper.NewHooks(k)
	return govtypes.GovHooksWrapper{GovHooks: h}, stakingtypes.StakingHooksWrapper{StakingHooks: h}
}
