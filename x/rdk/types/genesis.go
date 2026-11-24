package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Params holds module-level parameters for the RDK module.
type Params struct {
	MaxRollups             uint32 `json:"max_rollups"`
	MinStakeForRollup      int64  `json:"min_stake_for_rollup"`      // uqor
	RollupCreationBurnRate string `json:"rollup_creation_burn_rate"` // decimal (e.g. "0.01" = 1%)
	DefaultChallengeWindow uint64 `json:"default_challenge_window"`  // seconds
	MaxDABlobSize          uint64 `json:"max_da_blob_size"`          // bytes
	BlobRetentionBlocks    uint64 `json:"blob_retention_blocks"`
	MaxBatchesPerBlock     uint32 `json:"max_batches_per_block"`
	// FinalizedBatchRetentionBlocks is how many blocks a finalized batch
	// keeps its proof before EndBlock compacts it (from v3.2.8,
	// FeatureRDKBatchCompaction). Zero means DefaultFinalizedBatchRetentionBlocks.
	// It is omitted when zero, so a parameter record written before v3.2.8, and
	// every genesis file without the field, keeps its stored bytes.
	FinalizedBatchRetentionBlocks uint64 `json:"finalized_batch_retention_blocks,omitempty"`

	// The posted-data parameters (from v3.2.8, FeatureRDKPostRollupData).
	// Zero means the default for each, and each is omitted when zero, so a
	// parameter record written before v3.2.8 keeps its stored bytes.
	//
	// MaxPostedDataBytes caps one MsgPostRollupData payload
	// (DefaultMaxPostedDataBytes).
	MaxPostedDataBytes uint64 `json:"max_posted_data_bytes,omitempty"`
	// MaxPostedDataBytesPerBlock caps the billable bytes posted in one block,
	// all rollups together (DefaultMaxPostedDataBytesPerBlock).
	MaxPostedDataBytesPerBlock uint64 `json:"max_posted_data_bytes_per_block,omitempty"`
	// PostedDataFeePerByte is the uqor fee per billable byte
	// (DefaultPostedDataFeePerByte).
	PostedDataFeePerByte uint64 `json:"posted_data_fee_per_byte,omitempty"`
	// PostedDataRetentionBlocks is how many blocks a posted-data commitment
	// stays in state (DefaultPostedDataRetentionBlocks).
	PostedDataRetentionBlocks uint64 `json:"posted_data_retention_blocks,omitempty"`
	// MinSubmitterBond is the uqor bond a qledger-v1 rollup's submitter must
	// hold (v3.2.8, DefaultMinSubmitterBond when zero).
	MinSubmitterBond uint64 `json:"min_submitter_bond,omitempty"`
}

// Posted-data defaults and bounds (v3.2.8, FeatureRDKPostRollupData).
const (
	// DefaultMaxPostedDataBytes is 128 KiB per post.
	DefaultMaxPostedDataBytes uint64 = 128 << 10
	// DefaultMaxPostedDataBytesPerBlock is 512 KiB of billable bytes per block.
	DefaultMaxPostedDataBytesPerBlock uint64 = 512 << 10
	// DefaultPostedDataFeePerByte is 10 uqor per billable byte (10 QOR per MiB).
	DefaultPostedDataFeePerByte uint64 = 10
	// DefaultPostedDataRetentionBlocks is 7.5 days at 1 s blocks (about 24
	// days at 3.2 s), longer than the default 7-day challenge window on
	// every network (PostedDataRetentionFor).
	DefaultPostedDataRetentionBlocks uint64 = 648000

	// MaxPostedDataHardLimit is the ceiling of max_posted_data_bytes and the
	// stateless limit MsgPostRollupData.ValidateBasic applies.
	MaxPostedDataHardLimit uint64 = 1 << 20
	// MaxPostedDataPerBlockLimit is the ceiling of
	// max_posted_data_bytes_per_block.
	MaxPostedDataPerBlockLimit uint64 = 8 << 20
	// MaxPostedDataFeePerByte bounds the fee so a fee never overflows.
	MaxPostedDataFeePerByte uint64 = 1_000_000
	// MinPostedDataRetentionBlocks is the shortest non-zero retention.
	MinPostedDataRetentionBlocks uint64 = 100
	// MaxPostedDataRetentionBlocks is the longest retention: ten times the
	// longest challenge window (MaxChallengeWindowSec, 30 days) counted at
	// 1 s blocks, 25.92M blocks (300 days at 1 s). Every window in force is
	// covered with room to spare, and height + retention, the expiry the
	// keeper stores as an int64, cannot overflow (confirmation review 2, I3).
	MaxPostedDataRetentionBlocks uint64 = 10 * MaxChallengeWindowSec
	// MinBillablePostBytes is what a post is billed and counted at least,
	// so many tiny posts cannot fill the commitment store cheaply.
	MinBillablePostBytes uint64 = 1024
	// PostedDataGasPerByte is the gas the handler charges per billable byte
	// on top of the transaction's own size and store gas.
	PostedDataGasPerByte uint64 = 20
)

