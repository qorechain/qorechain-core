package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// ObservationDimensions is the number of features in a single observation vector.
const ObservationDimensions = 25

// Observation vector dimension indices.
const (
	ObsBlockUtilization      = 0  // block gas used / block gas limit
	ObsTxCount               = 1  // number of transactions in block
	ObsAvgTxSize             = 2  // mean transaction size in bytes
	ObsBlockTime             = 3  // time since previous block (ms)
	ObsBlockTimeDelta        = 4  // block time - target block time (ms)
	ObsGasPrice50th          = 5  // median gas price
	ObsGasPrice95th          = 6  // 95th-percentile gas price
	ObsMempoolSize           = 7  // number of pending transactions
	ObsMempoolBytes          = 8  // total bytes of pending transactions
	ObsValidatorCount        = 9  // active validator count
	ObsValidatorGini         = 10 // Gini coefficient of validator power
	ObsMissedBlockRatio      = 11 // fraction of validators that missed signing
	ObsAvgCommitLatency      = 12 // average commit round latency (ms)
	ObsMaxCommitLatency      = 13 // maximum commit round latency (ms)
	ObsPrecommitRatio        = 14 // fraction of precommits received
	ObsFailedTxRatio         = 15 // fraction of failed transactions
	ObsAvgGasPerTx           = 16 // mean gas consumed per tx
	ObsRewardPerValidator    = 17 // mean reward per validator in uqor
	ObsSlashCount            = 18 // number of slashing events in window
	ObsJailCount             = 19 // number of jail events in window
	ObsInflationRate         = 20 // current inflation rate
	ObsBondedRatio           = 21 // bonded tokens / total supply
	ObsReputationMean        = 22 // mean reputation score across validators
	ObsReputationStdDev      = 23 // standard deviation of reputation scores
	ObsMEVEstimate           = 24 // estimated MEV extracted (heuristic)
)

// FixedPointScale is the scaling factor for converting LegacyDec string
// representations to deterministic int64 fixed-point values.
// 10^8 provides 8 decimal places of precision.
const FixedPointScale = int64(100_000_000)

// Observation represents the state of the chain collected at a specific height.
// Values are stored as LegacyDec string representations for deterministic
// serialization.
type Observation struct {
	Height int64    `json:"height"`
	Values [ObservationDimensions]string `json:"values"`
}

// ToFixedPoint converts the string-encoded observation values to a deterministic
// int64 fixed-point representation with FixedPointScale precision.
func (o *Observation) ToFixedPoint() ([ObservationDimensions]int64, error) {
	var out [ObservationDimensions]int64
	scale := math.LegacyNewDec(FixedPointScale)
	for i := 0; i < ObservationDimensions; i++ {
		d, err := math.LegacyNewDecFromStr(o.Values[i])
		if err != nil {
			return out, fmt.Errorf("observation[%d]: invalid decimal %q: %w", i, o.Values[i], err)
		}
		product := d.Mul(scale)
		if !product.IsInteger() {
			// Truncate to integer after scaling.
			product = product.TruncateDec()
		}
		v := product.TruncateInt64()
		out[i] = v
	}
	return out, nil
}

// Validate performs basic validation of the observation.
func (o *Observation) Validate() error {
	if o.Height < 0 {
		return fmt.Errorf("observation height must be non-negative, got %d", o.Height)
	}
	for i := 0; i < ObservationDimensions; i++ {
		if _, err := math.LegacyNewDecFromStr(o.Values[i]); err != nil {
			return fmt.Errorf("observation[%d]: invalid decimal %q: %w", i, o.Values[i], err)
		}
	}
	return nil
}
