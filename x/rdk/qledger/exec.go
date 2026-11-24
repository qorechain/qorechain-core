package qledger

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/qorehash"
)

// WithdrawalDenom is the denom every qledger-v1 withdrawal pays.
const WithdrawalDenom = "uqor"

// WithdrawalCounterID is the reserved state slot holding the rollup's next
// withdrawal index (in its balance field). Withdrawal indices are global to
// the rollup, because the chain's replay protection is keyed by the
// withdrawal leaf, which carries the index but not the batch: per-batch
// indices would make a second identical withdrawal in a later batch
// unpayable. No account id can be it (account ids hash another prefix), and
// transfers and deposits to it are no-ops.
var WithdrawalCounterID = func() [32]byte {
	return sha256.Sum256([]byte("qledger/v1/withdrawal-counter"))
}()

// Env is the context of an execution.
type Env struct {
	ChainID  string
	RollupID string
	// Charge is called with the gas of each hash and signature check. The
	// chain passes its gas meter; nil charges nothing.
	Charge func(uint64)
}

func (e Env) charge(g uint64) {
	if e.Charge != nil {
		e.Charge(g)
	}
}

// Entry is one inbox entry a batch consumes.
type Entry struct {
	Height  int64
	Poster  string
	Payload []byte
}

// Withdrawal is one withdrawal a batch commits, at its index.
type Withdrawal struct {
	Index     uint64
	Recipient string
	Amount    uint64
}

// Result is what a batch must publish.
type Result struct {
	StateRoot       [32]byte
	Withdrawals     []Withdrawal
	WithdrawalsRoot []byte // empty when there is no withdrawal
}

// Execute applies entries to st in order. Entries that do not parse, carry
// a bad signature, the wrong nonce, a zero amount or more than the balance
// are no-ops: an open inbox must accept anything without making a batch
// invalid. An error is returned only when st cannot serve an account
// (ErrIncompleteWitness on a PartialTree).
func Execute(env Env, st State, entries []Entry) (Result, error) {
	var res Result
	for _, e := range entries {
		if strings.HasPrefix(e.Poster, DepositPosterPrefix) {
			id, amt, ok := DecodeDeposit(e.Payload)
			if !ok || id == WithdrawalCounterID {
				continue
			}
			bal, nonce, err := st.Get(id)
			if err != nil {
				return res, err
			}
			if bal > math.MaxUint64-amt {
				continue
			}
			if err := st.Set(id, bal+amt, nonce); err != nil {
				return res, err
			}
			continue
		}
		tx, ok := DecodeTx(e.Payload)
		if !ok {
			continue
		}
		env.charge(sigGas(tx.Scheme))
		if !tx.VerifySignature(env.ChainID, env.RollupID) {
			continue
		}
		from := tx.From()
		bal, nonce, err := st.Get(from)
		if err != nil {
			return res, err
		}
		if tx.Nonce != nonce || tx.Amount == 0 || bal < tx.Amount || nonce == math.MaxUint64 {
			continue
		}
		switch tx.Kind {
		case KindTransfer:
			if tx.To == WithdrawalCounterID {
				continue
			}
			if tx.To == from {
				if err := st.Set(from, bal, nonce+1); err != nil {
					return res, err
				}
				continue
			}
			tb, tn, err := st.Get(tx.To)
			if err != nil {
				return res, err
			}
			if tb > math.MaxUint64-tx.Amount {
				continue
			}
			if err := st.Set(from, bal-tx.Amount, nonce+1); err != nil {
				return res, err
			}
			if err := st.Set(tx.To, tb+tx.Amount, tn); err != nil {
				return res, err
			}
		case KindWithdraw:
			if tx.Amount > math.MaxInt64 || !canonicalAddress(tx.Recipient) {
				continue
			}
			next, _, err := st.Get(WithdrawalCounterID)
			if err != nil {
				return res, err
			}
			if next == math.MaxUint64 {
				continue
			}
			if err := st.Set(from, bal-tx.Amount, nonce+1); err != nil {
				return res, err
			}
			if err := st.Set(WithdrawalCounterID, next+1, 0); err != nil {
				return res, err
			}
			res.Withdrawals = append(res.Withdrawals, Withdrawal{Index: next, Recipient: tx.Recipient, Amount: tx.Amount})
		}
	}
	res.StateRoot = st.Root()
	res.WithdrawalsRoot = WithdrawalsRoot(env.ChainID, env.RollupID, res.Withdrawals)
	return res, nil
}

