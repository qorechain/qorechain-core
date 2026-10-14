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
	// The truncating conversion is the historical check: the field used to be
	// decoded straight into the uint8 AgentMode. ValidateBasic cannot tell the
	// height, so the exact range check on the wire value is in the handler,
	// behind FeatureRLParamBounds (ValidateModeRange).
	if !ValidAgentMode(m.AgentModeValue()) {
		return errorsmod.Wrapf(ErrInvalidAgentMode, "invalid mode: %d", m.Mode)
	}
	return nil
}

// AgentModeValue is the mode as the module stores it. For a value above 255
// it wraps, exactly as decoding into the uint8 AgentMode used to; callers that
// must not accept such a value check ValidateModeRange first.
func (m *MsgSetAgentMode) AgentModeValue() AgentMode {
	return AgentMode(m.Mode)
}

// ValidateModeRange checks the mode as it was sent on the wire, before any
// narrowing (QSR-2026-0434): 257 is refused, not read as 1.
func (m *MsgSetAgentMode) ValidateModeRange() error {
	if m.Mode > uint32(AgentModePaused) {
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
