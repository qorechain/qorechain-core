package types

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
