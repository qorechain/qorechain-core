package types

// v3.2.7 switch for k-of-n hybrid PQC multisig. Listed in
// V327MultisigFeatures below.
const (
	// FeaturePQCMultisig opens the post-quantum multisig registry: the
	// MsgRegisterPQCMultisig message, the PQCMultisigSignature transaction
	// extension and the ante verification of an account that registered a
	// member set (at least k of its n ML-DSA-87 members sign every
	// transaction, each verification charged PQCHybridVerifyGas before it
	// runs). An account holding a member set may no longer register a
	// single-signer key, and a single-signer account may not register a set.
	// Before it a multisig account could hold at most one ML-DSA key, which
	// one person held, so under HybridRequired it was either unable to send
	// anything but its own registration or had no k-of-n protection on the
	// post-quantum side; the extension was refused as an unsupported
	// extension option, and the message was routed to no handler.
	FeaturePQCMultisig = "pqc-multisig"
)

// V327MultisigFeatures are the v3.2.7 switches for k-of-n hybrid PQC multisig.
var V327MultisigFeatures = []string{
	FeaturePQCMultisig,
}
