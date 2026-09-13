package types_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

func window(expiry, maxTxs, usedTxs uint64, maxVal, usedVal int64) types.EVMAuthWindow {
	return types.EVMAuthWindow{
		ExpiryHeight: expiry,
		MaxTxs:       maxTxs,
		UsedTxs:      usedTxs,
		MaxValue:     sdkmath.NewInt(maxVal),
		UsedValue:    sdkmath.NewInt(usedVal),
	}
}

// Each of the three bounds has to stop the window on its own. If any one of them
// could be exhausted while the window still admitted transactions, the bound would
// be decorative and the holder would be protected by less than they were told.
func TestWindowStopsOnEachBoundIndependently(t *testing.T) {
	one := sdkmath.NewInt(1)

	if window(100, 10, 0, 1000, 0).IsLive(101, one) {
		t.Fatal("a window past its expiry height still admitted a transaction")
	}
	if window(100, 10, 10, 1000, 0).IsLive(50, one) {
		t.Fatal("a window with no transactions left still admitted one")
	}
	if window(100, 10, 0, 1000, 1000).IsLive(50, one) {
		t.Fatal("a window with no value left still admitted an outflow")
	}
	if !window(100, 10, 0, 1000, 0).IsLive(100, one) {
		t.Fatal("a live window refused a transaction it should admit")
	}
	// The expiry height itself is inside the window: a holder who authorises "until
	// block 100" means through block 100.
	if !window(100, 10, 9, 1000, 999).IsLive(100, one) {
		t.Fatal("the last admissible transaction of a window was refused")
	}
}

// The value bound is checked against the outflow being requested, not merely against
// what has already been spent. Checking only the latter would admit a transaction
// that overshoots the bound in one step, which is the whole thing the bound is for.
func TestValueBoundIsCheckedAgainstTheRequestedOutflow(t *testing.T) {
	w := window(100, 10, 0, 1000, 900)

	if w.IsLive(50, sdkmath.NewInt(101)) {
		t.Fatal("a single transaction was allowed to overshoot the value bound")
	}
	if !w.IsLive(50, sdkmath.NewInt(100)) {
		t.Fatal("a transaction that exactly fills the remaining value was refused")
	}
}

// A window whose amounts never got initialised must admit nothing. A nil Int here
// would otherwise panic on arithmetic, and a panic in the ante handler is a far
// worse outcome than a refusal.
func TestUninitialisedWindowAdmitsNothing(t *testing.T) {
	var w types.EVMAuthWindow
	w.ExpiryHeight = 1000
	w.MaxTxs = 10

	if w.IsLive(1, sdkmath.NewInt(1)) {
		t.Fatal("a window with uninitialised amounts admitted a transaction")
	}
}

// Rounding has to go UP. The EVM unit carries 18 decimals over a 6-decimal base, so
// a transfer can be worth less than one uqor. Rounding down would charge nothing for
// such a transfer, and a long series of them would drain an account through a window
// that never appeared to be consumed. Rounding up can only close a window early,
// which is the safe direction to be wrong in.
func TestSubUnitAmountsAreChargedNotFree(t *testing.T) {
	one := sdkmath.NewInt(1) // 1 wei, far below one uqor
	if got := types.WeiToUqorRoundUp(one); !got.Equal(sdkmath.OneInt()) {
		t.Fatalf("1 wei was charged %s uqor; a sub-unit transfer must never be free", got)
	}

	// 10^12 wei is exactly one uqor and must not be rounded to two.
	exact := sdkmath.NewInt(1_000_000_000_000)
	if got := types.WeiToUqorRoundUp(exact); !got.Equal(sdkmath.OneInt()) {
		t.Fatalf("an exact uqor was charged %s", got)
	}

	// One wei above an exact uqor rounds to two.
	over := exact.Add(sdkmath.OneInt())
	if got := types.WeiToUqorRoundUp(over); !got.Equal(sdkmath.NewInt(2)) {
		t.Fatalf("one wei over an exact uqor was charged %s, want 2", got)
	}

	if got := types.WeiToUqorRoundUp(sdkmath.ZeroInt()); !got.IsZero() {
		t.Fatalf("zero was charged %s", got)
	}
}

// Every bound must be stated by whoever opens the window. Defaulting a missing bound
// would mean the security parameter is chosen by whoever forgot it.
func TestOpeningRequiresEveryBound(t *testing.T) {
	ok := types.MsgOpenEVMWindow{Blocks: 100, MaxTxs: 5, MaxValue: sdkmath.NewInt(1000)}
	if err := ok.Validate(); err != nil {
		t.Fatalf("a fully specified window was refused: %v", err)
	}

	cases := map[string]types.MsgOpenEVMWindow{
		"no blocks": {Blocks: 0, MaxTxs: 5, MaxValue: sdkmath.NewInt(1000)},
		"no txs":    {Blocks: 100, MaxTxs: 0, MaxValue: sdkmath.NewInt(1000)},
		"no value":  {Blocks: 100, MaxTxs: 5, MaxValue: sdkmath.ZeroInt()},
		"negative":  {Blocks: 100, MaxTxs: 5, MaxValue: sdkmath.NewInt(-1)},
		"blocks too big": {Blocks: types.MaxEVMWindowBlocks + 1, MaxTxs: 5,
			MaxValue: sdkmath.NewInt(1000)},
		"txs too many": {Blocks: 100, MaxTxs: types.MaxEVMWindowTxs + 1,
			MaxValue: sdkmath.NewInt(1000)},
	}
	for name, m := range cases {
		if err := m.Validate(); err == nil {
			t.Fatalf("%s was accepted", name)
		}
	}
}

// An authorisation that never expires is not an authorisation, it is a permanent
// downgrade to classical security. The ceiling is what stops one being opened by
// accident or by habit.
func TestWindowLengthIsCapped(t *testing.T) {
	if types.MaxEVMWindowBlocks == 0 {
		t.Fatal("there is no ceiling on how long a window may stay open")
	}
	// About a day at five-second blocks. If this grows, someone should have to
	// justify it against the reason the window exists.
	if types.MaxEVMWindowBlocks > 17280 {
		t.Fatalf("the window ceiling is %d blocks, which is more than a day",
			types.MaxEVMWindowBlocks)
	}
}

// Remaining is what an operator sees when refused, so it must never report more than
// is actually left, and must not go negative when a bound is already exhausted.
func TestRemainingNeverOverstatesWhatIsLeft(t *testing.T) {
	w := window(100, 10, 10, 1000, 1000)
	blocks, txs, value := w.Remaining(200)

	if blocks != 0 || txs != 0 || !value.IsZero() {
		t.Fatalf("an exhausted, expired window reported %d blocks, %d txs, %s value",
			blocks, txs, value)
	}

	blocks, txs, value = window(100, 10, 3, 1000, 250).Remaining(60)
	if blocks != 40 || txs != 7 || !value.Equal(sdkmath.NewInt(750)) {
		t.Fatalf("got %d blocks, %d txs, %s value; want 40, 7, 750", blocks, txs, value)
	}
}
