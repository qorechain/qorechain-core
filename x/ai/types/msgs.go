package types

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgUpdateConfig{}

// ToConfig converts the message into the configuration it would store.
func (m *MsgUpdateConfig) ToConfig() (AIConfig, error) {
	anomaly, err := strconv.ParseFloat(m.AnomalyThreshold, 64)
	if err != nil {
		return AIConfig{}, ErrInvalidConfig.Wrapf("anomaly_threshold: %s", err)
	}
	risk, err := strconv.ParseFloat(m.RiskThreshold, 64)
	if err != nil {
		return AIConfig{}, ErrInvalidConfig.Wrapf("risk_threshold: %s", err)
	}
	cfg := AIConfig{
		AnomalyThreshold:  anomaly,
		RoutingStrategy:   m.RoutingStrategy,
		RiskThreshold:     risk,
		MaxTxPerMinute:    int(m.MaxTxPerMinute),
		MaxEVMTxPerMinute: int(m.MaxEvmTxPerMinute),
		AnomalyAction:     m.AnomalyAction,
	}
	return cfg, cfg.Validate()
}

// ValidateBasic performs stateless checks.
func (m *MsgUpdateConfig) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return ErrInvalidConfig.Wrapf("invalid authority address: %s", err)
	}
	_, err := m.ToConfig()
	return err
}
