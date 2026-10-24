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
