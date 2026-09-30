package app

import (
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
)

// upstreamFlattenSignaturesAtDepth is flattenSignaturesAtDepth from cosmos-sdk
// v0.53.8 x/auth/ante/sigverify.go, reduced to its verdict (the collected bytes
// are irrelevant to whether it errors). It is the reference the guard must agree
// with on every tree, so that the eventual SDK bump changes nothing.
func upstreamFlattenSignaturesAtDepth(data signing.SignatureData, depth, maxDepth, maxLength int) error {
	switch {
	case data == nil:
		return fmt.Errorf("SignatureData is required")
	case depth > maxDepth:
		return fmt.Errorf("max depth of %d reached", maxDepth)
	}

	if _, ok := data.(*signing.SingleSignatureData); ok {
		return nil
	}

	multi, ok := data.(*signing.MultiSignatureData)
	switch {
	case !ok:
		return fmt.Errorf("unexpected signature data type %T", data)
	case len(multi.Signatures) > maxLength:
		return fmt.Errorf("max breadth of %d reached", maxLength)
	}

	for _, sig := range multi.Signatures {
		if err := upstreamFlattenSignaturesAtDepth(sig, depth+1, maxDepth, maxLength); err != nil {
			return err
		}
	}
	return nil
}

func upstreamVerdict(data signing.SignatureData) error {
	return upstreamFlattenSignaturesAtDepth(data, 0, 2, 32)
}

// unknownSignatureData is a SignatureData the SDK does not know. Embedding the
// interface is the only way to satisfy its unexported marker method from here.
type unknownSignatureData struct{ signing.SignatureData }

func single() signing.SignatureData {
	return &signing.SingleSignatureData{SignMode: signing.SignMode_SIGN_MODE_DIRECT, Signature: []byte{1}}
}

func multiOf(members ...signing.SignatureData) *signing.MultiSignatureData {
	return &signing.MultiSignatureData{Signatures: members}
}

func nOf(n int, member func() signing.SignatureData) *signing.MultiSignatureData {
	members := make([]signing.SignatureData, n)
	for i := range members {
		members[i] = member()
	}
	return multiOf(members...)
}

// chain nests a single signature inside depth one-member multisigs.
func chain(depth int) signing.SignatureData {
	data := single()
	for i := 0; i < depth; i++ {
		data = multiOf(data)
	}
	return data
}

func TestSignatureBoundMatchesUpstreamOnTheEdges(t *testing.T) {
	cases := []struct {
		name  string
		data  signing.SignatureData
		admit bool
	}{
		{"single key", single(), true},
		{"flat 2-of-3", nOf(3, single), true},
		{"multisig of multisigs (two levels)", nOf(3, func() signing.SignatureData { return nOf(3, single) }), true},
		{"chain depth 1", chain(1), true},
		{"chain depth 2", chain(2), true},
		{"chain depth 3", chain(3), false},
		{"chain depth 4", chain(4), false},
		{"chain depth 5", chain(5), false},
		{"chain depth 100", chain(100), false},
		{"breadth 31", nOf(31, single), true},
		{"breadth 32", nOf(32, single), true},
		{"breadth 33", nOf(33, single), false},
		{"breadth 33 one level down", multiOf(nOf(33, single)), false},
		{"breadth 32 at both levels", nOf(32, func() signing.SignatureData { return nOf(32, single) }), true},
		// Upstream judges depth per node before type: an EMPTY multisig at the
		// last level is accepted because no member ever reaches the next level.
		{"empty multisig at the last level", multiOf(multiOf(multiOf())), true},
		{"empty multisig one level too deep", multiOf(multiOf(multiOf(multiOf()))), false},
		{"empty top-level multisig", multiOf(), true},
		{"nil signature", nil, false},
		{"nil member", multiOf(single(), nil), false},
		{"unknown type at the top", unknownSignatureData{}, false},
		{"unknown type as a member", multiOf(single(), unknownSignatureData{}), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkSignatureBound(tc.data, 0)
			want := upstreamVerdict(tc.data)
			if (got == nil) != (want == nil) {
				t.Fatalf("guard says %v, upstream says %v", got, want)
			}
			if (got == nil) != tc.admit {
				t.Fatalf("admit=%v, want %v (err %v)", got == nil, tc.admit, got)
			}
		})
	}
}

