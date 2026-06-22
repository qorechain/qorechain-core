package types

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ---------------------------------------------------------------------------
// MsgSetAgentMode
// ---------------------------------------------------------------------------

func (m *MsgSetAgentMode) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAgentMode, "invalid authority address: %s", err)
	}
	if !ValidAgentMode(m.Mode) {
		return errorsmod.Wrapf(ErrInvalidAgentMode, "invalid mode: %d", m.Mode)
	}
	return nil
}

func (m *MsgSetAgentMode) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ---------------------------------------------------------------------------
// MsgResumeAgent
// ---------------------------------------------------------------------------

func (m *MsgResumeAgent) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAgentMode, "invalid authority address: %s", err)
	}
	return nil
}

func (m *MsgResumeAgent) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ---------------------------------------------------------------------------
// MsgUpdatePolicy
// ---------------------------------------------------------------------------

// PolicyWeightsFromJSON decodes the WeightsJson field into a PolicyWeights.
func (m *MsgUpdatePolicy) PolicyWeightsFromJSON() (PolicyWeights, error) {
	var w PolicyWeights
	if err := json.Unmarshal([]byte(m.WeightsJson), &w); err != nil {
		return PolicyWeights{}, err
	}
	return w, nil
}

func (m *MsgUpdatePolicy) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidPolicyWeights, "invalid authority address: %s", err)
	}
	w, err := m.PolicyWeightsFromJSON()
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidPolicyWeights, "invalid weights json: %s", err)
	}
	if err := w.Validate(); err != nil {
		return errorsmod.Wrapf(ErrInvalidPolicyWeights, "invalid weights: %s", err)
	}
	return nil
}

func (m *MsgUpdatePolicy) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// ---------------------------------------------------------------------------
// MsgUpdateRewardWeights
// ---------------------------------------------------------------------------

// RewardWeightsValue reconstructs the RewardWeights struct from the flat fields.
func (m *MsgUpdateRewardWeights) RewardWeightsValue() RewardWeights {
	return RewardWeights{
		Throughput:       m.Throughput,
		Finality:         m.Finality,
		Decentralization: m.Decentralization,
		MEV:              m.MEV,
		FailedTxs:        m.FailedTxs,
	}
}

func (m *MsgUpdateRewardWeights) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidRewardWeights, "invalid authority address: %s", err)
	}
	if err := m.RewardWeightsValue().Validate(); err != nil {
		return errorsmod.Wrapf(ErrInvalidRewardWeights, "invalid weights: %s", err)
	}
	return nil
}

func (m *MsgUpdateRewardWeights) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}
