package types

const (
	ModuleName = "babylon"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ConfigKey           = []byte("babylon/config")
	StakingPositionPrefix = []byte("babylon/pos/")
	CheckpointPrefix    = []byte("babylon/cp/")
	EpochSnapshotPrefix = []byte("babylon/epoch/")
)
