package types

// v3.2.7 x/svm address switches. An SVM address is 32 bytes and a native
// (Cosmos/EVM) account is 20 bytes. A wallet's SVM address is its native
// account followed by 12 zero bytes (the canonical image); mapping any other
// 32-byte address to a native account by keeping its first 20 bytes is lossy,
// because 2^96 addresses share each prefix. Both switches only stop the chain
// from treating such an address as a native account: neither opens the SVM
// lane, which stays closed by the compile-time switch in the keeper.
const (
	// FeatureSVMAddrCanonicalIdentity makes a native account's SVM identity
	// always its canonical image. It applies to the sender of MsgDeployProgram,
	// MsgExecuteProgram and MsgCreateAccount, the caller of a cross-VM call into
	// SVM, the owner check of MsgRegisterSVMPQCKey and the account an
	// authenticator (a Phantom key) resolves to. A native account that is not 20
	// bytes (a CosmWasm contract, an ICA host account) has no SVM identity and is
	// refused with an error. Before it the sender's identity was read from the
	// native-to-SVM reverse map, which the store record of ANY 32-byte address
	// overwrites for its 20-byte prefix, so creating a record at "prefix + any
	// tail" replaced that native account's signer identity; the PQC key owner
	// check accepted a sender equal to the first 20 bytes of the account; and a
	// 32-byte native address was truncated to 20 bytes (the payer of
	// MsgCreateAccount, the authenticator account) or made AddrMapKey panic.
	FeatureSVMAddrCanonicalIdentity = "svm-addr-canonical-identity"
	// FeatureSVMAddrMapCanonical stops the reverse map from being written for a
	// non-canonical address: SetAccount records native -> SVM only when the
	// address is the canonical image of that native account, and DeleteAccount
	// removes the entry only when it names the account being deleted. Before it
	// every stored SVM account (programs, data accounts, PDAs, store-only
	// accounts at any address) wrote the entry for its first 20 bytes, and
	// deleting one removed whatever entry that prefix had.
	FeatureSVMAddrMapCanonical = "svm-addr-map-canonical"
)

// V327SVMAddrFeatures are the v3.2.7 switches for closing every lossy SVM address truncation.
var V327SVMAddrFeatures = []string{
	FeatureSVMAddrCanonicalIdentity,
	FeatureSVMAddrMapCanonical,
}
