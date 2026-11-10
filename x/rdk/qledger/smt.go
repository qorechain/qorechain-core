// Package qledger is the qledger-v1 execution model of RDK optimistic
// rollups (v3.2.8, FeatureRDKOptimisticReexec): an account ledger of uqor
// with signed transfers and withdrawals, small enough that the chain can
// re-execute a whole batch inside one fraud-proof transaction.
//
// Everything here is deterministic and pure: the chain uses it to decide a
// fraud proof, and sequencers, challengers and clients use the same code to
// compute the roots they publish. The specification is in
// testnet/handoff-v328/RDK-DATA-BASED-DESIGN.md, section 12.
package qledger

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sort"
)

// Depth is the depth of the state tree: one level per bit of an account id.
const Depth = 256

// HashGas is the gas charged per SHA-256 the chain computes while checking
// a fraud proof (tree nodes, leaves, inbox accumulators).
const HashGas = 50

var (
	// ErrIncompleteWitness: execution touched an account the proof did not
	// open. The proof is refused; it says nothing about the batch.
	ErrIncompleteWitness = errors.New("qledger: the witness does not open an account the batch touches")
	// ErrBadWitness: a witness does not verify against the root, or two
	// witnesses disagree about a node.
	ErrBadWitness = errors.New("qledger: witness does not match the state root")
)

// emptyHashes[h] is the root of an empty subtree of height h; emptyHashes[0]
// is the empty leaf.
var emptyHashes [Depth + 1][32]byte

func init() {
	emptyHashes[0] = sha256.Sum256(append([]byte{0x00}, "qledger/v1/empty"...))
	for h := 1; h <= Depth; h++ {
		emptyHashes[h] = hashNode(emptyHashes[h-1], emptyHashes[h-1])
	}
}

// EmptyRoot is the state root of a ledger with no account: the parent root
// of a qledger-v1 rollup's first batch.
func EmptyRoot() [32]byte { return emptyHashes[Depth] }

func hashNode(l, r [32]byte) [32]byte {
	var b [65]byte
	b[0] = 0x01
	copy(b[1:33], l[:])
	copy(b[33:], r[:])
	return sha256.Sum256(b[:])
}

// LeafHash is the leaf of an account. An account with a zero balance and a
// zero nonce is the empty leaf, so the root of a state is unique.
func LeafHash(id [32]byte, balance, nonce uint64) [32]byte {
	if balance == 0 && nonce == 0 {
		return emptyHashes[0]
	}
	var b [49]byte
	b[0] = 0x00
	copy(b[1:33], id[:])
	binary.BigEndian.PutUint64(b[33:41], balance)
	binary.BigEndian.PutUint64(b[41:49], nonce)
	return sha256.Sum256(b[:])
}

// bit returns bit i of id, most significant first: bit 0 chooses the side
// under the root, bit 255 the side of the leaf.
func bit(id [32]byte, i int) byte { return (id[i/8] >> (7 - uint(i%8))) & 1 }

type nodeKey struct {
	depth  uint16
	prefix [32]byte
}

// keyOf names the node at depth d on id's path: id's first d bits.
func keyOf(id [32]byte, d int) nodeKey {
	k := nodeKey{depth: uint16(d)}
	for i := 0; i < d/8; i++ {
		k.prefix[i] = id[i]
	}
	if r := d % 8; r != 0 {
		k.prefix[d/8] = id[d/8] & byte(0xff<<(8-uint(r)))
	}
	return k
}

// sibKey names the sibling of the node at depth d (d >= 1) on id's path.
func sibKey(id [32]byte, d int) nodeKey {
	flipped := id
	flipped[(d-1)/8] ^= 1 << (7 - uint((d-1)%8))
	return keyOf(flipped, d)
}

// Witness opens one account against a state root. Siblings are listed from
// the leaf up (depth 256 to depth 1), only the non-empty ones: bit d-1 of
// Bitmap (most significant bit first) is set when the sibling at depth d is
// given.
type Witness struct {
	ID       [32]byte
	Balance  uint64
	Nonce    uint64
	Bitmap   [32]byte
	Siblings [][32]byte
}

// State is what execution reads and writes. The chain uses a PartialTree
// built from a fraud proof's witness; sequencers and tests use a Ledger.
type State interface {
	Get(id [32]byte) (balance, nonce uint64, err error)
	Set(id [32]byte, balance, nonce uint64) error
	Root() [32]byte
}

// PartialTree is the part of a state tree the witnesses open. Reads of an
// account outside it fail with ErrIncompleteWitness; writes recompute the
// root from the nodes it holds.
type PartialTree struct {
	nodes  map[nodeKey][32]byte
	accts  map[[32]byte][2]uint64
	root   [32]byte
	charge func(uint64)
}

