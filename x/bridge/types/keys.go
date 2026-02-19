package types

const (
	ModuleName = "bridge"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ConfigKey           = []byte("bridge/config")
	ChainConfigPrefix   = []byte("bridge/chains/")
	ValidatorPrefix     = []byte("bridge/validators/")
	OperationPrefix     = []byte("bridge/operations/")
	LockedAmountPrefix  = []byte("bridge/locked/")
	CircuitBreakerPrefix = []byte("bridge/circuit-breakers/")
	OperationCounterKey = []byte("bridge/op-counter")
)