// PostedDataMaxBytes returns the per-post cap, resolving zero to the default.
func (p Params) PostedDataMaxBytes() uint64 {
	if p.MaxPostedDataBytes == 0 {
		return DefaultMaxPostedDataBytes
	}
	return p.MaxPostedDataBytes
}

// PostedDataMaxBytesPerBlock returns the per-block cap, resolving zero to the
// default.
func (p Params) PostedDataMaxBytesPerBlock() uint64 {
	if p.MaxPostedDataBytesPerBlock == 0 {
		return DefaultMaxPostedDataBytesPerBlock
	}
	return p.MaxPostedDataBytesPerBlock
}

// PostedDataFee returns the fee per billable byte, resolving zero to the
// default.
func (p Params) PostedDataFee() uint64 {
	if p.PostedDataFeePerByte == 0 {
		return DefaultPostedDataFeePerByte
	}
	return p.PostedDataFeePerByte
}

// PostedDataRetention returns the commitment retention in blocks, resolving
// zero to the default.
func (p Params) PostedDataRetention() uint64 {
	if p.PostedDataRetentionBlocks == 0 {
		return DefaultPostedDataRetentionBlocks
	}
	return p.PostedDataRetentionBlocks
}

// PostedDataRetentionFor is the shortest posted-data retention, in blocks,
// that covers a challenge window of windowSec seconds. A block is counted as
// at least one second, the shortest block time any QoreChain network runs
// (the testnet's), so the commitments of a batch's range are still in state
// for as long as the batch can be challenged.
func PostedDataRetentionFor(windowSec uint64) uint64 {
	return windowSec
}

// CheckPostedDataRetention refuses a parameter set whose posted-data
// retention is shorter than a challenge window of windowSec seconds
// (PostedDataRetentionFor). From v3.2.8 MsgUpdateParams checks it against
// the longest window in force: default_challenge_window and the window of
// every rollup that is not stopped.
func (p Params) CheckPostedDataRetention(windowSec uint64) error {
	if need := PostedDataRetentionFor(windowSec); p.PostedDataRetention() < need {
		return ErrInvalidParams.Wrapf("posted_data_retention_blocks %d is shorter than the challenge window (%d s): it must be at least %d",
			p.PostedDataRetention(), windowSec, need)
	}
	return nil
}

// HasPostedDataParams reports whether any posted-data parameter is set.
func (p Params) HasPostedDataParams() bool {
	return p.MaxPostedDataBytes != 0 || p.MaxPostedDataBytesPerBlock != 0 ||
		p.PostedDataFeePerByte != 0 || p.PostedDataRetentionBlocks != 0
}

// BillableBytes is what a post of n bytes is billed and counted at.
func BillableBytes(n uint64) uint64 {
	if n < MinBillablePostBytes {
		return MinBillablePostBytes
	}
	return n
}

