package types

// QueryRoutingStatsResponse holds the QCAI routing statistics
type QueryRoutingStatsResponse struct {
	TotalRouted                       uint64 `json:"total_routed"`
	RoutedToMain                      uint64 `json:"routed_to_main"`
	RoutedToSidechains                uint64 `json:"routed_to_sidechains"`
	RoutedToPaychains                 uint64 `json:"routed_to_paychains"`
	AverageGasSavingsPercent          string `json:"average_gas_savings_percent"`
	AverageLatencyImprovementPercent  string `json:"average_latency_improvement_percent"`
}
