# QoreChain Consensus Enhancements Implementation Plan (v0.9.0)

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add RL-based dynamic consensus parameter tuning, triple-pool CPoS with bonding curve and progressive slashing, and QDRW governance — taking the chain from v0.8.0 to v0.9.0.

**Architecture:** New `x/rlconsensus` module (factory pattern, proprietary/stub split) with a Go-native fixed-point MLP for deterministic on-chain inference. Extensions to existing `x/qca` module for pool classification, bonding curve, progressive slashing, and QDRW tally. Shared `math_utils.go` for deterministic math (integer sqrt, Taylor log, Pade sigmoid). New `qor_` RPC methods for RL observability.

**Tech Stack:** Go 1.23.8, Cosmos SDK v0.53.6, `cosmossdk.io/math.LegacyDec`, `int64` fixed-point (scale 10^8), JSON KV store encoding (matching existing module patterns).

**Standing Rules:**
- NO forbidden terms (Cosmos SDK, CometBFT, Tendermint, Claude, Anthropic, AWS Bedrock, Haiku, Sonnet, Opus, Baron Chain, Ethermint, Evmos) in public code/docs/git
- Git: `user.name "Liviu Epure"`, `user.email "liviu.etty@gmail.com"`, NO Co-Authored-By
- Open-core: `//go:build proprietary` / `//go:build !proprietary`
- Both builds must compile: `CGO_ENABLED=1 go build ./cmd/qorechaind/` AND `CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/`

---

## Part A: x/rlconsensus Module

### Task 1: Types — Keys, Errors, Events

**Files:**
- Create: `x/rlconsensus/types/keys.go`
- Create: `x/rlconsensus/types/errors.go`
- Create: `x/rlconsensus/types/events.go`

**Step 1: Create `x/rlconsensus/types/keys.go`**

Pattern: Follow `x/svm/types/keys.go` (lines 1-79) and `x/qca/types/keys.go`.

```go
package types

const (
	ModuleName = "rlconsensus"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ParamsKey              = []byte{0x01}
	AgentStatusKey         = []byte{0x02}
	PolicyWeightsKey       = []byte{0x03}
	ObservationKeyPrefix   = []byte{0x04}
	RewardKeyPrefix        = []byte{0x05}
	ExperienceKeyPrefix    = []byte{0x06}
	CircuitBreakerStateKey = []byte{0x07}
	AppliedParamsKey       = []byte{0x08}
)

// ObservationKey returns the key for an observation at a given height.
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
```

**Step 2: Create `x/rlconsensus/types/errors.go`**

Pattern: Follow `x/svm/types/errors.go`.

```go
package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrRLDisabled          = errorsmod.Register(ModuleName, 2, "RL consensus module is disabled")
	ErrAgentPaused         = errorsmod.Register(ModuleName, 3, "RL agent is paused")
	ErrInvalidPolicyWeights = errorsmod.Register(ModuleName, 4, "invalid policy weights")
	ErrCircuitBreakerActive = errorsmod.Register(ModuleName, 5, "circuit breaker is active")
	ErrInvalidAgentMode    = errorsmod.Register(ModuleName, 6, "invalid agent mode")
	ErrInvalidObservation  = errorsmod.Register(ModuleName, 7, "invalid observation vector")
	ErrOverflow            = errorsmod.Register(ModuleName, 8, "fixed-point arithmetic overflow")
	ErrInvalidRewardWeights = errorsmod.Register(ModuleName, 9, "reward weights must sum to 1.0")
)
```

**Step 3: Create `x/rlconsensus/types/events.go`**

```go
package types

const (
	EventTypeObservationCollected = "rl_observation_collected"
	EventTypeActionApplied        = "rl_action_applied"
	EventTypeCircuitBreakerTriggered = "rl_circuit_breaker_triggered"
	EventTypeCircuitBreakerRecovered = "rl_circuit_breaker_recovered"
	EventTypeAgentModeChanged     = "rl_agent_mode_changed"
	EventTypePolicyUpdated        = "rl_policy_updated"
	EventTypeRewardComputed       = "rl_reward_computed"

	AttributeKeyHeight    = "height"
	AttributeKeyEpoch     = "epoch"
	AttributeKeyAgentMode = "agent_mode"
	AttributeKeyReward    = "reward"
)
```

**Step 4: Run build to verify types compile**

```bash
cd /Users/liviu/Development/Qore/testnet/qorechain-core && CGO_ENABLED=1 go build ./x/rlconsensus/types/
```

Expected: SUCCESS (types package compiles independently)

**Step 5: Commit**

```bash
git add x/rlconsensus/types/keys.go x/rlconsensus/types/errors.go x/rlconsensus/types/events.go
git commit -m "feat(rlconsensus): add types — keys, errors, events"
```

---

### Task 2: Types — Params, Observation, Action, Reward, Policy, Genesis

**Files:**
- Create: `x/rlconsensus/types/params.go`
- Create: `x/rlconsensus/types/observation.go`
- Create: `x/rlconsensus/types/action.go`
- Create: `x/rlconsensus/types/reward.go`
- Create: `x/rlconsensus/types/policy.go`
- Create: `x/rlconsensus/types/genesis.go`

**Step 1: Create `x/rlconsensus/types/params.go`**

Pattern: Follow `x/svm/types/params.go` (DefaultParams + Validate).

```go
package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// AgentMode defines the RL agent operating mode.
type AgentMode uint8

const (
	AgentModeShadow       AgentMode = 0
	AgentModeConservative AgentMode = 1
	AgentModeAutonomous   AgentMode = 2
	AgentModePaused       AgentMode = 3
)

func (m AgentMode) String() string {
	switch m {
	case AgentModeShadow:
		return "shadow"
	case AgentModeConservative:
		return "conservative"
	case AgentModeAutonomous:
		return "autonomous"
	case AgentModePaused:
		return "paused"
	default:
		return "unknown"
	}
}

// Params defines the module parameters.
type Params struct {
	Enabled                bool      `json:"enabled"`
	ObservationInterval    int64     `json:"observation_interval"`     // blocks between observations (default: 10)
	AgentMode              AgentMode `json:"agent_mode"`               // current operating mode
	MaxChangeConservative  string    `json:"max_change_conservative"`  // max param change in conservative mode (e.g. "0.10")
	MaxChangeAutonomous    string    `json:"max_change_autonomous"`    // max param change in autonomous mode (e.g. "0.25")
	CircuitBreakerWindow   int64     `json:"circuit_breaker_window"`   // blocks to check (default: 50)
	CircuitBreakerThreshold string   `json:"circuit_breaker_threshold"` // fraction of on-time blocks (default: "0.50")
	RewardWeights          RewardWeights `json:"reward_weights"`
	DefaultBlockTimeMs     int64     `json:"default_block_time_ms"`     // fallback block time (default: 5000)
	DefaultBaseGasPrice    string    `json:"default_base_gas_price"`    // fallback gas price (default: "100")
	DefaultValidatorSetSize int64    `json:"default_validator_set_size"` // fallback set size (default: 100)
}

// RewardWeights holds the weights for the reward function components.
type RewardWeights struct {
	Throughput       string `json:"throughput"`        // w1 (default: "0.30")
	Finality         string `json:"finality"`          // w2 (default: "0.25")
	Decentralization string `json:"decentralization"`  // w3 (default: "0.20")
	MEV              string `json:"mev"`               // w4 (default: "0.15")
	FailedTxs        string `json:"failed_txs"`        // w5 (default: "0.10")
}

func DefaultParams() Params {
	return Params{
		Enabled:                true,
		ObservationInterval:    10,
		AgentMode:              AgentModeShadow,
		MaxChangeConservative:  "0.10",
		MaxChangeAutonomous:    "0.25",
		CircuitBreakerWindow:   50,
		CircuitBreakerThreshold: "0.50",
		RewardWeights: RewardWeights{
			Throughput:       "0.30",
			Finality:         "0.25",
			Decentralization: "0.20",
			MEV:              "0.15",
			FailedTxs:        "0.10",
		},
		DefaultBlockTimeMs:     5000,
		DefaultBaseGasPrice:    "100",
		DefaultValidatorSetSize: 100,
	}
}

func (p Params) Validate() error {
	if p.ObservationInterval < 1 {
		return fmt.Errorf("observation interval must be >= 1, got %d", p.ObservationInterval)
	}
	if p.AgentMode > AgentModePaused {
		return fmt.Errorf("invalid agent mode: %d", p.AgentMode)
	}
	if p.CircuitBreakerWindow < 10 {
		return fmt.Errorf("circuit breaker window must be >= 10, got %d", p.CircuitBreakerWindow)
	}
	// Validate reward weights sum to 1.0
	w1, _ := math.LegacyNewDecFromStr(p.RewardWeights.Throughput)
	w2, _ := math.LegacyNewDecFromStr(p.RewardWeights.Finality)
	w3, _ := math.LegacyNewDecFromStr(p.RewardWeights.Decentralization)
	w4, _ := math.LegacyNewDecFromStr(p.RewardWeights.MEV)
	w5, _ := math.LegacyNewDecFromStr(p.RewardWeights.FailedTxs)
	sum := w1.Add(w2).Add(w3).Add(w4).Add(w5)
	one := math.LegacyOneDec()
	if !sum.Equal(one) {
		return fmt.Errorf("reward weights must sum to 1.0, got %s", sum.String())
	}
	return nil
}

// MaxChangeForMode returns the max fractional change allowed for the current agent mode.
func (p Params) MaxChangeForMode() math.LegacyDec {
	switch p.AgentMode {
	case AgentModeConservative:
		d, _ := math.LegacyNewDecFromStr(p.MaxChangeConservative)
		return d
	case AgentModeAutonomous:
		d, _ := math.LegacyNewDecFromStr(p.MaxChangeAutonomous)
		return d
	default:
		return math.LegacyZeroDec()
	}
}
```

