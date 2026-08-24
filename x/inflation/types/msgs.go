package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgUpdateParams{}

// Params converts the message into the module's parameter struct.
func (m MsgUpdateParams) ToParams() Params {
	tiers := make([]EmissionTier, 0, len(m.Schedule))
	for _, t := range m.Schedule {
		tiers = append(tiers, EmissionTier{Year: t.Year, InflationRate: t.InflationRate})
	}
	return Params{
		Schedule:           tiers,
		EpochLength:        m.EpochLength,
		Enabled:            m.Enabled,
		EmissionMode:       m.EmissionMode,
		FixedEpochEmission: OptionalInt(m.FixedEpochEmission),
		MaxTotalEmission:   OptionalInt(m.MaxTotalEmission),
	}
}

// ValidateBasic performs stateless checks.
func (m MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return fmt.Errorf("invalid authority address %q: %w", m.Authority, err)
	}
	return m.ToParams().Validate()
}