// randomSignatureData builds a tree that sometimes stays in bounds and sometimes
// crosses one of the limits, so the differential test sees both verdicts often.
func randomSignatureData(r *rand.Rand, depth int) signing.SignatureData {
	switch p := r.Intn(100); {
	case p < 2:
		return nil
	case p < 4:
		return unknownSignatureData{}
	case p < 40 || depth > 4:
		return single()
	}
	n := r.Intn(5)
	switch r.Intn(20) {
	case 0:
		n = 31 + r.Intn(4) // 31..34, straddling the breadth limit
	case 1:
		n = 0
	}
	members := make([]signing.SignatureData, n)
	for i := range members {
		members[i] = randomSignatureData(r, depth+1)
	}
	return multiOf(members...)
}

func TestSignatureBoundAgreesWithUpstreamOnRandomTrees(t *testing.T) {
	r := rand.New(rand.NewSource(20260930))
	var admitted, refused int
	for i := 0; i < 20000; i++ {
		data := randomSignatureData(r, 0)
		got := checkSignatureBound(data, 0)
		want := upstreamVerdict(data)
		if (got == nil) != (want == nil) {
			t.Fatalf("tree %d: guard says %v, upstream says %v", i, got, want)
		}
		if got == nil {
			admitted++
		} else {
			refused++
		}
	}
	if admitted < 1000 || refused < 1000 {
		t.Fatalf("differential run is lopsided (admitted %d, refused %d); the generator no longer exercises both verdicts",
			admitted, refused)
	}
}

// The walk stops at the first violation. A tree that is wide at every level and
// far too deep must be refused after a bounded number of visits, not expanded.
func TestSignatureBoundStopsAtTheFirstViolation(t *testing.T) {
	// 32 members per node, 6 levels: about a billion leaves if walked in full.
	var build func(level int) signing.SignatureData
	shared := map[int]signing.SignatureData{}
	build = func(level int) signing.SignatureData {
		if level == 0 {
			return single()
		}
		if d, ok := shared[level]; ok {
			return d
		}
		members := make([]signing.SignatureData, 32)
		child := build(level - 1)
		for i := range members {
			members[i] = child
		}
		shared[level] = multiOf(members...)
		return shared[level]
	}
	huge := build(6)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	err := checkSignatureBound(huge, 0)
	runtime.ReadMemStats(&after)
	if err == nil {
		t.Fatal("a six-level tree was admitted")
	}
	if delta := after.TotalAlloc - before.TotalAlloc; delta > 64<<10 {
		t.Fatalf("the bound check allocated %d bytes on a hostile tree", delta)
	}
}

// Every mode that never executes during replay enforces the bound; block
// execution and genesis delivery do not.
func TestNodeLocalPolicyModes(t *testing.T) {
	base := sdk.Context{}
	cases := []struct {
		name     string
		ctx      sdk.Context
		simulate bool
		want     bool
	}{
		{"CheckTx", base.WithIsCheckTx(true), false, true},
		{"ReCheckTx", base.WithIsCheckTx(true).WithIsReCheckTx(true), false, true},
		{"simulate (gRPC/REST, check state)", base.WithIsCheckTx(true).WithExecMode(sdk.ExecModeSimulate), true, true},
		{"simulate flag on a query context", base, true, true},
		{"simulate mode without the flag", base.WithExecMode(sdk.ExecModeSimulate), false, true},
		{"PrepareProposal", base.WithExecMode(sdk.ExecModePrepareProposal), false, true},
		{"ProcessProposal", base.WithExecMode(sdk.ExecModeProcessProposal), false, true},
		{"FinalizeBlock", base.WithExecMode(sdk.ExecModeFinalize), false, false},
		{"InitChain gentx delivery (zero mode, not CheckTx)", base, false, false},
		{"vote extension", base.WithExecMode(sdk.ExecModeVoteExtension), false, false},
	}
	for _, tc := range cases {
		if got := enforcesNodeLocalPolicy(tc.ctx, tc.simulate); got != tc.want {
			t.Errorf("%s: enforces=%v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMultisigGuardErrorIsTooManySignatures(t *testing.T) {
	tx := buildSignedTx(t, chain(3))
	_, err := NewMultisigDepthGuardDecorator(nil).AnteHandle(
		sdk.Context{}.WithIsCheckTx(true), tx, false, failIfReached(t))
	if !errors.Is(err, errortypes.ErrTooManySignatures) {
		t.Fatalf("got %v, want ErrTooManySignatures (the code v0.53.8 returns)", err)
	}
}
