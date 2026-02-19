package types

// Event types for the bridge module.
const (
	EventTypeBridgeDeposit       = "bridge_deposit"
	EventTypeBridgeWithdraw      = "bridge_withdraw"
	EventTypeBridgeAttestation   = "bridge_attestation"
	EventTypeOperationExecuted   = "bridge_operation_executed"
	EventTypeCircuitBreakerTrip  = "bridge_circuit_breaker_trip"
	EventTypeValidatorRegistered = "bridge_validator_registered"
	EventTypePQCVerification     = "bridge_pqc_verification"

	AttributeKeyOperationID    = "operation_id"
	AttributeKeyChain          = "chain"
	AttributeKeyAsset          = "asset"
	AttributeKeyAmount         = "amount"
	AttributeKeySender         = "sender"
	AttributeKeyReceiver       = "receiver"
	AttributeKeyValidator      = "validator"
	AttributeKeyStatus         = "status"
	AttributeKeyPQCVerified    = "pqc_verified"
	AttributeKeyAttestations   = "attestations"
	AttributeKeyThreatLevel    = "threat_level"
)
