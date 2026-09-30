package app

import (
	"errors"
	"runtime"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// The block-execution leg of the multisig bound (QSR-2026-0035, QSR-2026-0164):
// FinalizeBlock and InitChain enforce it only while FeatureMultisigSignatureBound
// is on, so blocks below the v3.2.2 plan height replay exactly as they ran.

func featureSwitch(on bool) func(sdk.Context) bool {
	return func(sdk.Context) bool { return on }
}

// fakeFeatureKeeper records which switch the guard asked about.
type fakeFeatureKeeper struct {
	enabled map[string]bool
	asked   []string
}

func (k *fakeFeatureKeeper) ProtocolFeatureEnabled(_ sdk.Context, name string) bool {
	k.asked = append(k.asked, name)
	return k.enabled[name]
}

func reached(hit *int) sdk.AnteHandler {
	return func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		*hit++
		return ctx, nil
	}
}

var blockExecutionContexts = map[string]sdk.Context{
	"FinalizeBlock":      sdk.Context{}.WithExecMode(sdk.ExecModeFinalize).WithBlockHeight(10),
	"InitChain delivery": sdk.Context{},
}

// Feature off: block execution passes every tree straight through, including the
// hostile ones, without looking at it. This is the replay guarantee.
func TestBlockBoundOffPassesBlockExecutionThrough(t *testing.T) {
	for _, gate := range []struct {
		name string
		fn   func(sdk.Context) bool
	}{{"nil switch", nil}, {"switch off", featureSwitch(false)}} {
		for mode, ctx := range blockExecutionContexts {
			for _, data := range []signing.SignatureData{chain(3), chain(100), nOf(33, single), single()} {
				hit := 0
				tx := buildSignedTx(t, data)
				_, err := NewMultisigDepthGuardDecorator(gate.fn).AnteHandle(ctx, tx, false, reached(&hit))
				if err != nil || hit != 1 {
					t.Fatalf("%s / %s: err=%v next=%d, want passthrough", gate.name, mode, err, hit)
				}
			}
		}
	}
}

// Feature on: block execution applies exactly the node-local bound. Depth 2 and
// breadth 32 pass; depth 3, breadth 33 and a 100-deep chain are refused with
// ErrTooManySignatures before anything downstream runs.
func TestBlockBoundOnEnforcesInBlockExecution(t *testing.T) {
	cases := []struct {
		name  string
		data  signing.SignatureData
		admit bool
	}{
		{"single key", single(), true},
		{"flat 2-of-3", nOf(3, single), true},
		{"flat 32", nOf(32, single), true},
		{"two-level multisig", nOf(3, func() signing.SignatureData { return nOf(3, single) }), true},
		{"breadth 32 at both levels", nOf(32, func() signing.SignatureData { return nOf(32, single) }), true},
		{"chain depth 2", chain(2), true},
		{"chain depth 3", chain(3), false},
		{"breadth 33", nOf(33, single), false},
		{"chain depth 100", chain(100), false},
	}
	for mode, ctx := range blockExecutionContexts {
		for _, tc := range cases {
			hit := 0
			tx := buildSignedTx(t, tc.data)
			_, err := NewMultisigDepthGuardDecorator(featureSwitch(true)).AnteHandle(ctx, tx, false, reached(&hit))
			if tc.admit {
				if err != nil || hit != 1 {
					t.Errorf("%s / %s: err=%v next=%d, want admitted", mode, tc.name, err, hit)
				}
				continue
			}
			if !errors.Is(err, errortypes.ErrTooManySignatures) || hit != 0 {
				t.Errorf("%s / %s: err=%v next=%d, want ErrTooManySignatures and no next", mode, tc.name, err, hit)
			}
		}
	}
}

// The node-local modes enforce whatever the switch says, and never need to ask.
func TestNodeLocalModesIgnoreTheSwitch(t *testing.T) {
	k := &fakeFeatureKeeper{}
	guard := NewMultisigDepthGuardDecorator(multisigBoundFeature(k))
	for name, ctx := range map[string]sdk.Context{
		"CheckTx":         sdk.Context{}.WithIsCheckTx(true),
		"PrepareProposal": sdk.Context{}.WithExecMode(sdk.ExecModePrepareProposal),
		"ProcessProposal": sdk.Context{}.WithExecMode(sdk.ExecModeProcessProposal),
	} {
		_, err := guard.AnteHandle(ctx, buildSignedTx(t, chain(3)), false, failIfReached(t))
		if !errors.Is(err, errortypes.ErrTooManySignatures) {
			t.Fatalf("%s: got %v", name, err)
		}
	}
	if len(k.asked) != 0 {
		t.Fatalf("node-local modes consulted the switch: %v", k.asked)
	}
}

