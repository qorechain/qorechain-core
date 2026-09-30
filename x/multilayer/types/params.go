package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Params defines the module parameters for the QoreChain multi-layer architecture
type Params struct {
	MaxSidechains              uint64 `json:"max_sidechains"`               // Maximum number of active sidechains
	MaxPaychains               uint64 `json:"max_paychains"`                // Maximum number of active paychains
	MinAnchorInterval          uint64 `json:"min_anchor_interval"`          // Minimum blocks between state anchors
	MaxAnchorInterval          uint64 `json:"max_anchor_interval"`          // Maximum blocks between state anchors (force anchor)
	DefaultChallengePeriod     uint64 `json:"default_challenge_period"`     // Default challenge period in seconds
	MinSidechainStake          string `json:"min_sidechain_stake"`          // Minimum stake to create a sidechain (uqor)
	MinPaychainStake           string `json:"min_paychain_stake"`           // Minimum stake to create a paychain (uqor)
	RoutingEnabled             bool   `json:"routing_enabled"`              // Enable QCAI-based TX routing
	RoutingConfidenceThreshold string `json:"routing_confidence_threshold"` // Min confidence for QCAI routing (decimal)
	CrossLayerFeeBundling      bool   `json:"cross_layer_fee_bundling"`     // Global CLFB toggle
}

// DefaultParams returns the default module parameters for the multi-layer architecture
func DefaultParams() Params {
	return Params{
		MaxSidechains:              10,
		MaxPaychains:               50,
		MinAnchorInterval:          100,
		MaxAnchorInterval:          1000,
		DefaultChallengePeriod:     86400,        // 24 hours
		MinSidechainStake:          "1000000000", // 1,000 QOR in uqor
		MinPaychainStake:           "100000000",  // 100 QOR in uqor
		RoutingEnabled:             true,
		RoutingConfidenceThreshold: "0.6",
		CrossLayerFeeBundling:      true,
	}
}

// Validate checks that the parameters are valid
func (p Params) Validate() error {
	if p.MaxSidechains == 0 {
		return ErrInvalidLayerTransition.Wrap("max_sidechains must be > 0")
	}
	if p.MaxPaychains == 0 {
		return ErrInvalidLayerTransition.Wrap("max_paychains must be > 0")
	}
	if p.MinAnchorInterval == 0 {
		return ErrInvalidAnchor.Wrap("min_anchor_interval must be > 0")
	}
	if p.MaxAnchorInterval <= p.MinAnchorInterval {
		return ErrInvalidAnchor.Wrap("max_anchor_interval must be > min_anchor_interval")
	}
	if p.DefaultChallengePeriod == 0 {
		return ErrInvalidAnchor.Wrap("default_challenge_period must be > 0")
	}
	return nil
}

// ValidateUpdate is Validate plus the checks a governance update must also pass:
// the stakes parse as non-negative amounts and the routing threshold is a
// decimal in [0, 1]. Genesis keeps plain Validate, so the historical genesis
// files are judged by the rules they were written under.
func (p Params) ValidateUpdate() error {
	if err := p.Validate(); err != nil {
		return err
	}
	for _, s := range []struct{ name, v string }{
		{"min_sidechain_stake", p.MinSidechainStake},
		{"min_paychain_stake", p.MinPaychainStake},
	} {
		amt, ok := math.NewIntFromString(s.v)
		if !ok || amt.IsNegative() {
			return ErrInvalidLayerTransition.Wrap(fmt.Sprintf("%s must be a non-negative integer amount, got %q", s.name, s.v))
		}
	}
	th, err := math.LegacyNewDecFromStr(p.RoutingConfidenceThreshold)
	if err != nil || th.IsNegative() || th.GT(math.LegacyOneDec()) {
		return ErrInvalidLayerTransition.Wrap(fmt.Sprintf("routing_confidence_threshold must be a decimal between 0 and 1, got %q", p.RoutingConfidenceThreshold))
	}
	return nil
}
