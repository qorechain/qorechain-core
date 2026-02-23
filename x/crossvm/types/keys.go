package types

const (
	ModuleName = "crossvm"
	StoreKey   = ModuleName
	RouterKey  = ModuleName

	// MessageKeyPrefix is the prefix for cross-VM message storage.
	MessageKeyPrefix = "msg/"

	// QueueKeyPrefix is the prefix for the pending message queue.
	QueueKeyPrefix = "queue/"

	// ParamsKey is the key for module parameters.
	ParamsKey = "params"
)

// CrossVM precompile address: 0x0000000000000000000000000000000000000901
const PrecompileAddress = "0x0000000000000000000000000000000000000901"

// MessageStoreKey returns the store key for a cross-VM message by ID.
func MessageStoreKey(id string) []byte {
	return append([]byte(MessageKeyPrefix), []byte(id)...)
}

// QueueStoreKey returns the store key for a queued message by ID.
func QueueStoreKey(id string) []byte {
	return append([]byte(QueueKeyPrefix), []byte(id)...)
}
