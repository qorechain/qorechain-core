package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// GasAbstractionConfig holds gas abstraction configuration.
type GasAbstractionConfig struct {
	Enabled        bool               `json:"enabled"`
	NativeDenom    string             `json:"native_denom"`
	AcceptedTokens []AcceptedFeeToken `json:"accepted_tokens"`
}

// AcceptedFeeToken defines an accepted fee token and its conversion rate.
type AcceptedFeeToken struct {
	Denom          string `json:"denom"`
	ConversionRate string `json:"conversion_rate"`
}

// DefaultGasAbstractionConfig returns default config.
func DefaultGasAbstractionConfig() GasAbstractionConfig {
	return GasAbstractionConfig{
		Enabled:     true,
		NativeDenom: "uqor",
		AcceptedTokens: []AcceptedFeeToken{
			{Denom: "uqor", ConversionRate: "1.0"},
			{Denom: "ibc/USDC", ConversionRate: "1.0"},
			{Denom: "ibc/ATOM", ConversionRate: "10.0"},
		},
	}
}

// Validate checks that all accepted fee tokens have valid conversion rates.
func (cfg GasAbstractionConfig) Validate() error {
	if cfg.NativeDenom == "" {
		return fmt.Errorf("native denom cannot be empty")
	}
	for _, token := range cfg.AcceptedTokens {
		if token.Denom == "" {
			return fmt.Errorf("accepted token denom cannot be empty")
		}
		rate, err := math.LegacyNewDecFromStr(token.ConversionRate)
		if err != nil {
			return fmt.Errorf("invalid conversion rate %q for token %s: %w", token.ConversionRate, token.Denom, err)
		}
		if !rate.IsPositive() {
			return fmt.Errorf("conversion rate must be positive for token %s, got %s", token.Denom, rate)
		}
	}
	return nil
}

// FeeQuote represents a fee quote in a non-native denom.
type FeeQuote struct {
	OriginalDenom  string `json:"original_denom"`
	OriginalAmount string `json:"original_amount"`
	NativeEquiv    string `json:"native_equivalent"`
	ConversionRate string `json:"conversion_rate"`
}
