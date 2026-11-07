package app

import (
	"errors"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
)

// feeMemTx is a memTx that carries a real fee, so the production priority
// (cosmosTxPriority) can rank it.
type feeMemTx struct {
	*memTx
	fee sdk.Coins
}

func (t feeMemTx) GetFee() sdk.Coins  { return t.fee }
func (t feeMemTx) FeePayer() []byte   { return t.signer }
func (t feeMemTx) FeeGranter() []byte { return nil }
func (t feeMemTx) Unwrap() *memTx     { return t.memTx }
func uqorFee(amount int64) sdk.Coins  { return sdk.NewCoins(sdk.NewInt64Coin("uqor", amount)) }
func uqorDenom(sdk.Context) string    { return "uqor" }
func priceTx(id int, fee int64, gas uint64) feeMemTx {
	return feeMemTx{memTx: &memTx{id: id, signer: addr(id), gas: gas}, fee: uqorFee(fee)}
}

// The pool ranked on whole uqor per gas, so every price below 1 uqor/gas
// (the live minimum is 0.1) ranked 0 and a better-paying tx could never
// evict a cheaper one. It now ranks on the 18-decimal fee per gas.
func TestCosmosTxPriorityBelowOneUqorPerGas(t *testing.T) {
	p := cosmosTxPriority(uqorDenom)
	ctx := proposalCtx(-1)
	rank := func(fee int64, gas uint64) math.Int { return p.GetTxPriority(ctx, priceTx(0, fee, gas)) }

	tenth := rank(20_000, 200_000) // 0.1 uqor/gas
	fifth := rank(40_000, 200_000) // 0.2 uqor/gas
	if !tenth.IsPositive() {
		t.Fatalf("0.1 uqor/gas ranks %s, want > 0", tenth)
	}
	if p.Compare(fifth, tenth) <= 0 {
		t.Fatalf("0.2 uqor/gas (%s) does not outrank 0.1 uqor/gas (%s)", fifth, tenth)
	}
	// 1 aqor/gas is the finest step: 1 uqor over 10^12 gas is not, but
	// the same price keeps ranking equal whatever the gas.
	if got := rank(1, 1_000_000); !got.Equal(math.NewInt(1_000_000)) {
		t.Fatalf("1 uqor over 1M gas ranks %s, want 1000000 (aqor per gas)", got)
	}
	if !rank(10, 100).Equal(rank(1_000, 10_000)) {
		t.Fatal("the same price per gas ranks differently")
	}
	// Unchanged edges: no gas, no fee in the EVM denom.
	if !rank(5, 0).IsZero() {
		t.Fatal("a tx without gas ranks above zero")
	}
	other := feeMemTx{memTx: &memTx{gas: 10}, fee: sdk.NewCoins(sdk.NewInt64Coin("other", 1_000))}
	if !p.GetTxPriority(ctx, other).IsZero() {
		t.Fatal("a fee in another denom ranks above zero")
	}
}

// The reporter's scenario: a pool full of 0.1 uqor/gas txs refused a 0.2
// uqor/gas one (ErrMempoolTxMaxCapacity); now the cheapest is evicted.
func TestEvictingMempoolEvictsBelowOneUqorPerGas(t *testing.T) {
	p := cosmosTxPriority(uqorDenom)
	inner := sdkmempool.NewPriorityMempool(sdkmempool.PriorityNonceMempoolConfig[math.Int]{
		TxPriority:      p,
		SignerExtractor: feeMemTxSigners{},
		MaxTx:           3,
	})
	m := newEvictingMempool(inner, p, log.NewNopLogger())
	m.signers = feeMemTxSigners{}
	ctx := proposalCtx(-1)
	for i := 0; i < 3; i++ {
		if err := m.Insert(ctx, priceTx(i, 20_000, 200_000)); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Insert(ctx, priceTx(9, 40_000, 200_000)); err != nil {
		if errors.Is(err, sdkmempool.ErrMempoolTxMaxCapacity) {
			t.Fatal("a 0.2 uqor/gas tx was refused by a pool of 0.1 uqor/gas txs")
		}
		t.Fatal(err)
	}
	if m.CountTx() != 3 {
		t.Fatalf("pool size %d, want 3", m.CountTx())
	}
	// Equal price still does not evict.
	if err := m.Insert(ctx, priceTx(10, 20_000, 200_000)); !errors.Is(err, sdkmempool.ErrMempoolTxMaxCapacity) {
		t.Fatalf("an equal-price tx: %v, want ErrMempoolTxMaxCapacity", err)
	}
}

type feeMemTxSigners struct{}

func (feeMemTxSigners) GetSigners(tx sdk.Tx) ([]sdkmempool.SignerData, error) {
	return memTxSigners{}.GetSigners(tx.(feeMemTx).memTx)
}
