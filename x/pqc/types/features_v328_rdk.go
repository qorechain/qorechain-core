package types

// v3.2.8 RDK area. Below the v3.2.8 plan height every one of these is off, so
// history replays byte-identically.
const (
	// FeatureRDKStopRefundLegacy makes StopRollup refund a rollup that has no
	// escrow ledger entry at all: a record from before the ledger existed, or
	// a rollup brought in by a genesis file that did not carry the ledger. Such
	// a rollup gets back what it escrowed, the smaller of its escrow figure and
	// its stake (the stake alone when the escrow figure is zero), the same rule
	// the v3.2.3 backfill used. A ledger entry that is present and reads 0 is a
	// drained ledger and still refunds nothing. Before it the v3.2.3 ledger
	// path refunded only the ledger, so such a rollup was stopped with nothing
	// returned and its escrow stayed in the module account for good.
	FeatureRDKStopRefundLegacy = "rdk-stop-refund-legacy"

	// FeatureRDKMaxBatchesPerBlock enforces params.max_batches_per_block: a
	// rollup can settle at most that many batches in one block, and the next
	// MsgSubmitBatch for it in the same block fails with ErrBatchLimitReached.
	// The count is per rollup, so one rollup filling its quota cannot keep the
	// others from settling. It is refused in block execution only: a CheckTx
	// or simulate context runs on the last committed block's height, where the
	// count is stale, so there the batch is counted but never refused. Before
	// it the parameter was validated and queried but nothing read it.
	FeatureRDKMaxBatchesPerBlock = "rdk-max-batches-per-block"

	// FeatureRDKBatchCompaction compacts finalized batches: in EndBlock, a
	// finalized batch whose finality is older than
	// params.finalized_batch_retention_blocks (0 means the default,
	// DefaultFinalizedBatchRetentionBlocks) loses its stored validity or fraud
	// proof, the bulk of the record (a STARK proof is about 230 KB). Every
	// field a withdrawal, the latest-batch pointer or a query needs is kept,
	// so withdrawals stay executable without a time limit. The walk follows a
	// per-rollup cursor in index order and is bounded per rollup and per
	// block; the rollups take turns (each block starts after the rollup the
	// previous one visited last), so a busy rollup cannot starve the others.
	// MsgUpdateParams may set the retention from the switch; before it
	// a non-zero value is refused. Before it no finalized batch was ever
	// pruned and the store grew by every proof ever settled.
	FeatureRDKBatchCompaction = "rdk-batch-compaction"
)

// V328RDKFeatures are the v3.2.8 switches for RDK escrow ledger, batch limits and pruning.
var V328RDKFeatures = []string{
	FeatureRDKStopRefundLegacy,
	FeatureRDKMaxBatchesPerBlock,
	FeatureRDKBatchCompaction,
}
