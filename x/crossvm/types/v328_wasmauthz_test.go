package types_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

func addr32(seed string) string {
	b := make([]byte, 32)
	copy(b, seed)
	return sdk.AccAddress(b).String()
}

var gov = sdk.AccAddress([]byte("governance-module-ac")).String()

func lookupOf(m map[string]types.BridgeContractState) types.BridgeContractLookup {
	return func(c string) types.BridgeContractState { return m[c] }
}

// Params that never set the new fields are stored exactly as before.
func TestParamsV328FieldsKeepStoredBytes(t *testing.T) {
	p := types.DefaultParams()
	p.BridgeAllowedTargets = []string{"cosmwasm:" + addr32("a")}
	bz, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bz), "bridge_target_pins") || strings.Contains(string(bz), "max_pending_per_sender") {
		t.Fatalf("unset v3.2.8 fields appear in the stored params: %s", bz)
	}
}

func TestBridgePinsValidate(t *testing.T) {
	a, b := "cosmwasm:"+addr32("a"), "cosmwasm:"+addr32("b")
	base := types.DefaultParams()
	base.BridgeAllowedTargets = []string{a, b}
	for name, pins := range map[string][]types.BridgeTargetPin{
		"unlisted target": {{Target: "cosmwasm:" + addr32("c"), CodeID: 1, HistoryLen: 1}},
		"pinned twice":    {{Target: a, CodeID: 1, HistoryLen: 1}, {Target: a, CodeID: 2, HistoryLen: 1}},
		"zero code":       {{Target: a, CodeID: 0, HistoryLen: 1}},
		"no history":      {{Target: a, CodeID: 1}},
	} {
		p := base
		p.BridgeTargetPins = pins
		if err := p.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	p := base
	p.BridgeTargetPins = []types.BridgeTargetPin{{Target: a, CodeID: 5, HistoryLen: 1}}
	if err := p.ValidateUpdate(); err != nil {
		t.Fatalf("valid pin refused: %v", err)
	}
	p.MaxPendingPerSender = p.MaxQueueSize + 1
	if err := p.ValidateUpdate(); err == nil {
		t.Fatal("a per-sender bound above the queue size was accepted")
	}
}

func TestValidateBridgePins(t *testing.T) {
	eoa, none, govd, ghost := addr32("eoa"), addr32("none"), addr32("gov"), addr32("ghost")
	look := lookupOf(map[string]types.BridgeContractState{
		eoa:  {Found: true, CodeID: 5, Admin: addr32("someone"), HistoryLen: 3},
		none: {Found: true, CodeID: 6, HistoryLen: 1},
		govd: {Found: true, CodeID: 7, Admin: gov, HistoryLen: 1},
	})
	p := types.DefaultParams()
	p.BridgeAllowedTargets = []string{"cosmwasm:" + none, "cosmwasm:" + govd}
	if err := p.ValidateBridgePins(look, gov); err != nil {
		t.Fatalf("governed targets refused: %v", err)
	}
	p.BridgeAllowedTargets = append(p.BridgeAllowedTargets, "cosmwasm:"+eoa)
	if err := p.ValidateBridgePins(look, gov); err == nil || !strings.Contains(err.Error(), "admin") {
		t.Fatalf("an unpinned account-administered target: %v", err)
	}
	for name, pin := range map[string]types.BridgeTargetPin{
		"other code":    {Target: "cosmwasm:" + eoa, CodeID: 4, HistoryLen: 3},
		"other history": {Target: "cosmwasm:" + eoa, CodeID: 5, HistoryLen: 1},
	} {
		p.BridgeTargetPins = []types.BridgeTargetPin{pin}
		if err := p.ValidateBridgePins(look, gov); err == nil {
			t.Fatalf("%s: a pin that does not match the contract was accepted", name)
		}
	}
	p.BridgeTargetPins = []types.BridgeTargetPin{{Target: "cosmwasm:" + eoa, CodeID: 5, HistoryLen: 3}}
	if err := p.ValidateBridgePins(look, gov); err != nil {
		t.Fatalf("a correctly pinned target: %v", err)
	}
	p.BridgeAllowedTargets = append(p.BridgeAllowedTargets, "cosmwasm:"+ghost)
	if err := p.ValidateBridgePins(look, gov); err == nil {
		t.Fatal("a target naming no contract was accepted")
	}
}

func TestCheckBridgeTargetCode(t *testing.T) {
	c := addr32("c")
	entry := "cosmwasm:" + c
	p := types.DefaultParams()
	p.BridgeAllowedTargets = []string{entry}
	p.BridgeTargetPins = []types.BridgeTargetPin{{Target: entry, CodeID: 5, HistoryLen: 1}}
	x := addr32("x")
	for name, tc := range map[string]struct {
		st types.BridgeContractState
		ok bool
	}{
		"unchanged":              {types.BridgeContractState{Found: true, CodeID: 5, Admin: x, HistoryLen: 1}, true},
		"migrated":               {types.BridgeContractState{Found: true, CodeID: 9, Admin: x, HistoryLen: 2}, false},
		"migrated away and back": {types.BridgeContractState{Found: true, CodeID: 5, Admin: x, HistoryLen: 3}, false},
		"gone":                   {types.BridgeContractState{}, false},
	} {
		err := p.CheckBridgeTargetCode(types.VMTypeCosmWasm, c, tc.st, gov)
		if (err == nil) != tc.ok {
			t.Errorf("pinned, %s: %v", name, err)
		}
	}
	p.BridgeTargetPins = nil
	for admin, ok := range map[string]bool{"": true, gov: true, x: false} {
		err := p.CheckBridgeTargetCode(types.VMTypeCosmWasm, c, types.BridgeContractState{Found: true, CodeID: 9, Admin: admin, HistoryLen: 4}, gov)
		if (err == nil) != ok {
			t.Fatalf("unpinned, admin %q: %v", admin, err)
		}
	}
	if err := p.CheckBridgeTargetCode(types.VMTypeCosmWasm, c, types.BridgeContractState{}, gov); err == nil {
		t.Fatal("unpinned, naming no contract: called")
	}
}

// The release step pins every existing entry to the code and history it has,
// keeps any pin already there, removes an entry naming no contract, and
// reports no change on an empty allowlist (nothing is written on mainnet).
func TestPinBridgeTargets(t *testing.T) {
	if _, changed, _ := types.DefaultParams().PinBridgeTargets(lookupOf(nil)); changed {
		t.Fatal("an empty allowlist changed")
	}
	a, b, c, ghost := addr32("a"), addr32("b"), addr32("c"), addr32("ghost")
	look := lookupOf(map[string]types.BridgeContractState{
		a: {Found: true, CodeID: 5, Admin: addr32("eoa"), HistoryLen: 2},
		b: {Found: true, CodeID: 6, Admin: addr32("eoa"), HistoryLen: 1},
		c: {Found: true, CodeID: 7, HistoryLen: 1},
	})
	p := types.DefaultParams()
	p.BridgeAllowedTargets = []string{"cosmwasm:" + a, "cosmwasm:" + ghost, "cosmwasm:" + b, "cosmwasm:" + c}
	p.BridgeTargetPins = []types.BridgeTargetPin{{Target: "cosmwasm:" + b, CodeID: 3, HistoryLen: 1}}
	got, changed, removed := p.PinBridgeTargets(look)
	if !changed {
		t.Fatal("nothing pinned")
	}
	if len(removed) != 1 || removed[0] != "cosmwasm:"+ghost {
		t.Fatalf("removed %v", removed)
	}
	if strings.Join(got.BridgeAllowedTargets, ",") != strings.Join([]string{"cosmwasm:" + a, "cosmwasm:" + b, "cosmwasm:" + c}, ",") {
		t.Fatalf("allowlist %v", got.BridgeAllowedTargets)
	}
	want := map[string]types.BridgeTargetPin{
		"cosmwasm:" + a: {Target: "cosmwasm:" + a, CodeID: 5, HistoryLen: 2},
		"cosmwasm:" + b: {Target: "cosmwasm:" + b, CodeID: 3, HistoryLen: 1},
		"cosmwasm:" + c: {Target: "cosmwasm:" + c, CodeID: 7, HistoryLen: 1},
	}
	if len(got.BridgeTargetPins) != len(want) {
		t.Fatalf("pins %+v", got.BridgeTargetPins)
	}
	for _, pin := range got.BridgeTargetPins {
		if want[pin.Target] != pin {
			t.Fatalf("pin %+v", pin)
		}
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("migrated params invalid: %v", err)
	}
	if len(p.BridgeTargetPins) != 1 || len(p.BridgeAllowedTargets) != 4 {
		t.Fatal("the input params were modified")
	}
	if _, again, _ := got.PinBridgeTargets(look); again {
		t.Fatal("pinning twice changed something")
	}
}

// Error text that becomes consensus data carries no source annotation, and a
// plain error still unwraps to its cause.
func TestPlainErrorText(t *testing.T) {
	cause := errorsmod.Wrap(types.ErrExecutionFailed, "callee refused")
	annotated := fmt.Errorf("outer: %w", cause)
	if !strings.Contains(annotated.Error(), ".go:") {
		t.Skipf("this errors version adds no annotation: %q", annotated.Error())
	}
	got := types.PlainErrorText(annotated)
	if strings.Contains(got, ".go:") || !strings.HasPrefix(got, "outer: callee refused") {
		t.Fatalf("plain text %q", got)
	}
	e := types.NewPlainError("cross-VM call failed", annotated)
	if strings.Contains(e.Error(), ".go:") || !errors.Is(e, types.ErrExecutionFailed) {
		t.Fatalf("plain error %q, unwraps: %v", e.Error(), errors.Is(e, types.ErrExecutionFailed))
	}
	plain := errors.New("no annotation [here]")
	if types.PlainErrorText(plain) != plain.Error() {
		t.Fatal("text without an annotation changed")
	}
}
