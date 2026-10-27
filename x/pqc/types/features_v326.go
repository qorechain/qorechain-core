package types

const (
	// FeatureRLParamsPrecompileLive makes the rlConsensusParams precompile
	// (0x0C01) return the parameters x/rlconsensus holds: the applied block
	// time and gas price floor, the module's validator set size and the
	// agent's current epoch. Before it the precompile answered the static
	// genesis defaults (5000, 100, 100, 0) whatever the module held. Read-only;
	// the flat 1,500 gas stays and the store reads are charged on top at the
	// store's read cost.
	FeatureRLParamsPrecompileLive = "rl-params-precompile-live"

	// FeatureIBCRateLimit puts the ICS-20 rate limiter (cosmos/ibc-apps
	// rate-limiting) in the transfer stack, IBC v1 and v2, and starts its hour
	// epoch. Limits are per denom and channel (v1) or client (v2) and are set
	// by governance only; none exist at the switch, and a path without a limit
	// is unlimited, so the switch alone changes no transfer. Before it the
	// stack is the v3.2.5 one, the hour epoch is not kept and the module's
	// messages are refused.
	FeatureIBCRateLimit = "ibc-rate-limit"

	// FeatureIBCPacketForward puts Packet Forward Middleware in the ICS-20 v1
	// stack, between the rate limiter and the erc20 middleware: a transfer
	// whose memo carries a "forward" object is received into a derived
	// intermediate account and sent on to the next hop, with the
	// acknowledgement written back once that hop answers. Before it the memo
	// is not read and the transfer is received by its named receiver.
	FeatureIBCPacketForward = "ibc-packet-forward"

	// FeatureIBCWasmLightClient opens the 08-wasm light client's messages
	// (store, migrate and remove light client code), all of them signed by the
	// governance authority only. A wasm client still cannot be created until
	// governance adds "08-wasm" to the IBC allowed_clients parameter, which
	// this switch does not do.
	FeatureIBCWasmLightClient = "ibc-wasm-light-client"
)

// V326Features is the v3.2.6 release set. Every consensus change of the release
// is guarded by one of these switches, so below the v3.2.6 plan height history
// replays byte-identically.
var V326Features = []string{
	FeatureRLParamsPrecompileLive,
	FeatureIBCRateLimit,
	FeatureIBCPacketForward,
	FeatureIBCWasmLightClient,
}