// NewPartialTree checks every witness against root and joins them into one
// tree. A witness that does not reach root, or that disagrees with another
// about any node, refuses the whole set.
func NewPartialTree(root [32]byte, ws []Witness, charge func(uint64)) (*PartialTree, error) {
	if charge == nil {
		charge = func(uint64) {}
	}
	t := &PartialTree{nodes: map[nodeKey][32]byte{}, accts: map[[32]byte][2]uint64{}, root: root, charge: charge}
	put := func(k nodeKey, v [32]byte) error {
		if old, ok := t.nodes[k]; ok && old != v {
			return ErrBadWitness
		}
		t.nodes[k] = v
		return nil
	}
	for _, w := range ws {
		charge(HashGas)
		cur := LeafHash(w.ID, w.Balance, w.Nonce)
		if err := put(keyOf(w.ID, Depth), cur); err != nil {
			return nil, err
		}
		si := 0
		for d := Depth; d >= 1; d-- {
			sib := emptyHashes[Depth-d]
			if (w.Bitmap[(d-1)/8]>>(7-uint((d-1)%8)))&1 == 1 {
				if si >= len(w.Siblings) {
					return nil, ErrBadWitness
				}
				sib = w.Siblings[si]
				si++
			}
			if err := put(sibKey(w.ID, d), sib); err != nil {
				return nil, err
			}
			if bit(w.ID, d-1) == 0 {
				cur = hashNode(cur, sib)
			} else {
				cur = hashNode(sib, cur)
			}
			charge(HashGas)
			if err := put(keyOf(w.ID, d-1), cur); err != nil {
				return nil, err
			}
		}
		if si != len(w.Siblings) || cur != root {
			return nil, ErrBadWitness
		}
		t.accts[w.ID] = [2]uint64{w.Balance, w.Nonce}
	}
	return t, nil
}

// Get returns an opened account.
func (t *PartialTree) Get(id [32]byte) (uint64, uint64, error) {
	a, ok := t.accts[id]
	if !ok {
		return 0, 0, ErrIncompleteWitness
	}
	return a[0], a[1], nil
}

// Set updates an opened account and the root.
func (t *PartialTree) Set(id [32]byte, balance, nonce uint64) error {
	if _, ok := t.accts[id]; !ok {
		return ErrIncompleteWitness
	}
	t.accts[id] = [2]uint64{balance, nonce}
	cur := LeafHash(id, balance, nonce)
	t.charge(HashGas)
	t.nodes[keyOf(id, Depth)] = cur
	for d := Depth; d >= 1; d-- {
		sib, ok := t.nodes[sibKey(id, d)]
		if !ok {
			return ErrIncompleteWitness
		}
		if bit(id, d-1) == 0 {
			cur = hashNode(cur, sib)
		} else {
			cur = hashNode(sib, cur)
		}
		t.charge(HashGas)
		t.nodes[keyOf(id, d-1)] = cur
	}
	t.root = cur
	return nil
}

// Root is the current root.
func (t *PartialTree) Root() [32]byte { return t.root }

// Ledger is a whole qledger-v1 state in memory: the reference a sequencer or
// a challenger keeps, and what builds witnesses.
type Ledger struct {
	accts map[[32]byte][2]uint64
}

// NewLedger returns an empty ledger (root EmptyRoot()).
func NewLedger() *Ledger { return &Ledger{accts: map[[32]byte][2]uint64{}} }

// Clone returns an independent copy.
func (l *Ledger) Clone() *Ledger {
	c := NewLedger()
	for k, v := range l.accts {
		c.accts[k] = v
	}
	return c
}

// Get returns an account (zero when absent).
func (l *Ledger) Get(id [32]byte) (uint64, uint64, error) {
	a := l.accts[id]
	return a[0], a[1], nil
}

// Set writes an account; a zero account is removed.
func (l *Ledger) Set(id [32]byte, balance, nonce uint64) error {
	if balance == 0 && nonce == 0 {
		delete(l.accts, id)
		return nil
	}
	l.accts[id] = [2]uint64{balance, nonce}
	return nil
}

func (l *Ledger) sortedIDs() [][32]byte {
	ids := make([][32]byte, 0, len(l.accts))
	for id := range l.accts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		for k := 0; k < 32; k++ {
			if ids[i][k] != ids[j][k] {
				return ids[i][k] < ids[j][k]
			}
		}
		return false
	})
	return ids
}

// subtree returns the root of the subtree at depth d holding ids (all of
// which share their first d bits).
func (l *Ledger) subtree(d int, ids [][32]byte) [32]byte {
	switch {
	case len(ids) == 0:
		return emptyHashes[Depth-d]
	case len(ids) == 1:
		a := l.accts[ids[0]]
		cur := LeafHash(ids[0], a[0], a[1])
		for dd := Depth; dd > d; dd-- {
			if bit(ids[0], dd-1) == 0 {
				cur = hashNode(cur, emptyHashes[Depth-dd])
			} else {
				cur = hashNode(emptyHashes[Depth-dd], cur)
			}
		}
		return cur
	}
	split := sort.Search(len(ids), func(i int) bool { return bit(ids[i], d) == 1 })
	return hashNode(l.subtree(d+1, ids[:split]), l.subtree(d+1, ids[split:]))
}

// Root computes the state root.
func (l *Ledger) Root() [32]byte { return l.subtree(0, l.sortedIDs()) }

// Witness opens id (present or not) against Root().
func (l *Ledger) Witness(id [32]byte) Witness {
	a := l.accts[id]
	w := Witness{ID: id, Balance: a[0], Nonce: a[1]}
	ids := l.sortedIDs()
	for d := Depth; d >= 1; d-- {
		want := sibKey(id, d)
		var under [][32]byte
		for _, x := range ids {
			if keyOf(x, d) == want {
				under = append(under, x)
			}
		}
		sib := l.subtree(d, under)
		if sib != emptyHashes[Depth-d] {
			w.Bitmap[(d-1)/8] |= 1 << (7 - uint((d-1)%8))
			w.Siblings = append(w.Siblings, sib)
		}
	}
	return w
}
