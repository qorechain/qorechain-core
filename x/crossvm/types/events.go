package types

const (
	EventTypeCrossVMRequest  = "crossvm_request"
	EventTypeCrossVMResponse = "crossvm_response"
	EventTypeCrossVMTimeout  = "crossvm_timeout"
	EventTypeParamsUpdated   = "crossvm_params_updated"

	AttributeKeyMessageID            = "message_id"
	AttributeKeySourceVM             = "source_vm"
	AttributeKeyTargetVM             = "target_vm"
	AttributeKeySourceContract       = "source_contract"
	AttributeKeyTargetContract       = "target_contract"
	AttributeKeySender               = "sender"
	AttributeKeyStatus               = "status"
	AttributeKeyError                = "error"
	AttributeKeyEnabled              = "enabled"
	AttributeKeyBridgeAllowedTargets = "bridge_allowed_targets"
)
