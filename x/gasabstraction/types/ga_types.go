package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// GasAbstractionConfig holds gas abstraction configuration.
//
// The fields after AcceptedTokens were added in v3.2.2 for fee conversion.
// They are omitted while empty, so a config stored before they existed
// serialises byte-identically.
type GasAbstractionConfig struct {
	Enabled        bool               `json:"enabled"`
	NativeDenom    string             `json:"native_denom"`
	AcceptedTokens []AcceptedFeeToken `json:"accepted_tokens"`

	// PremiumBps is the share of a converted fee's value the float keeps, in
	// basis points: a fee of F is worth F x rate x (1 - premium) in the
	// native denom. It pays for the float's price risk.
	PremiumBps uint32 `json:"premium_bps,omitempty"`
	// MaxTipMultiplier bounds a converted fee at this multiple of the minimum
	// fee for its gas; zero means no bound. It limits what one transaction can
	// draw from the float.
	MaxTipMultiplier uint32 `json:"max_tip_multiplier,omitempty"`
	// FloatMin is the native amount the float always keeps; a conversion that
	// would take it below is refused.
	FloatMin string `json:"float_min,omitempty"`
	// BlockConversionCap is the most the float pays out in one block, in the
	// native denom. Empty or zero means conversion is off.
	BlockConversionCap string `json:"block_conversion_cap,omitempty"`
}

// Price sources for a non-native fee token.
const (
	// PriceSourceStatic is a governance-set rate that expires.
	PriceSourceStatic = "static"
	// PriceSourceAMMTWAP is the x/amm pool's time-weighted price, guarded by a
	// minimum depth and a maximum deviation from the spot price.
	PriceSourceAMMTWAP = "amm_twap"
)

// AcceptedFeeToken defines an accepted fee token and its conversion rate.
type AcceptedFeeToken struct {
	Denom string `json:"denom"`
	// ConversionRate is native units per unit of Denom; the rate for the
	// static source.
	ConversionRate string `json:"conversion_rate"`

	// PriceSource is PriceSourceStatic or PriceSourceAMMTWAP. A token without
	// one is listed but cannot pay fees.
	PriceSource string `json:"price_source,omitempty"`
	// RateExpiryHeight is the last height at which a static rate is used;
	// required for the static source, so a stale rate stops on its own.
	RateExpiryHeight int64 `json:"rate_expiry_height,omitempty"`
	// PoolID is the x/amm pool pairing Denom with the native denom.
	PoolID uint64 `json:"pool_id,omitempty"`
	// MinLiquidity is the native-side reserve the pool must hold.
	MinLiquidity string `json:"min_liquidity,omitempty"`
	// MaxDeviationBps bounds |TWAP - spot| / spot; above it the pool is
	// considered manipulated and the token refused. It also bounds the
	// slippage of the float's refill swap.
	MaxDeviationBps uint32 `json:"max_deviation_bps,omitempty"`
}

// DefaultGasAbstractionConfig returns default config: native fees only.
// Foreign tokens are added by governance once a real (ibc/<hash>) denom and a
// pool deep enough to price it exist; the conversion cap starts at zero.
func DefaultGasAbstractionConfig() GasAbstractionConfig {
	return GasAbstractionConfig{
		Enabled:     true,
		NativeDenom: "uqor",
		AcceptedTokens: []AcceptedFeeToken{
			{Denom: "uqor", ConversionRate: "1.0"},
		},
		PremiumBps:         DefaultPremiumBps,
		MaxTipMultiplier:   DefaultMaxTipMultiplier,
		FloatMin:           "0",
		BlockConversionCap: "0",
	}
}

const (
	// DefaultPremiumBps is 3%.
	DefaultPremiumBps = 300
	// DefaultMaxTipMultiplier lets a converted fee be up to three times the
	// minimum for its gas.
	DefaultMaxTipMultiplier = 3
	// MaxPremiumBps caps the premium at 50%.
	MaxPremiumBps = 5000
)

func parseNonNegInt(field, s string) error {
	if s == "" {
		return nil
	}
	v, ok := math.NewIntFromString(s)
	if !ok || v.IsNegative() {
		return fmt.Errorf("%s %q is not a non-negative integer", field, s)
	}
	return nil
}

// Validate checks that all accepted fee tokens have valid conversion rates,
// and, for tokens that name a price source, that the source is usable.
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
		if err := token.validateSource(cfg.NativeDenom); err != nil {
			return err
		}
	}
	if cfg.PremiumBps > MaxPremiumBps {
		return fmt.Errorf("premium_bps %d above %d", cfg.PremiumBps, MaxPremiumBps)
	}
	if err := parseNonNegInt("float_min", cfg.FloatMin); err != nil {
		return err
	}
	return parseNonNegInt("block_conversion_cap", cfg.BlockConversionCap)
}

func (t AcceptedFeeToken) validateSource(native string) error {
	switch t.PriceSource {
	case "":
		return nil
	case PriceSourceStatic:
		if t.Denom == native {
			return fmt.Errorf("token %s: the native denom needs no price source", t.Denom)
		}
		if t.RateExpiryHeight <= 0 {
			return fmt.Errorf("token %s: a static rate needs rate_expiry_height", t.Denom)
		}
	case PriceSourceAMMTWAP:
		if t.Denom == native {
			return fmt.Errorf("token %s: the native denom needs no price source", t.Denom)
		}
		if t.PoolID == 0 {
			return fmt.Errorf("token %s: amm_twap needs pool_id", t.Denom)
		}
		min, ok := math.NewIntFromString(t.MinLiquidity)
		if !ok || !min.IsPositive() {
			return fmt.Errorf("token %s: amm_twap needs a positive min_liquidity", t.Denom)
		}
		if t.MaxDeviationBps == 0 || t.MaxDeviationBps > 10_000 {
			return fmt.Errorf("token %s: max_deviation_bps must be in (0, 10000]", t.Denom)
		}
	default:
		return fmt.Errorf("token %s: unknown price_source %q", t.Denom, t.PriceSource)
	}
	return nil
}

// Token returns the accepted token entry for denom.
func (cfg GasAbstractionConfig) Token(denom string) (AcceptedFeeToken, bool) {
	for _, t := range cfg.AcceptedTokens {
		if t.Denom == denom {
			return t, true
		}
	}
	return AcceptedFeeToken{}, false
}

// BlockCap returns BlockConversionCap as an Int (zero when unset).
func (cfg GasAbstractionConfig) BlockCap() math.Int { return intOrZero(cfg.BlockConversionCap) }

// FloatMinimum returns FloatMin as an Int (zero when unset).
func (cfg GasAbstractionConfig) FloatMinimum() math.Int { return intOrZero(cfg.FloatMin) }

func intOrZero(s string) math.Int {
	if v, ok := math.NewIntFromString(s); ok && !v.IsNegative() {
		return v
	}
	return math.ZeroInt()
}

// FeeQuote represents a fee quote in a non-native denom.
type FeeQuote struct {
	OriginalDenom  string `json:"original_denom"`
	OriginalAmount string `json:"original_amount"`
	NativeEquiv    string `json:"native_equivalent"`
	ConversionRate string `json:"conversion_rate"`
}
