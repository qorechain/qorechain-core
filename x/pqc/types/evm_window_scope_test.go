package types_test

import (
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

func open(native bool, targets ...string) types.MsgOpenEVMWindow {
	return types.MsgOpenEVMWindow{
		Sender: "x", Blocks: 10, MaxTxs: 1, MaxValue: sdkmath.NewInt(1),
		NativeOnly: native, AllowedTargets: targets,
	}
}

func TestOpenEVMWindowScopeValidation(t *testing.T) {
	a := "0x63c0c19a282a1B52b07dD5a65b58948A07DAE32B"
	if m := open(false); m.Validate() != nil || m.HasScope() {
		t.Fatal("an unscoped request changed meaning")
	}
	if m := open(true); m.Validate() != nil || !m.HasScope() {
		t.Fatal("native_only request refused or not scoped")
	}
	if m := open(false, a); m.Validate() != nil || !m.HasScope() {
		t.Fatal("allow-list request refused or not scoped")
	}
	for name, m := range map[string]types.MsgOpenEVMWindow{
		"no 0x":     open(false, strings.TrimPrefix(a, "0x")),
		"short":     open(false, "0x1234"),
		"duplicate": open(false, a, strings.ToUpper("0x"+a[2:])),
	} {
		if m.Validate() == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
	many := make([]string, types.MaxEVMWindowTargets+1)
	for i := range many {
		many[i] = common.BigToAddress(sdkmath.NewInt(int64(i + 1)).BigInt()).Hex()
	}
	if m := open(false, many...); m.Validate() == nil {
		t.Fatal("an over-long allow-list was accepted")
	}
	if m := open(false, many[:types.MaxEVMWindowTargets]...); m.Validate() != nil {
		t.Fatal("an allow-list at the maximum was refused")
	}
}

func TestEVMWindowCheckScope(t *testing.T) {
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	other := common.HexToAddress("0x3333333333333333333333333333333333333333")

	unscoped := types.EVMAuthWindow{}
	for _, c := range []types.EVMWindowCall{
		{To: nil, DataLen: 10},
		{To: &to, DataLen: 68, ToHasCode: true, Authorizations: 1},
	} {
		if unscoped.CheckScope(c) != nil {
			t.Fatal("a window without a scope refused a call")
		}
	}

	native := types.EVMAuthWindow{NativeOnly: true}
	if native.CheckScope(types.EVMWindowCall{To: &to}) != nil {
		t.Fatal("native_only refused a plain transfer")
	}
	for name, c := range map[string]types.EVMWindowCall{
		"create":   {To: nil},
		"calldata": {To: &to, DataLen: 4},
		"code":     {To: &to, ToHasCode: true},
		"7702":     {To: &to, Authorizations: 1},
	} {
		if native.CheckScope(c) == nil {
			t.Fatalf("native_only admitted %s", name)
		}
	}

	listed := types.EVMAuthWindow{AllowedTargets: types.NormalizeEVMWindowTargets([]string{to.Hex()})}
	if listed.AllowedTargets[0] != strings.ToLower(to.Hex()) {
		t.Fatalf("targets not normalised: %v", listed.AllowedTargets)
	}
	if listed.CheckScope(types.EVMWindowCall{To: &to, DataLen: 68, ToHasCode: true}) != nil {
		t.Fatal("allow-list refused a listed target")
	}
	if listed.CheckScope(types.EVMWindowCall{To: &other}) == nil {
		t.Fatal("allow-list admitted an unlisted target")
	}
	if listed.CheckScope(types.EVMWindowCall{To: nil}) == nil {
		t.Fatal("allow-list admitted contract creation")
	}
}
