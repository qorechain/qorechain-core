package app

import (
	"errors"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func newTestEvicting(maxTx int) (*evictingMempool, *sdkmempool.PriorityNonceMempool[math.Int]) {
	inner := newMemPool(maxTx)
	m := newEvictingMempool(inner, memTxPriority(), log.NewNopLogger())
	m.signers = memTxSigners{}
	return m, inner
}

func poolTxs(mp sdkmempool.Mempool) map[int]bool {
	out := map[int]bool{}
	sdkmempool.SelectBy(proposalCtx(-1), mp, nil, func(tx sdk.Tx) bool {
		out[tx.(*memTx).id] = true
		return true
	})
	return out
}

func TestEvictingMempoolRefusesWhenFullAndNotBetter(t *testing.T) {
	const n = 10
	m, _ := newTestEvicting(n)
	ctx := proposalCtx(-1)
	for i := 0; i < n; i++ {
		if err := m.Insert(ctx, &memTx{id: i, signer: addr(i), fee: 5}); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	// Same fee as everything in the pool: no eviction.
	err := m.Insert(ctx, &memTx{id: n, signer: addr(n), fee: 5})
	if !errors.Is(err, sdkmempool.ErrMempoolTxMaxCapacity) {
		t.Fatalf("expected ErrMempoolTxMaxCapacity, got %v", err)
	}
	if m.CountTx() != n {
		t.Fatalf("pool size %d, want %d", m.CountTx(), n)
	}
}

func TestEvictingMempoolHigherFeeDisplacesLowest(t *testing.T) {
	m, _ := newTestEvicting(3)
	ctx := proposalCtx(-1)
	for i, fee := range []int64{4, 1, 7} {
		if err := m.Insert(ctx, &memTx{id: i, signer: addr(i), fee: fee}); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Insert(ctx, &memTx{id: 9, signer: addr(9), fee: 2}); err != nil {
		t.Fatalf("a better-paying tx should get in: %v", err)
	}
	got := poolTxs(m)
	if m.CountTx() != 3 || got[1] || !got[9] || !got[0] || !got[2] {
		t.Fatalf("expected tx 1 (fee 1) evicted for tx 9, pool is %v", got)
	}
}

// The victim is always some sender's highest nonce, so eviction never leaves a
// gap in front of a pooled tx, and the newcomer's own sender is never chosen.
func TestEvictingMempoolEvictsTailNonce(t *testing.T) {
	m, _ := newTestEvicting(3)
	ctx := proposalCtx(-1)
	a, b := addr(1), addr(2)
	for _, tx := range []*memTx{
		{id: 0, signer: a, seq: 0, fee: 1},
		{id: 1, signer: a, seq: 1, fee: 3},
		{id: 2, signer: b, seq: 0, fee: 2},
	} {
		if err := m.Insert(ctx, tx); err != nil {
			t.Fatal(err)
		}
	}
	// a's tail (seq 1, fee 3) is not below 2.5-ish, b's only tx (fee 2) is.
	if err := m.Insert(ctx, &memTx{id: 3, signer: addr(3), fee: 3}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got := poolTxs(m)
	if got[2] || !got[0] || !got[1] || !got[3] {
		t.Fatalf("expected b's tx evicted and a's nonces kept, pool is %v", got)
	}

	// b comes back paying more than a's tail: a's seq 1 goes, seq 0 stays.
	if err := m.Insert(ctx, &memTx{id: 4, signer: b, seq: 0, fee: 4}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	got = poolTxs(m)
	if got[1] || !got[0] || !got[4] {
		t.Fatalf("expected a's seq 1 evicted before its seq 0, pool is %v", got)
	}

	// A sender never evicts its own txs to make room.
	err := m.Insert(ctx, &memTx{id: 5, signer: a, seq: 1, fee: 100})
	if err != nil {
		// a's seq 1 slot is free again only through eviction of someone else.
		t.Fatalf("insert: %v", err)
	}
	got = poolTxs(m)
	if !got[0] || !got[5] {
		t.Fatalf("a's own seq 0 must survive a's insert, pool is %v", got)
	}
}

func TestEvictingMempoolIndexFollowsRemovals(t *testing.T) {
	m, inner := newTestEvicting(2)
	ctx := proposalCtx(-1)
	tx0 := &memTx{id: 0, signer: addr(0), fee: 1}
	tx1 := &memTx{id: 1, signer: addr(1), fee: 2}
	for _, tx := range []*memTx{tx0, tx1} {
		if err := m.Insert(ctx, tx); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Remove(tx1); err != nil {
		t.Fatal(err)
	}
	if m.size() != 1 {
		t.Fatalf("index has %d entries after removal, want 1", m.size())
	}

	// Drift: tx0 leaves the pool without the wrapper seeing it. The pool
	// then has room, and the stale index entry is dropped when it is picked.
	if err := inner.Remove(tx0); err != nil {
		t.Fatal(err)
	}
	for i, fee := range []int64{3, 4} {
		if err := m.Insert(ctx, &memTx{id: 10 + i, signer: addr(10 + i), fee: fee}); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Insert(ctx, &memTx{id: 20, signer: addr(20), fee: 9}); err != nil {
		t.Fatalf("eviction should skip the stale entry: %v", err)
	}
	got := poolTxs(m)
	if m.CountTx() != 2 || !got[20] || !got[11] || got[10] {
		t.Fatalf("unexpected pool %v", got)
	}
}

// EVM transactions go straight to the inner pool and are not indexed.
func TestEvictingMempoolPassesEVMThrough(t *testing.T) {
	if !isEVMTx(&memTx{msgs: []sdk.Msg{&evmtypes.MsgEthereumTx{}}}) {
		t.Fatal("a single MsgEthereumTx is an EVM tx")
	}
	if isEVMTx(&memTx{msgs: []sdk.Msg{&evmtypes.MsgEthereumTx{}, &banktypes.MsgSend{}}}) {
		t.Fatal("two messages is not an EVM tx")
	}
	if isEVMTx(&memTx{msgs: []sdk.Msg{&banktypes.MsgSend{}}}) {
		t.Fatal("a bank send is not an EVM tx")
	}
}

func TestEvictingMempoolUnderLoadStaysBounded(t *testing.T) {
	const n = 500
	m, _ := newTestEvicting(n)
	ctx := proposalCtx(-1)
	start := time.Now()
	for i := 0; i < 5*n; i++ {
		_ = m.Insert(ctx, &memTx{id: i, signer: addr(i), fee: int64(i % 97)})
		if m.CountTx() > n {
			t.Fatalf("pool grew to %d over the cap %d", m.CountTx(), n)
		}
	}
	if m.size() != m.CountTx() {
		t.Fatalf("index has %d entries, pool %d", m.size(), m.CountTx())
	}
	// Nothing that paid more than the cheapest survivor was lost.
	got := poolTxs(m)
	minFee := int64(-1)
	for id := range got {
		if f := int64(id % 97); minFee < 0 || f < minFee {
			minFee = f
		}
	}
	for i := 0; i < 5*n; i++ {
		if int64(i%97) > minFee && !got[i] {
			t.Fatalf("tx %d (fee %d) is gone while a fee-%d tx remains", i, i%97, minFee)
		}
	}
	t.Logf("%d inserts in %v", 5*n, time.Since(start))
}
