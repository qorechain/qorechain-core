package types

const (
	ModuleName = "burn"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

// MaxExportRecords is the upper bound on burn records exported in genesis.
const MaxExportRecords = 1_000_000

var (
	ParamsKey         = []byte("burn/params")
	TotalBurnedKey    = []byte("burn/total-burned")
	BurnRecordPrefix  = []byte("burn/records/")
	BurnStatsPrefix   = []byte("burn/stats/") // reserved for future per-source stats indexing
	MilestoneStateKey = []byte("burn/milestone-state")
	BlockTxCountKey   = []byte("burn/block-tx-count")
	// BlockFailedFeesKey accumulates, within one block, the uqor fees of the
	// transactions whose messages failed (FeatureBurnFailedTx). Written only
	// from the v3.2.2 plan height, emptied by the EndBlocker every block.
	BlockFailedFeesKey = []byte("burn/block-failed-fees")
)

// BondDenom is the chain's base denomination, the only one this module moves.
const BondDenom = "uqor"
