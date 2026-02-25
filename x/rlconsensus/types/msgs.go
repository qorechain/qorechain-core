package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ---------------------------------------------------------------------------
// MsgSetAgentMode
// ---------------------------------------------------------------------------

// MsgSetAgentMode changes the RL agent's operating mode.
type MsgSetAgentMode struct {
	Authority string    `json:"authority"`
	Mode      AgentMode `json:"mode"`
}

func (m *MsgSetAgentMode) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAgentMode, "invalid authority address: %s", err)
	}
	if !ValidAgentMode(m.Mode) {
		return errorsmod.Wrapf(ErrInvalidAgentMode, "invalid mode: %d", m.Mode)
	}
	return nil
}

func (m *MsgSetAgentMode) Reset()         { *m = MsgSetAgentMode{} }
func (m *MsgSetAgentMode) String() string { return fmt.Sprintf("MsgSetAgentMode{authority=%s, mode=%s}", m.Authority, m.Mode) }
func (m *MsgSetAgentMode) ProtoMessage()  {}

// ---------------------------------------------------------------------------
// MsgResumeAgent
// ---------------------------------------------------------------------------

// MsgResumeAgent resumes the RL agent from a paused state back to shadow mode.
type MsgResumeAgent struct {
	Authority string `json:"authority"`
}

func (m *MsgResumeAgent) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAgentMode, "invalid authority address: %s", err)
	}
	return nil
}

func (m *MsgResumeAgent) Reset()         { *m = MsgResumeAgent{} }
func (m *MsgResumeAgent) String() string { return fmt.Sprintf("MsgResumeAgent{authority=%s}", m.Authority) }
func (m *MsgResumeAgent) ProtoMessage()  {}

// ---------------------------------------------------------------------------
// MsgUpdatePolicy
// ---------------------------------------------------------------------------

// MsgUpdatePolicy replaces the current policy weights with a new set.
type MsgUpdatePolicy struct {
	Authority string       `json:"authority"`
	Weights   PolicyWeights `json:"weights"`
}

func (m *MsgUpdatePolicy) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidPolicyWeights, "invalid authority address: %s", err)
	}
	if err := m.Weights.Validate(); err != nil {
		return errorsmod.Wrapf(ErrInvalidPolicyWeights, "invalid weights: %s", err)
	}
	return nil
}

func (m *MsgUpdatePolicy) Reset()         { *m = MsgUpdatePolicy{} }
func (m *MsgUpdatePolicy) String() string { return fmt.Sprintf("MsgUpdatePolicy{authority=%s, epoch=%d}", m.Authority, m.Weights.Epoch) }
func (m *MsgUpdatePolicy) ProtoMessage()  {}

// ---------------------------------------------------------------------------
// MsgUpdateRewardWeights
// ---------------------------------------------------------------------------

// MsgUpdateRewardWeights changes the reward function weighting.
type MsgUpdateRewardWeights struct {
	Authority string       `json:"authority"`
	Weights   RewardWeights `json:"weights"`
}

func (m *MsgUpdateRewardWeights) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidRewardWeights, "invalid authority address: %s", err)
	}
	if err := m.Weights.Validate(); err != nil {
		return errorsmod.Wrapf(ErrInvalidRewardWeights, "invalid weights: %s", err)
	}
	return nil
}

func (m *MsgUpdateRewardWeights) Reset()         { *m = MsgUpdateRewardWeights{} }
func (m *MsgUpdateRewardWeights) String() string { return fmt.Sprintf("MsgUpdateRewardWeights{authority=%s}", m.Authority) }
func (m *MsgUpdateRewardWeights) ProtoMessage()  {}
