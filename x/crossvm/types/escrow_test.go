package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

// async_max_factor is for async calls into CosmWasm only, from 1x to the
// highest maximum factor governance may set.
func TestAsyncMaxFactorValidateBasic(t *testing.T) {
	sender := sdk.AccAddress(make([]byte, 20)).String()
	base := MsgCrossVMCall{
		Sender: sender, SourceVM: PlaceholderSourceVM(VMTypeCosmWasm), TargetVM: VMTypeCosmWasm,
		TargetContract: "c", Payload: []byte("{}"), Async: true,
	}
	for _, c := range []struct {
		name string
		edit func(*MsgCrossVMCall)
		ok   bool
	}{
		{"unset", func(*MsgCrossVMCall) {}, true},
		{"1x", func(m *MsgCrossVMCall) { m.AsyncMaxFactor = CongestionFactorOne }, true},
		{"the highest maximum", func(m *MsgCrossVMCall) { m.AsyncMaxFactor = uint64(MaxWasmCongestionMaxFactor) * CongestionFactorOne }, true},
		{"below 1x", func(m *MsgCrossVMCall) { m.AsyncMaxFactor = CongestionFactorOne - 1 }, false},
		{"above the highest maximum", func(m *MsgCrossVMCall) { m.AsyncMaxFactor = uint64(MaxWasmCongestionMaxFactor)*CongestionFactorOne + 1 }, false},
		{"synchronous", func(m *MsgCrossVMCall) { m.Async = false; m.AsyncMaxFactor = 2 * CongestionFactorOne }, false},
		{"into the EVM", func(m *MsgCrossVMCall) { m.TargetVM = VMTypeEVM; m.AsyncMaxFactor = 2 * CongestionFactorOne }, false},
	} {
		m := base
		c.edit(&m)
		if m.TargetVM == VMTypeEVM {
			m.SourceVM = PlaceholderSourceVM(VMTypeEVM)
		}
		if err := m.ValidateBasic(); (err == nil) != c.ok {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

// Only a pending call may hold escrow, and escrow must be valid coins.
func TestEscrowOnlyOnPendingCalls(t *testing.T) {
	m := CrossVMMessage{
		ID: "1", SourceVM: VMTypeNative, TargetVM: VMTypeCosmWasm, Sender: sdk.AccAddress(make([]byte, 20)).String(),
		TargetContract: "c", Payload: []byte("{}"), Status: StatusPending,
		Escrow: sdk.NewCoins(sdk.NewCoin("uqor", sdkmath.NewInt(5))),
	}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	m.Status = StatusTimedOut
	if err := m.Validate(); err == nil {
		t.Fatal("a timed-out call holding escrow passed")
	}
	m.Escrow, m.EscrowFee = nil, sdk.Coins{sdk.Coin{Denom: "uqor", Amount: sdkmath.NewInt(-1)}}
	if err := m.Validate(); err == nil {
		t.Fatal("a negative escrow fee passed")
	}
	if EscrowAccountName == ModuleName || authtypes.NewModuleAddress(EscrowAccountName).Empty() {
		t.Fatal("escrow account name")
	}
}
