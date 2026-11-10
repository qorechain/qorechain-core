package qledger

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// Bounds that keep every fraud proof sendable in one transaction.
const (
	// MaxBatchEntries is the most inbox entries one qledger-v1 batch may
	// consume.
	MaxBatchEntries = 64
	// MaxPostBytes is the largest post a qledger-v1 rollup accepts (an
	// ML-DSA-44 transaction is about 3.8 KiB).
	MaxPostBytes = 6 * 1024
	// maxWitnesses: an entry touches at most two accounts.
	maxWitnesses = 2 * MaxBatchEntries
	maxPosterLen = 256
)

const fraudMagic = "QLFP"

// FraudProof is what a challenger sends against a qledger-v1 batch: the
// inbox accumulator at the batch's start, every entry it consumed, and a
// witness for every account those entries touch, against the batch's
// previous state root.
type FraudProof struct {
	AccFrom   [32]byte
	Entries   []Entry
	Witnesses []Witness
}

// Encode is the proof's wire form:
//
//	"QLFP" | 0x01 | acc_from[32] | n (u16) | n x (height u64 | lp16 poster | lp32 payload) |
//	m (u16) | m x (id[32] | balance u64 | nonce u64 | bitmap[32] | popcount(bitmap) x sibling[32])
func (p FraudProof) Encode() []byte {
	b := append([]byte(fraudMagic), 1)
	b = append(b, p.AccFrom[:]...)
	b = binary.BigEndian.AppendUint16(b, uint16(len(p.Entries)))
	for _, e := range p.Entries {
		b = binary.BigEndian.AppendUint64(b, uint64(e.Height))
		b = binary.BigEndian.AppendUint16(b, uint16(len(e.Poster)))
		b = append(b, e.Poster...)
		b = binary.BigEndian.AppendUint32(b, uint32(len(e.Payload)))
		b = append(b, e.Payload...)
	}
	b = binary.BigEndian.AppendUint16(b, uint16(len(p.Witnesses)))
	for _, w := range p.Witnesses {
		b = append(b, w.ID[:]...)
		b = binary.BigEndian.AppendUint64(b, w.Balance)
		b = binary.BigEndian.AppendUint64(b, w.Nonce)
		b = append(b, w.Bitmap[:]...)
		for _, s := range w.Siblings {
			b = append(b, s[:]...)
		}
	}
	return b
}

var errMalformed = errors.New("qledger: malformed fraud proof")

// DecodeFraudProof parses a proof, refusing anything over the bounds or with
// trailing bytes.
func DecodeFraudProof(b []byte) (FraudProof, error) {
	var p FraudProof
	r := bytes.NewReader(b)
	take := func(n int) ([]byte, bool) {
		if n < 0 || r.Len() < n {
			return nil, false
		}
		out := make([]byte, n)
		_, _ = r.Read(out)
		return out, true
	}
	head, ok := take(5)
	if !ok || string(head[:4]) != fraudMagic || head[4] != 1 {
		return p, errMalformed
	}
	acc, ok := take(32)
	if !ok {
		return p, errMalformed
	}
	copy(p.AccFrom[:], acc)
	nb, ok := take(2)
	if !ok {
		return p, errMalformed
	}
	n := int(binary.BigEndian.Uint16(nb))
	if n > MaxBatchEntries {
		return p, fmt.Errorf("%w: %d entries, at most %d", errMalformed, n, MaxBatchEntries)
	}
	for i := 0; i < n; i++ {
		hb, ok1 := take(8)
		lb, ok2 := take(2)
		if !ok1 || !ok2 {
			return p, errMalformed
		}
		pl := int(binary.BigEndian.Uint16(lb))
		if pl > maxPosterLen {
			return p, errMalformed
		}
		poster, ok3 := take(pl)
		plb, ok4 := take(4)
		if !ok3 || !ok4 {
			return p, errMalformed
		}
		ln := binary.BigEndian.Uint32(plb)
		if ln > MaxPostBytes {
			return p, errMalformed
		}
		payload, ok5 := take(int(ln))
		if !ok5 {
			return p, errMalformed
		}
		p.Entries = append(p.Entries, Entry{Height: int64(binary.BigEndian.Uint64(hb)), Poster: string(poster), Payload: payload})
	}
	mb, ok := take(2)
	if !ok {
		return p, errMalformed
	}
	m := int(binary.BigEndian.Uint16(mb))
	if m > maxWitnesses {
		return p, errMalformed
	}
	for i := 0; i < m; i++ {
		fixed, ok := take(32 + 8 + 8 + 32)
		if !ok {
			return p, errMalformed
		}
		var w Witness
		copy(w.ID[:], fixed[:32])
		w.Balance = binary.BigEndian.Uint64(fixed[32:40])
		w.Nonce = binary.BigEndian.Uint64(fixed[40:48])
		copy(w.Bitmap[:], fixed[48:80])
		cnt := 0
		for _, x := range w.Bitmap {
			cnt += bits.OnesCount8(x)
		}
		for j := 0; j < cnt; j++ {
			s, ok := take(32)
			if !ok {
				return p, errMalformed
			}
			var sib [32]byte
			copy(sib[:], s)
			w.Siblings = append(w.Siblings, sib)
		}
		p.Witnesses = append(p.Witnesses, w)
	}
	if r.Len() != 0 {
		return p, errMalformed
	}
	return p, nil
}

