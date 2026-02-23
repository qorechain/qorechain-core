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
	AccountPrefix   = []byte("pqc/accounts/")
	ParamsKey       = []byte("pqc/params")
	StatsKey        = []byte("pqc/stats")

	// Algorithm agility keys (v0.6.0)
	AlgorithmPrefix = []byte("pqc/algorithms/")
	MigrationPrefix = []byte("pqc/migrations/")
)

// AlgorithmKey returns the KV store key for an algorithm by ID.
func AlgorithmKey(id AlgorithmID) []byte {
	return append(AlgorithmPrefix, []byte(id.String())...)
}

// MigrationKey returns the KV store key for a migration by source algorithm ID.
func MigrationKey(fromID AlgorithmID) []byte {
	return append(MigrationPrefix, []byte(fromID.String())...)
}
