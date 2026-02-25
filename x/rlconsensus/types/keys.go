package types

const (
	// ModuleName defines the module name for the RL consensus parameter tuning system.
	ModuleName = "rlconsensus"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key.
	RouterKey = ModuleName
)

// KVStore key prefixes for the rlconsensus module.
var (
	// ParamsKey stores module parameters: single key -> Params
	ParamsKey = []byte{0x01}

	// AgentStatusKey stores the RL agent status: single key -> AgentStatus
	AgentStatusKey = []byte{0x02}

	// PolicyWeightsKey stores the current policy weights: single key -> PolicyWeights
	PolicyWeightsKey = []byte{0x03}

	// ObservationKeyPrefix stores observations: 0x04 | height(8 bytes BE) -> Observation
	ObservationKeyPrefix = []byte{0x04}

	// RewardKeyPrefix stores reward records: 0x05 | height(8 bytes BE) -> Reward
	RewardKeyPrefix = []byte{0x05}

	// ExperienceKeyPrefix stores experience tuples: 0x06 | height(8 bytes BE) -> Experience
	ExperienceKeyPrefix = []byte{0x06}

	// CircuitBreakerStateKey stores circuit breaker state: single key -> bool
	CircuitBreakerStateKey = []byte{0x07}

	// AppliedParamsKey stores the most recently applied consensus parameters: single key
	AppliedParamsKey = []byte{0x08}
)

// ObservationKey returns the key for an observation at a given height.
// Uses big-endian encoding for proper ordering.
func ObservationKey(height int64) []byte {
	key := make([]byte, 1, 1+8)
	key[0] = ObservationKeyPrefix[0]
	bz := make([]byte, 8)
	bz[0] = byte(height >> 56)
	bz[1] = byte(height >> 48)
	bz[2] = byte(height >> 40)
	bz[3] = byte(height >> 32)
	bz[4] = byte(height >> 24)
	bz[5] = byte(height >> 16)
	bz[6] = byte(height >> 8)
	bz[7] = byte(height)
	return append(key, bz...)
}

// RewardKey returns the key for a reward record at a given height.
func RewardKey(height int64) []byte {
	key := make([]byte, 1, 1+8)
	key[0] = RewardKeyPrefix[0]
	bz := make([]byte, 8)
	bz[0] = byte(height >> 56)
	bz[1] = byte(height >> 48)
	bz[2] = byte(height >> 40)
	bz[3] = byte(height >> 32)
	bz[4] = byte(height >> 24)
	bz[5] = byte(height >> 16)
	bz[6] = byte(height >> 8)
	bz[7] = byte(height)
	return append(key, bz...)
}
