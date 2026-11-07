package types

const (
	EventTypeCreateAccount       = "create_abstract_account"
	EventTypeUpdateSpendingRules = "update_spending_rules"
	EventTypeAddSessionKey       = "add_session_key"
	EventTypeRevokeSessionKey    = "revoke_session_key"
	EventTypeUpdateConfig        = "update_abstractaccount_config"

	AttributeKeyAccountAddress  = "account_address"
	AttributeKeyContractAddress = "contract_address"
	AttributeKeyAccountType     = "account_type"
	AttributeKeySessionKey      = "session_key"
	AttributeKeyOwner           = "owner"
	AttributeKeyEnabled         = "enabled"
)

// v3.2.8 (FeatureAAFailedExecutionBurnsNonce): a signed authenticator
// execution that failed after its nonce was accepted. The nonce is consumed and
// the message returns success=false.
const (
	EventTypeExecutionFailed = "abstractaccount_execution_failed"

	AttributeKeyAccount = "account"
	AttributeKeyLane    = "lane"
	AttributeKeyNonce   = "nonce"
	AttributeKeyReason  = "reason"
)
