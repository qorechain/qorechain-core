package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgCreateAbstractAccount and MsgUpdateSpendingRules are generated from
// proto/qorechain/abstractaccount/v1/tx.proto (see tx.pb.go). The methods
// below are attached to those generated types.

func (msg MsgCreateAbstractAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	switch msg.AccountType {
	case "multisig", "social_recovery", "session_based":
		return nil
	default:
		return fmt.Errorf("invalid account type: %s", msg.AccountType)
	}
}

func (msg MsgCreateAbstractAccount) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (msg MsgUpdateSpendingRules) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.AccountAddress == "" {
		return fmt.Errorf("account address cannot be empty")
	}
	for _, rule := range msg.Rules {
		if rule.DailyLimit < 0 {
			return fmt.Errorf("daily limit must be non-negative")
		}
		if rule.PerTxLimit < 0 {
			return fmt.Errorf("per-tx limit must be non-negative")
		}
	}
	return nil
}

func (msg MsgUpdateSpendingRules) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}
