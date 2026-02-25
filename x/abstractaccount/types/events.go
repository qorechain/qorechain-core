package types

const (
	EventTypeCreateAccount      = "create_abstract_account"
	EventTypeUpdateSpendingRules = "update_spending_rules"
	EventTypeAddSessionKey      = "add_session_key"
	EventTypeRevokeSessionKey   = "revoke_session_key"

	AttributeKeyAccountAddress  = "account_address"
	AttributeKeyContractAddress = "contract_address"
	AttributeKeyAccountType     = "account_type"
	AttributeKeySessionKey      = "session_key"
	AttributeKeyOwner           = "owner"
)
