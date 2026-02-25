package types

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

// FeeQuote represents a fee quote in a non-native denom.
type FeeQuote struct {
	OriginalDenom  string `json:"original_denom"`
	OriginalAmount string `json:"original_amount"`
	NativeEquiv    string `json:"native_equivalent"`
	ConversionRate string `json:"conversion_rate"`
}
