package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	_ sdk.Msg = &MsgUpdateConfig{}
	_ sdk.Msg = &MsgFundFloat{}
)

// ValidateBasic performs stateless checks.
func (m *MsgUpdateConfig) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address: %s", err)
	}
	return m.Config.ToConfig().Validate()
}

// ToConfig converts the wire form to the stored configuration.
func (c FeeConfig) ToConfig() GasAbstractionConfig {
	out := GasAbstractionConfig{
		Enabled:            c.Enabled,
		NativeDenom:        c.NativeDenom,
		AcceptedTokens:     make([]AcceptedFeeToken, 0, len(c.AcceptedTokens)),
		PremiumBps:         c.PremiumBps,
		MaxTipMultiplier:   c.MaxTipMultiplier,
		FloatMin:           c.FloatMin,
		BlockConversionCap: c.BlockConversionCap,
	}
	for _, t := range c.AcceptedTokens {
		out.AcceptedTokens = append(out.AcceptedTokens, AcceptedFeeToken{
			Denom:            t.Denom,
			ConversionRate:   t.ConversionRate,
			PriceSource:      t.PriceSource,
			RateExpiryHeight: t.RateExpiryHeight,
			PoolID:           t.PoolId,
			MinLiquidity:     t.MinLiquidity,
			MaxDeviationBps:  t.MaxDeviationBps,
		})
	}
	return out
}

// FeeConfigFrom converts the stored configuration to the wire form.
func FeeConfigFrom(c GasAbstractionConfig) FeeConfig {
	out := FeeConfig{
		Enabled:            c.Enabled,
		NativeDenom:        c.NativeDenom,
		AcceptedTokens:     make([]FeeToken, 0, len(c.AcceptedTokens)),
		PremiumBps:         c.PremiumBps,
		MaxTipMultiplier:   c.MaxTipMultiplier,
		FloatMin:           c.FloatMin,
		BlockConversionCap: c.BlockConversionCap,
	}
	for _, t := range c.AcceptedTokens {
		out.AcceptedTokens = append(out.AcceptedTokens, FeeToken{
			Denom:            t.Denom,
			ConversionRate:   t.ConversionRate,
			PriceSource:      t.PriceSource,
			RateExpiryHeight: t.RateExpiryHeight,
			PoolId:           t.PoolID,
			MinLiquidity:     t.MinLiquidity,
			MaxDeviationBps:  t.MaxDeviationBps,
		})
	}
	return out
}

// ValidateBasic performs stateless checks.
func (m *MsgFundFloat) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid sender address: %s", err)
	}
	if !m.Amount.IsValid() || !m.Amount.IsPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, "amount must be positive")
	}
	return nil
}
