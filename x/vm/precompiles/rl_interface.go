package precompiles

import (
	"math/big"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RLConsensusParamsProvider is the interface that the future RL module (Chapter 2, Section 2.1)
// will implement. Until then, StaticRLProvider returns genesis defaults.
type RLConsensusParamsProvider interface {
	GetCurrentBlockTime(ctx sdk.Context) time.Duration
	GetCurrentBaseGasPrice(ctx sdk.Context) math.LegacyDec
	GetValidatorSetSize(ctx sdk.Context) uint64
	GetCurrentEpoch(ctx sdk.Context) uint64
	IsRLActive(ctx sdk.Context) bool
}

// StaticRLProvider returns static genesis parameters.
// Used as the default until the RL module is implemented.
type StaticRLProvider struct {
	BlockTimeMs      uint64
	BaseGasPrice     uint64
	ValidatorSetSize uint64
}

// DefaultStaticRLProvider returns a provider with testnet default values.
func DefaultStaticRLProvider() *StaticRLProvider {
	return &StaticRLProvider{
		BlockTimeMs:      5000, // 5 seconds
		BaseGasPrice:     100,  // 100 uqor
		ValidatorSetSize: 100,
	}
}

func (p *StaticRLProvider) GetCurrentBlockTime(_ sdk.Context) time.Duration {
	return time.Duration(p.BlockTimeMs) * time.Millisecond
}

func (p *StaticRLProvider) GetCurrentBaseGasPrice(_ sdk.Context) math.LegacyDec {
	return math.LegacyNewDecFromBigInt(new(big.Int).SetUint64(p.BaseGasPrice))
}

func (p *StaticRLProvider) GetValidatorSetSize(_ sdk.Context) uint64 {
	return p.ValidatorSetSize
}

func (p *StaticRLProvider) GetCurrentEpoch(_ sdk.Context) uint64 {
	return 0 // No RL training yet
}

func (p *StaticRLProvider) IsRLActive(_ sdk.Context) bool {
	return false
}
