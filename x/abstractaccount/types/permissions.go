package types

// Permission taxonomy for authenticator / session-key scoping (v3.1.84).
//
// An authenticator holds a SET of these action-class permissions. A transaction
// authorized by an authenticator (rather than the account's root key) is allowed
// only if EVERY message in it maps to a permission the authenticator holds — or
// the authenticator holds PermAll. The mapping is fail-closed: any message whose
// type URL is not in RequiredPermission is DENIED.
//
// Key-management messages (register/revoke authenticator, register PQC key) are
// NEVER delegable to an authenticator, regardless of its permissions — only the
// account root key may manage keys. See IsKeyManagementMsg.
//
// This vocabulary is the canonical contract shared by the chain, the SDK
// (@qorechain/wallet-adapter), the dashboard, the relayer and QoreX. It is
// surfaced on-chain via the permission-schema query so clients never drift.
const (
	PermAll      = "all"      // wildcard — every action (high-risk, explicit opt-in only)
	PermSend     = "send"     // bank transfers out
	PermDelegate = "delegate" // staking delegate / undelegate / redelegate
	PermWithdraw = "withdraw" // claim staking rewards / commission
	PermVote     = "vote"     // governance vote / deposit
	PermEVM      = "evm"      // EVM txs / calls
	PermWasm     = "wasm"     // CosmWasm execute
	PermSVM      = "svm"      // SVM program execution
	PermAMM      = "amm"      // AMM swaps + liquidity
	PermIBC      = "ibc"      // IBC transfer
	PermDeploy   = "deploy"   // code / program upload (wasm store, evm/svm deploy)

	// PermSchemaVersion is bumped whenever the taxonomy or the mapping below
	// changes, so clients can detect drift against the on-chain schema query.
	PermSchemaVersion = "1"
)

// AllPermissions returns the canonical permission vocabulary (for the schema
// query and for client-side validation of a requested scope).
func AllPermissions() []string {
	return []string{
		PermAll, PermSend, PermDelegate, PermWithdraw, PermVote,
		PermEVM, PermWasm, PermSVM, PermAMM, PermIBC, PermDeploy,
	}
}

// IsValidPermission reports whether p is a known permission string.
func IsValidPermission(p string) bool {
	for _, k := range AllPermissions() {
		if k == p {
			return true
		}
	}
	return false
}

// msgPermission maps a message type URL to the action-class permission it
// requires. Kept explicit (not reflection-based) so the security boundary is
// auditable. New message types MUST be added here or they are denied.
var msgPermission = map[string]string{
	// bank
	"/cosmos.bank.v1beta1.MsgSend":      PermSend,
	"/cosmos.bank.v1beta1.MsgMultiSend": PermSend,
	// staking
	"/cosmos.staking.v1beta1.MsgDelegate":                   PermDelegate,
	"/cosmos.staking.v1beta1.MsgUndelegate":                 PermDelegate,
	"/cosmos.staking.v1beta1.MsgBeginRedelegate":            PermDelegate,
	"/cosmos.staking.v1beta1.MsgCancelUnbondingDelegation":  PermDelegate,
	// distribution (rewards)
	"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward":  PermWithdraw,
	"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission": PermWithdraw,
	// governance
	"/cosmos.gov.v1.MsgVote":          PermVote,
	"/cosmos.gov.v1.MsgVoteWeighted":  PermVote,
	"/cosmos.gov.v1.MsgDeposit":       PermVote,
	"/cosmos.gov.v1beta1.MsgVote":     PermVote,
	"/cosmos.gov.v1beta1.MsgVoteWeighted": PermVote,
	"/cosmos.gov.v1beta1.MsgDeposit":  PermVote,
	// EVM
	"/cosmos.evm.vm.v1.MsgEthereumTx": PermEVM,
	// CosmWasm
	"/cosmwasm.wasm.v1.MsgExecuteContract": PermWasm,
	"/cosmwasm.wasm.v1.MsgInstantiateContract": PermDeploy,
	"/cosmwasm.wasm.v1.MsgStoreCode":       PermDeploy,
	// SVM
	"/qorechain.svm.v1.MsgExecuteProgram": PermSVM,
	"/qorechain.svm.v1.MsgDeployProgram":  PermDeploy,
	// AMM
	"/qorechain.amm.v1.MsgSwapExactIn":    PermAMM,
	"/qorechain.amm.v1.MsgSwapExactOut":   PermAMM,
	"/qorechain.amm.v1.MsgAddLiquidity":   PermAMM,
	"/qorechain.amm.v1.MsgRemoveLiquidity": PermAMM,
	// IBC
	"/ibc.applications.transfer.v1.MsgTransfer": PermIBC,
}

// keyManagementMsgs are actions that manage an account's keys/authenticators and
// therefore can NEVER be performed by a delegated authenticator (anti privilege
// escalation) — only the account root key may do them.
var keyManagementMsgs = map[string]struct{}{
	"/qorechain.abstractaccount.v1.MsgRegisterAuthenticator": {},
	"/qorechain.abstractaccount.v1.MsgRevokeAuthenticator":   {},
	"/qorechain.pqc.v1.MsgRegisterPQCKeyV2":                  {},
	"/qorechain.pqc.v1.MsgRegisterPQCKey":                    {},
	"/qorechain.pqc.v1.MsgMigratePQCKey":                     {},
	"/qorechain.pqc.v1.MsgRotatePQCKey":                      {},
}

// RequiredPermission returns the permission a message type URL requires and
// whether the type URL is known. Unknown ⇒ (‑, false) ⇒ the caller MUST deny
// (fail-closed).
func RequiredPermission(typeURL string) (string, bool) {
	p, ok := msgPermission[typeURL]
	return p, ok
}

// IsKeyManagementMsg reports whether a message manages keys/authenticators and
// is thus non-delegable to an authenticator.
func IsKeyManagementMsg(typeURL string) bool {
	_, ok := keyManagementMsgs[typeURL]
	return ok
}

// MessageAllowed reports whether an authenticator with the given permission set
// may execute a message of typeURL. Fail-closed: unknown type URLs and
// key-management messages are always denied. PermAll grants everything EXCEPT
// key-management messages.
func MessageAllowed(perms []string, typeURL string) bool {
	if IsKeyManagementMsg(typeURL) {
		return false
	}
	required, known := RequiredPermission(typeURL)
	if !known {
		return false
	}
	for _, p := range perms {
		if p == PermAll || p == required {
			return true
		}
	}
	return false
}

// MsgPermissionSchema returns the full type-URL → permission mapping (for the
// on-chain schema query so clients stay in lockstep).
func MsgPermissionSchema() map[string]string {
	out := make(map[string]string, len(msgPermission))
	for k, v := range msgPermission {
		out[k] = v
	}
	return out
}

// KeyManagementMsgs returns the non-delegable message type URLs (for the schema
// query).
func KeyManagementMsgs() []string {
	out := make([]string, 0, len(keyManagementMsgs))
	for k := range keyManagementMsgs {
		out = append(out, k)
	}
	return out
}