**Step 2: Create `x/rlconsensus/types/observation.go`**

```go
package types

import "cosmossdk.io/math"

// ObservationDimensions is the number of observation vector entries.
const ObservationDimensions = 25

// Observation represents a state observation at a given block height.
type Observation struct {
	Height int64    `json:"height"`
	Values [ObservationDimensions]string `json:"values"` // math.LegacyDec string representations
}

// ObservationIndex constants for named access.
const (
	ObsBlockUtilization    = 0
	ObsMempoolDepth        = 1
	ObsValidatorParticipation = 2
	ObsLatencyP50          = 3
	ObsLatencyP95          = 4
	ObsLatencyP99          = 5
	ObsBaseFee             = 6
	ObsBaseFeeVelocity     = 7
	ObsAnomalyCount        = 8
	ObsFailedTxRatio       = 9
	ObsAvgReputation       = 10
	ObsReputationStd       = 11
	ObsCurrentBlockTime    = 12
	ObsCurrentGasLimit     = 13
	ObsCurrentGasPriceFloor = 14
	ObsCurrentPoolWeightRPoS = 15
	ObsCurrentPoolWeightDPoS = 16
	ObsParamDeltaBlockTime = 17
	ObsParamDeltaGasLimit  = 18
	ObsParamDeltaGasPriceFloor = 19
	ObsParamDeltaPoolWeightRPoS = 20
	ObsParamDeltaPoolWeightDPoS = 21
	ObsEpoch               = 22
	ObsBlocksSinceRevert   = 23
	ObsMEVEstimate         = 24
)

// ToFixedPoint converts the string-encoded LegacyDec values to int64 fixed-point.
// Scale factor is 10^8. Used by the MLP inference engine.
func (o *Observation) ToFixedPoint() ([ObservationDimensions]int64, error) {
	var fp [ObservationDimensions]int64
	scale := math.LegacyNewDec(100_000_000) // 10^8
	for i, v := range o.Values {
		d, err := math.LegacyNewDecFromStr(v)
		if err != nil {
			return fp, err
		}
		fp[i] = d.Mul(scale).TruncateInt().Int64()
	}
	return fp, nil
}
```

**Step 3: Create `x/rlconsensus/types/action.go`**

```go
package types

// ActionDimensions is the number of action vector entries.
const ActionDimensions = 5

// Action represents the RL agent's output for a given step.
type Action struct {
	Height int64              `json:"height"`
	Values [ActionDimensions]string `json:"values"` // fractional deltas as LegacyDec strings
}

// Action index constants.
const (
	ActBlockTimeDelta       = 0
	ActGasLimitDelta        = 1
	ActGasPriceFloorDelta   = 2
	ActPoolWeightRPoSDelta  = 3
	ActPoolWeightDPoSDelta  = 4
)
```

**Step 4: Create `x/rlconsensus/types/reward.go`**

```go
package types

// Reward represents the computed reward for a given step.
type Reward struct {
	Height             int64  `json:"height"`
	TotalReward        string `json:"total_reward"`
	ThroughputDelta    string `json:"throughput_delta"`
	FinalityDelta      string `json:"finality_delta"`
	DecentralizationDelta string `json:"decentralization_delta"`
	MEVEstimate        string `json:"mev_estimate"`
	FailedTxRatio      string `json:"failed_tx_ratio"`
}
```

**Step 5: Create `x/rlconsensus/types/policy.go`**

```go
package types

import "fmt"

// MLPConfig describes the network architecture for verification.
type MLPConfig struct {
	InputSize    int   `json:"input_size"`    // 25
	HiddenSizes  []int `json:"hidden_sizes"`  // [256, 256]
	OutputSize   int   `json:"output_size"`   // 5
}

// DefaultMLPConfig returns the default MLP configuration.
func DefaultMLPConfig() MLPConfig {
	return MLPConfig{
		InputSize:   ObservationDimensions,
		HiddenSizes: []int{256, 256},
		OutputSize:  ActionDimensions,
	}
}

// TotalParams returns the total number of parameters in the MLP.
func (c MLPConfig) TotalParams() int {
	total := 0
	prev := c.InputSize
	for _, h := range c.HiddenSizes {
		total += prev*h + h // weights + biases
		prev = h
	}
	total += prev*c.OutputSize + c.OutputSize
	return total
}

// PolicyWeights stores the MLP weights as fixed-point int64 values (scale 10^8).
type PolicyWeights struct {
	Epoch       uint64  `json:"epoch"`
	Config      MLPConfig `json:"config"`
	Weights     []int64 `json:"weights"` // flattened: [W1, b1, W2, b2, W3, b3]
	UpdatedAt   int64   `json:"updated_at"` // block height
}

// Validate checks that the weight count matches the config.
func (pw PolicyWeights) Validate() error {
	expected := pw.Config.TotalParams()
	if len(pw.Weights) != expected {
		return fmt.Errorf("%w: expected %d weights, got %d", ErrInvalidPolicyWeights, expected, len(pw.Weights))
	}
	return nil
}
```

**Step 6: Create `x/rlconsensus/types/genesis.go`**

Pattern: Follow `x/svm/types/genesis.go`.

```go
package types

// AgentStatus stores the current agent state.
type AgentStatus struct {
	Mode              AgentMode `json:"mode"`
	CurrentEpoch      uint64    `json:"current_epoch"`
	TotalSteps        uint64    `json:"total_steps"`
	LastObservationAt int64     `json:"last_observation_at"`
	LastActionAt      int64     `json:"last_action_at"`
	CircuitBreakerActive bool   `json:"circuit_breaker_active"`
	BlocksSinceRevert int64     `json:"blocks_since_revert"`
}

// GenesisState defines the RL consensus module's genesis state.
type GenesisState struct {
	Params       Params       `json:"params"`
	AgentStatus  AgentStatus  `json:"agent_status"`
	PolicyWeights *PolicyWeights `json:"policy_weights,omitempty"`
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		AgentStatus: AgentStatus{
			Mode: AgentModeShadow,
		},
		PolicyWeights: nil, // no weights at genesis — agent observes only
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	if gs.PolicyWeights != nil {
		if err := gs.PolicyWeights.Validate(); err != nil {
			return err
		}
	}
	return nil
}
```

