package app

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/btree"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// DefaultCosmosMempoolMaxTxs caps the app-side Cosmos transaction pool. It
// matches the default mempool.size in config.toml.
const DefaultCosmosMempoolMaxTxs = 5000

// cosmosTxPriority is the priority the cosmos/evm mempool gives a Cosmos tx by
// default (fee in the EVM denom per unit of gas wanted). It is spelled out here
// so the evicting wrapper ranks entries exactly as the pool does.
func cosmosTxPriority(evmDenom func(ctx sdk.Context) string) sdkmempool.TxPriority[math.Int] {
	return sdkmempool.TxPriority[math.Int]{
		GetTxPriority: func(goCtx context.Context, tx sdk.Tx) math.Int {
			ctx := sdk.UnwrapSDKContext(goCtx)
			feeTx, ok := tx.(sdk.FeeTx)
			if !ok || feeTx.GetGas() == 0 {
				return math.ZeroInt()
			}
			found, coin := feeTx.GetFee().Find(evmDenom(ctx))
			if !found {
				return math.ZeroInt()
			}
			return coin.Amount.Quo(math.NewIntFromUint64(feeTx.GetGas()))
		},
		Compare: func(a, b math.Int) int {
			return a.BigInt().Cmp(b.BigInt())
		},
		MinValue: math.ZeroInt(),
	}
}

// isEVMTx reports whether the unified mempool routes tx to its EVM pool, using
// the same rule as ExperimentalEVMMempool: exactly one message, a
// MsgEthereumTx.
func isEVMTx(tx sdk.Tx) bool {
	msgs := tx.GetMsgs()
	if len(msgs) != 1 {
		return false
	}
	_, ok := msgs[0].(*evmtypes.MsgEthereumTx)
	return ok
}

type poolKey struct {
	sender string
	nonce  uint64
}

type poolEntry struct {
	tx       sdk.Tx
	priority math.Int
}

// senderEntries is one sender's indexed txs and the tail item (its highest
// nonce) currently in the tails tree.
type senderEntries struct {
	nonces map[uint64]poolEntry
	tail   tailItem
}

// tailItem orders senders' tail entries for eviction: lowest priority first,
// then the higher nonce, then the lexically larger sender, so the choice does
// not depend on insertion or map order.
type tailItem struct {
	priority math.Int
	nonce    uint64
	sender   string
}

func tailLess(a, b tailItem) bool {
	if !a.priority.Equal(b.priority) {
		return a.priority.LT(b.priority)
	}
	if a.nonce != b.nonce {
		return a.nonce > b.nonce
	}
	return a.sender > b.sender
}

// evictingMempool wraps the unified EVM+Cosmos mempool so that a full Cosmos
// pool does not lock out fee-paying users (QSR-2026-0286).
//
// The Cosmos side is a PriorityNonceMempool bounded by MaxTx. When it is full
// it refuses every new tx, so a spammer who fills it at the floor price would
// delay everyone else until the pool drains. When an insert is refused for
// capacity, this wrapper evicts the lowest-priority tail entry (the highest
// nonce of some other sender, so no sender is left with a gap) provided the
// newcomer pays strictly more per gas, then retries. EVM transactions pass
// straight through; the EVM pool has its own limits.
//
// The wrapper keeps a small index of the Cosmos entries it has seen go in. All
// Cosmos inserts and removals reach the pool through baseapp's mempool, which
// is this wrapper, so the index follows the pool. If it ever drifts, eviction
// of a stale entry reports not-found, the entry is dropped and the next
// candidate is tried.
type evictingMempool struct {
	sdkmempool.ExtMempool

	priority sdkmempool.TxPriority[math.Int]
	signers  sdkmempool.SignerExtractionAdapter
	logger   log.Logger

	mtx     sync.Mutex
	entries map[string]*senderEntries
	tails   *btree.BTreeG[tailItem]
}

var _ sdkmempool.ExtMempool = (*evictingMempool)(nil)

func newEvictingMempool(inner sdkmempool.ExtMempool, priority sdkmempool.TxPriority[math.Int], logger log.Logger) *evictingMempool {
	return &evictingMempool{
		ExtMempool: inner,
		priority:   priority,
		signers:    sdkmempool.NewDefaultSignerExtractionAdapter(),
		logger:     logger,
		entries:    make(map[string]*senderEntries),
		tails:      btree.NewG(8, tailLess),
	}
}

// key mirrors how PriorityNonceMempool indexes a tx: first signer, and the
// nonce ChooseNonce picks for it.
func (m *evictingMempool) key(tx sdk.Tx) (poolKey, error) {
	sigs, err := m.signers.GetSigners(tx)
	if err != nil {
		return poolKey{}, err
	}
	if len(sigs) == 0 {
		return poolKey{}, fmt.Errorf("tx must have at least one signer")
	}
	nonce, err := sdkmempool.ChooseNonce(sigs[0].Sequence, tx)
	if err != nil {
		return poolKey{}, err
	}
	return poolKey{sender: sigs[0].Signer.String(), nonce: nonce}, nil
}

