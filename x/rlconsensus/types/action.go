package types

// ActionDimensions is the number of tunable parameters in a single action vector.
const ActionDimensions = 5

// Action vector dimension indices.
const (
	// ActBlockTimeDelta is the proposed change to target block time (ms).
	ActBlockTimeDelta = 0
	// ActGasPriceDelta is the proposed change to base gas price.
	ActGasPriceDelta = 1
	// ActValidatorSetSizeDelta is the proposed change to target validator set size.
	ActValidatorSetSizeDelta = 2
	// ActPoolWeightPQCDelta is the proposed change to PQC pool priority weight.
	ActPoolWeightPQCDelta = 3
	// ActPoolWeightDPoSDelta is the proposed change to DPoS pool priority weight.
	ActPoolWeightDPoSDelta = 4
)

// Action represents the consensus parameter adjustments proposed by the RL agent
// at a specific height. Values are stored as LegacyDec string representations.
type Action struct {
	Height int64                  `json:"height"`
	Values [ActionDimensions]string `json:"values"`
}
