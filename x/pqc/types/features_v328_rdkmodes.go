package types

// v3.2.8 RDK settlement modes: safe optimistic rollups, the STARK statement
// gate and private (encapsulated) rollups. Design:
// handoff-v328/RDK-DATA-BASED-DESIGN.md, sections 11-14. Below the v3.2.8 plan
// height all three are off and nothing new is accepted or written.
const (
	// FeatureRDKOptimisticReexec serves optimistic rollups with the
	// qledger-v1 execution VM: MsgCreateRollup.execution_vm, deposits into a
	// per-rollup pool (MsgDepositToRollup), bonded submitters
	// (MsgBondRollupSubmitter / MsgUnbondRollupSubmitter) and fraud proofs
	// decided at once by re-executing the whole batch on chain. Such a rollup
	// is based and anyone with a bond may submit; an upheld fraud proof
	// rejects the batch and its descendants, rolls the inbox back and
	// slashes the submitter's bond: half is burned, half goes to the
	// challenger, and the rejected batches no longer count against
	// max_batches_per_block. A batch consumes at least one inbox entry (an
	// idle rollup needs none), and every entry older than 20 blocks up to the
	// 64 a batch carries; one submitter has at most a quarter of the
	// rollup's 4,096 pending batches. Withdrawals are paid from the deposit
	// pool after the challenge window. Deposits, like posts, count against
	// the posted-data block cap; from this switch every post and deposit also
	// counts against a per-rollup and a per-poster share of it (a quarter of
	// the cap, at least one full-size post). An unfinalized qledger-v1
	// commitment is kept for at most 4 extra retention periods, none once the
	// rollup is stopped. Account keys are ML-DSA-44 only. From the switch no
	// new optimistic rollup is created without an execution VM (existing ones
	// are unchanged). Requires FeatureRDKPostRollupData and
	// FeatureRDKBasedSequencing.
	FeatureRDKOptimisticReexec = "rdk-optimistic-reexec"

	// FeatureRDKStarkStatement refuses the demonstration STARK AIR
	// (iterated squaring, "QSTK" keys): such a key cannot be set or moved
	// to, and a batch of a rollup on one is refused instead of settling on
	// a proof that binds neither its withdrawals nor its data.
	FeatureRDKStarkStatement = "rdk-stark-statement"

	// FeatureRDKPrivateRollups serves private rollups: MsgCreateRollup.privacy
	// "encapsulated" (zk with a Groth16 key only), MsgSetRollupMembers, and a
	// structure check on every post of such a rollup: an envelope whose
	// ML-KEM-1024 encapsulations name every current member's registered KEM
	// key. The chain never decrypts. A member's key is read from the KEM
	// registry at each post: after a member rotates its key, posts must
	// name the new one; earlier posts stay encrypted to the old key, which
	// the member keeps to read them. Pausing and resuming a based rollup
	// moves to governance under FeatureRDKBasedSequencing, not here.
	FeatureRDKPrivateRollups = "rdk-private-rollups"
)

// V328RDKModeFeatures are the v3.2.8 switches for RDK settlement modes.
var V328RDKModeFeatures = []string{
	FeatureRDKOptimisticReexec,
	FeatureRDKStarkStatement,
	FeatureRDKPrivateRollups,
}
