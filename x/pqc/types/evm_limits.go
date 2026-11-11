package types

import "fmt"

// EVM execution limits (proto: evm_limits.proto). Governance may change them
// with MsgUpdateEVMLimits; a zero field, or no stored value, means the
// default below.

const (
	// DefaultEVMTxMaxGas is the most gas one EVM transaction may ask for,
	// from FeatureEVMTxGasCap. Large contract deployments fit well within it
	// (a 24 KiB contract deploys for about 5 to 6 million gas); it bounds the
	// work any single transaction can buy, whatever the block allows.
	DefaultEVMTxMaxGas uint64 = 30_000_000
	// MinEVMTxMaxGas and MaxEVMTxMaxGas bound what governance may set.
	MinEVMTxMaxGas uint64 = 5_000_000
	MaxEVMTxMaxGas uint64 = 200_000_000

	// DefaultPrecompileInputMaxBytes is the largest input a guarded stateful
	// precompile accepts, from FeatureEVMPrecompileInputCanonical. Every
	// method's ordinary use fits: a staking or distribution call is a few
	// hundred bytes, an ICS-20 transfer with a memo of a few kilobytes is
	// under 8 KiB. A governance proposal larger than this (a code upload)
	// is submitted as a Cosmos transaction.
	DefaultPrecompileInputMaxBytes uint64 = 16 * 1024
	// MinPrecompileInputMaxBytes and MaxPrecompileInputMaxBytes bound what
	// governance may set.
	MinPrecompileInputMaxBytes uint64 = 4 * 1024
	MaxPrecompileInputMaxBytes uint64 = 1024 * 1024
)

// EVMLimitsKey is where the governance-set limits are stored.
var EVMLimitsKey = []byte("pqc/evmlimits")

// TxMaxGasOrDefault is the transaction gas cap in force.
func (l EVMLimits) TxMaxGasOrDefault() uint64 {
	if l.TxMaxGas == 0 {
		return DefaultEVMTxMaxGas
	}
	return l.TxMaxGas
}

// PrecompileInputMaxBytesOrDefault is the precompile input cap in force.
func (l EVMLimits) PrecompileInputMaxBytesOrDefault() uint64 {
	if l.PrecompileInputMaxBytes == 0 {
		return DefaultPrecompileInputMaxBytes
	}
	return l.PrecompileInputMaxBytes
}

// Validate bounds the limits; zero keeps the default.
func (l EVMLimits) Validate() error {
	if l.TxMaxGas != 0 && (l.TxMaxGas < MinEVMTxMaxGas || l.TxMaxGas > MaxEVMTxMaxGas) {
		return fmt.Errorf("tx_max_gas %d outside [%d, %d]", l.TxMaxGas, MinEVMTxMaxGas, MaxEVMTxMaxGas)
	}
	if l.PrecompileInputMaxBytes != 0 &&
		(l.PrecompileInputMaxBytes < MinPrecompileInputMaxBytes || l.PrecompileInputMaxBytes > MaxPrecompileInputMaxBytes) {
		return fmt.Errorf("precompile_input_max_bytes %d outside [%d, %d]",
			l.PrecompileInputMaxBytes, MinPrecompileInputMaxBytes, MaxPrecompileInputMaxBytes)
	}
	return nil
}
