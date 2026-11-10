package types_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

func TestStandardPrecompileAddressesMatchCosmosEVM(t *testing.T) {
	for got, want := range map[common.Address]string{
		types.EVMStakingPrecompile:      evmtypes.StakingPrecompileAddress,
		types.EVMDistributionPrecompile: evmtypes.DistributionPrecompileAddress,
		types.EVMICS20Precompile:        evmtypes.ICS20PrecompileAddress,
		types.EVMBankPrecompile:         evmtypes.BankPrecompileAddress,
		types.EVMGovPrecompile:          evmtypes.GovPrecompileAddress,
		types.EVMSlashingPrecompile:     evmtypes.SlashingPrecompileAddress,
	} {
		if got != common.HexToAddress(want) {
			t.Fatalf("%s is not the cosmos/evm address %s", got.Hex(), want)
		}
	}
}

func TestValueMovingPrecompileSet(t *testing.T) {
	for _, a := range []common.Address{
		types.EVMStakingPrecompile, types.EVMDistributionPrecompile,
		types.EVMICS20Precompile, types.EVMGovPrecompile,
	} {
		if !types.IsEVMValueMovingPrecompile(a) {
			t.Fatalf("%s is not treated as value-moving", a.Hex())
		}
	}
	for _, a := range []common.Address{
		types.EVMBankPrecompile, types.EVMSlashingPrecompile,
		common.HexToAddress("0x0000000000000000000000000000000000000A01"),
		common.HexToAddress("0x0000000000000000000000000000000000000901"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	} {
		if types.IsEVMValueMovingPrecompile(a) {
			t.Fatalf("%s is treated as value-moving", a.Hex())
		}
	}
}

func TestCheckValueMovingPrecompile(t *testing.T) {
	staking := types.EVMStakingPrecompile
	gov := types.EVMGovPrecompile
	plain := common.HexToAddress("0x2222222222222222222222222222222222222222")
	lower := func(a common.Address) string { return types.NormalizeEVMWindowTargets([]string{a.Hex()})[0] }

	unscoped := types.EVMAuthWindow{}
	if unscoped.CheckValueMovingPrecompile(&staking) == nil {
		t.Fatal("an unscoped window admitted the staking precompile")
	}
	if unscoped.CheckValueMovingPrecompile(&plain) != nil || unscoped.CheckValueMovingPrecompile(nil) != nil {
		t.Fatal("the precompile rule refused an ordinary recipient or a creation")
	}

	listed := types.EVMAuthWindow{AllowedTargets: []string{lower(staking)}}
	if listed.CheckValueMovingPrecompile(&staking) != nil {
		t.Fatal("a window listing the staking precompile refused it")
	}
	if listed.CheckValueMovingPrecompile(&gov) == nil {
		t.Fatal("a window listing staking admitted gov")
	}
	// Listing is checked the way CheckScope checks it, so upper or mixed case
	// in a stored window (which NormalizeEVMWindowTargets never produces)
	// still matches.
	mixed := types.EVMAuthWindow{AllowedTargets: []string{staking.Hex()}}
	if mixed.CheckValueMovingPrecompile(&staking) != nil {
		t.Fatal("a checksummed listing did not match")
	}

	nativeListed := types.EVMAuthWindow{NativeOnly: true, AllowedTargets: []string{lower(staking)}}
	if nativeListed.CheckValueMovingPrecompile(&staking) == nil {
		t.Fatal("a native_only window admitted a precompile call")
	}
}

// From evm-window-crossvm-bridge the cross-VM bridge precompile (0x0901) is
// value-moving: an unscoped window refuses it, a window that lists it admits
// it, and native_only refuses it. Without the switch it stays an ordinary
// recipient.
func TestCrossVMBridgeIsValueMovingWithSwitch(t *testing.T) {
	bridge := types.EVMCrossVMBridgePrecompile
	if bridge != common.HexToAddress("0x0000000000000000000000000000000000000901") {
		t.Fatalf("bridge address %s", bridge.Hex())
	}
	if types.IsEVMValueMovingPrecompile(bridge) || types.IsEVMValueMovingPrecompileWith(bridge, false) {
		t.Fatal("without the switch the bridge must stay an ordinary recipient")
	}
	if !types.IsEVMValueMovingPrecompileWith(bridge, true) {
		t.Fatal("with the switch the bridge is value-moving")
	}
	for _, a := range types.EVMValueMovingPrecompiles {
		if !types.IsEVMValueMovingPrecompileWith(a, true) || !types.IsEVMValueMovingPrecompileWith(a, false) {
			t.Fatalf("%s lost its classification", a.Hex())
		}
	}

	unscoped := types.EVMAuthWindow{}
	if unscoped.CheckValueMovingPrecompile(&bridge) != nil || unscoped.CheckValueMovingPrecompileWith(&bridge, false) != nil {
		t.Fatal("switch off: an unscoped window must admit the bridge as before")
	}
	if unscoped.CheckValueMovingPrecompileWith(&bridge, true) == nil {
		t.Fatal("switch on: an unscoped window admitted the bridge")
	}
	lower := types.NormalizeEVMWindowTargets([]string{bridge.Hex()})
	listed := types.EVMAuthWindow{AllowedTargets: lower}
	if listed.CheckValueMovingPrecompileWith(&bridge, true) != nil {
		t.Fatal("a window listing the bridge refused it")
	}
	other := types.EVMAuthWindow{AllowedTargets: types.NormalizeEVMWindowTargets([]string{types.EVMStakingPrecompile.Hex()})}
	if other.CheckValueMovingPrecompileWith(&bridge, true) == nil {
		t.Fatal("a window listing only staking admitted the bridge")
	}
	native := types.EVMAuthWindow{NativeOnly: true, AllowedTargets: lower}
	if native.CheckValueMovingPrecompileWith(&bridge, true) == nil {
		t.Fatal("a native_only window admitted the bridge")
	}
}

// Every QoreChain precompile address is classified on purpose: listed here as
// value-moving under the switch or as not value-moving, so one added later
// fails this test until someone decides.
func TestQoreChainPrecompilesClassified(t *testing.T) {
	valueMoving := map[string]bool{
		"0x0000000000000000000000000000000000000800": true,  // staking
		"0x0000000000000000000000000000000000000801": true,  // distribution
		"0x0000000000000000000000000000000000000802": true,  // ics20
		"0x0000000000000000000000000000000000000804": false, // bank, read-only
		"0x0000000000000000000000000000000000000805": true,  // gov
		"0x0000000000000000000000000000000000000806": false, // slashing, unjail only
		"0x0000000000000000000000000000000000000901": true,  // cross-VM bridge
		"0x0000000000000000000000000000000000000A01": false, // PQC verify, pure
		"0x0000000000000000000000000000000000000A02": false, // PQC key status, read-only
		"0x0000000000000000000000000000000000000B01": false, // AI risk score, read-only
		"0x0000000000000000000000000000000000000B02": false, // AI anomaly check, read-only
		"0x0000000000000000000000000000000000000C01": false, // RL consensus params, read-only
	}
	for hex, want := range valueMoving {
		if got := types.IsEVMValueMovingPrecompileWith(common.HexToAddress(hex), true); got != want {
			t.Fatalf("%s: value-moving=%v, want %v", hex, got, want)
		}
	}
}
