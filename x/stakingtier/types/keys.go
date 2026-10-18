package types

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"

	inflationtypes "github.com/qorechain/qorechain-core/x/inflation/types"
)

const (
	ModuleName = "stakingtier"
	RouterKey  = ModuleName
)

// HostStoreKey names the KV store this module keeps its state in: the
// x/inflation store, under KeyPrefix.
//
// WHY NOT A STORE OF ITS OWN. Every mounted store is part of the app hash from
// the first block, empty or not. A "stakingtier" store mounted by this binary
// would give every block it replays below the v3.2.4 plan height a different
// app hash than the one the network committed, so a node could no longer
// follow the chain's history on this binary, which the release process relies
// on. Adding the store at the plan height through a store loader fixes the
// live upgrade but not that replay. The module's records therefore live in an
// existing store, under a prefix no other record uses: x/inflation's own keys
// all start with "inflation/", and nothing iterates that store as a whole.
// The bonus is an x/inflation emission channel, so its accounting sits next
// to the emission it is funded from.
const HostStoreKey = inflationtypes.StoreKey

// KeyPrefix starts every key this module writes.
var KeyPrefix = []byte("stakingtier/")

var (
	ParamsKey            = []byte("stakingtier/params")
	PoolKey              = []byte("stakingtier/pool")
	NextLockIDKey        = []byte("stakingtier/next_lock_id")
	LockPrefix           = []byte("stakingtier/lock/")
	DelegatorIndexPrefix = []byte("stakingtier/dl/")
	ExpiryQueuePrefix    = []byte("stakingtier/exp/")
	ClaimablePrefix      = []byte("stakingtier/claim/")
)

// MaxLocksPerDelegator bounds the open locks one delegator may hold. Every
// undelegation and redelegation of a holder reads that holder's open locks, so
// the bound keeps the gas of those messages bounded too.
const MaxLocksPerDelegator = 16

func u64(v uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, v)
	return bz
}

// ParseU64 decodes a big-endian uint64 written by this package.
func ParseU64(bz []byte) uint64 {
	if len(bz) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(bz[:8])
}

// EncodeU64 encodes a big-endian uint64.
func EncodeU64(v uint64) []byte { return u64(v) }

// LockKey is where lock id is stored.
func LockKey(id uint64) []byte {
	return append(append([]byte{}, LockPrefix...), u64(id)...)
}

// DelegatorIndexPrefixFor prefixes every index entry of one delegator.
func DelegatorIndexPrefixFor(del sdk.AccAddress) []byte {
	out := append([]byte{}, DelegatorIndexPrefix...)
	out = append(out, byte(len(del)))
	return append(out, del.Bytes()...)
}

// DelegatorIndexKey indexes lock id under its delegator.
func DelegatorIndexKey(del sdk.AccAddress, id uint64) []byte {
	return append(DelegatorIndexPrefixFor(del), u64(id)...)
}

// ExpiryQueueKey orders a lock by its end time, then id, so the end blocker
// reads only the locks that are due.
func ExpiryQueueKey(endTime int64, id uint64) []byte {
	out := append([]byte{}, ExpiryQueuePrefix...)
	out = append(out, u64(uint64(endTime))...)
	return append(out, u64(id)...)
}

// ExpiryQueueEnd is the exclusive end of the queue entries due at or before
// now.
func ExpiryQueueEnd(now int64) []byte {
	out := append([]byte{}, ExpiryQueuePrefix...)
	return append(out, u64(uint64(now)+1)...)
}

// ClaimableKey holds what an address can claim outside its open locks.
func ClaimableKey(addr sdk.AccAddress) []byte {
	return append(append([]byte{}, ClaimablePrefix...), addr.Bytes()...)
}
