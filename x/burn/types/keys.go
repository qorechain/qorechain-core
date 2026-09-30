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
	// AMMPendingProtocolFeeKey holds the uqor the AMM has routed to the fee
	// collector as protocol fees since the last fee split (v3.2.2). The split
	// in the same block's EndBlocker consumes and deletes it, so it is empty
	// between blocks and in every export.
	AMMPendingProtocolFeeKey = []byte("burn/amm-pending-uqor")
)

// BondDenom is the chain's base denomination, the only one this module moves.
const BondDenom = "uqor"
