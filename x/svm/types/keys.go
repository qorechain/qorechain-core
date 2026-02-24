package types

import (
	"encoding/binary"
	"fmt"
)

const (
	// ModuleName defines the module name for the SVM runtime
	ModuleName = "svm"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_svm"
)

// KVStore key prefixes for the SVM module
var (
	// AccountKeyPrefix stores SVM accounts: 0x01 | 32-byte-addr -> SVMAccount
	AccountKeyPrefix = []byte{0x01}

	// ProgramKeyPrefix stores program metadata: 0x02 | 32-byte-addr -> ProgramMeta
	ProgramKeyPrefix = []byte{0x02}

	// AddrMapKeyPrefix maps native addresses to SVM addresses: 0x03 | 20-byte-cosmos-addr -> [32]byte
	AddrMapKeyPrefix = []byte{0x03}

	// RentEpochKey stores the current rent epoch: single key -> uint64
	RentEpochKey = []byte{0x04}

	// ParamsKey stores module parameters: single key -> Params
	ParamsKey = []byte{0x05}

	// SlotKey stores the current SVM slot: single key -> uint64
	SlotKey = []byte{0x06}

	// RecentBlockhashPrefix stores recent blockhashes: 0x07 | height(8 bytes) -> [32]byte
	RecentBlockhashPrefix = []byte{0x07}
)

// AccountKey returns the store key for an SVM account
func AccountKey(addr [32]byte) []byte {
	key := make([]byte, 1, 1+32)
	key[0] = AccountKeyPrefix[0]
	return append(key, addr[:]...)
}

// ProgramKey returns the store key for a program's metadata
func ProgramKey(addr [32]byte) []byte {
	key := make([]byte, 1, 1+32)
	key[0] = ProgramKeyPrefix[0]
	return append(key, addr[:]...)
}

// AddrMapKey returns the store key for a native-to-SVM address mapping.
// cosmosAddr must be exactly 20 bytes.
func AddrMapKey(cosmosAddr []byte) []byte {
	if len(cosmosAddr) != 20 {
		panic(fmt.Sprintf("AddrMapKey expects 20-byte address, got %d", len(cosmosAddr)))
	}
	key := make([]byte, 1, 1+20)
	key[0] = AddrMapKeyPrefix[0]
	return append(key, cosmosAddr...)
}

// RecentBlockhashKey returns the store key for a recent blockhash at a given height
func RecentBlockhashKey(height uint64) []byte {
	key := make([]byte, 1, 1+8)
	key[0] = RecentBlockhashPrefix[0]
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)
	return append(key, heightBytes...)
}
