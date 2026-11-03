package types

const (
	// FeatureRDKZKVerificationKey makes x/rdk zk settlement usable. A rollup's
	// creator can register its ZK verification key with MsgSetVerificationKey,
	// or in MsgCreateRollup's new verification_key field, until the rollup's
	// first batch; after that only governance can change it, with
	// MsgUpdateVerificationKey. The key is validated when it is set: a Groth16
	// BLS12-381 key in gnark's encoding for exactly four public inputs, or a
	// 16-byte QSTK key for the STARK verifier. A zk batch of a rollup without a
	// usable key is refused at the start of SubmitBatch, and this node refuses
	// it in CheckTx, before the fee. Before it no message could set a key
	// (MsgCreateRollup had no field for one and no profile set one), so every
	// zk batch with a proof failed with code 21 after its fee was paid, the
	// two new messages are refused and a MsgCreateRollup that carries a key is
	// refused.
	FeatureRDKZKVerificationKey = "rdk-zk-verification-key"
)

// V327ZKVKFeatures are the v3.2.7 switches for the RDK ZK verification key.
var V327ZKVKFeatures = []string{
	FeatureRDKZKVerificationKey,
}