// DefaultFinalizedBatchRetentionBlocks is the retention a zero
// finalized_batch_retention_blocks stands for: about 30 days at 6 s blocks,
// the same as blob_retention_blocks.
const DefaultFinalizedBatchRetentionBlocks uint64 = 432000

// MinFinalizedBatchRetentionBlocks is the shortest non-zero retention a
// parameter update may set.
const MinFinalizedBatchRetentionBlocks uint64 = 100

// FinalizedBatchRetention returns the retention in blocks, resolving zero to
// the default.
func (p Params) FinalizedBatchRetention() uint64 {
	if p.FinalizedBatchRetentionBlocks == 0 {
		return DefaultFinalizedBatchRetentionBlocks
	}
	return p.FinalizedBatchRetentionBlocks
}

// DefaultParams returns sensible default parameters.
func DefaultParams() Params {
	return Params{
		MaxRollups:             100,
		MinStakeForRollup:      10000000000, // 10,000 QOR in uqor
		RollupCreationBurnRate: "0.01",      // 1%
		DefaultChallengeWindow: 604800,      // 7 days in seconds
		MaxDABlobSize:          2097152,     // 2 MB
		BlobRetentionBlocks:    432000,      // ~30 days at 6s blocks
		MaxBatchesPerBlock:     10,
	}
}

// Bounds on a governance parameter update (ValidateUpdate). They keep a
// proposal from switching the module off by accident; they are not policy.
const (
	// MinChallengeWindowSec is the shortest default challenge window: one
	// minute, so a watcher always has at least a few blocks to challenge.
	MinChallengeWindowSec uint64 = 60
	// MaxChallengeWindowSec is the longest: 30 days.
	MaxChallengeWindowSec uint64 = 30 * 24 * 3600
	// MaxDABlobSizeLimit caps max_da_blob_size at 8 MiB, under any block size
	// the chain runs with.
	MaxDABlobSizeLimit uint64 = 8 << 20
)

// ValidateUpdate checks a parameter set proposed through MsgUpdateParams
// (from v3.2.5, FeatureRDKParamsRoute). It is stricter than the genesis check,
// which existing genesis files must keep passing.
func (p Params) ValidateUpdate() error {
	if p.MaxRollups == 0 {
		return ErrInvalidParams.Wrap("max_rollups must be positive")
	}
	if p.MinStakeForRollup <= 0 {
		return ErrInvalidParams.Wrap("min_stake_for_rollup must be positive")
	}
	rate, err := math.LegacyNewDecFromStr(p.RollupCreationBurnRate)
	if err != nil || rate.IsNegative() || rate.GTE(math.LegacyOneDec()) {
		return ErrInvalidParams.Wrapf("rollup_creation_burn_rate must be a decimal in [0, 1), got %q", p.RollupCreationBurnRate)
	}
	if p.DefaultChallengeWindow < MinChallengeWindowSec || p.DefaultChallengeWindow > MaxChallengeWindowSec {
		return ErrInvalidParams.Wrapf("default_challenge_window must be %d..%d seconds, got %d",
			MinChallengeWindowSec, MaxChallengeWindowSec, p.DefaultChallengeWindow)
	}
	if p.MaxDABlobSize == 0 || p.MaxDABlobSize > MaxDABlobSizeLimit {
		return ErrInvalidParams.Wrapf("max_da_blob_size must be 1..%d bytes, got %d", MaxDABlobSizeLimit, p.MaxDABlobSize)
	}
	if p.BlobRetentionBlocks == 0 {
		return ErrInvalidParams.Wrap("blob_retention_blocks must be positive")
	}
	if p.MaxBatchesPerBlock == 0 {
		return ErrInvalidParams.Wrap("max_batches_per_block must be positive")
	}
	if r := p.FinalizedBatchRetentionBlocks; r != 0 && r < MinFinalizedBatchRetentionBlocks {
		return ErrInvalidParams.Wrapf("finalized_batch_retention_blocks must be 0 (default) or at least %d, got %d",
			MinFinalizedBatchRetentionBlocks, r)
	}
	if err := p.validatePostedData(); err != nil {
		return err
	}
	return p.validateModes()
}

