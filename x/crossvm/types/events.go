package types

const (
	EventTypeCrossVMRequest  = "crossvm_request"
	EventTypeCrossVMResponse = "crossvm_response"
	EventTypeCrossVMTimeout  = "crossvm_timeout"
	EventTypeParamsUpdated   = "crossvm_params_updated"

	// EventTypeBridgeTargetRemoved: the v3.2.8 release step removed a bridge
	// allowlist entry that names no contract.
	EventTypeBridgeTargetRemoved = "crossvm_bridge_target_removed"

	// EventTypeEscrowSettled: a finished async call's escrow was paid out
	// (crossvm-async-escrow): the fee to validators, the refund to its sender.
	EventTypeEscrowSettled = "crossvm_escrow_settled"

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
	AttributeKeyTarget               = "target"
	AttributeKeyReason               = "reason"
	AttributeKeyEscrowFee            = "escrow_fee"
	AttributeKeyEscrowRefund         = "escrow_refund"
	// AttributeKeyEscrowStranded is, on EventTypeEscrowSettled, an escrow
	// that could be neither paid out nor moved to the community pool: the
	// call finished as failed and the coins stay in the escrow account.
	AttributeKeyEscrowStranded = "escrow_stranded"
)
