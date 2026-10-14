package types

// v3.2.3 economics fixes. The release integrator appends V323EconFeatures to
// ReleaseV3_2_3ProtocolFeatures.
const (
	// x/inflation: the schedule year advances after params.EpochsPerYear()
	// epochs instead of after a fixed 365 (QSR-2026-0443). Identical for the
	// mainnet epoch length of 43200 blocks.
	FeatureInflationYearByEpochLength = "inflation-year-by-epoch-length"
)

// V323EconFeatures are the economics switches the v3.2.3 handler turns on.
var V323EconFeatures = []string{
	FeatureInflationYearByEpochLength,
}