// validatePostedData checks the posted-data parameters. Zero is the default
// for each.
func (p Params) validatePostedData() error {
	if n := p.MaxPostedDataBytes; n > MaxPostedDataHardLimit {
		return ErrInvalidParams.Wrapf("max_posted_data_bytes must be 0 (default) or at most %d, got %d",
			MaxPostedDataHardLimit, n)
	}
	if n := p.MaxPostedDataBytesPerBlock; n != 0 {
		if n > MaxPostedDataPerBlockLimit || n < p.PostedDataMaxBytes() {
			return ErrInvalidParams.Wrapf("max_posted_data_bytes_per_block must be 0 (default) or %d..%d, got %d",
				p.PostedDataMaxBytes(), MaxPostedDataPerBlockLimit, n)
		}
	} else if p.PostedDataMaxBytes() > DefaultMaxPostedDataBytesPerBlock {
		return ErrInvalidParams.Wrapf("max_posted_data_bytes %d is above the default per-block cap %d; set max_posted_data_bytes_per_block",
			p.PostedDataMaxBytes(), DefaultMaxPostedDataBytesPerBlock)
	}
	if n := p.PostedDataFeePerByte; n > MaxPostedDataFeePerByte {
		return ErrInvalidParams.Wrapf("posted_data_fee_per_byte must be 0 (default) or at most %d, got %d",
			MaxPostedDataFeePerByte, n)
	}
	if r := p.PostedDataRetentionBlocks; r != 0 && (r < MinPostedDataRetentionBlocks || r > MaxPostedDataRetentionBlocks) {
		return ErrInvalidParams.Wrapf("posted_data_retention_blocks must be 0 (default) or %d..%d, got %d",
			MinPostedDataRetentionBlocks, MaxPostedDataRetentionBlocks, r)
	}
	return nil
}

// GenesisState defines the rdk module genesis state.
type GenesisState struct {
	Params  Params            `json:"params"`
	Rollups []RollupConfig    `json:"rollups"`
	Batches []SettlementBatch `json:"batches"`
	// EscrowLedger carries each rollup's share of the pooled module escrow.
	// Before v3.2.8 it was not exported, so an export and re-import left every
	// rollup with no ledger entry. Omitted when empty, so genesis files without
	// it read and export as before.
	EscrowLedger []RollupEscrowEntry `json:"escrow_ledger,omitempty"`
	// PostedData and Inboxes carry the v3.2.8 posted-data commitments and
	// inbox states (FeatureRDKPostRollupData). Omitted when empty, so genesis
	// files without them read and export as before.
	PostedData []PostedDataRecord `json:"posted_data,omitempty"`
	Inboxes    []InboxState       `json:"inboxes,omitempty"`
	// SubmitterBonds and DepositPools carry the v3.2.8 qledger-v1 state
	// (FeatureRDKOptimisticReexec). Omitted when empty.
	SubmitterBonds []SubmitterBond     `json:"submitter_bonds,omitempty"`
	DepositPools   []RollupEscrowEntry `json:"deposit_pools,omitempty"`
}

