package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgCreateAbstractAccount creates a new abstract account.
type MsgCreateAbstractAccount struct {
	Owner       string `json:"owner"`
	AccountType string `json:"account_type"` // multisig, social_recovery, session_based
}

func (msg MsgCreateAbstractAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	switch msg.AccountType {
	case "multisig", "social_recovery", "session_based":
		// valid
	default:
		return fmt.Errorf("invalid account type: %s", msg.AccountType)
	}
	return nil
}

func (msg MsgCreateAbstractAccount) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{addr}
}

// MsgUpdateSpendingRules updates spending rules for an abstract account.
type MsgUpdateSpendingRules struct {
	Owner         string         `json:"owner"`
	AccountAddress string        `json:"account_address"`
	Rules         []SpendingRule `json:"rules"`
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
	addr, _ := sdk.AccAddressFromBech32(msg.Owner)
	return []sdk.AccAddress{addr}
}
