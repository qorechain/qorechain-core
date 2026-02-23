//go:build !proprietary

package app

// registerEVMPrecompiles is a no-op in the public build.
// The proprietary build registers standard QoreChain EVM precompiles
// (bank, staking, distribution, gov, etc.) plus QoreChain custom
// precompiles (CrossVM).
func (app *QoreChainApp) registerEVMPrecompiles() {
	// No precompiles in public build.
	// EVM will still function with the default geth precompiles
	// (ecrecover, sha256, ripemd160, identity, modexp, ecAdd, ecMul, ecPairing, blake2f).
}
