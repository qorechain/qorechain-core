package types

const (
	EventTypeLicenseGranted   = "license_granted"
	EventTypeLicenseRevoked   = "license_revoked"
	EventTypeLicenseSuspended = "license_suspended"
	EventTypeLicenseResumed   = "license_resumed"
	EventTypeLicenseExpired   = "license_expired"

	AttributeKeyGrantee   = "grantee"
	AttributeKeyFeatureID = "feature_id"
	AttributeKeyGrantedBy = "granted_by"
	AttributeKeyExpiresAt = "expires_at"
)