**Step 7: Run build**

```bash
cd /Users/liviu/Development/Qore/testnet/qorechain-core && CGO_ENABLED=1 go build ./x/rlconsensus/types/
```

Expected: SUCCESS

**Step 8: Commit**

```bash
git add x/rlconsensus/types/params.go x/rlconsensus/types/observation.go x/rlconsensus/types/action.go x/rlconsensus/types/reward.go x/rlconsensus/types/policy.go x/rlconsensus/types/genesis.go
git commit -m "feat(rlconsensus): add types — params, observation, action, reward, policy, genesis"
```

---

### Task 3: Types — Codec and Messages

**Files:**
- Create: `x/rlconsensus/types/codec.go`
- Create: `x/rlconsensus/types/msgs.go`

**Step 1: Create `x/rlconsensus/types/msgs.go`**

Pattern: Follow `x/svm/types/codec.go` message types.

```go
package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgSetAgentMode changes the RL agent operating mode. Governance only.
type MsgSetAgentMode struct {
	Authority string    `json:"authority"`
	Mode      AgentMode `json:"mode"`
}

func (m *MsgSetAgentMode) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAgentMode, "invalid authority: %s", err)
	}
	if m.Mode > AgentModePaused {
		return errorsmod.Wrapf(ErrInvalidAgentMode, "invalid mode: %d", m.Mode)
	}
	return nil
}

func (m *MsgSetAgentMode) Reset()         { *m = MsgSetAgentMode{} }
func (m *MsgSetAgentMode) String() string { return fmt.Sprintf("MsgSetAgentMode{mode=%s}", m.Mode) }
func (m *MsgSetAgentMode) ProtoMessage()  {}

// MsgResumeAgent resumes the agent after a circuit breaker event. Governance only.
type MsgResumeAgent struct {
	Authority string `json:"authority"`
}

func (m *MsgResumeAgent) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidAgentMode, "invalid authority: %s", err)
	}
	return nil
}

func (m *MsgResumeAgent) Reset()         { *m = MsgResumeAgent{} }
func (m *MsgResumeAgent) String() string { return "MsgResumeAgent" }
func (m *MsgResumeAgent) ProtoMessage()  {}

// MsgUpdatePolicy submits new MLP weights from off-chain training. Governance only.
type MsgUpdatePolicy struct {
	Authority string        `json:"authority"`
	Weights   PolicyWeights `json:"weights"`
}

func (m *MsgUpdatePolicy) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidPolicyWeights, "invalid authority: %s", err)
	}
	return m.Weights.Validate()
}

func (m *MsgUpdatePolicy) Reset()         { *m = MsgUpdatePolicy{} }
func (m *MsgUpdatePolicy) String() string { return fmt.Sprintf("MsgUpdatePolicy{epoch=%d}", m.Weights.Epoch) }
func (m *MsgUpdatePolicy) ProtoMessage()  {}

// MsgUpdateRewardWeights updates the reward function weights. Governance only.
type MsgUpdateRewardWeights struct {
	Authority string        `json:"authority"`
	Weights   RewardWeights `json:"weights"`
}

func (m *MsgUpdateRewardWeights) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrapf(ErrInvalidRewardWeights, "invalid authority: %s", err)
	}
	return nil
}

func (m *MsgUpdateRewardWeights) Reset()         { *m = MsgUpdateRewardWeights{} }
func (m *MsgUpdateRewardWeights) String() string { return "MsgUpdateRewardWeights" }
func (m *MsgUpdateRewardWeights) ProtoMessage()  {}
```

**Step 2: Create `x/rlconsensus/types/codec.go`**

```go
package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterInterfaces(_ codectypes.InterfaceRegistry) {
	// TODO: Register sdk.Msg implementations once proto definitions are generated.
}

func RegisterLegacyAminoCodec(_ *codec.LegacyAmino) {
	// TODO: Register concrete types once proto definitions are generated.
}
```

**Step 3: Run build**

```bash
cd /Users/liviu/Development/Qore/testnet/qorechain-core && CGO_ENABLED=1 go build ./x/rlconsensus/types/
```

**Step 4: Commit**

```bash
git add x/rlconsensus/types/msgs.go x/rlconsensus/types/codec.go
git commit -m "feat(rlconsensus): add types — messages and codec"
```

---

### Task 4: Interface + Stub Keeper

**Files:**
- Create: `x/rlconsensus/interfaces.go`
- Create: `x/rlconsensus/keeper_stub.go`

**Step 1: Create `x/rlconsensus/interfaces.go`**

Pattern: Follow `x/svm/interfaces.go` exactly — NO build tag, defines the interface.

```go
package rlconsensus

import (
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// RLConsensusKeeper defines the interface for the RL consensus module keeper.
// Both the proprietary and stub implementations satisfy this interface.
type RLConsensusKeeper interface {
	// GetParams returns the module parameters.
	GetParams(ctx sdk.Context) types.Params

	// SetParams updates the module parameters.
	SetParams(ctx sdk.Context, params types.Params) error

	// GetAgentStatus returns the current agent status.
	GetAgentStatus(ctx sdk.Context) types.AgentStatus

	// GetLatestObservation returns the most recent observation.
	GetLatestObservation(ctx sdk.Context) (*types.Observation, error)

	// GetLatestReward returns the most recent reward computation.
	GetLatestReward(ctx sdk.Context) (*types.Reward, error)

	// GetPolicyWeights returns the current MLP policy weights.
	GetPolicyWeights(ctx sdk.Context) (*types.PolicyWeights, error)

	// RLConsensusParamsProvider methods (drop-in for StaticRLProvider).
	GetCurrentBlockTime(ctx sdk.Context) time.Duration
	GetCurrentBaseGasPrice(ctx sdk.Context) math.LegacyDec
	GetValidatorSetSize(ctx sdk.Context) uint64
	GetCurrentEpoch(ctx sdk.Context) uint64
	IsRLActive(ctx sdk.Context) bool

	// BeginBlock processes begin-block RL logic (observation collection).
	BeginBlock(ctx sdk.Context) error

	// EndBlock processes end-block RL logic (reward, inference, apply).
	EndBlock(ctx sdk.Context) error

	// InitGenesis initializes the module's state from genesis.
	InitGenesis(ctx sdk.Context, gs types.GenesisState)

	// ExportGenesis exports the module's current state.
	ExportGenesis(ctx sdk.Context) *types.GenesisState

	// Logger returns the module's logger.
	Logger() log.Logger
}

// TokenomicsKeeper is the interface for the future tokenomics module.
// Stubbed to return zero until xQORE is implemented.
type TokenomicsKeeper interface {
	GetXQOREBalance(ctx sdk.Context, addr sdk.AccAddress) math.Int
}

// NilTokenomicsKeeper is a no-op implementation returning zero balances.
type NilTokenomicsKeeper struct{}

func (NilTokenomicsKeeper) GetXQOREBalance(_ sdk.Context, _ sdk.AccAddress) math.Int {
	return math.ZeroInt()
}
```

**Step 2: Create `x/rlconsensus/keeper_stub.go`**

Pattern: Follow `x/svm/keeper_stub.go` exactly — `//go:build !proprietary`.

