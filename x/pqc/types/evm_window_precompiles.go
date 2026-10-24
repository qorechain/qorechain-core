package types

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// The value-moving standard precompiles and the EVM authorisation window
// (FeatureEVMPrecompileWindowScope).
//
// WHY THEY NEED A RULE OF THEIR OWN. The window bounds what an EVM-lane
// transaction can take out of the account by the value it carries plus its
// maximum fee, and the settlement after execution charges any further fall
// in the sender's native balance. The standard precompiles below act on the
// caller's funds through the Cosmos modules with a transaction value of zero,
// and several of their actions move no balance at all: an undelegation, a
// redelegation to a validator the caller of the classical key runs, a new
// withdraw address that diverts every future reward, a deposit or transfer in
// a denomination other than uqor. None of that can be priced in uqor, so,
// as for tokens, the bound is on WHERE the account may go: such a precompile
// is reachable only when the holder of the post-quantum key listed it in the
// window's allowed_targets.
//
// The bank precompile (0x0804) is read-only (balances, totalSupply,
// supplyOf) and slashing (0x0806) can only unjail the caller's own
// validator, so neither is listed. Both still honour the circuit breaker
// (FeatureEVMPrecompileCircuit).

// The standard precompile addresses, as cosmos/evm defines them.
var (
	EVMStakingPrecompile      = common.HexToAddress("0x0000000000000000000000000000000000000800")
	EVMDistributionPrecompile = common.HexToAddress("0x0000000000000000000000000000000000000801")
	EVMICS20Precompile        = common.HexToAddress("0x0000000000000000000000000000000000000802")
	EVMBankPrecompile         = common.HexToAddress("0x0000000000000000000000000000000000000804")
	EVMGovPrecompile          = common.HexToAddress("0x0000000000000000000000000000000000000805")
	EVMSlashingPrecompile     = common.HexToAddress("0x0000000000000000000000000000000000000806")
)

// EVMValueMovingPrecompiles are the precompiles whose transaction methods move
// or commit the caller's funds. Order is the address order.
var EVMValueMovingPrecompiles = []common.Address{
	EVMStakingPrecompile,
	EVMDistributionPrecompile,
	EVMICS20Precompile,
	EVMGovPrecompile,
}

// IsEVMValueMovingPrecompile reports whether a is one of
// EVMValueMovingPrecompiles.
func IsEVMValueMovingPrecompile(a common.Address) bool {
	for _, p := range EVMValueMovingPrecompiles {
		if p == a {
			return true
		}
	}
	return false
}

// ListsTarget reports whether the window's allowed_targets names a.
func (w EVMAuthWindow) ListsTarget(a common.Address) bool {
	for _, t := range w.AllowedTargets {
		if strings.EqualFold(t, a.Hex()) {
			return true
		}
	}
	return false
}

// CheckValueMovingPrecompile refuses a call to a value-moving precompile that
// the window does not list. Only allowed_targets can admit one: an unscoped
// window cannot, and native_only never admits calldata. Any other recipient
// is left to CheckScope.
func (w EVMAuthWindow) CheckValueMovingPrecompile(to *common.Address) error {
	if to == nil || !IsEVMValueMovingPrecompile(*to) {
		return nil
	}
	if !w.NativeOnly && w.ListsTarget(*to) {
		return nil
	}
	return fmt.Errorf("%s is a precompile that moves or commits the account's funds outside the "+
		"window's value bound; open a window that lists it in allowed_targets", to.Hex())
}