// Claim is what a qledger-v1 batch published.
type Claim struct {
	InboxStart      uint64
	InboxEnd        uint64
	DataHash        []byte
	PrevStateRoot   []byte
	StateRoot       []byte
	WithdrawalsRoot []byte
}

// ErrProofNotApplicable: the proof is not about this batch (wrong inputs or
// a witness that does not open the batch's parent state). Nothing is decided.
var ErrProofNotApplicable = errors.New("qledger: the fraud proof does not apply to this batch")

// CheckFraud decides a fraud proof. It returns (true, nil) when re-executing
// the batch's inputs from its parent state gives a different state root or
// withdrawals root than the batch published, (false, nil) when the batch is
// right, and an error when the proof cannot decide (malformed, not about this
// batch, incomplete witness).
func CheckFraud(env Env, c Claim, proof []byte) (bool, error) {
	p, err := DecodeFraudProof(proof)
	if err != nil {
		return false, err
	}
	if c.InboxEnd < c.InboxStart || uint64(len(p.Entries)) != c.InboxEnd-c.InboxStart {
		return false, fmt.Errorf("%w: %d entries for inbox range %d..%d", ErrProofNotApplicable,
			len(p.Entries), c.InboxStart, c.InboxEnd)
	}
	// 1. The entries are exactly the posts the batch consumed, in order.
	acc := p.AccFrom[:]
	for i, e := range p.Entries {
		env.charge(2 * HashGas)
		acc = types.InboxAccumulate(env.ChainID, env.RollupID, acc, c.InboxStart+uint64(i)+1, e.Height, e.Poster,
			types.PostedDataHash(e.Payload))
	}
	want := types.InboxRangeHash(env.ChainID, env.RollupID, c.InboxStart, c.InboxEnd, p.AccFrom[:], acc)
	if !bytes.Equal(want, c.DataHash) {
		return false, fmt.Errorf("%w: the entries do not hash to the batch's inbox range", ErrProofNotApplicable)
	}
	// 2. The witnesses open the batch's parent state.
	if len(c.PrevStateRoot) != 32 {
		return false, fmt.Errorf("%w: the batch has no 32-byte parent root", ErrProofNotApplicable)
	}
	var prev [32]byte
	copy(prev[:], c.PrevStateRoot)
	tree, err := NewPartialTree(prev, p.Witnesses, env.Charge)
	if err != nil {
		return false, err
	}
	// 3. Re-execute.
	res, err := Execute(env, tree, p.Entries)
	if err != nil {
		return false, err
	}
	// 4. Compare.
	return !bytes.Equal(res.StateRoot[:], c.StateRoot) || !bytes.Equal(res.WithdrawalsRoot, c.WithdrawalsRoot), nil
}

// recorder notes every account an execution touches.
type recorder struct {
	State
	seen  map[[32]byte]bool
	order [][32]byte
}

func (r *recorder) note(id [32]byte) {
	if !r.seen[id] {
		r.seen[id] = true
		r.order = append(r.order, id)
	}
}
func (r *recorder) Get(id [32]byte) (uint64, uint64, error) { r.note(id); return r.State.Get(id) }
func (r *recorder) Set(id [32]byte, b, n uint64) error      { r.note(id); return r.State.Set(id, b, n) }

// BuildFraudProof is the challenger's side: given the parent state, the
// accumulator at the batch's start and its entries, it returns the proof
// and the result an honest batch would have published.
func BuildFraudProof(env Env, parent *Ledger, accFrom []byte, entries []Entry) ([]byte, Result, error) {
	rec := &recorder{State: parent.Clone(), seen: map[[32]byte]bool{}}
	res, err := Execute(Env{ChainID: env.ChainID, RollupID: env.RollupID}, rec, entries)
	if err != nil {
		return nil, res, err
	}
	p := FraudProof{Entries: entries}
	copy(p.AccFrom[:], types.AccOrEmpty(accFrom))
	for _, id := range rec.order {
		p.Witnesses = append(p.Witnesses, parent.Witness(id))
	}
	return p.Encode(), res, nil
}