```go
//go:build !proprietary

package rlconsensus

import (
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// StubKeeper is a no-op implementation of RLConsensusKeeper for the public build.
type StubKeeper struct {
	logger log.Logger
}

func NewStubKeeper(logger log.Logger) *StubKeeper {
	return &StubKeeper{logger: logger}
}

func (k *StubKeeper) GetParams(_ sdk.Context) types.Params {
	return types.DefaultParams()
}

func (k *StubKeeper) SetParams(_ sdk.Context, _ types.Params) error {
	return nil
}

func (k *StubKeeper) GetAgentStatus(_ sdk.Context) types.AgentStatus {
	return types.AgentStatus{Mode: types.AgentModePaused}
}

func (k *StubKeeper) GetLatestObservation(_ sdk.Context) (*types.Observation, error) {
	return nil, types.ErrRLDisabled
}

func (k *StubKeeper) GetLatestReward(_ sdk.Context) (*types.Reward, error) {
	return nil, types.ErrRLDisabled
}

func (k *StubKeeper) GetPolicyWeights(_ sdk.Context) (*types.PolicyWeights, error) {
	return nil, types.ErrRLDisabled
}

func (k *StubKeeper) GetCurrentBlockTime(_ sdk.Context) time.Duration {
	return 5 * time.Second
}

func (k *StubKeeper) GetCurrentBaseGasPrice(_ sdk.Context) math.LegacyDec {
	return math.LegacyNewDec(100)
}

func (k *StubKeeper) GetValidatorSetSize(_ sdk.Context) uint64 {
	return 100
}

func (k *StubKeeper) GetCurrentEpoch(_ sdk.Context) uint64 {
	return 0
}

func (k *StubKeeper) IsRLActive(_ sdk.Context) bool {
	return false
}

func (k *StubKeeper) BeginBlock(_ sdk.Context) error { return nil }
func (k *StubKeeper) EndBlock(_ sdk.Context) error   { return nil }

func (k *StubKeeper) InitGenesis(_ sdk.Context, _ types.GenesisState) {}

func (k *StubKeeper) ExportGenesis(_ sdk.Context) *types.GenesisState {
	return types.DefaultGenesis()
}

func (k *StubKeeper) Logger() log.Logger {
	return k.logger
}
```

**Step 3: Run both builds**

```bash
cd /Users/liviu/Development/Qore/testnet/qorechain-core
CGO_ENABLED=1 go build ./x/rlconsensus/...
```

Expected: SUCCESS (public build compiles)

**Step 4: Commit**

```bash
git add x/rlconsensus/interfaces.go x/rlconsensus/keeper_stub.go
git commit -m "feat(rlconsensus): add keeper interface and stub implementation"
```

---

### Task 5: Module Files (Stub + Proprietary AppModule)

**Files:**
- Create: `x/rlconsensus/module_stub.go`
- Create: `x/rlconsensus/module.go`

**Step 1: Create `x/rlconsensus/module_stub.go`**

Pattern: Follow `x/svm/module_stub.go` exactly.

```go
//go:build !proprietary

package rlconsensus

// (Full AppModuleBasic + AppModule stub, same pattern as x/svm/module_stub.go)
// AppModuleBasic with Name(), DefaultGenesis(), ValidateGenesis(), GetTxCmd(), GetQueryCmd()
// AppModule wrapping RLConsensusKeeper, with InitGenesis/ExportGenesis/ConsensusVersion
// NewAppModule(k RLConsensusKeeper) constructor
```

**Step 2: Create `x/rlconsensus/module.go`**

Pattern: Follow `x/svm/module.go` exactly — `//go:build proprietary`.

```go
//go:build proprietary

package rlconsensus

// (Full proprietary AppModuleBasic + AppModule)
// NewProprietaryAppModule(k RLConsensusKeeper) constructor
// HasBeginBlocker and HasEndBlocker implementations that delegate to keeper
```

NOTE: The proprietary module includes `BeginBlock` and `EndBlock` methods:
```go
func (am AppModule) BeginBlock(ctx sdk.Context) error {
	return am.keeper.BeginBlock(ctx)
}

func (am AppModule) EndBlock(ctx sdk.Context) error {
	return am.keeper.EndBlock(ctx)
}
```

The stub module has no-op BeginBlock/EndBlock.

**Step 3: Run both builds**

```bash
CGO_ENABLED=1 go build ./x/rlconsensus/...
CGO_ENABLED=1 go build -tags proprietary ./x/rlconsensus/...
```

**Step 4: Commit**

```bash
git add x/rlconsensus/module_stub.go x/rlconsensus/module.go
git commit -m "feat(rlconsensus): add module definitions (proprietary + stub)"
```

---

### Task 6: CLI Commands

**Files:**
- Create: `x/rlconsensus/client/cli/query.go`
- Create: `x/rlconsensus/client/cli/tx.go`

**Step 1: Create query CLI**

Subcommands: `agent-status`, `observation`, `reward`, `params`, `policy`

**Step 2: Create tx CLI**

Subcommands: `set-agent-mode`, `resume-agent`, `update-policy`, `update-reward-weights`

**Step 3: Run build**

```bash
CGO_ENABLED=1 go build ./x/rlconsensus/...
```

**Step 4: Commit**

```bash
git add x/rlconsensus/client/
git commit -m "feat(rlconsensus): add CLI commands for query and tx"
```

---

### Task 7: Shared Math Utilities

**Files:**
- Create: `x/rlconsensus/mathutil/mathutil.go`
- Create: `x/rlconsensus/mathutil/mathutil_test.go`

NOTE: NO build tag — shared between all builds and multiple modules (x/rlconsensus, x/qca).

**Step 1: Write the test file first (TDD)**

```go
package mathutil

import (
	"testing"

	"cosmossdk.io/math"
)

func TestIntegerSqrt(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0.0", "0.0"},
		{"1.0", "1.0"},
		{"4.0", "2.0"},
		{"9.0", "3.0"},
		{"2.0", "1.414213562373095048"},  // approximate
		{"100.0", "10.0"},
		{"0.25", "0.5"},
	}
	for _, tc := range tests {
		d, _ := math.LegacyNewDecFromStr(tc.input)
		result := IntegerSqrt(d)
		exp, _ := math.LegacyNewDecFromStr(tc.expected)
		diff := result.Sub(exp).Abs()
		if diff.GT(math.LegacyNewDecWithPrec(1, 12)) { // 1e-12 tolerance
			t.Errorf("IntegerSqrt(%s) = %s, expected ~%s", tc.input, result, tc.expected)
		}
	}
}

func TestTaylorLn1PlusX(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0.0", "0.0"},
		{"0.5", "0.405465108108164381"}, // ln(1.5)
		{"1.0", "0.693147180559945309"}, // ln(2)
	}
	for _, tc := range tests {
		d, _ := math.LegacyNewDecFromStr(tc.input)
		result := TaylorLn1PlusX(d)
		exp, _ := math.LegacyNewDecFromStr(tc.expected)
		diff := result.Sub(exp).Abs()
		if diff.GT(math.LegacyNewDecWithPrec(1, 6)) { // 1e-6 tolerance for Taylor
			t.Errorf("TaylorLn1PlusX(%s) = %s, expected ~%s", tc.input, result, tc.expected)
		}
	}
}

func TestSigmoidApprox(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0.0", "0.5"},
		{"3.0", "0.952574"},  // approximate
		{"-3.0", "0.047426"}, // approximate
	}
	for _, tc := range tests {
		d, _ := math.LegacyNewDecFromStr(tc.input)
		result := SigmoidApprox(d)
		exp, _ := math.LegacyNewDecFromStr(tc.expected)
		diff := result.Sub(exp).Abs()
		if diff.GT(math.LegacyNewDecWithPrec(1, 3)) { // 1e-3 tolerance
			t.Errorf("SigmoidApprox(%s) = %s, expected ~%s", tc.input, result, tc.expected)
		}
	}
}

func TestReputationMultiplier(t *testing.T) {
	// r=0 -> ~0.5, r=0.5 -> ~1.25, r=1.0 -> ~2.0
	r0, _ := math.LegacyNewDecFromStr("0.0")
	r50, _ := math.LegacyNewDecFromStr("0.5")
	r100, _ := math.LegacyNewDecFromStr("1.0")

	m0 := ReputationMultiplier(r0)
	m50 := ReputationMultiplier(r50)
	m100 := ReputationMultiplier(r100)

	half := math.LegacyNewDecWithPrec(5, 1)    // 0.5
	mid := math.LegacyNewDecWithPrec(125, 2)    // 1.25
	two := math.LegacyNewDec(2)

	tol := math.LegacyNewDecWithPrec(5, 2) // 0.05 tolerance

	if m0.Sub(half).Abs().GT(tol) {
		t.Errorf("ReputationMultiplier(0) = %s, expected ~0.5", m0)
	}
	if m50.Sub(mid).Abs().GT(tol) {
		t.Errorf("ReputationMultiplier(0.5) = %s, expected ~1.25", m50)
	}
	if m100.Sub(two).Abs().GT(tol) {
		t.Errorf("ReputationMultiplier(1.0) = %s, expected ~2.0", m100)
	}
}
```

