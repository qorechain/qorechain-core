//go:build proprietary

package app

func init() {
	ConfigureLanes = func() []LaneConfig {
		return []LaneConfig{
			{
				Name:          "pqc",
				Priority:      100,
				MaxBlockSpace: 0.15,
				Description:   "Post-quantum secured transactions (hybrid PQC signatures)",
			},
			{
				Name:          "mev",
				Priority:      90,
				MaxBlockSpace: 0.20,
				Description:   "MEV-protected transactions (FairBlock tIBE encrypted)",
			},
			{
				Name:          "ai",
				Priority:      80,
				MaxBlockSpace: 0.15,
				Description:   "AI-prioritized transactions (anomaly-scored, fee-optimized)",
			},
			{
				Name:          "default",
				Priority:      50,
				MaxBlockSpace: 0.40,
				Description:   "Standard transaction lane",
			},
			{
				Name:          "free",
				Priority:      10,
				MaxBlockSpace: 0.10,
				Description:   "Gas-abstracted and sponsored transactions",
			},
		}
	}
}
