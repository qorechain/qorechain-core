package types

import (
	"crypto/sha256"
	"encoding/binary"
)

// Posted rollup data and the rollup inbox (v3.2.8, FeatureRDKPostRollupData
// and FeatureRDKBasedSequencing). Design: handoff-v328/RDK-DATA-BASED-DESIGN.md.

// PostedDataRecord is the commitment the chain keeps for one
// MsgPostRollupData. The payload itself is not stored in state: it is in the
// transaction of the block at Height.
type PostedDataRecord struct {
	RollupID string `json:"rollup_id"`
	// Sequence is the post's position in the rollup's inbox, from 1.
	Sequence uint64 `json:"sequence"`
	// DataHash is SHA-256 of the payload: what a batch references.
	DataHash []byte `json:"data_hash"`
	Size     uint64 `json:"size"`
	Poster   string `json:"poster"`
	Height   int64  `json:"height"`
	// InboxAcc is the inbox accumulator after this post.
	InboxAcc []byte `json:"inbox_acc"`
	// ExpiresAt is the height from which EndBlock prunes the commitment.
	ExpiresAt int64 `json:"expires_at"`
	// Holds counts the retention periods a qledger-v1 rollup kept the
	// commitment past its expiry because no finalized batch covered it yet;
	// at MaxQLedgerPostHolds it is pruned like any other.
	Holds uint32 `json:"holds,omitempty"`
}

// InboxState is a rollup's inbox: the last post (head) and the position the
// next based batch starts from (consumed). An empty accumulator stands for
// the 32 zero bytes of an empty inbox.
type InboxState struct {
	RollupID         string `json:"rollup_id"`
	HeadSequence     uint64 `json:"head_sequence"`
	HeadAcc          []byte `json:"head_acc,omitempty"`
	ConsumedSequence uint64 `json:"consumed_sequence"`
	ConsumedAcc      []byte `json:"consumed_acc,omitempty"`
	// FinalSequence is the inbox end of the rollup's latest finalized based
	// batch (v3.2.8). Commitments of a qledger-v1 rollup after it are not
	// pruned, so a range a fraud proof rolls back can be consumed again.
	FinalSequence uint64 `json:"final_sequence,omitempty"`
}

// EmptyInboxAcc is the accumulator of an inbox with no post.
var EmptyInboxAcc = make([]byte, 32)

// AccOrEmpty returns acc, or EmptyInboxAcc when acc is empty.
func AccOrEmpty(acc []byte) []byte {
	if len(acc) == 0 {
		return EmptyInboxAcc
	}
	return acc
}

// PostedDataHash is the hash a post is committed and referenced by.
func PostedDataHash(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

const (
	inboxAccDomain   = "qorechain/rdk/inbox/v1"
	inboxRangeDomain = "qorechain/rdk/inbox-range/v1"
)

func writeLP(h interface{ Write([]byte) (int, error) }, b []byte) {
	var n [4]byte
	binary.BigEndian.PutUint32(n[:], uint32(len(b)))
	h.Write(n[:])
	h.Write(b)
}

func writeU64(h interface{ Write([]byte) (int, error) }, v uint64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], v)
	h.Write(b[:])
}

// InboxAccumulate extends a rollup's inbox accumulator by one post:
//
//	acc_n = SHA-256("qorechain/rdk/inbox/v1" | lp(chain_id) | lp(rollup_id) |
//	                acc_{n-1} | be64(n) | be64(height) | lp(poster) | data_hash)
//
// lp is a 4-byte big-endian length prefix. The chain id makes an inbox, and
// a proof over it, specific to one network.
func InboxAccumulate(chainID, rollupID string, prev []byte, seq uint64, height int64, poster string, dataHash []byte) []byte {
	h := sha256.New()
	h.Write([]byte(inboxAccDomain))
	writeLP(h, []byte(chainID))
	writeLP(h, []byte(rollupID))
	h.Write(AccOrEmpty(prev))
	writeU64(h, seq)
	writeU64(h, uint64(height))
	writeLP(h, []byte(poster))
	h.Write(dataHash)
	return h.Sum(nil)
}

// InboxRangeHash is the data_hash a based batch consuming the posts after
// from up to to must carry:
//
//	SHA-256("qorechain/rdk/inbox-range/v1" | lp(chain_id) | lp(rollup_id) |
//	        be64(from) | be64(to) | acc_from | acc_to)
func InboxRangeHash(chainID, rollupID string, from, to uint64, accFrom, accTo []byte) []byte {
	h := sha256.New()
	h.Write([]byte(inboxRangeDomain))
	writeLP(h, []byte(chainID))
	writeLP(h, []byte(rollupID))
	writeU64(h, from)
	writeU64(h, to)
	h.Write(AccOrEmpty(accFrom))
	h.Write(AccOrEmpty(accTo))
	return h.Sum(nil)
}
