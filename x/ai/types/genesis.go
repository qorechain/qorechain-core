package types

// GenesisState defines the ai module's genesis state.
type GenesisState struct {
	Config AIConfig `json:"config"`
	Stats  AIStats  `json:"stats"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Config: DefaultAIConfig(),
		Stats:  AIStats{},
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if gs.Config.AnomalyThreshold < 0 || gs.Config.AnomalyThreshold > 1 {
		return ErrInvalidConfig.Wrap("anomaly_threshold must be between 0 and 1")
	}
	if gs.Config.RiskThreshold < 0 || gs.Config.RiskThreshold > 1 {
		return ErrInvalidConfig.Wrap("risk_threshold must be between 0 and 1")
	}
	return nil
}