// The switch the ante chain wires is FeatureMultisigSignatureBound, read from the
// PQC keeper, and it is consulted only in block execution.
func TestMultisigBoundFeatureReadsTheV322Switch(t *testing.T) {
	if multisigBoundFeature(nil) != nil {
		t.Fatal("a nil keeper must mean no block-execution check")
	}
	finalize := blockExecutionContexts["FinalizeBlock"]

	off := &fakeFeatureKeeper{}
	hit := 0
	if _, err := NewMultisigDepthGuardDecorator(multisigBoundFeature(off)).AnteHandle(
		finalize, buildSignedTx(t, chain(3)), false, reached(&hit)); err != nil || hit != 1 {
		t.Fatalf("switch off: err=%v next=%d", err, hit)
	}
	if len(off.asked) != 1 || off.asked[0] != pqctypes.FeatureMultisigSignatureBound {
		t.Fatalf("asked %v, want [%s]", off.asked, pqctypes.FeatureMultisigSignatureBound)
	}

	on := &fakeFeatureKeeper{enabled: map[string]bool{pqctypes.FeatureMultisigSignatureBound: true}}
	if _, err := NewMultisigDepthGuardDecorator(multisigBoundFeature(on)).AnteHandle(
		finalize, buildSignedTx(t, chain(3)), false, failIfReached(t)); !errors.Is(err, errortypes.ErrTooManySignatures) {
		t.Fatalf("switch on: got %v", err)
	}

	// Some other feature being on changes nothing here.
	other := &fakeFeatureKeeper{enabled: map[string]bool{pqctypes.FeatureAMMSwapIntegrity: true}}
	hit = 0
	if _, err := NewMultisigDepthGuardDecorator(multisigBoundFeature(other)).AnteHandle(
		finalize, buildSignedTx(t, chain(3)), false, reached(&hit)); err != nil || hit != 1 {
		t.Fatalf("unrelated feature: err=%v next=%d", err, hit)
	}
}

func TestMultisigBoundIsAV322Feature(t *testing.T) {
	for _, f := range pqctypes.ReleaseV3_2_2ProtocolFeatures {
		if f == pqctypes.FeatureMultisigSignatureBound {
			return
		}
	}
	t.Fatal("FeatureMultisigSignatureBound is not enabled by the v3.2.2 release")
}

// End to end through baseapp, with the switch flipping at a height the way the
// release handler flips it in PreBlock of the plan height. Below it, a block
// carrying a depth-3 multisig executes exactly as before (the stand-in for
// SetPubKey runs and the tx succeeds). From it, the same tx fails in DeliverTx
// with ErrTooManySignatures, reaching nothing, and a deep hostile tree is refused
// cheaply. A legitimate two-level multisig executes on both sides.
func TestFinalizeBlockEnforcesFromThePlanHeight(t *testing.T) {
	const planHeight = 3
	s := &sentinel{}
	gate := func(ctx sdk.Context) bool { return ctx.BlockHeight() >= planHeight }
	bapp := newGuardedBaseApp(t, sdk.ChainAnteDecorators(NewMultisigDepthGuardDecorator(gate), s))

	depth3 := buildSignedTxBytes(t, chain(3))
	legit := buildSignedTxBytes(t, nOf(3, func() signing.SignatureData { return nOf(3, single) }))

	finalize := func(h int64, txs ...[]byte) []*abci.ExecTxResult {
		t.Helper()
		res, err := bapp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: h, Txs: txs})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := bapp.Commit(); err != nil {
			t.Fatal(err)
		}
		return res.TxResults
	}

	// Below the plan height: unchanged.
	for h := int64(1); h < planHeight; h++ {
		before := s.hits.Load()
		rs := finalize(h, depth3, legit)
		for i, r := range rs {
			if r.Code != 0 {
				t.Fatalf("height %d tx %d: code %d (%s), want the old success", h, i, r.Code, r.Log)
			}
		}
		if got := s.hits.Load() - before; got != 2 {
			t.Fatalf("height %d: SetPubKey stand-in reached %d times, want 2", h, got)
		}
	}

	// From the plan height: refused in block execution.
	deep := buildSignedTxBytes(t, chain(100))
	var m0, m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m0)
	before := s.hits.Load()
	rs := finalize(planHeight, depth3, deep, legit)
	runtime.ReadMemStats(&m1)

	for i, want := range []uint32{errortypes.ErrTooManySignatures.ABCICode(), errortypes.ErrTooManySignatures.ABCICode(), 0} {
		if rs[i].Code != want {
			t.Fatalf("plan height tx %d: code %d (%s), want %d", i, rs[i].Code, rs[i].Log, want)
		}
	}
	for i := 0; i < 2; i++ {
		if len(rs[i].Events) != 0 {
			t.Fatalf("refused tx %d emitted events: %v", i, rs[i].Events)
		}
	}
	if got := s.hits.Load() - before; got != 1 {
		t.Fatalf("plan height: SetPubKey stand-in reached %d times, want 1 (the legitimate tx only)", got)
	}
	if delta := m1.TotalAlloc - m0.TotalAlloc; delta > 1<<20 {
		t.Fatalf("executing the block allocated %d bytes", delta)
	}
}