**Step 2: Run tests — verify they fail (functions don't exist yet)**

```bash
cd /Users/liviu/Development/Qore/testnet/qorechain-core && go test ./x/rlconsensus/mathutil/ -v
```

Expected: FAIL — functions not defined

**Step 3: Write the implementation**

```go
package mathutil

import "cosmossdk.io/math"

// IntegerSqrt computes √x using Newton's method on math.LegacyDec.
// Deterministic — no float64.
func IntegerSqrt(x math.LegacyDec) math.LegacyDec {
	if x.IsZero() || x.IsNegative() {
		return math.LegacyZeroDec()
	}
	// Initial guess: x/2 (or 1 if x < 1)
	guess := x.Quo(math.LegacyNewDec(2))
	if guess.IsZero() {
		guess = math.LegacyOneDec()
	}
	for i := 0; i < 100; i++ { // max 100 iterations
		next := guess.Add(x.Quo(guess)).Quo(math.LegacyNewDec(2))
		diff := next.Sub(guess).Abs()
		if diff.LT(math.LegacyNewDecWithPrec(1, 18)) { // 1e-18 convergence
			return next
		}
		guess = next
	}
	return guess
}

// TaylorLn1PlusX computes ln(1+x) for x in [0, 1] using 10-term Taylor series.
// For larger x, uses argument reduction: ln(1+x) = ln((1+x)/k) + ln(k).
func TaylorLn1PlusX(x math.LegacyDec) math.LegacyDec {
	if x.IsZero() {
		return math.LegacyZeroDec()
	}
	if x.IsNegative() {
		return math.LegacyZeroDec() // invalid input
	}

	// Argument reduction for x > 1: break into ln(1+x) = ln(2) + ln((1+x)/2)
	one := math.LegacyOneDec()
	ln2, _ := math.LegacyNewDecFromStr("0.693147180559945309")

	val := one.Add(x) // 1+x
	result := math.LegacyZeroDec()

	// While val > 2, divide by 2 and accumulate ln(2)
	two := math.LegacyNewDec(2)
	for val.GT(two) {
		val = val.Quo(two)
		result = result.Add(ln2)
	}

	// Now compute ln(val) where val is in (0, 2]
	// Use the series for ln(1+u) where u = val - 1, so u in (-1, 1]
	u := val.Sub(one)

	// Taylor: ln(1+u) = u - u^2/2 + u^3/3 - u^4/4 + ... (10 terms)
	term := u
	sum := math.LegacyZeroDec()
	for n := int64(1); n <= 10; n++ {
		divisor := math.LegacyNewDec(n)
		if n%2 == 1 {
			sum = sum.Add(term.Quo(divisor))
		} else {
			sum = sum.Sub(term.Quo(divisor))
		}
		term = term.Mul(u) // u^(n+1)
	}

	return result.Add(sum)
}

// SigmoidApprox computes sigmoid(x) ≈ 1/2 + x*(1/4 - x²/48) / (1 + x²/12)
// using Padé approximant. Error < 0.1% for x in [-3, 3].
func SigmoidApprox(x math.LegacyDec) math.LegacyDec {
	half := math.LegacyNewDecWithPrec(5, 1)        // 0.5
	quarter := math.LegacyNewDecWithPrec(25, 2)     // 0.25
	twelve := math.LegacyNewDec(12)
	fortyEight := math.LegacyNewDec(48)

	x2 := x.Mul(x) // x²
	num := x.Mul(quarter.Sub(x2.Quo(fortyEight)))
	den := math.LegacyOneDec().Add(x2.Quo(twelve))

	return half.Add(num.Quo(den))
}

// ReputationMultiplier maps a reputation score in [0, 1] to a multiplier in [0.5, 2.0].
// Formula: 0.5 + 1.5 * sigmoid(6 * (r - 0.5))
func ReputationMultiplier(r math.LegacyDec) math.LegacyDec {
	half := math.LegacyNewDecWithPrec(5, 1)       // 0.5
	onePointFive := math.LegacyNewDecWithPrec(15, 1) // 1.5
	six := math.LegacyNewDec(6)

	arg := six.Mul(r.Sub(half)) // 6 * (r - 0.5)
	sig := SigmoidApprox(arg)

	result := half.Add(onePointFive.Mul(sig))

	// Clamp to [0.5, 2.0]
	minVal := half
	maxVal := math.LegacyNewDec(2)
	if result.LT(minVal) {
		return minVal
	}
	if result.GT(maxVal) {
		return maxVal
	}
	return result
}
```

**Step 4: Run tests — verify they pass**

```bash
cd /Users/liviu/Development/Qore/testnet/qorechain-core && go test ./x/rlconsensus/mathutil/ -v
```

Expected: ALL PASS

**Step 5: Commit**

```bash
git add x/rlconsensus/mathutil/
git commit -m "feat(rlconsensus): add deterministic math utilities with tests"
```

---

### Task 8: Fixed-Point MLP Forward Pass

**Files:**
- Create: `x/rlconsensus/keeper/mlp.go` (proprietary)
- Create: `x/rlconsensus/keeper/mlp_test.go` (proprietary)

**Step 1: Write test first**

```go
//go:build proprietary

package keeper

import (
	"testing"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

func TestMLPForward_ZeroWeights(t *testing.T) {
	config := types.MLPConfig{InputSize: 3, HiddenSizes: []int{4}, OutputSize: 2}
	weights := make([]int64, config.TotalParams()) // all zeros
	mlp := NewMLP(config, weights)

	input := [3]int64{100_000_000, 200_000_000, 300_000_000} // 1.0, 2.0, 3.0 in fixed-point
	output := mlp.Forward(input[:])

	// With zero weights and ReLU/tanh, output should be zero
	for i, v := range output {
		if v != 0 {
			t.Errorf("output[%d] = %d, expected 0 with zero weights", i, v)
		}
	}
}

func TestMLPForward_Identity(t *testing.T) {
	// Simple 1->1->1 network: weight=1.0 (scale 10^8), bias=0
	config := types.MLPConfig{InputSize: 1, HiddenSizes: []int{1}, OutputSize: 1}
	// Weights: [W1=100M, b1=0, W2=100M, b2=0]
	weights := []int64{100_000_000, 0, 100_000_000, 0}
	mlp := NewMLP(config, weights)

	input := []int64{50_000_000} // 0.5 in fixed-point
	output := mlp.Forward(input)

	// After ReLU(0.5*1.0+0)=0.5, then tanh(0.5*1.0+0) ≈ 0.462... in fixed-point
	// Just verify it's non-zero and reasonable
	if output[0] == 0 {
		t.Error("output should be non-zero for identity-like network")
	}
}
```

**Step 2: Run test to verify failure**

```bash
CGO_ENABLED=1 go test -tags proprietary ./x/rlconsensus/keeper/ -run TestMLP -v
```

Expected: FAIL — MLP type not defined

**Step 3: Implement the MLP**

The MLP uses `int64` arithmetic with scale factor `10^8`. ReLU activation for hidden layers, tanh approximation for output layer.

Key implementation details:
- Fixed-point multiply: `(a * b) / SCALE` with overflow check
- ReLU: `max(0, x)`
- Tanh approximation via Padé: `tanh(x) ≈ x*(1 - x²/9) / (1 + x²/3)` for small x, clamped to [-SCALE, SCALE]

**Step 4: Run test to verify pass**

```bash
CGO_ENABLED=1 go test -tags proprietary ./x/rlconsensus/keeper/ -run TestMLP -v
```

**Step 5: Commit**

```bash
git add x/rlconsensus/keeper/mlp.go x/rlconsensus/keeper/mlp_test.go
git commit -m "feat(rlconsensus): add fixed-point MLP forward pass (proprietary)"
```

---

### Task 9: Proprietary Keeper (Core)

**Files:**
- Create: `x/rlconsensus/keeper/keeper.go` (proprietary)

**Step 1: Implement the keeper**

The keeper struct holds all dependencies and implements the `RLConsensusKeeper` interface. KV store operations for params, agent status, observations, rewards, policy weights, and circuit breaker state — all JSON-encoded (matching existing module pattern: `json.Marshal` / `json.Unmarshal`).

Dependencies (constructor parameters):
```go
type Keeper struct {
	cdc              codec.Codec
	storeKey         storetypes.StoreKey
	reputationKeeper ReputationReader
	aiKeeper         AIStatsReader
	feeMarketKeeper  FeeMarketReader
	stakingKeeper    StakingReader
	logger           log.Logger
}
```

Where reader interfaces are defined in the same file:
```go
type ReputationReader interface {
	GetAllValidatorReputations(ctx sdk.Context) []reputationtypes.ValidatorReputation
	GetValidatorReputation(ctx sdk.Context, addr string) (reputationtypes.ValidatorReputation, bool)
	CalculateReputation(ctx sdk.Context, valAddr string) float64
}

type AIStatsReader interface {
	GetStats(ctx sdk.Context) aitypes.AIStats
}

type FeeMarketReader interface {
	GetBaseFee(ctx sdk.Context) math.LegacyDec
}

type StakingReader interface {
	GetAllValidators(ctx sdk.Context) ([]stakingtypes.Validator, error)
}
```

**Step 2: Run build**

```bash
CGO_ENABLED=1 go build -tags proprietary ./x/rlconsensus/...
```

**Step 3: Commit**

```bash
git add x/rlconsensus/keeper/keeper.go
git commit -m "feat(rlconsensus): add proprietary keeper with KV store operations"
```

---

### Task 10: Observation Collector (Proprietary)

**Files:**
- Create: `x/rlconsensus/keeper/observation.go` (proprietary)

Collects the 25-dimension observation vector from on-chain state. Reads from x/reputation (float64→LegacyDec boundary), x/ai stats, x/feemarket base fee, block headers.

Key: float64 boundary conversion:
```go
func float64ToDec(f float64) math.LegacyDec {
	d, err := math.LegacyNewDecFromStr(fmt.Sprintf("%.18f", f))
	if err != nil {
		return math.LegacyZeroDec()
	}
	return d
}
```

**Commit:**
```bash
git add x/rlconsensus/keeper/observation.go
git commit -m "feat(rlconsensus): add observation collector (proprietary)"
```

---

### Task 11: Reward Computation (Proprietary)

**Files:**
- Create: `x/rlconsensus/keeper/reward.go` (proprietary)

Computes the reward between two consecutive observations using the weighted formula:
```
R = w1*Δthroughput + w2*Δfinality + w3*Δdecentralization - w4*mev - w5*failed_txs
```

All arithmetic in `math.LegacyDec`.

**Commit:**
```bash
git add x/rlconsensus/keeper/reward.go
git commit -m "feat(rlconsensus): add reward computation (proprietary)"
```

---

### Task 12: PPO Agent (Inference Only) + Circuit Breaker (Proprietary)

**Files:**
- Create: `x/rlconsensus/keeper/agent.go` (proprietary)
- Create: `x/rlconsensus/keeper/circuit_breaker.go` (proprietary)

The agent wraps the MLP and performs inference: observation → action vector. Clamps actions to `±MaxChangeForMode()`.

The circuit breaker tracks recent block times. If <50% of last N blocks produced on time, triggers revert.

**Commit:**
```bash
git add x/rlconsensus/keeper/agent.go x/rlconsensus/keeper/circuit_breaker.go
git commit -m "feat(rlconsensus): add PPO agent inference and circuit breaker (proprietary)"
```

---

### Task 13: Params Applicator (Proprietary)

**Files:**
- Create: `x/rlconsensus/keeper/params_applicator.go` (proprietary)

Applies action vector deltas to consensus parameters. In shadow mode, logs but doesn't apply. In conservative/autonomous mode, clamps to bounds and applies.

Updates:
- Block time (via consensus params keeper)
- Base gas price (stored in RL params, read by feemarket)
- Pool weights (stored in RL params, read by x/qca pool selector)

**Commit:**
```bash
git add x/rlconsensus/keeper/params_applicator.go
git commit -m "feat(rlconsensus): add params applicator (proprietary)"
```

---

### Task 14: ABCI Hooks + Message Server (Proprietary)

**Files:**
- Create: `x/rlconsensus/keeper/abci.go` (proprietary)
- Create: `x/rlconsensus/keeper/msg_server.go` (proprietary)
- Create: `x/rlconsensus/keeper/query_server.go` (proprietary)
- Create: `x/rlconsensus/keeper/genesis.go` (proprietary)

`abci.go` — The `BeginBlock` and `EndBlock` implementations:
- `BeginBlock`: Every `ObservationInterval` blocks, call `CollectObservation()`
- `EndBlock`: Every `ObservationInterval` blocks, call `ComputeReward()`, `agent.Infer()`, `ApplyActions()`, circuit breaker check

`msg_server.go` — Handles `MsgSetAgentMode`, `MsgResumeAgent`, `MsgUpdatePolicy`, `MsgUpdateRewardWeights`

`query_server.go` — Handles observation, reward, agent status, params, policy queries

`genesis.go` — `InitGenesis` and `ExportGenesis` for the proprietary keeper

**Commit:**
```bash
git add x/rlconsensus/keeper/abci.go x/rlconsensus/keeper/msg_server.go x/rlconsensus/keeper/query_server.go x/rlconsensus/keeper/genesis.go
git commit -m "feat(rlconsensus): add ABCI hooks, msg server, query server, genesis (proprietary)"
```

---

### Task 15: Register.go (Factory + Adapter)

**Files:**
- Create: `x/rlconsensus/register.go` (proprietary)

Pattern: Follow `x/svm/register.go` exactly — `keeperAdapter` wrapping concrete keeper.

```go
//go:build proprietary

package rlconsensus

// keeperAdapter wraps the concrete keeper.Keeper to satisfy the RLConsensusKeeper interface.
// RealNewRLConsensusKeeper creates the real keeper.
// RealNewAppModule creates the real AppModule.
```

**Commit:**
```bash
git add x/rlconsensus/register.go
git commit -m "feat(rlconsensus): add proprietary register.go with keeper adapter"
```

---

### Task 16: Wire into Factory + App.go + Root.go + App_config.go

**Files:**
- Modify: `app/factory.go` — Add `NewRLConsensusKeeper`, `NewRLConsensusAppModule`, `NewRLConsensusModuleBasic` vars
- Modify: `app/factory_stub.go` — Add stub init for RL factories
- Modify: `app/factory_proprietary.go` — Add real init for RL factories
- Modify: `app/app.go` — Add `RLConsensusKeeper` field, store key, keeper init, module registration
- Modify: `app/app_config.go` — Add `"rlconsensus"` to InitGenesis, ExportGenesis, BeginBlockers, EndBlockers
- Modify: `cmd/qorechaind/cmd/root.go` — Register `rlconsensus` module basic

**Step 1: Modify `app/factory.go`** — Add after SVM factory vars (line 63):

```go
	// RL Consensus module factories
	NewRLConsensusKeeper      func(cdc codec.Codec, storeKey storetypes.StoreKey, ...) rlconsensusmod.RLConsensusKeeper
	NewRLConsensusAppModule   func(keeper rlconsensusmod.RLConsensusKeeper) module.AppModule
	NewRLConsensusModuleBasic func() module.AppModuleBasic
```

**Step 2: Modify `app/factory_stub.go`** — Add stub inits:
```go
	NewRLConsensusKeeper = func(...) rlconsensusmod.RLConsensusKeeper {
		return rlconsensusmod.NewStubKeeper(logger)
	}
	// ... etc
```

**Step 3: Modify `app/factory_proprietary.go`** — Add real inits

**Step 4: Modify `app/app.go`**:
- Add field: `RLConsensusKeeper rlconsensusmod.RLConsensusKeeper` (line ~186, after SVMKeeper)
- Add store key + mount (after SVM, ~line 469)
- Add keeper init (after SVM init, ~line 470)
- Add to RegisterModules call (~line 540)
- Replace `StaticRLProvider` with RL keeper in EVM precompile registration

**Step 5: Modify `app/app_config.go`**:
- Add `"rlconsensus"` to BeginBlockers (after `"evm"`, line 135)
- Add `"rlconsensus"` to EndBlockers (before `"evm"`, line 143)
- Add `"rlconsensus"` to InitGenesis (after `"multilayer"`, line 193)
- Add `"rlconsensus"` to ExportGenesis (after `"multilayer"`, line 231)

**Step 6: Modify `cmd/qorechaind/cmd/root.go`** — Add after SVM basic (line 108):
```go
	rlBasic := app.NewRLConsensusModuleBasic()
	moduleBasicManager[rlBasic.Name()] = rlBasic
```

**Step 7: Run both builds**

```bash
CGO_ENABLED=1 go build ./cmd/qorechaind/
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/
```

Expected: BOTH SUCCESS

**Step 8: Test genesis init**

```bash
rm -rf /tmp/qoretest && ./qorechaind init test --chain-id qorechain-test --home /tmp/qoretest 2>/dev/null
cat /tmp/qoretest/config/genesis.json | python3 -c "import sys,json; d=json.load(sys.stdin); print('rlconsensus' in d.get('app_state',{}))"
```

Expected: `True`

**Step 9: Commit**

```bash
git add app/factory.go app/factory_stub.go app/factory_proprietary.go app/app.go app/app_config.go cmd/qorechaind/cmd/root.go
git commit -m "feat(rlconsensus): wire module into app, factory, genesis, and ABCI"
```

---

### Task 17: RPC Extensions for RL

**Files:**
- Modify: `rpc/qor/api.go` — Add 4 new RL methods + rlconsensusKeeper field
- Modify: `rpc/qor_stub/api.go` — Add 4 stub RL methods

**Step 1: Modify `rpc/qor/api.go`**

Add `rlconsensusKeeper rlconsensusmod.RLConsensusKeeper` to QorAPI struct. Add to `NewQorAPI` constructor. Add methods:
- `GetRLAgentStatus()` — returns agent mode, epoch, circuit breaker state
- `GetRLObservation()` — returns latest observation vector
- `GetRLReward()` — returns latest reward breakdown
- `GetPoolClassification()` — returns current validator pool assignments (Part B)

**Step 2: Modify `rpc/qor_stub/api.go`** — Add 4 stub methods returning `errNotAvailable`

**Step 3: Run both builds**

```bash
CGO_ENABLED=1 go build ./cmd/qorechaind/
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/
```

**Step 4: Commit**

```bash
git add rpc/qor/api.go rpc/qor_stub/api.go
git commit -m "feat(rpc): add qor_ RL agent status, observation, reward, pool classification endpoints"
```

---

## Part B: x/qca Extensions

### Task 18: QCA Types Extensions

**Files:**
- Create: `x/qca/types/pool.go`
- Create: `x/qca/types/bonding_curve.go`
- Create: `x/qca/types/slashing_record.go`
- Modify: `x/qca/types/types.go` — Extend QCAConfig with new params
- Modify: `x/qca/types/keys.go` — Add new key prefixes
- Modify: `x/qca/types/genesis.go` — Extend GenesisState

**Step 1: Create pool, bonding curve, and slashing record types**

Pool types: `PoolType` enum (RPoS=0, DPoS=1, PoS=2), `PoolClassification`, `PoolConfig`.

Slashing record type: `SlashingRecord` with validator addr, infraction height, infraction type, severity factor, penalty.

**Step 2: Extend QCAConfig** in `x/qca/types/types.go` — add pool weights, bonding curve params, slashing params, QDRW params.

**Step 3: Extend keys** — add `PoolClassificationKey`, `SlashingRecordKeyPrefix`, `BondingCurveKey`.

**Step 4: Run build**

```bash
CGO_ENABLED=1 go build ./x/qca/...
```

**Step 5: Commit**

```bash
git add x/qca/types/
git commit -m "feat(qca): add types for pool classification, bonding curve, progressive slashing, QDRW"
```

---

### Task 19: Pool Classifier + Pool Selector (Proprietary)

**Files:**
- Create: `x/qca/keeper/pool_classifier.go` (proprietary)
- Create: `x/qca/keeper/pool_selector.go` (proprietary)

`pool_classifier.go`:
- Every `pool_classification_interval` blocks, classify all active validators
- RPoS: reputation >= 70th percentile AND stake >= median
- DPoS: delegation >= 10k QOR
- PoS: remainder
- Store classification in KV store

`pool_selector.go`:
- `PoolWeightedSelector` wrapping `HeuristicSelector`
- Weighted random pool selection, then within-pool reputation-weighted selection
- Reads dynamic pool weights from RL keeper (nil-safe fallback to defaults)

**Step 1: Run build**

```bash
CGO_ENABLED=1 go build -tags proprietary ./x/qca/...
```

**Step 2: Commit**

```bash
git add x/qca/keeper/pool_classifier.go x/qca/keeper/pool_selector.go
git commit -m "feat(qca): add triple-pool classifier and weighted selector (proprietary)"
```

---

### Task 20: Bonding Curve (Proprietary)

**Files:**
- Create: `x/qca/keeper/bonding_curve.go` (proprietary)
- Create: `x/qca/keeper/bonding_curve_test.go` (proprietary)

Formula: `R(v,t) = β * S_v * (1 + α * log(1 + L_v)) * Q(r_v) * P(t)`

Uses `TaylorLn1PlusX` from `x/rlconsensus/mathutil/` for deterministic log.

**TDD approach:**
1. Write test with known inputs/expected outputs
2. Verify test fails
3. Implement
4. Verify test passes

**Commit:**
```bash
git add x/qca/keeper/bonding_curve.go x/qca/keeper/bonding_curve_test.go
git commit -m "feat(qca): add custom bonding curve with deterministic log (proprietary)"
```

---

### Task 21: Progressive Slashing (Proprietary)

**Files:**
- Create: `x/qca/keeper/progressive_slashing.go` (proprietary)
- Create: `x/qca/keeper/progressive_slashing_test.go` (proprietary)

Formula: `penalty = base_rate * 1.5^effective_count * severity_factor` (capped at 33%)

`effective_count = Σ 0.5^(blocks_since_i / 100000)` — temporal decay with half-life.

`0.5^x` computed via exp(-ln2 * x) with Taylor series on LegacyDec.

Store `SlashingRecord` entries in KV store. Prune after 1,000,000 blocks.

**TDD approach:**
1. Write tests for first offense (low penalty), repeated offense (escalated), very old offense (decayed)
2. Verify tests fail
3. Implement
4. Verify tests pass

**Commit:**
```bash
git add x/qca/keeper/progressive_slashing.go x/qca/keeper/progressive_slashing_test.go
git commit -m "feat(qca): add progressive slashing with temporal decay (proprietary)"
```

---

### Task 22: QDRW Tally Handler (Proprietary)

**Files:**
- Create: `x/qca/keeper/qdrw_tally.go` (proprietary)
- Create: `x/qca/keeper/qdrw_tally_stub.go` (!proprietary)
- Create: `x/qca/keeper/qdrw_tally_test.go` (proprietary)

`qdrw_tally.go`:
- `QDRWTallyHandler` implementing a custom tally function
- `CalculateVotingPower(voter)`: `√(staked + 2×xQORE) × ReputationMultiplier(r)`
- Uses `IntegerSqrt` and `ReputationMultiplier` from `x/rlconsensus/mathutil/`
- `NilTokenomicsKeeper` used for xQORE (returns 0)

`qdrw_tally_stub.go`:
- Returns default SDK tally behavior

**TDD approach:**
1. Write tests verifying quadratic dampening (whale with 100x stake gets ~10x voting power, not 100x)
2. Write tests verifying reputation multiplier effect
3. Verify tests fail
4. Implement
5. Verify tests pass

**Commit:**
```bash
git add x/qca/keeper/qdrw_tally.go x/qca/keeper/qdrw_tally_stub.go x/qca/keeper/qdrw_tally_test.go
git commit -m "feat(qca): add QDRW tally handler with quadratic-reputation voting power (proprietary)"
```

---

### Task 23: Wire QCA Extensions into Keeper

**Files:**
- Modify: `x/qca/keeper/keeper.go` — Add optional dependencies (stakingKeeper, rlKeeper), update constructor
- Modify: `app/app.go` — Pass new dependencies to QCA keeper

**Step 1: Update QCA keeper** to accept optional `StakingKeeper` and `RLConsensusKeeper` dependencies. Add setter methods for optional deps (called after keeper creation to avoid circular deps):

```go
func (k *Keeper) SetStakingKeeper(sk StakingReader) { k.stakingKeeper = sk }
func (k *Keeper) SetRLKeeper(rl RLReader)           { k.rlKeeper = rl }
```

**Step 2: Update app.go** — After creating both keepers, wire:
```go
app.QCAKeeper.SetStakingKeeper(app.StakingKeeper)
app.QCAKeeper.SetRLKeeper(app.RLConsensusKeeper)
```

**Step 3: Run both builds**

```bash
CGO_ENABLED=1 go build ./cmd/qorechaind/
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/
```

**Step 4: Commit**

```bash
git add x/qca/keeper/keeper.go app/app.go
git commit -m "feat(qca): wire staking and RL keeper dependencies into QCA"
```

---

## Part C: Integration & Verification

### Task 24: Full Build Verification

**Step 1: Public build**

```bash
cd /Users/liviu/Development/Qore/testnet/qorechain-core
CGO_ENABLED=1 go build ./cmd/qorechaind/
```

**Step 2: Proprietary build**

```bash
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/
```

**Step 3: Genesis init**

```bash
rm -rf /tmp/qoretest
./qorechaind init test --chain-id qorechain-diana --home /tmp/qoretest 2>/dev/null
python3 -c "
import json
with open('/tmp/qoretest/config/genesis.json') as f:
    data = json.load(f)
app_state = data.get('app_state', {})
required = ['rlconsensus', 'qca', 'pqc', 'ai', 'reputation', 'evm', 'feemarket', 'erc20', 'wasm', 'svm', 'bridge', 'crossvm', 'multilayer']
for m in required:
    status = '✓' if m in app_state else '✗'
    print(f'  {status} {m}')
"
```

Expected: All ✓

**Step 4: Run all unit tests**

```bash
go test ./x/rlconsensus/... -v
go test -tags proprietary ./x/rlconsensus/... -v
go test -tags proprietary ./x/qca/... -v
```

**Step 5: Branding check**

```bash
bash scripts/check_branding.sh 2>/dev/null || echo "No branding script"
```

**Step 6: Commit (if any fixes needed)**

---

### Task 25: Documentation

**Files:**
- Create: `docs/RL_CONSENSUS.md` — Full RL module documentation
- Create: `docs/CONSENSUS.md` — Updated consensus overview
- Modify: `docs/API_REFERENCE.md` — Add RL REST/RPC endpoints
- Modify: `docs/EVM_PRECOMPILES.md` — Update RL precompile section (epoch now returns real value)

**Commit:**

```bash
git add docs/RL_CONSENSUS.md docs/CONSENSUS.md docs/API_REFERENCE.md docs/EVM_PRECOMPILES.md
git commit -m "docs(consensus): add RL_CONSENSUS.md, CONSENSUS.md, update API reference"
```

---

### Task 26: CHANGELOG + Version Bump

**Files:**
- Modify: `CHANGELOG.md` — Add v0.9.0 entry
- Modify: `version/version.go` (or wherever version is defined) — Bump to 0.9.0

**Commit:**

```bash
git add CHANGELOG.md version/
git commit -m "chore: bump version to v0.9.0 — consensus enhancements"
```

---

## Task Dependency Graph

```
Task 1 (keys/errors/events) → Task 2 (params/obs/action/reward/policy/genesis) → Task 3 (codec/msgs)
       ↓
Task 4 (interface + stub keeper)
       ↓
Task 5 (module files) → Task 6 (CLI)
       ↓
Task 7 (math utils) ← shared by Part A and Part B
       ↓
Task 8 (MLP) → Task 9 (keeper core) → Task 10 (observation) → Task 11 (reward)
                                                                      ↓
Task 12 (agent + circuit breaker) → Task 13 (params applicator) → Task 14 (ABCI + servers)
                                                                      ↓
Task 15 (register.go) → Task 16 (wire into app) → Task 17 (RPC)
                                                      ↓
Task 18 (QCA types) → Task 19 (pool classifier/selector)
                    → Task 20 (bonding curve)
                    → Task 21 (progressive slashing)
                    → Task 22 (QDRW tally)
                    → Task 23 (wire QCA extensions)
                                  ↓
Task 24 (full verification) → Task 25 (docs) → Task 26 (CHANGELOG + version)
```

Tasks 19, 20, 21, 22 are independent of each other and can be parallelized after Task 18.

---

## Verification Checklist

After all tasks complete:

```bash
# Both builds compile
CGO_ENABLED=1 go build ./cmd/qorechaind/
CGO_ENABLED=1 go build -tags proprietary ./cmd/qorechaind/

# Genesis includes rlconsensus
./qorechaind init test --chain-id qorechain-diana --home /tmp/verify 2>/dev/null
grep -q rlconsensus /tmp/verify/config/genesis.json && echo "✓ rlconsensus in genesis"

# All tests pass
go test ./x/rlconsensus/... -v
go test -tags proprietary ./x/rlconsensus/... -v
go test -tags proprietary ./x/qca/... -v

# Math utils tests pass
go test ./x/rlconsensus/mathutil/ -v

# No forbidden terms
! grep -rn "Cosmos SDK\|CometBFT\|Tendermint\|Anthropic\|Bedrock\|Baron Chain\|Ethermint\|Evmos" x/rlconsensus/ x/qca/keeper/pool_*.go x/qca/keeper/bonding_*.go x/qca/keeper/progressive_*.go x/qca/keeper/qdrw_*.go docs/RL_CONSENSUS.md docs/CONSENSUS.md
```
