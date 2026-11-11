package types

// v3.2.8 switches for authorisation on the EVM lane and for the consent
// messages. Below the v3.2.8 plan height all are off, so history replays
// byte-identically.
const (
	// FeatureEVMDelegatedActorInFrame decides the window rule of a delegated
	// account (EIP-7702) at the moment it acts, inside the call or create it
	// makes, instead of only once the whole execution is over. A delegated
	// account other than the sender that calls code or a stateful precompile,
	// or creates a contract, needs a usable post-quantum key and a live window
	// at that moment; a value transfer out of it needs the window to cover the
	// account's net outflow so far. A refusal fails that one CALL or CREATE
	// (it returns failure to the code that made it, gas returned), exactly as
	// an out-of-balance call does on Ethereum, so the caller can handle it.
	// Before it the rule was applied only after execution, and a refusal
	// reverted the whole transaction: a delegated recipient with no window
	// could make any payment to it, however the payer guarded it (try/catch,
	// a 2300 gas stipend, an ignored success flag), revert the payer's whole
	// transaction. The same rule applies, in frame, to the debits a call hook
	// does not see: a SELFDESTRUCT of the account (checked before its balance
	// moves; a refusal fails that frame as a REVERT does) and a debit by a
	// stateful precompile acting for the account (checked once the debit is
	// mirrored into the EVM state; a refusal fails that precompile call). Only
	// a CALL counts as an action: a STATICCALL changes nothing, and a
	// DELEGATECALL or CALLCODE runs other code as the account itself, whose
	// own calls, creations and value are checked in turn. The
	// end-of-execution settlement stays as the backstop. Value a stateful
	// precompile pays the account (a reward claim, a module payout), and
	// any value another contract or account pays it (a vault withdrawal, an
	// unwrap, a refund, an escrow or withdrawal queue finalised by anyone, a
	// SELFDESTRUCT naming it), do not offset what it sends. Only value the
	// execution's sender pays it (the transaction's own value, or a payment
	// by the sender's own code) does, so the account can forward what the
	// sender paid it.
	// Runs that commit nothing (eth_call, eth_estimateGas, a module's
	// read-only call) apply the same rules to every account but the sender,
	// whose window the ante handler checks at delivery.
	FeatureEVMDelegatedActorInFrame = "evm-delegated-actor-in-frame"

	// FeatureEVMDelegatedActorScope applies the window's scope (native_only,
	// allowed_targets) to what a delegated account does as an actor: the
	// recipient of every call to code or a stateful precompile it makes, and
	// any contract creation. native_only admits no such action; an
	// allowed_targets window admits calls to listed recipients only and no
	// creation. It also applies to the sender when the sender is itself
	// delegated and its code acts (a listed contract calling back into the
	// sender). Checked in the call hook (the call fails in frame) and again in
	// the settlement from the recorded recipients. Before it the scope bounded
	// only the recipient of the transaction itself, and an account's own
	// delegated code could move its tokens anywhere while it held a window.
	FeatureEVMDelegatedActorScope = "evm-delegated-actor-scope"

	// FeatureEVMEcrecoverWindow makes the ecrecover precompile (0x01) honour
	// the post-quantum requirement of the address it recovers. In a committed
	// EVM execution, when the recovered address holds a usable post-quantum
	// key and is not the sender of the transaction, the recovery succeeds only
	// if the address has a live window whose scope admits the verifying
	// contract (native_only admits none; allowed_targets must list the
	// contract that called 0x01). Otherwise 0x01 returns the empty result, as
	// for an invalid signature. An accepted signature is never charged to the
	// window: a signature is public once used, and a charge would let anyone
	// replay it to use the window up. The sender's own signature is bounded
	// by its window's scope when the window has one. A delegated account's
	// own code checking its own signature is not bounded by the scope (what
	// that code then does is, action by action). Addresses without any
	// post-quantum record (contracts, accounts that never registered one)
	// recover as before; an account whose record authorises nothing (empty,
	// classical-only, degenerate) is treated as a key without a window, its
	// signature refused. The gas is unchanged (3000) and the reads are off the
	// meter. Before it a stolen classical key could sign an EIP-2612 permit, a
	// Permit2 or order-book signature, or a meta transaction for a protected
	// account and have someone else submit it, without the account's window
	// being consulted at all. Known limit: signature checks a contract does
	// with its own arithmetic, without 0x01, are outside what the chain sees.
	FeatureEVMEcrecoverWindow = "evm-ecrecover-window"

	// FeaturePQCRegistrationEndsEVMConsent makes MsgRegisterPQCKeyV2 that
	// replaces an existing record (a degenerate, classical-only or empty one,
	// which is what registration may replace) delete the account's EVM window
	// and EIP-7702 consent, as a rotation and a migration do from
	// FeaturePQCRotationEndsEVMConsent. Before it what the replaced record had
	// authorised (possibly with a forged signature of a degenerate key)
	// survived the repair until it expired.
	FeaturePQCRegistrationEndsEVMConsent = "pqc-registration-ends-evm-consent"

	// FeatureEVMWindowUsableKey makes the EVM lane require a usable
	// post-quantum key, not only a stored record: the window decorator, the
	// delegated-actor settlement and the in-frame checks refuse an account
	// whose record is replaceable (empty, classical-only, or degenerate once
	// FeaturePQCDegenerateKeyRefused is on) or whose algorithm governance has
	// disabled. A window opened while the key was usable stops admitting
	// anything once it is not. Before it a window outlived the key's
	// usability until it expired.
	FeatureEVMWindowUsableKey = "evm-window-usable-key"

	// FeatureEVMNativeAllowanceWindow bounds ERC-20 allowances of native QOR
	// by the owner's window. If a token pair for the EVM denomination is ever
	// registered, an approve (or an allowance increase) on it by an owner that
	// holds a usable post-quantum key is a value outflow of the owner: it must
	// fit in what remains of the owner's live window and is charged to it
	// (the amount by which the allowance grows, in uqor rounded up), in frame,
	// with the approve failing otherwise (an amount no window can hold, such
	// as approve(max), fails the approve as an ordinary error). A
	// transferFrom then spends an allowance the owner's window already paid
	// for: it is not charged again, and stays usable after the owner's window
	// expires, as an ERC-20 allowance does. No such pair exists on any
	// network; the guard makes one safe whenever it appears.
	FeatureEVMNativeAllowanceWindow = "evm-native-allowance-window"

	// FeaturePQCConsentOwnMessage tightens FeaturePQCConsentOuterSigner: a
	// consent or account-management message (the EVM window and EIP-7702
	// consent messages, the abstract account management messages) must be
	// one of the transaction's own top-level messages, signed by the account
	// it names, not merely a message naming an account that signed the
	// transaction. A message dispatched from inside another (authz MsgExec,
	// an x/group proposal, a contract's sub-message) never qualifies, even
	// when the account it names signed the transaction that ran the
	// dispatcher. The same switch makes the authz keeper refuse, at dispatch,
	// an inner message of a type the consent authz limiter lists
	// (ConsentAuthzLimitedMsgTypeURLs, MsgGrant and the key management
	// messages included), so the limit no longer depends on the ante handler
	// alone. Before it a contract holding an authz grant from an account
	// could open a window, consent to a delegation or register an
	// authenticator in that account's name whenever the account called it.
	FeaturePQCConsentOwnMessage = "pqc-consent-own-message"

	// FeatureEVMPraguePrecompiles makes the EVM serve the Ethereum
	// precompiles of the Prague fork rules the chain runs under (the KZG point
	// evaluation at 0x0a and the BLS12-381 operations at 0x0b..0x11) from
	// the fork's own set, where the application registers only the Berlin
	// set. Before it a call to one of them, directly or from a contract,
	// panicked in the EVM keeper ("precompiled contract not stored in
	// memory"): a transaction failed with a recovered panic, a query returned
	// an internal error, and an end-of-block cross-VM call that did not
	// contain panics failed the block.
	FeatureEVMPraguePrecompiles = "evm-prague-precompiles"

	// FeatureEVMDelegatedActorChargeOnCost changes what the delegated-actor
	// settlement (evm-delegated-actor-settlement) charges to the window of a
	// delegated account that is not the sender. Its actions are still
	// checked (usable key, live window, scope), in frame and at settlement,
	// but window transactions are charged only for what the execution costs
	// the account and it authorised: one for each action its code takes in
	// a frame where its own code had its own signature accepted (verified
	// in its frame, by a library or proxy implementation it DELEGATECALLed,
	// counted at once and withdrawn if the verifying frame reverts, or by a
	// validator it STATICCALLed, once that frame returns), or after a frame
	// of its code entered by a state-changing call verified it and returned
	// (a wallet that validates and executes in separate calls, ERC-4337);
	// an action beyond what the window has left fails in frame. At least one
	// is charged when it lost native value or granted a native allowance. A
	// signature accepted by a contract the account's code CALLed (a
	// callback), or by a read-only check, counts for no later action, so a
	// stranger replaying an old public signature there uses nothing. Before it every action was
	// charged, so anyone could use up the window of an account whose
	// delegated code calls out on an incoming call by calling it repeatedly.
	FeatureEVMDelegatedActorChargeOnCost = "evm-delegated-actor-charge-on-cost"

	// FeatureEVMWindowValidators adds the optional validators list to
	// MsgOpenEVMWindow: the contracts whose check of the account's signature
	// counts for the account's actions under evm-delegated-actor-charge-on-cost
	// (a validator module its code STATICCALLs, or an ERC-4337 entry point
	// that validates and then executes it). Only this list counts, not
	// allowed_targets. A check by an unnamed contract counts for nothing.
	// Below the switch a request with validators is refused.
	FeatureEVMWindowValidators = "evm-window-validators"

	// FeatureEVMPrecompileAuthorityRefused refuses, for an account that only
	// the EVM authorisation window authorises (the EVM-lane sender, or a
	// delegated account's code), the standard precompile methods that change
	// a lasting authority or position of the account rather than move value
	// it can see: the distribution withdraw address; creating or editing a
	// validator; undelegating; redelegating (admitted from
	// evm-window-staking-validators only to a validator the window names);
	// and governance votes, weighted votes and proposal cancellation. A stolen classical key inside a window could
	// otherwise redirect every future reward, move the whole stake to a
	// validator of its choosing, or vote with it. They stay available as
	// post-quantum signed Cosmos messages; claiming rewards stays available
	// through a window that lists the precompile.
	FeatureEVMPrecompileAuthorityRefused = "evm-precompile-authority-refused"

	// FeatureEVMWindowStakingValidators adds the optional staking_validators
	// list to MsgOpenEVMWindow, set with the post-quantum key: delegate and
	// redelegate through the window (the EVM-lane sender, or a delegated
	// account's code, calling the staking precompile) are admitted only to a
	// validator it names; with the list empty both are refused through the
	// window. They stay available as post-quantum signed Cosmos messages. The
	// account's current delegations grant nothing: a dust delegation made
	// through the window cannot make a validator "its own". Below the switch
	// a request with staking_validators is refused.
	FeatureEVMWindowStakingValidators = "evm-window-staking-validators"

	// FeatureEVMPrecompileMethodAllowlist makes the standard precompiles
	// (staking, distribution, ICS-20, gov, slashing, the ERC-20 token
	// precompiles and any other non-QoreChain stateful precompile)
	// deny-by-default, per method, for an account that only the EVM
	// authorisation window authorises: the EVM-lane sender, or a delegated
	// account's code. Only these run through the window:
	//   - distribution claimRewards, withdrawDelegatorRewards and (for the
	//     account's own validator) withdrawValidatorCommission, when the
	//     rewards are paid to the account itself;
	//   - staking delegate and redelegate to a validator the window names in
	//     staking_validators, and distribution depositValidatorRewardsPool
	//     to such a validator in native QOR only;
	//   - ICS-20 transfer of native QOR, without a memo, over the transfer
	//     port, to a "<source channel>/<receiver>" the window names in
	//     ibc_receivers (a new optional window field);
	//   - ERC-20 precompile transfer, transferFrom and approve of native QOR;
	//     of another denomination only when the window lists that token
	//     precompile in allowed_targets.
	// Native QOR they move stays within the window's value. Every other
	// transaction method is refused through the window and stays available as
	// a post-quantum signed Cosmos message. Who the window binds is decided
	// from the execution's sender (a module-authorised sender is not bound)
	// and from the caller's EIP-7702 delegation, which always binds, never
	// from the EVM origin. A contract acting for itself is unaffected. Below
	// the switch a request with ibc_receivers is refused.
	FeatureEVMPrecompileMethodAllowlist = "evm-precompile-method-allowlist"
)

// V328EVMAuthzFeatures are the v3.2.8 switches for EVM-lane authorisation (delegated actors in frame and in scope, ecrecover, usable keys, native allowances), consent messages being the account's own, and the Prague precompiles.
var V328EVMAuthzFeatures = []string{
	FeatureEVMDelegatedActorInFrame,
	FeatureEVMDelegatedActorScope,
	FeatureEVMEcrecoverWindow,
	FeaturePQCRegistrationEndsEVMConsent,
	FeatureEVMWindowUsableKey,
	FeatureEVMNativeAllowanceWindow,
	FeaturePQCConsentOwnMessage,
	FeatureEVMPraguePrecompiles,
	FeatureEVMDelegatedActorChargeOnCost,
	FeatureEVMWindowValidators,
	FeatureEVMPrecompileAuthorityRefused,
	FeatureEVMWindowStakingValidators,
	FeatureEVMPrecompileMethodAllowlist,
}
