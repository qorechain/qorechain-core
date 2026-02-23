package types

const (
	EventTypeCrossVMRequest  = "crossvm_request"
	EventTypeCrossVMResponse = "crossvm_response"
	EventTypeCrossVMTimeout  = "crossvm_timeout"

	AttributeKeyMessageID      = "message_id"
	AttributeKeySourceVM       = "source_vm"
	AttributeKeyTargetVM       = "target_vm"
	AttributeKeySourceContract = "source_contract"
	AttributeKeyTargetContract = "target_contract"
	AttributeKeySender         = "sender"
	AttributeKeyStatus         = "status"
	AttributeKeyError          = "error"
)