// RollupEscrowEntry is one rollup's escrow ledger entry. An entry with amount
// 0 is a drained ledger, which is not the same as no entry.
type RollupEscrowEntry struct {
	RollupID string `json:"rollup_id"`
	Amount   uint64 `json:"amount,string"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:  DefaultParams(),
		Rollups: []RollupConfig{},
		Batches: []SettlementBatch{},
	}
}

// Validate validates the genesis state.
func (gs GenesisState) Validate() error {
	if gs.Params.MaxRollups == 0 {
		return fmt.Errorf("max rollups must be positive")
	}
	if gs.Params.MinStakeForRollup <= 0 {
		return fmt.Errorf("min stake for rollup must be positive")
	}
	if gs.Params.MaxDABlobSize == 0 {
		return fmt.Errorf("max DA blob size must be positive")
	}
	if gs.Params.MaxBatchesPerBlock == 0 {
		return fmt.Errorf("max batches per block must be positive")
	}
	rollups := make(map[string]bool, len(gs.Rollups))
	for i, r := range gs.Rollups {
		if err := r.Validate(); err != nil {
			return fmt.Errorf("rollup %d: %w", i, err)
		}
		rollups[r.RollupID] = true
	}
	seen := make(map[string]bool, len(gs.EscrowLedger))
	for i, e := range gs.EscrowLedger {
		if !rollups[e.RollupID] {
			return fmt.Errorf("escrow ledger entry %d: rollup %q is not in genesis", i, e.RollupID)
		}
		if seen[e.RollupID] {
			return fmt.Errorf("escrow ledger entry %d: duplicate rollup %q", i, e.RollupID)
		}
		seen[e.RollupID] = true
	}
	if err := gs.Params.validatePostedData(); err != nil {
		return err
	}
	if err := gs.Params.validateModes(); err != nil {
		return err
	}
	byID := make(map[string]RollupConfig, len(gs.Rollups))
	for _, r := range gs.Rollups {
		byID[r.RollupID] = r
	}
	if err := validateGenesisModes(byID, gs.SubmitterBonds, gs.DepositPools); err != nil {
		return err
	}
	return validateGenesisInboxes(rollups, gs.Inboxes, gs.PostedData)
}

// validateGenesisInboxes checks the posted-data commitments and inbox states
// against the genesis rollups.
func validateGenesisInboxes(rollups map[string]bool, inboxes []InboxState, records []PostedDataRecord) error {
	heads := make(map[string]uint64, len(inboxes))
	for i, ib := range inboxes {
		if !rollups[ib.RollupID] {
			return fmt.Errorf("inbox %d: rollup %q is not in genesis", i, ib.RollupID)
		}
		if _, dup := heads[ib.RollupID]; dup {
			return fmt.Errorf("inbox %d: duplicate rollup %q", i, ib.RollupID)
		}
		if ib.ConsumedSequence > ib.HeadSequence {
			return fmt.Errorf("inbox %d: consumed %d is above head %d", i, ib.ConsumedSequence, ib.HeadSequence)
		}
		if err := checkAcc(ib.HeadAcc); err != nil {
			return fmt.Errorf("inbox %d head: %w", i, err)
		}
		if err := checkAcc(ib.ConsumedAcc); err != nil {
			return fmt.Errorf("inbox %d consumed: %w", i, err)
		}
		heads[ib.RollupID] = ib.HeadSequence
	}
	type key struct {
		id  string
		seq uint64
	}
	seen := make(map[key]bool, len(records))
	for i, r := range records {
		head, ok := heads[r.RollupID]
		if !ok {
			return fmt.Errorf("posted data %d: rollup %q has no inbox in genesis", i, r.RollupID)
		}
		if r.Sequence == 0 || r.Sequence > head {
			return fmt.Errorf("posted data %d: sequence %d outside 1..%d", i, r.Sequence, head)
		}
		if seen[key{r.RollupID, r.Sequence}] {
			return fmt.Errorf("posted data %d: duplicate sequence %d of rollup %q", i, r.Sequence, r.RollupID)
		}
		seen[key{r.RollupID, r.Sequence}] = true
		if len(r.DataHash) != 32 || len(r.InboxAcc) != 32 {
			return fmt.Errorf("posted data %d: data_hash and inbox_acc must be 32 bytes", i)
		}
	}
	return nil
}

func checkAcc(acc []byte) error {
	if len(acc) != 0 && len(acc) != 32 {
		return fmt.Errorf("accumulator is %d bytes, want 32 or empty", len(acc))
	}
	return nil
}
