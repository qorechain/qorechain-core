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
	// FeatureSVMZeroLamportAccountsClosed deletes, at the end of an
	// instruction, every account other than a wallet that it left with zero
	// lamports, as Solana's runtime does. Before it a closed token account
	// stayed in the store owned by the token program, uninitialized, and anyone
	// could initialize it again with themselves as holder, including at a
	// wallet's associated token account address.
	FeatureSVMZeroLamportAccountsClosed = "svm-zero-lamport-accounts-closed"
	// FeatureSVMATAIdempotentVerified makes the associated token program's
	// CreateIdempotent accept an existing account only if it is the token
	// account it would have created: owned by the token program, initialized,
	// for the same mint and held by the same wallet. Before it any account with
	// data at the address was reported as already existing.
	FeatureSVMATAIdempotentVerified = "svm-ata-idempotent-verified"
	// FeatureSVMCPICalleeRules applies Solana's account rules to every
	// cross-program invocation: an account not writable for the callee does not
	// change, an account the callee does not own keeps its data and owner and
	// may only gain lamports, the executable flag never changes, the
	// invocation's lamports balance, and an invocation names each account once.
	// Before it a callee's changes to data and lamports were copied back to the
	// caller unchecked; only the top-level result was judged.
	FeatureSVMCPICalleeRules = "svm-cpi-callee-rules"
	// FeatureSVMNoStrandedCredits refuses an instruction that leaves lamports
	// added to a system-owned account with no data at an address that is not
	// canonical, unless the running program signed for it as its PDA in the
	// same instruction. No signer can ever be produced for such an address (a
	// raw ed25519 key, a seed address), so the value could never move again.
	FeatureSVMNoStrandedCredits = "svm-no-stranded-credits"
	// FeatureSVMCustodyCapNet applies the custody cap to the net change of the
	// svm module account over an instruction's settlement instead of to each
	// inflow, so a wallet-to-wallet transfer that leaves custody unchanged is
	// not refused when custody is at the cap.
	FeatureSVMCustodyCapNet = "svm-custody-cap-net"
	// FeatureSVMATAPrefundedCreate makes the associated token program create a
	// token account at an address that already holds lamports but is otherwise
	// blank, topping it up to the rent-exempt minimum, as SPL does. Before it
	// one lamport sent to the address blocked the account forever.
	FeatureSVMATAPrefundedCreate = "svm-ata-prefunded-create"
	// FeatureSVMTokenCloseFrozenRefused makes CloseAccount refuse a frozen
	// token account, as SPL Token does.
	FeatureSVMTokenCloseFrozenRefused = "svm-token-close-frozen-refused"
	// FeatureSVMRelayedGrossOutflow makes a relayed (linked-key) execution
	// charge the key's spending rule with the gross lamports that left every
	// account of the instruction, not only the net change of the canonical
	// account, and refuses a relayed execution that changes an existing token
	// account: spending rules are in QOR and cannot bound token value.
	FeatureSVMRelayedGrossOutflow = "svm-relayed-gross-outflow"
	// FeatureSVMRelayedIdentityLocked refuses a relayed (linked-key) execution
	// that changes the owner, data or executable flag of the canonical account
	// it acts for, or of an account that account owns, and counts a canonical
	// wallet that stops being a wallet as an outflow of its whole previous
	// balance. Before it a linked key could assign the holder's canonical
	// account to a program of its choice: the balance moved into custody as that
	// program's store lamports and no lamports "left", so the spending rule was
	// charged nothing. The holder's own signature (a message from the native
	// account) is not affected.
	FeatureSVMRelayedIdentityLocked = "svm-relayed-identity-locked"
	// FeatureSVMZeroLamportTouchedOnly limits svm-zero-lamport-accounts-closed
	// to accounts the instruction itself changed (or brought to zero lamports);
	// an untouched account listed writable is never deleted. The v3.2.8 handler
	// reports, in store order, every account other than a wallet that already
	// holds zero lamports and leaves it as it is.
	FeatureSVMZeroLamportTouchedOnly = "svm-zero-lamport-touched-only"
	// FeatureSVMCustodyCapRentExempt exempts from the custody cap the inflow
	// that only pays the rent-exempt minimum of an account being created (a
	// program deploy, MsgCreateAccount at exactly the minimum, an account a
	// built-in or program creates in an instruction). One holder filling the
	// shared cap with free value can then no longer block other users from
	// creating accounts or deploying. Rent deposits stay bound to the account
	// they fund.
	FeatureSVMCustodyCapRentExempt = "svm-custody-cap-rent-exempt"
	// FeatureSVMCreateAccountOwnerChecked refuses MsgCreateAccount with the
	// loader as owner (the account could never be writable again). An account
	// created for the token program is uninitialized until InitializeMint or
	// InitializeAccount runs; clients must put both messages in the same
	// transaction, as on Solana, or anyone may initialize it first.
	FeatureSVMCreateAccountOwnerChecked = "svm-create-account-owner-checked"
	// FeatureSVMNoStrandedZeroedData extends svm-no-stranded-credits to a
	// system-owned account whose data is all zero but not empty (what a program
	// leaves when it gives an account back to System).
	FeatureSVMNoStrandedZeroedData = "svm-no-stranded-zeroed-data"
	// FeatureSVMRelayedChainBound binds the chain id into the message a linked
	// key signs for a relayed execution (types.SVMAuthSignBytesV3). Before it
	// the v2 message carried no chain id.
	FeatureSVMRelayedChainBound = "svm-relayed-chain-bound"
	// FeatureSVMCustodyLiabilityWindow measures the custody cap against what
	// the SVM owes (store lamports above each account's rent reserve, plus
	// wallet dust), kept as a running total, instead of the svm module
	// account's raw balance, and turns it into a limit on the net growth of
	// that liability per window of CustodyWindowBlocks blocks. Rent reserves
	// stay bound to their accounts and are released when an account drops
	// below its reserve or is deleted. When the liability itself exceeds the
	// cap an svm_custody_above_cap event is emitted for governance. Before it
	// one holder could fill the shared cap with recoverable value (or the
	// module balance could already exceed it from faucet funding) and block
	// every other user's deposits.
	FeatureSVMCustodyLiabilityWindow = "svm-custody-liability-window"
	// FeatureSVMRelayedProgramScope refuses a relayed (linked-key) execution
	// that changes the data or owner of an account owned by (or handed to) a
	// program other than the built-ins, unless the key holds
	// "svm:program:<program id>" or "all" among its permissions. Lamport moves,
	// System transfers, ATA creation and Memo are unaffected. Before it value a
	// user program kept for the account in its own records moved under a key
	// whose QOR spending rule could not bound it.
	FeatureSVMRelayedProgramScope = "svm-relayed-program-scope"
	// FeatureSVMNoCreditCustodyOrProgram refuses lamports sent to the svm
	// module account's own SVM address (nobody can ever sign for it) and to an
	// executable account (a program account can never pay them out).
	FeatureSVMNoCreditCustodyOrProgram = "svm-no-credit-custody-or-program"
	// FeatureSVMCPICallerRules makes the runtime apply Solana's ownership rules
	// to a program's own changes since the host last validated its accounts,
	// at every cross-program invocation and at the end of every run: only the
	// owner of an account changes its data or takes its lamports, only the
	// owner gives it away (data zeroed), a read-only account does not change.
	// Before it a caller could forge another program's state (a token
	// balance) in its buffer and a callee took it as real.
	FeatureSVMCPICallerRules = "svm-cpi-caller-rules"
	// FeatureSVMCPIComposable lets cross-program invocations compose as on
	// Solana: accounts may grow by up to 10 KiB in an instruction (padded
	// layout), a callee's owner change reaches its caller, and the runtime
	// attests every program that ran and every PDA a program signed for. The
	// keeper then judges a user program's instruction per account by the
	// programs that actually ran instead of requiring the top-level program to
	// own every changed account. Token transfers from PDA-held accounts, ATA
	// and PDA creation through invocations, state updates by called programs
	// and debits of system-owned PDA vaults all work. Needs
	// svm-cpi-caller-rules.
	FeatureSVMCPIComposable = "svm-cpi-composable"
	// FeatureSVMSystemWithSeed accepts System CreateAccountWithSeed,
	// AllocateWithSeed, AssignWithSeed and TransferWithSeed at the top level
	// when the base account signed and the derived address matches, as on
	// Solana. Before it the derived account never signed, so all four were
	// refused and System could not create an account in one message.
	FeatureSVMSystemWithSeed = "svm-system-with-seed"
	// FeatureSVMPQCKeyForWallets lets a plain wallet (no SVM store record)
	// register its PQC key with MsgRegisterSVMPQCKey: the record is created on
	// first registration, still only for the sender's own canonical address,
	// and the key length is always checked.
	FeatureSVMPQCKeyForWallets = "svm-pqc-key-for-wallets"
	// FeatureSVMCreateAccountOwnerWritable refuses MsgCreateAccount for an
	// owner that could never write the account: only the token program or a
	// deployed (executable) program may own it. Before it the sender could
	// lock its own lamports in an account owned by ATA, Memo, System or any
	// address that is not a program.
	FeatureSVMCreateAccountOwnerWritable = "svm-create-account-owner-writable"
	// FeatureSVMPDASigningMetered charges every signer-seed set of
	// sol_invoke_signed like sol_create_program_address (1,500 compute units)
	// before deriving it, and refuses an instruction that signs for more than
	// 1,024 distinct PDAs. Before it the derivations were free and the logs
	// grew quadratically, so a flood cost validators far more CPU than gas.
	FeatureSVMPDASigningMetered = "svm-pda-signing-metered"
	// FeatureSVMCPIReentrancyGuard refuses, as Solana does, an invocation of a
	// program that is already executing unless it is the immediate caller
	// invoking itself; direct self-recursion, the top-level program included,
	// works. Before it V -> Y -> V re-entered V through a third program.
	FeatureSVMCPIReentrancyGuard = "svm-cpi-reentrancy-guard"
	// FeatureSVMATAPayerSystem requires a system-owned ATA payer and lets the
	// associated token program change only system-owned accounts.
	FeatureSVMATAPayerSystem = "svm-ata-payer-system"
	// FeatureSVMTokenLamportsCloseOnly lets a token-program account lose
	// lamports only when it is closed (zero lamports, data zeroed), whoever
	// ran. Before it any decrease was accepted once the token program ran.
	FeatureSVMTokenLamportsCloseOnly = "svm-token-lamports-close-only"
	// FeatureSVMResultSizeExact makes the runtime report the size a result
	// needs instead of cutting it to the buffer, and the keeper run once more
	// with a buffer of that size, so large legitimate results (accounts grown
	// in an invocation, System creations over 64 KiB) settle.
	FeatureSVMResultSizeExact = "svm-result-size-exact"
	// FeatureSVMAttestedCreatedAccounts makes the keeper's attested rule check
	// a created account independently: its new owner is System or a program
	// that ran, and its data is zero unless that owner ran.
	FeatureSVMAttestedCreatedAccounts = "svm-attested-created-accounts"
	// FeatureSVMAccountSizeRentChecked refuses an account larger than
	// max_account_data_size after execution, and an account whose data grew
	// without holding the rent-exempt minimum for its new size, as Solana does.
	FeatureSVMAccountSizeRentChecked = "svm-account-size-rent-checked"
	// FeatureSVMLiabilityLiveRent computes the custody liability's rent
	// reserves with the live rent params and recounts the liability when
	// governance changes them. Before it the default rates were used.
	FeatureSVMLiabilityLiveRent = "svm-liability-live-rent"
	// FeatureSVMSystemWithSeedCPI accepts System *WithSeed instructions run
	// through an invocation: the runtime attests the derived account when its
	// base signed in the calling frame, and the keeper treats it as signed.
	FeatureSVMSystemWithSeedCPI = "svm-system-with-seed-cpi"
)

