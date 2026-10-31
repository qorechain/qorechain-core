package types

// v3.2.5 RDK area. Below the v3.2.5 plan height every one of these is off, so
// history replays byte-identically.
const (
	// FeatureRDKChallengeSettlement makes an optimistic batch final only in
	// order and only on an undisputed chain of batches, and makes the fraud
	// challenge resolvable by anyone:
	//   - a batch finalizes when its own challenge window has passed AND the
	//     batch before it is final; a challenged batch holds back every batch
	//     after it, so no withdrawal can be paid against a state built on a
	//     disputed root;
	//   - MsgResolveChallenge may be sent by any account (the outcome was
	//     already computed on chain from the stored fraud proof; only the
	//     rollup creator, the accused party, could trigger it before);
	//   - an upheld fraud rejects the batch and every later batch that is not
	//     final, refunds the bonds of open challenges on those, and moves the
	//     latest-batch pointer back, so the sequencer resubmits from the last
	//     good state root (a rejected batch record may be overwritten);
	//   - a rollup with a submitted or challenged batch cannot be stopped, so a
	//     creator cannot take the escrow back ahead of a challenge;
	//   - optimistic and based auto-finalization walk a per-rollup cursor
	//     instead of every batch the rollup ever settled.
	FeatureRDKChallengeSettlement = "rdk-challenge-settlement"

	// FeatureRDKRollupParams makes two rdk parameters mean what they say: a new
	// optimistic rollup takes its challenge window from
	// params.default_challenge_window (it was the preset's fixed 7 days), and
	// a stopped rollup no longer holds one of params.max_rollups slots.
	FeatureRDKRollupParams = "rdk-rollup-params"

	// FeatureRDKParamsRoute serves MsgUpdateParams for x/rdk, governance only.
	// Before it the module had no way to change its parameters.
	FeatureRDKParamsRoute = "rdk-params-route"

	// FeatureRDKWithdrawalsBinding binds a batch's withdrawals root, and the
	// chain id, into what settles the batch, for batches submitted from the
	// switch (SettlementBatch.WithdrawalsBound):
	//   - the optimistic transition binding becomes
	//     H("qorechain-rdk-transition/v2", chain id, rollup id, prev root,
	//     data hash, tx count, withdrawals root), length-prefixed, so a batch
	//     that publishes the honest root for one withdrawals list and commits
	//     another is provably fraudulent;
	//   - a SNARK proof takes the withdrawals root as its fourth public input,
	//     and a STARK proof's transcript label carries it, so a validity proof
	//     cannot be reused with another withdrawals list;
	//   - a withdrawal leaf starts with the chain id, so a withdrawal proved on
	//     one network cannot be executed on another, and replay protection
	//     checks the leaf in both forms;
	//   - a withdrawals root is either empty or exactly 32 bytes.
	FeatureRDKWithdrawalsBinding = "rdk-withdrawals-binding"
)

// V325RDKFeatures is this area's share of the v3.2.5 release set. Every
// consensus change of the area is guarded by one of these switches, so below
// the v3.2.5 plan height history replays byte-identically.
var V325RDKFeatures = []string{
	FeatureRDKChallengeSettlement,
	FeatureRDKRollupParams,
	FeatureRDKParamsRoute,
	FeatureRDKWithdrawalsBinding,
}
