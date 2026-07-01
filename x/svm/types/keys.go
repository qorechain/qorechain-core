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

	// TxRecordPrefix stores SVM transaction records: 0x08 | signature -> SVMTxRecord
	TxRecordPrefix = []byte{0x08}

	// TxBySeqPrefix maps a monotonic sequence to a signature (for pruning):
	// 0x09 | seq(8 bytes) -> signature
	TxBySeqPrefix = []byte{0x09}

	// AddrTxPrefix indexes signatures by involved address, newest-first:
	// 0x0A | 32-byte-addr | invSeq(8 bytes) | signature -> {1}
	AddrTxPrefix = []byte{0x0A}

	// TxSeqCounterKey stores the next transaction sequence: single key -> uint64
	TxSeqCounterKey = []byte{0x0B}

	// DustKeyPrefix stores sub-uqor lamport dust per SVM address so no value is
	// lost when converting between the uqor (x/bank) and lamport (SVM) ledgers:
	// 0x0C | 32-byte-addr -> uint64 (big-endian, 0..LamportsPerUqor-1)
	DustKeyPrefix = []byte{0x0C}
)

// DustKey returns the store key for an SVM address's sub-uqor lamport dust.
func DustKey(addr [32]byte) []byte {
	key := make([]byte, 1, 1+32)
	key[0] = DustKeyPrefix[0]
	return append(key, addr[:]...)
}

// TxRecordKey returns the store key for an SVM transaction record by signature.
func TxRecordKey(signature []byte) []byte {
	key := make([]byte, 1, 1+len(signature))
	key[0] = TxRecordPrefix[0]
	return append(key, signature...)
}

// TxBySeqKey returns the store key mapping a sequence number to a signature.
func TxBySeqKey(seq uint64) []byte {
	key := make([]byte, 1, 1+8)
	key[0] = TxBySeqPrefix[0]
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, seq)
	return append(key, b...)
}

// AddrTxIterPrefix returns the iteration prefix for an address's tx index.
func AddrTxIterPrefix(addr [32]byte) []byte {
	key := make([]byte, 1, 1+32)
	key[0] = AddrTxPrefix[0]
	return append(key, addr[:]...)
}

// AddrTxKey returns the per-address tx-index key. invSeq = MaxUint64 - seq so
// that ascending iteration yields newest-first ordering.
func AddrTxKey(addr [32]byte, invSeq uint64, signature []byte) []byte {
	prefix := AddrTxIterPrefix(addr)
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, invSeq)
	key := append(prefix, b...)
	return append(key, signature...)
}

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
