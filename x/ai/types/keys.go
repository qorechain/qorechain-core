package types

import "encoding/binary"

const (
	ModuleName = "ai"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ConfigKey       = []byte("ai/config")
	StatsKey        = []byte("ai/stats")
	FlaggedTxPrefix = []byte("ai/flagged/")

	// Phase 2 keys
	InvestigationPrefix         = []byte("ai/investigations/")
	FeeHistoryPrefix            = []byte("ai/fee-history/")
	NetworkRecommendationPrefix = []byte("ai/network-recommendations/")
	CircuitBreakerPrefix        = []byte("ai/circuit-breakers/")
	ExtendedStatsKey            = []byte("ai/extended-stats")
	TxHistoryPrefix             = []byte("ai/tx-history/")

	// FlaggedPruneCursorKey holds the next flagged-record key the retention
	// sweep looks at (ai-flagged-tx-retention). It sorts outside
	// FlaggedTxPrefix.
	FlaggedPruneCursorKey = []byte("ai/flagged-cursor")
)

const (
	// AIFlaggedRetentionBlocks is how long a flagged-transaction record is
	// kept from ai-flagged-tx-retention: about six days at five-second blocks.
	AIFlaggedRetentionBlocks int64 = 100_000
	// AIFlaggedPrunePerWrite is how many records the sweep looks at each time
	// a record is written, so the work per transaction stays bounded and the
	// records kept since genesis drain over time.
	AIFlaggedPrunePerWrite = 16
)

// HeightKey builds a store key by appending a big-endian encoded height
// to the given prefix, ensuring correct lexicographic ordering.
func HeightKey(prefix []byte, height int64) []byte {
	bz := make([]byte, len(prefix)+8)
	copy(bz, prefix)
	binary.BigEndian.PutUint64(bz[len(prefix):], uint64(height))
	return bz
}
