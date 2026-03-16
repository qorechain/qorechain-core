package types

import "encoding/binary"

const (
	ModuleName = "ai"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ConfigKey        = []byte("ai/config")
	StatsKey         = []byte("ai/stats")
	FlaggedTxPrefix  = []byte("ai/flagged/")

	// Phase 2 keys
	InvestigationPrefix        = []byte("ai/investigations/")
	FeeHistoryPrefix           = []byte("ai/fee-history/")
	NetworkRecommendationPrefix = []byte("ai/network-recommendations/")
	CircuitBreakerPrefix       = []byte("ai/circuit-breakers/")
	ExtendedStatsKey           = []byte("ai/extended-stats")
)

// HeightKey builds a store key by appending a big-endian encoded height
// to the given prefix, ensuring correct lexicographic ordering.
func HeightKey(prefix []byte, height int64) []byte {
	bz := make([]byte, len(prefix)+8)
	copy(bz, prefix)
	binary.BigEndian.PutUint64(bz[len(prefix):], uint64(height))
	return bz
}
