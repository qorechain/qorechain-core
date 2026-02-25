//go:build !proprietary

package app

func init() {
	ConfigureLanes = func() []LaneConfig {
		return []LaneConfig{
			{
				Name:          "default",
				Priority:      50,
				MaxBlockSpace: 1.0,
				Description:   "Default transaction lane",
			},
		}
	}
}
