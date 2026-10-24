package types

// Released in v3.2.5: what it takes for a cross-VM call to be usable on a live
// network. Below the v3.2.5 plan height all three are off, so history replays
// byte-identically.
const (
	// x/crossvm: MsgUpdateParams, signed by the governance module account,
	// replaces the module parameters, among them bridge_allowed_targets. Before
	// it the allowlist the hardened bridge precompile checks could only be set
	// in genesis, so on a running chain the bridge could never be opened.
	FeatureCrossVMParamsUpdate = "crossvm-params-update"
	// The CrossVM bridge precompile (0x0901) answers in its published ABI: the
	// selector must be executeCrossVMCall(uint8,string,bytes), the call may not
	// carry value, and the callee's answer comes back ABI-encoded as bytes. Bank
	// movements made by the callee are mirrored into the calling EVM state, and
	// nothing reached through the bridge may enter the EVM again.
	FeatureCrossVMBridgeABI = "crossvm-bridge-abi"
	// x/crossvm: a call into a lane is refused while that lane's own entry
	// message is circuit-tripped (MsgEthereumTx, MsgExecuteContract,
	// MsgExecuteProgram), and an async call prepays a fixed execution budget
	// that bounds what the end-of-block queue may spend on it.
	FeatureCrossVMLaneBounds = "crossvm-lane-bounds"
)

// V325CrossVMFeatures is this area's share of the v3.2.5 release set. Every
// consensus change of the area is guarded by one of these switches, so below
// the v3.2.5 plan height history replays byte-identically.
var V325CrossVMFeatures = []string{
	FeatureCrossVMParamsUpdate,
	FeatureCrossVMBridgeABI,
	FeatureCrossVMLaneBounds,
}
