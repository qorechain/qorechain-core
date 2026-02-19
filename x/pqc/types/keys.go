package types

const (
	// ModuleName defines the module name.
	ModuleName = "pqc"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key.
	RouterKey = ModuleName
)

// KV store key prefixes.
var (
	AccountPrefix = []byte("pqc/accounts/")
	ParamsKey     = []byte("pqc/params")
	StatsKey      = []byte("pqc/stats")
)
