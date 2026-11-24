package types

// v3.2.8 switches for post-quantum consent messages. Below the v3.2.8 plan
// height all are off, so history replays byte-identically.
const (
	// FeaturePQCConsentAuthzLimiter extends the Cosmos ante's authz limiter
	// (MsgExec nested to any depth, and MsgGrant of the type, top-level or
	// nested) with the messages that state an account's post-quantum consent
	// or manage how the account is authorised: the EVM window and EIP-7702
	// consent messages (PQCConsentMsgTypeURLs), the abstract account
	// management messages, the staking tier early unlock (a 5% burn of the
	// delegator's locked stake), and MsgGrant itself, so a grant cannot hand
	// out further grants in the granter's name and an existing grant of
	// MsgGrant cannot be used. Before it a grantee holding a grant of one of
	// these types could send it for the granter with the grantee's own
	// post-quantum signature; the granter's key never signed it.
	FeaturePQCConsentAuthzLimiter = "pqc-consent-authz-limiter"

	// FeaturePQCConsentOuterSigner makes the handlers of the EVM window and
	// EIP-7702 consent messages (open, close, authorise, clear) refuse unless
	// the account the message names is an outer signer of the transaction
	// whose post-quantum signature the ante handler verified
	// (RequireVerifiedOuterSigner). The abstract account management handlers
	// use the same check under the same switch. This also covers what the
	// ante limiter never sees: an x/group proposal, a contract's
	// sub-message, an interchain-account packet. Before it the handlers
	// accepted any message whose Sender held a registered key, whoever had
	// signed the transaction that carried it.
	FeaturePQCConsentOuterSigner = "pqc-consent-outer-signer"

	// FeaturePQCRotationEndsEVMConsent makes a key rotation
	// (MsgRotatePQCKey) and a key migration (MsgMigratePQCKey) delete the
	// account's EVM window and EIP-7702 consent, as MsgCloseEVMWindow does:
	// what the old key authorised does not outlive it. Before it both
	// survived the rotation until they expired.
	FeaturePQCRotationEndsEVMConsent = "pqc-rotation-ends-evm-consent"

	// FeaturePQCDegenerateKeyRefused refuses, in the hybrid signature check,
	// a signature from a stored ML-DSA-87 key that IsDegeneratePublicKey
	// flags (all-zero t1 and the other shapes refused at registration since
	// FeaturePQCRegistrationIntegrity), and makes such a record repairable,
	// so its owner can register a real key with MsgRegisterPQCKeyV2 signed
	// classically. Before it a degenerate key registered before that switch
	// kept verifying signatures and could not be replaced by registration.
	FeaturePQCDegenerateKeyRefused = "pqc-degenerate-key-refused"
)

// V328PQCAuthzFeatures are the v3.2.8 switches for PQC consent and authz/group sender guards, key rotation closing EVM consent, degenerate keys.
var V328PQCAuthzFeatures = []string{
	FeaturePQCConsentAuthzLimiter,
	FeaturePQCConsentOuterSigner,
	FeaturePQCRotationEndsEVMConsent,
	FeaturePQCDegenerateKeyRefused,
}
