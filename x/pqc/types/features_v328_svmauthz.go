package types

// v3.2.8 x/svm authorization rules. Like the rest of V328SVMFeatures they do not
// open the SVM lane; they are what has to hold before governance may open it.
// Below the v3.2.8 plan height every one of them is off, so history replays
// byte-identically: the keeper keeps the earlier rules and the runtime is
// called without the new flags.
const (
	// FeatureSVMClaimNeedsSignature changes how an account may leave the
	// wallet state (system-owned, no data):
	//   - a user program may take ownership of such an account only if the
	//     account signed the transaction, or if it is a program derived address
	//     of that program which the runtime itself saw the program sign for with
	//     sol_invoke_signed (the PDA set returned by the v3 entry, computed from
	//     the program id the runtime holds, never from program memory);
	//   - a canonical address (the image of a native account) never leaves the
	//     wallet state without its own signature, under any program and whatever
	//     its balance, and settlement requires that signature even when the
	//     wallet held nothing.
	// Before it a user program could claim any system-owned blank account that
	// held zero lamports without a signature, including another user's unused
	// canonical address or a not yet created associated token account, and then
	// sweep whatever was later sent there.
	FeatureSVMClaimNeedsSignature = "svm-claim-needs-signature"
	// FeatureSVMBytecodeLoaderOwned makes a deployed program's bytecode account
	// owned by the loader (BPFLoaderUpgradeab1e...) instead of by the program
	// itself, and refuses any instruction that lists a loader-owned account as
	// writable. The v3.2.8 handler moves the bytecode accounts of programs
	// deployed earlier to the loader. Before it a program owned its own
	// bytecode, so it could rewrite it or take the rent locked in it.
	FeatureSVMBytecodeLoaderOwned = "svm-bytecode-loader-owned"
	// FeatureSVMCPIHostPrivileges makes the runtime keep each execution's
	// account keys, signer and writable flags, executable flags, data lengths
	// and program id in host memory, take cross-program invocation privileges
	// and PDA derivations only from there, and fail any execution that changed
	// one of those bytes in its input buffer. With it the runtime also reports
	// the PDAs the top-level program signed for (FeatureSVMClaimNeedsSignature).
	// Before it a calling program could set the signer or writable byte of any
	// account, or the program id, in its own input buffer and the runtime
	// honoured it for the invocation.
	FeatureSVMCPIHostPrivileges = "svm-cpi-host-privileges"
	// FeatureSVMUniqueAccountMetas refuses an instruction that lists the same
	// address twice, before anything is loaded, and sums lamports over unique
	// addresses. Before it the keeper relied on the runtime returning every
	// duplicate for the conservation check to hold.
	FeatureSVMUniqueAccountMetas = "svm-unique-account-metas"
	// FeatureSVMTokenSPLSemantics aligns the token program with SPL Token where
	// authority is concerned: changing an account's owner clears its delegate
	// (and a native account's close authority); only the close authority, or
	// the owner when there is none, may close an account; Burn, Approve,
	// Revoke and SetAuthority refuse a frozen account; every instruction
	// refuses an uninitialized account or mint where SPL does; a mint and a
	// token account are told apart by their exact length; InitializeAccount
	// needs an account the token program owns and that holds the rent-exempt
	// minimum, and the associated token program funds that minimum.
	FeatureSVMTokenSPLSemantics = "svm-token-spl-semantics"
	// FeatureSVMMetadataNotNative stops the runtime from running the token
	// metadata program as a built-in. The keeper never treated it as one, so it
	// was reachable only through a cross-program invocation that no keeper rule
	// was written for; the two sides now agree.
	FeatureSVMMetadataNotNative = "svm-metadata-not-native"
	// FeatureSVMExecuteResultChecked makes MsgExecuteProgram fail when the
	// runtime reports an unsuccessful execution, as the cross-VM path already
	// did. Before it an unsuccessful result from a pre-v3 entry was returned as
	// a successful message.
	FeatureSVMExecuteResultChecked = "svm-execute-result-checked"
	// FeatureSVMRelayedWalletKeyOnly substitutes the relayed wallet's address by
	// the canonical account only for a 32-byte (ed25519) wallet key. Before it a
	// 33-byte key left the wallet address all zero, so every account meta equal
	// to the System program id was replaced by the canonical account and marked
	// as a signer.
	FeatureSVMRelayedWalletKeyOnly = "svm-relayed-wallet-key-only"
)

// V328SVMAuthzFeatures are the v3.2.8 SVM authorization switches.
var V328SVMAuthzFeatures = []string{
	FeatureSVMClaimNeedsSignature,
	FeatureSVMBytecodeLoaderOwned,
	FeatureSVMCPIHostPrivileges,
	FeatureSVMUniqueAccountMetas,
	FeatureSVMTokenSPLSemantics,
	FeatureSVMMetadataNotNative,
	FeatureSVMExecuteResultChecked,
	FeatureSVMRelayedWalletKeyOnly,
}
