package types

// v3.2.8 abstract-account switches. Abstract accounts are disabled on mainnet;
// these must be on before the module is ever enabled there. Below the v3.2.8
// plan height every one is off, so history replays byte-identically.
//
// The fourth abstract-account change of the release, the management messages
// (MsgCreateAbstractAccount, MsgRegisterAuthenticator, MsgRevokeAuthenticator,
// MsgUpdateSpendingRules) refusing an owner that is not a post-quantum verified
// outer signer of the transaction, is guarded by FeaturePQCConsentOuterSigner
// (features_v328_pqcauthz.go), the switch of the same check on the EVM window
// and EIP-7702 consent messages.
const (
	// FeatureAACreateNoOverwrite makes MsgCreateAbstractAccount refuse an
	// address that already holds an abstract account. Before it the handler
	// wrote a fresh record over the existing one, which erased the account's
	// authenticators, session keys and spending rules (the reverse index of
	// each authenticator stayed behind, pointing at an account that no longer
	// listed it).
	FeatureAACreateNoOverwrite = "abstractaccount-create-no-overwrite"

	// FeatureAAFailedExecutionBurnsNonce makes a signed authenticator execution
	// (MsgExecuteCosmos, MsgExecuteEVM) consume its nonce when it fails after
	// the signature, the key, its permission, the account and the nonce were
	// all accepted. The spending check and the transfer or call run in a cache
	// branch; on failure the branch is dropped, the nonce still advances, the
	// handler returns success=false (code 0) and emits an
	// abstractaccount_execution_failed event with the reason. On the EVM lane
	// the call policy and the value pre-check move after the nonce check so
	// that they burn it too; the reason goes in vm_error and a reverted call's
	// gas is charged as a successful call's is. A refusal of the transaction envelope (a signed
	// gas limit the transaction does not provision) still fails the
	// transaction without burning. Before it every failure reverted the whole
	// transaction including the nonce, so the same signed authorisation could
	// be relayed again by anyone until the key expired, for example the next
	// day once a daily limit reset.
	FeatureAAFailedExecutionBurnsNonce = "abstractaccount-failed-execution-burns-nonce"

	// FeatureAAValueRuleRequired requires an outflow limit on every
	// authenticator that may move value (permission "send", "evm", "deploy" or
	// "all"): its effective spending rule must bound every denom it admits (a
	// per_tx_limit or daily_limit, or an allowed_denoms list whose every denom
	// has a denom_limits amount). MsgRegisterAuthenticator refuses such a key
	// without one; MsgUpdateSpendingRules refuses rules that would leave an
	// active one unbounded; MsgExecuteCosmos and MsgExecuteEVM refuse an
	// execution by an unbounded one (a key linked before the switch), before
	// the nonce is checked. So that an owner can set rules on the account at
	// its own address before linking a value key, MsgUpdateSpendingRules
	// creates that account (as registration already does) when it does not
	// exist yet and the module is enabled. Before it a key with no rule moved
	// any amount: a classical key worth the whole balance with no window bound.
	FeatureAAValueRuleRequired = "abstractaccount-value-rule-required"
)

// V328AAFeatures are the v3.2.8 switches for abstract account hardening.
var V328AAFeatures = []string{
	FeatureAACreateNoOverwrite,
	FeatureAAFailedExecutionBurnsNonce,
	FeatureAAValueRuleRequired,
}
