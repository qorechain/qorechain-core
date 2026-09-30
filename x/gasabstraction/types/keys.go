package types

const (
	ModuleName = "gasabstraction"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ConfigKey = []byte("ga/config")
	// BlockConversionUsedKey holds the native amount the float has paid out
	// in the current block: height (8 bytes BE) | amount (decimal string).
	BlockConversionUsedKey = []byte("ga/block_used")
)
