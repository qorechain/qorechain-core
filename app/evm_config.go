package app

// QoreChain native-token denominations.
//
//	BaseDenom     — bank base unit, 6 decimals (the staking/mint/gas token)
//	ExtendedDenom — 18-decimal representation the EVM works in (x/precisebank
//	                bridges the 12 sub-uqor digits)
//	DisplayDenom  — human-facing unit (1 QOR = 10^6 uqor)
//
// The global cosmos/evm coin metadata is NOT configured here: the x/vm module
// derives it from the genesis (evm_denom + bank denom metadata, see
// genesis_denom.go) and installs it during InitGenesis / the first block via
// SetGlobalConfigVariables. x/vm must therefore init before x/precisebank,
// which reads those globals (see the InitGenesis order in app_config.go).
const (
	BaseDenom     = "uqor"
	ExtendedDenom = "aqor"
	DisplayDenom  = "qor"
	BaseDecimals  = 6
)
