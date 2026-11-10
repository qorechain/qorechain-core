package types

// v3.2.8 RDK data posting (backlog C2) and based sequencing (backlog C4).
// Design: handoff-v328/RDK-DATA-BASED-DESIGN.md. Below the v3.2.8 plan height
// both are off: the new messages are refused, a message that carries one of
// the new fields is refused, EndBlock does no new work and no new key is
// written, so history replays byte-identically.
const (
	// FeatureRDKPostRollupData serves MsgPostRollupData. The payload stays in
	// the transaction; the chain stores a commitment (SHA-256 of the data,
	// size, poster, height) under the rollup's next inbox sequence and
	// extends the rollup's inbox accumulator. Posting is limited per message
	// and per block, costs a fee per billable byte (at least 1 KiB per post)
	// paid into the fee collector, and commitments are pruned in EndBlock
	// after posted_data_retention_blocks. A batch of a native-DA rollup may
	// reference a retained post by its data_hash. Only the rollup's sequencer
	// (its creator when no sequencer address is set) may post, except on a
	// based rollup, where anyone may. MsgUpdateParams may set the four
	// posted-data parameters from the switch.
	FeatureRDKPostRollupData = "rdk-post-rollup-data"

	// FeatureRDKBasedSequencing serves based sequencing: MsgCreateRollup may
	// set sequencing "based" and governance may move a rollup with
	// MsgSetRollupSequencing. A based rollup's batch must consume its inbox in
	// order (MsgSubmitBatch.inbox_end, data_hash = the inbox range
	// commitment). A zk rollup with a Groth16 (snark) key accepts such batches
	// from any account, since the proof binds the range, the roots and the
	// withdrawals; a based-settlement rollup (no proof) keeps its sequencer as
	// the only submitter. Optimistic and STARK rollups cannot be based in this
	// release. Requires FeatureRDKPostRollupData.
	FeatureRDKBasedSequencing = "rdk-based-sequencing"
)

// V328RDKDataFeatures are the v3.2.8 switches for RDK data posting and based
// sequencing.
var V328RDKDataFeatures = []string{
	FeatureRDKPostRollupData,
	FeatureRDKBasedSequencing,
}
