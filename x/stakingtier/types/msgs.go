package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg = &MsgLockDelegation{}
	_ sdk.Msg = &MsgUnlockEarly{}
	_ sdk.Msg = &MsgClaimBonus{}
	_ sdk.Msg = &MsgUpdateParams{}
)

// ValidateBasic performs stateless checks.
func (m MsgLockDelegation) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Delegator); err != nil {
		return fmt.Errorf("invalid delegator address: %w", err)
	}
	if _, err := sdk.ValAddressFromBech32(m.Validator); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if m.Amount.IsNil() || !m.Amount.IsPositive() {
		return ErrInvalidAmount.Wrap("amount must be positive")
	}
	if !allowedMonths(m.Months) {
		return ErrInvalidTerm.Wrapf("months must be one of %v, got %d", AllowedLockMonths, m.Months)
	}
	return nil
}

// ValidateBasic performs stateless checks.
func (m MsgUnlockEarly) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Delegator); err != nil {
		return fmt.Errorf("invalid delegator address: %w", err)
	}
	if m.LockId == 0 {
		return ErrLockNotFound.Wrap("lock_id must be positive")
	}
	return nil
}

// ValidateBasic performs stateless checks.
func (m MsgClaimBonus) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Holder); err != nil {
		return fmt.Errorf("invalid holder address: %w", err)
	}
	return nil
}

// ToParams converts the message into the module's parameter struct.
func (m MsgUpdateParams) ToParams() Params {
	tiers := make([]LockTier, 0, len(m.LockTiers))
	for _, t := range m.LockTiers {
		tiers = append(tiers, LockTier{Months: t.Months, BonusBps: t.BonusBps})
	}
	return Params{
		LockTiers:             tiers,
		LightNodeBonusBps:     m.LightNodeBonusBps,
		ValidatorBonusBps:     m.ValidatorBonusBps,
		MaxTotalAPYBps:        m.MaxTotalApyBps,
		EarlyUnlockPenaltyBps: m.EarlyUnlockPenaltyBps,
		Enabled:               m.Enabled,
	}
}

// ValidateBasic performs stateless checks. Who signed is checked by the
// handler (govauth.Assert): the authority field alone proves nothing.
func (m MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address %q: %w", m.Authority, err)
	}
	return m.ToParams().Validate()
}
