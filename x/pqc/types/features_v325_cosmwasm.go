package types

// v3.2.5 CosmWasm: contracts are deployed through governance. Code upload stays
// closed to accounts (code_upload_access = Nobody, MsgStoreCode on the circuit
// breaker's disable list); governance stores reviewed code with a proposal that
// resets the breaker for MsgStoreCode, stores, and trips it again. Anyone may
// then instantiate and execute that code.
const (
	// Mempool policy: MsgInstantiateContract and MsgInstantiateContract2 are
	// admitted into the mempool again. Before the switch the node refuses them
	// in CheckTx, as it has since the upload lockdown; block execution is not
	// touched either way, since the chain parameters and the circuit breaker
	// decide whether an instantiation runs. Upload messages stay refused in the
	// mempool, because the governance path never passes through it.
	FeatureWasmGovernedDeploy = "wasm-governed-deploy"
)

// V325WasmFeatures is this area's share of the v3.2.5 release set. Every
// consensus change of the area is guarded by one of these switches, so below
// the v3.2.5 plan height history replays byte-identically.
var V325WasmFeatures = []string{FeatureWasmGovernedDeploy}
