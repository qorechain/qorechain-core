package types

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	ModuleName = "lightnode"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ParamsKey        = []byte("lightnode/params")
	LightNodePrefix  = []byte("lightnode/nodes/")
	RewardPoolKey    = []byte("lightnode/reward-pool")
	StatsKey         = []byte("lightnode/stats")
)

// LightNodeKey returns the store key for a specific light node by operator address.
func LightNodeKey(address sdk.AccAddress) []byte {
	return append(LightNodePrefix, address.Bytes()...)
}