// CustodyWindowBlocks is the window over which svm-custody-liability-window
// limits the net growth of the SVM liability to max_custody_uqor.
const CustodyWindowBlocks int64 = 3600

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
	FeatureSVMZeroLamportAccountsClosed,
	FeatureSVMATAIdempotentVerified,
	FeatureSVMCPICalleeRules,
	FeatureSVMNoStrandedCredits,
	FeatureSVMCustodyCapNet,
	FeatureSVMATAPrefundedCreate,
	FeatureSVMTokenCloseFrozenRefused,
	FeatureSVMRelayedGrossOutflow,
	FeatureSVMRelayedIdentityLocked,
	FeatureSVMZeroLamportTouchedOnly,
	FeatureSVMCustodyCapRentExempt,
	FeatureSVMCreateAccountOwnerChecked,
	FeatureSVMNoStrandedZeroedData,
	FeatureSVMRelayedChainBound,
	FeatureSVMCustodyLiabilityWindow,
	FeatureSVMRelayedProgramScope,
	FeatureSVMNoCreditCustodyOrProgram,
	FeatureSVMCPICallerRules,
	FeatureSVMCPIComposable,
	FeatureSVMSystemWithSeed,
	FeatureSVMPQCKeyForWallets,
	FeatureSVMCreateAccountOwnerWritable,
	FeatureSVMPDASigningMetered,
	FeatureSVMCPIReentrancyGuard,
	FeatureSVMATAPayerSystem,
	FeatureSVMTokenLamportsCloseOnly,
	FeatureSVMResultSizeExact,
	FeatureSVMAttestedCreatedAccounts,
	FeatureSVMAccountSizeRentChecked,
	FeatureSVMLiabilityLiveRent,
	FeatureSVMSystemWithSeedCPI,
}
