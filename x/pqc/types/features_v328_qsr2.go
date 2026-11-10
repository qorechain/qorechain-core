package types

// v3.2.8 hardening switches for governance deposits, the EVM authorisation
// window, rollup validity-proof inputs, abstract-account key linking and the
// bridge messages. Below the v3.2.8 plan height every one is off, so history
// replays byte-identically.
const (
	// FeatureGovDepositSolvency keeps the governance module account able to
	// settle the deposits it holds (cosmos-sdk patch
	// deps/patches/cosmos-sdk/0012). The end blocker refunds or burns a
	// passed proposal's deposits before its messages run, so a message that
	// spends from the governance account (a bank send, a delegation or a
	// community-pool funding with the governance address as the payer) spends
	// the deposits of other proposals; the later refund of those failed, and
	// the end blocker returned the error, which halts the chain. From the
	// switch:
	//   - after a passed proposal's messages run (in their cache branch), the
	//     governance account must still cover every deposit held for other
	//     proposals, or at least not be short of more than it was before the
	//     messages; otherwise the results are discarded and the proposal is
	//     marked failed with the reason;
	//   - a refund or burn the account cannot cover no longer fails the end
	//     blocker: each deposit is settled on its own, one the balance cannot
	//     cover is settled with what is there, and the unpaid part is logged
	//     and reported in a proposal_deposit_shortfall event.
	FeatureGovDepositSolvency = "gov-deposit-solvency"

	// FeatureEVMWindowCrossVMBridge treats the cross-VM bridge precompile
	// (0x0901) as a value-moving precompile for the EVM authorisation window
	// (see EVMValueMovingPrecompiles): a call to it for an EVM-lane account
	// (an EVM transaction addressed to it, or the account's EIP-7702 code
	// calling it) needs a window that lists it in allowed_targets, as the
	// staking, distribution, ics20 and gov precompiles do. The bridge runs
	// CosmWasm contracts as the caller, so an unscoped window would otherwise
	// admit any allowlisted CosmWasm action of the account. It applies
	// together with FeatureEVMPrecompileWindowScope.
	FeatureEVMWindowCrossVMBridge = "evm-window-crossvm-bridge"

	// FeatureRDKZKCanonicalInputs makes a rollup batch refuse public inputs
	// that are not canonical field elements, instead of reducing them. For a
	// SNARK (Groth16 over BLS12-381) batch, prev_state_root, state_root,
	// data_hash and (when bound) withdrawals_root must each be, read as a
	// big-endian integer, below the BLS12-381 scalar modulus; for a STARK
	// batch, prev_state_root and state_root must be below the Goldilocks
	// modulus. Before it each was reduced modulo the field, so a stored root
	// could differ from the proven value by a multiple of the modulus and
	// still verify.
	FeatureRDKZKCanonicalInputs = "rdk-zk-canonical-inputs"

	// FeatureAAAuthenticatorPossession requires proof of possession when a
	// key is linked to an account: MsgRegisterAuthenticator must carry, in
	// possession_proof, a signature by the linked key over
	// AuthenticatorLinkMessage (chain id, account, scheme and key), verified
	// under the key's scheme. A key a third party linked to its own account
	// first used to block the key's holder from ever linking it. The same
	// switch frees a key's index entry once its binding is no longer active:
	// a binding that has expired or was revoked no longer holds the key, and
	// a new (proven) link of the key replaces it.
	FeatureAAAuthenticatorPossession = "abstractaccount-authenticator-possession"

	// FeatureBridgeMsgIdentifiers runs the identifier checks of the bridge
	// configuration (ValidateChainID, ValidateAsset) on the chain and asset
	// fields of MsgBridgeDeposit, MsgBridgeWithdraw and MsgBridgeAttestation
	// in their handlers, before any state is read. Valid messages are not
	// affected; a malformed identifier fails with the configuration's error
	// instead of a lookup miss further down.
	FeatureBridgeMsgIdentifiers = "bridge-msg-identifiers"
)

// V328QSR2Features are the v3.2.8 hardening switches listed above.
var V328QSR2Features = []string{
	FeatureGovDepositSolvency,
	FeatureEVMWindowCrossVMBridge,
	FeatureRDKZKCanonicalInputs,
	FeatureAAAuthenticatorPossession,
	FeatureBridgeMsgIdentifiers,
}
