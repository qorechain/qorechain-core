package types

import (
	"fmt"
	"time"

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
	for i, rule := range msg.Rules {
		if rule.DailyLimit < 0 {
			return fmt.Errorf("daily limit must be non-negative")
		}
		if rule.PerTxLimit < 0 {
			return fmt.Errorf("per-tx limit must be non-negative")
		}
		// An ENABLED rule that constrains nothing is refused.
		//
		// Zero means "no bound on this dimension", which is a useful thing to be
		// able to say for one dimension at a time. A rule that is switched on and
		// says it for every dimension at once enforces nothing at all, while
		// appearing in every query and every interface as an active spending limit.
		// Whoever configured it believed they had set a limit. Refusing it is the
		// only way that belief and reality stay in step.
		if rule.Enabled && rule.DailyLimit == 0 && rule.PerTxLimit == 0 && len(rule.AllowedDenoms) == 0 {
			return fmt.Errorf(
				"rule %d is enabled but sets no per-tx limit, no daily limit and no "+
					"denom list, so it would permit everything while appearing to be a "+
					"spending limit; set at least one bound or disable it", i)
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

func (msg MsgRegisterAuthenticator) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.AccountAddress == "" {
		return fmt.Errorf("account address cannot be empty")
	}
	// Reuse the full authenticator policy check (scheme, key length, TTL, perms).
	return ValidateAuthenticator(Authenticator{
		Scheme:      msg.Scheme,
		PubKey:      msg.Pubkey,
		Permissions: msg.Permissions,
		Expiry:      time.Unix(msg.ExpiryUnix, 0).UTC(),
	})
}

func (msg MsgRegisterAuthenticator) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (msg MsgRevokeAuthenticator) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Owner); err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	if msg.AccountAddress == "" {
		return fmt.Errorf("account address cannot be empty")
	}
	if len(msg.Pubkey) == 0 || msg.Scheme == "" {
		return fmt.Errorf("scheme and pubkey are required")
	}
	return nil
}

func (msg MsgRevokeAuthenticator) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Owner)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}
