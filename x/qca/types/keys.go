package types

const (
	ModuleName = "qca"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ConfigKey                = []byte("qca/config")
	StatsKey                 = []byte("qca/stats")
	PoolClassificationPrefix = []byte("qca/pool/")        // + validator_addr
	SlashingRecordPrefix     = []byte("qca/slash/")       // + validator_addr + "/" + height
	BondingCurveStateKey     = []byte("qca/bonding_state")
)

// PoolClassificationKey returns the key for a validator's pool classification.
func PoolClassificationKey(validatorAddr string) []byte {
	return append(PoolClassificationPrefix, []byte(validatorAddr)...)
}

// SlashingRecordKey returns the key for a validator's slashing record at a height.
func SlashingRecordKey(validatorAddr string, height int64) []byte {
	key := append(SlashingRecordPrefix, []byte(validatorAddr)...)
	key = append(key, '/')
	// Big-endian height encoding for sorted iteration
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
