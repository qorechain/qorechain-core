package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The rdk Msg types are generated from proto/qorechain/rdk/v1/tx.proto
// (see tx.pb.go). The ValidateBasic methods below are attached to them.

func (m *MsgCreateRollup) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	if m.StakeAmount <= 0 {
		return fmt.Errorf("stake_amount must be positive")
	}
	return nil
}

func (m *MsgSubmitBatch) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sequencer); err != nil {
		return fmt.Errorf("invalid sequencer address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	if len(m.StateRoot) == 0 {
		return fmt.Errorf("state_root cannot be empty")
	}
	return nil
}

func (m *MsgChallengeBatch) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	return nil
}

func (m *MsgPauseRollup) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	return nil
}

func (m *MsgResumeRollup) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	return nil
}

func (m *MsgStopRollup) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if m.RollupID == "" {
		return fmt.Errorf("rollup_id cannot be empty")
	}
	return nil
}