// maxEvictAttempts bounds how many stale index entries one insert may skip
// over before giving up.
const maxEvictAttempts = 4

func (m *evictingMempool) Insert(ctx context.Context, tx sdk.Tx) error {
	if isEVMTx(tx) {
		return m.ExtMempool.Insert(ctx, tx)
	}

	m.mtx.Lock()
	defer m.mtx.Unlock()

	err := m.ExtMempool.Insert(ctx, tx)
	if err == nil {
		m.record(ctx, tx)
		return nil
	}
	if !errors.Is(err, sdkmempool.ErrMempoolTxMaxCapacity) {
		return err
	}

	k, kerr := m.key(tx)
	if kerr != nil {
		return err
	}
	p := m.priority.GetTxPriority(ctx, tx)

	for attempt := 0; attempt < maxEvictAttempts; attempt++ {
		victimKey, victim, ok := m.victim(k.sender, p)
		if !ok {
			return err
		}
		rerr := m.ExtMempool.Remove(victim.tx)
		m.forget(victimKey)
		if rerr != nil {
			if errors.Is(rerr, sdkmempool.ErrTxNotFound) {
				continue // stale index entry; try the next one
			}
			return err
		}

		ierr := m.ExtMempool.Insert(ctx, tx)
		if ierr != nil {
			// The victim is gone either way; report why the newcomer failed.
			return ierr
		}
		m.record(ctx, tx)
		m.logger.Debug("cosmos mempool full, evicted lowest-priority tx",
			"evicted_sender", victimKey.sender,
			"evicted_nonce", victimKey.nonce,
			"evicted_priority", victim.priority.String(),
			"new_priority", p.String(),
		)
		return nil
	}
	return err
}

func (m *evictingMempool) Remove(tx sdk.Tx) error {
	err := m.ExtMempool.Remove(tx)
	if isEVMTx(tx) {
		return err
	}
	if err == nil || errors.Is(err, sdkmempool.ErrTxNotFound) {
		if k, kerr := m.key(tx); kerr == nil {
			m.mtx.Lock()
			m.forget(k)
			m.mtx.Unlock()
		}
	}
	return err
}

func (m *evictingMempool) record(ctx context.Context, tx sdk.Tx) {
	k, err := m.key(tx)
	if err != nil {
		return
	}
	se, ok := m.entries[k.sender]
	if !ok {
		se = &senderEntries{nonces: make(map[uint64]poolEntry)}
		m.entries[k.sender] = se
	}
	se.nonces[k.nonce] = poolEntry{tx: tx, priority: m.priority.GetTxPriority(ctx, tx)}
	m.retail(k.sender, se, ok)
}

func (m *evictingMempool) forget(k poolKey) {
	se, ok := m.entries[k.sender]
	if !ok {
		return
	}
	if _, ok := se.nonces[k.nonce]; !ok {
		return
	}
	delete(se.nonces, k.nonce)
	if len(se.nonces) == 0 {
		m.tails.Delete(se.tail)
		delete(m.entries, k.sender)
		return
	}
	m.retail(k.sender, se, true)
}

// retail puts the sender's current highest nonce in the tails tree in place
// of the previous one.
func (m *evictingMempool) retail(sender string, se *senderEntries, hadTail bool) {
	if hadTail {
		m.tails.Delete(se.tail)
	}
	first := true
	var top uint64
	for n := range se.nonces {
		if first || n > top {
			top, first = n, false
		}
	}
	se.tail = tailItem{priority: se.nonces[top].priority, nonce: top, sender: sender}
	m.tails.ReplaceOrInsert(se.tail)
}

// size is the number of indexed entries.
func (m *evictingMempool) size() int {
	n := 0
	for _, se := range m.entries {
		n += len(se.nonces)
	}
	return n
}

// victim picks the lowest-priority tail entry of another sender whose priority
// is strictly below p. The tails tree is ordered for exactly this, so the walk
// stops at the first candidate (or skips only the newcomer's own sender).
func (m *evictingMempool) victim(newSender string, p math.Int) (poolKey, poolEntry, bool) {
	var (
		best  poolKey
		entry poolEntry
		found bool
	)
	m.tails.Ascend(func(it tailItem) bool {
		if !it.priority.LT(p) {
			return false
		}
		if it.sender == newSender {
			return true
		}
		best = poolKey{sender: it.sender, nonce: it.nonce}
		entry = m.entries[it.sender].nonces[it.nonce]
		found = true
		return false
	})
	return best, entry, found
}
