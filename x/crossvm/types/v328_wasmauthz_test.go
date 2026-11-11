package types_test

import (
	"encoding/json"
	"strings"
	"testing"

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
		"unlisted target": {{Target: "cosmwasm:" + addr32("c"), CodeID: 1}},
		"pinned twice":    {{Target: a, CodeID: 1}, {Target: a, CodeID: 2}},
		"zero code":       {{Target: a, CodeID: 0}},
	} {
		p := base
		p.BridgeTargetPins = pins
		if err := p.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	p := base
	p.BridgeTargetPins = []types.BridgeTargetPin{{Target: a, CodeID: 5}}
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
		eoa:  {Found: true, CodeID: 5, Admin: addr32("someone")},
		none: {Found: true, CodeID: 6},
		govd: {Found: true, CodeID: 7, Admin: gov},
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
	p.BridgeTargetPins = []types.BridgeTargetPin{{Target: "cosmwasm:" + eoa, CodeID: 4}}
	if err := p.ValidateBridgePins(look, gov); err == nil {
		t.Fatal("a pin to code the contract does not run was accepted")
	}
	p.BridgeTargetPins = []types.BridgeTargetPin{{Target: "cosmwasm:" + eoa, CodeID: 5}}
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
	p.BridgeTargetPins = []types.BridgeTargetPin{{Target: entry, CodeID: 5}}
	if err := p.CheckBridgeTargetCode(types.VMTypeCosmWasm, c, types.BridgeContractState{Found: true, CodeID: 5, Admin: addr32("x")}, gov); err != nil {
		t.Fatalf("pinned and unchanged: %v", err)
	}
	if err := p.CheckBridgeTargetCode(types.VMTypeCosmWasm, c, types.BridgeContractState{Found: true, CodeID: 9, Admin: addr32("x")}, gov); err == nil {
		t.Fatal("pinned and migrated: called")
	}
	if err := p.CheckBridgeTargetCode(types.VMTypeCosmWasm, c, types.BridgeContractState{}, gov); err == nil {
		t.Fatal("pinned and gone: called")
	}
	p.BridgeTargetPins = nil
	for admin, ok := range map[string]bool{"": true, gov: true, addr32("x"): false} {
		err := p.CheckBridgeTargetCode(types.VMTypeCosmWasm, c, types.BridgeContractState{Found: true, CodeID: 9, Admin: admin}, gov)
		if (err == nil) != ok {
			t.Fatalf("unpinned, admin %q: %v", admin, err)
		}
	}
}

// The release step pins every existing entry to the code it runs, keeps any
// pin already there, skips an entry naming no contract, and reports no change
// on an empty allowlist (nothing is written on mainnet).
func TestPinBridgeTargets(t *testing.T) {
	if _, changed := types.DefaultParams().PinBridgeTargets(lookupOf(nil)); changed {
		t.Fatal("an empty allowlist changed")
	}
	a, b, c, ghost := addr32("a"), addr32("b"), addr32("c"), addr32("ghost")
	look := lookupOf(map[string]types.BridgeContractState{
		a: {Found: true, CodeID: 5, Admin: addr32("eoa")},
		b: {Found: true, CodeID: 6, Admin: addr32("eoa")},
		c: {Found: true, CodeID: 7},
	})
	p := types.DefaultParams()
	p.BridgeAllowedTargets = []string{"cosmwasm:" + a, "cosmwasm:" + b, "cosmwasm:" + c, "cosmwasm:" + ghost}
	p.BridgeTargetPins = []types.BridgeTargetPin{{Target: "cosmwasm:" + b, CodeID: 3}}
	got, changed := p.PinBridgeTargets(look)
	if !changed {
		t.Fatal("nothing pinned")
	}
	want := map[string]uint64{"cosmwasm:" + a: 5, "cosmwasm:" + b: 3, "cosmwasm:" + c: 7}
	if len(got.BridgeTargetPins) != len(want) {
		t.Fatalf("pins %+v", got.BridgeTargetPins)
	}
	for _, pin := range got.BridgeTargetPins {
		if want[pin.Target] != pin.CodeID {
			t.Fatalf("pin %+v", pin)
		}
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("migrated params invalid: %v", err)
	}
	if len(p.BridgeTargetPins) != 1 {
		t.Fatal("the input params were modified")
	}
	if _, again := got.PinBridgeTargets(look); again {
		t.Fatal("pinning twice changed something")
	}
}