// canonicalAddress reports whether s is an account address in the chain's
// canonical bech32 form, which is what ExecuteWithdrawal recomputes the leaf
// from.
func canonicalAddress(s string) bool {
	a, err := sdk.AccAddressFromBech32(s)
	return err == nil && a.String() == s
}

// WithdrawalLeaf is the chain-bound withdrawal leaf (the rdk keeper's
// WithdrawalLeafV2, SHAKE-256): every field length-prefixed.
func WithdrawalLeaf(chainID, rollupID string, index uint64, recipient string, amount uint64) [32]byte {
	h := qorehash.New()
	lp := func(b []byte) {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(b)))
		h.Write(n[:])
		h.Write(b)
	}
	lp([]byte("qorechain-rdk-withdrawal/v2"))
	lp([]byte(chainID))
	lp([]byte(rollupID))
	var u8 [8]byte
	binary.BigEndian.PutUint64(u8[:], index)
	h.Write(u8[:])
	lp([]byte(recipient))
	lp([]byte(WithdrawalDenom))
	binary.BigEndian.PutUint64(u8[:], amount)
	h.Write(u8[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func pairHash(l, r [32]byte) [32]byte {
	h := qorehash.New()
	h.Write(l[:])
	h.Write(r[:])
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func withdrawalLevels(chainID, rollupID string, ws []Withdrawal) [][][32]byte {
	if len(ws) == 0 {
		return nil
	}
	n := 1
	for n < len(ws) {
		n *= 2
	}
	// A batch's withdrawals have consecutive global indices, so their
	// positions index mod n are distinct; ExecuteWithdrawal reads the
	// position from the index's low bits (VerifyWithdrawalMerkle).
	level := make([][32]byte, n)
	for _, w := range ws {
		level[w.Index%uint64(n)] = WithdrawalLeaf(chainID, rollupID, w.Index, w.Recipient, w.Amount)
	}
	levels := [][][32]byte{level}
	for len(level) > 1 {
		next := make([][32]byte, len(level)/2)
		for i := range next {
			next[i] = pairHash(level[2*i], level[2*i+1])
		}
		levels = append(levels, next)
		level = next
	}
	return levels
}

// WithdrawalsRoot is the Merkle root ExecuteWithdrawal checks withdrawals
// against: n leaves in a tree of the next power of two, each at its index
// modulo that size, the rest zero; empty when there is no withdrawal.
func WithdrawalsRoot(chainID, rollupID string, ws []Withdrawal) []byte {
	levels := withdrawalLevels(chainID, rollupID, ws)
	if levels == nil {
		return nil
	}
	root := levels[len(levels)-1][0]
	return root[:]
}

// WithdrawalProof returns the sibling path of withdrawal i, as
// MsgExecuteWithdrawal takes it.
func WithdrawalProof(chainID, rollupID string, ws []Withdrawal, i int) ([][]byte, error) {
	if i < 0 || i >= len(ws) {
		return nil, errors.New("qledger: no such withdrawal")
	}
	levels := withdrawalLevels(chainID, rollupID, ws)
	var path [][]byte
	idx := int(ws[i].Index % uint64(len(levels[0])))
	for _, level := range levels[:len(levels)-1] {
		s := level[idx^1]
		path = append(path, append([]byte(nil), s[:]...))
		idx /= 2
	}
	return path, nil
}
