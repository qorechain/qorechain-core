# The EVM authorisation window

QoreChain requires an ML-DSA-87 (post-quantum) signature on every Cosmos-lane transaction. An
Ethereum transaction cannot carry one: MetaMask and every EVM tool sign secp256k1 only. So the
post-quantum authorisation travels ahead of the transactions instead. The account opens a bounded
**window** with a Cosmos-lane message, signed with its post-quantum key, and the EVM lane then admits
the account's transactions only while the window covers them. A stolen classical (secp256k1) key is
worth only what an open window admits.

This page is the policy: what a window bounds, what it counts, how to narrow it, and what is outside
it. The step-by-step for deploying contracts is in [EVM_DEPLOY.md](./EVM_DEPLOY.md).

## Opening, closing, reading

| Action | Message | CLI |
|---|---|---|
| open (replaces any open window) | `/qorechain.pqc.v1.MsgOpenEVMWindow` | `qorechaind tx pqc open-evm-window [blocks] [max-txs] [max-value]` |
| close at once | `/qorechain.pqc.v1.MsgCloseEVMWindow` | `qorechaind tx pqc close-evm-window` |
| read | `GET /qorechain/pqc/v1/evm_window/{address}` | `qorechaind q pqc evm-window <address>` |

Both messages are Cosmos-lane transactions and need the account's hybrid (classical plus ML-DSA-87)
signature. The account must hold a registered post-quantum key to open a window. Closing a window
also withdraws a pending EIP-7702 delegation consent.

## What a window bounds

Every bound must be given (the CLI fills defaults for a deploy session: 3,600 blocks, 10
transactions, 25 QOR):

| Bound | Limit | |
|---|---|---|
| `blocks` | at most 17,280 | the window admits nothing after `opened height + blocks` |
| `max_txs` | at most 1,000 | how many EVM transactions it admits |
| `max_value` | positive | total uqor its transactions may take out of the account (below) |

Any bound running out ends the window in practice.

## What the window counts

**At admission** (the EVM-lane ante handler, before the transaction runs), each transaction uses one
of `max_txs` and adds to the used value:

- the value it transfers, plus
- its maximum fee: gas limit times fee cap (the gas price of a legacy transaction),

converted from the 18-decimal EVM unit to uqor and rounded up. The fee is counted at its maximum
because whoever holds the classical key chooses the gas price. A transaction that is admitted and
then reverts has still used its share: the authorisation was given.

**After execution** (from v3.2.2, `evm-delegated-actor-settlement`), before anything is committed:

- if the sender's native balance fell by more than the value it declared, the difference is charged
  to the sender's window, and the execution fails when the window does not cover it. This is how QOR
  moved by a precompile (a staking delegation, a governance deposit, an IBC transfer of uqor), which
  carries a transaction value of zero, still counts against the value bound;
- an EIP-7702 delegated account whose code acts in the transaction (calls code or a stateful
  precompile, creates a contract, or loses value) needs its own live window, which is charged one
  transaction and its outflow.

**Not counted:** `eth_call`, `eth_estimateGas` and simulation run no window check and use none of it.
Tokens are not counted at all: an ERC-20 transfer or approval carries value zero and a token cannot be
priced in uqor. Bound those with a scope.

## Usage event (from v3.2.7, `evm-window-used-event`)

Every charge to a window emits one `pqc_evm_window_used` event in the transaction's result, so a
wallet can show what a transaction took without querying the window before and after. There are two
kinds of charge, told apart by `source`:

- `ante`: the EVM-lane ante handler admitted the transaction against the sender's window. This event
  is in the result whether the transaction then succeeds or reverts, the same as the charge.
- `settlement`: after a successful execution, the delegated-actor settlement (above) charged a
  window: the sender's for value it lost beyond what it declared, or a delegated account's that
  acted. One event per window charged. It is in the result only when the execution's state is
  committed, again the same as the charge.

A transaction can therefore carry an `ante` event and one or more `settlement` events, possibly for
the same account. A refused transaction, `eth_call`, `eth_estimateGas` and simulation emit none.

| Attribute | Value |
|---|---|
| `address` | the account in `qor1...` form, the form the window is queried by |
| `evm_address` | the same account in `0x` form (checksummed) |
| `window_open_height` | the height the window was opened at |
| `used_txs` | transactions the window has admitted, after this charge |
| `used_value` | uqor the window has counted, after this charge (decimal string) |
| `tx_value` | uqor this charge added to `used_value` (decimal string) |
| `source` | `ante` or `settlement` |

`tx_value` is always the difference in `used_value` before and after the charge. For `ante` it is
the transaction's value plus its maximum fee, rounded up, not the bare value it transfers: the fee is
counted at its maximum (above). For `settlement` it is the value the account lost, beyond the
declared value for the sender, rounded up; it can be 0 for a delegated account that acted without
losing value, which still uses one transaction. A settlement charge to the sender leaves `used_txs`
unchanged, because the ante handler already counted the transaction.

The other window events are `pqc_evm_window_opened` (`address`, `expiry_height`, `max_txs`,
`max_value`, and `native_only` / `allowed_targets` when set) and `pqc_evm_window_closed`
(`address`).

## Narrowing the scope (from v3.2.3, `evm-pqc-window-targets`)

| Option | Admits | Refuses |
|---|---|---|
| no scope | any transaction within the bounds, except a value-moving precompile (below) | |
| `--native-only` | plain QOR transfers | calldata, contract creation, a recipient carrying code, EIP-7702 authorizations |
| `--allowed-targets a,b,...` (at most 16) | transactions to the listed addresses | any other recipient, contract creation |

A scope is the only bound on what a contract call does with tokens: a window listing one token
contract can move that token and nothing else.

## Value-moving precompiles

From v3.2.5 (`evm-precompile-window-scope`). The standard precompiles below act on their caller's
funds through the Cosmos modules with a transaction value of zero, and several of their actions move
no native balance at all: an undelegation, a redelegation, a new withdraw address that diverts every
future reward, a deposit or transfer in a denomination other than uqor. The value bound cannot price
those, so they are bounded by where the account may go:

| Precompile | Address |
|---|---|
| staking | `0x0000000000000000000000000000000000000800` |
| distribution | `0x0000000000000000000000000000000000000801` |
| ics20 (IBC transfer) | `0x0000000000000000000000000000000000000802` |
| gov | `0x0000000000000000000000000000000000000805` |

The rule:

- A transaction method of one of them runs for an account the window authorises only when that
  account's window lists the precompile in `allowed_targets` (and is not `--native-only`). An
  unscoped window cannot reach them.
- That account is the sender of the EVM transaction when it calls the precompile itself or through
  its own EIP-7702 code, and any EIP-7702 delegated account whose code calls the precompile in
  someone else's transaction (its own window must list it).
- A transaction addressed straight to one of them from a window that does not list it is refused
  before it enters a block. A call reached through a contract is refused by the precompile, as a
  revert with a reason.
- The QOR they move is still charged to the window's value by the settlement above, so a listing
  window bounds both where the account goes and how much it spends there.
- A contract that calls them acts with its own funds (the precompiles act for `msg.sender`), like any
  other contract call the window admits.
- Queries are never affected, and the bank precompile (read-only) and slashing (unjail only) need no
  listing.

Separately, from v3.2.5 (`evm-precompile-circuit`), every transaction method of every standard
precompile refuses while the circuit-breaker entry of the Cosmos message it executes is tripped.

Example: delegate from MetaMask through the staking precompile, with a window that allows it and
nothing else:

```bash
qorechaind tx pqc open-evm-window 600 5 100qor \
  --allowed-targets 0x0000000000000000000000000000000000000800 \
  --from me --generate-only > w.json
qorechaind tx pqc cosign w.json --from me --pqc-key me-mldsa --chain-id $QCHAIN --node $QNODE
```

The standard precompiles are inactive on both networks until governance activates them.

### Deny by default, per method (from v3.2.8, `evm-precompile-method-allowlist`)

For an account the window authorises (the EVM-lane sender, or a delegated account's code), every
transaction method of a standard precompile (staking, distribution, ICS-20, gov, slashing, the
ERC-20 token precompiles, and any other non-QoreChain stateful precompile) is refused through the
window unless it is one of these, and the window lists the precompile in `--allowed-targets`:

| Precompile | Method | Admitted when |
|---|---|---|
| distribution | `claimRewards`, `withdrawDelegatorRewards` | the rewards are paid to the account itself (its withdraw address is its own) |
| distribution | `withdrawValidatorCommission` | the validator is the account's own and pays the account itself |
| distribution | `depositValidatorRewardsPool` | the validator is named in `--staking-validators` and every coin is native QOR |
| staking | `delegate`, `redelegate` | the destination validator is named in `--staking-validators` |
| ics20 | `transfer` | native QOR, over the `transfer` port, with no memo, to a `<source channel>/<receiver>` named in `--ibc-receivers` |
| ERC-20 token precompiles | `transfer`, `transferFrom`, `approve` | native QOR; another token only when the window lists that token's address in `--allowed-targets` |

Everything else is refused through the window: setting the withdraw address, funding the community
pool, undelegating, cancelling an unbonding, creating or editing a validator, unjailing, submitting,
depositing on, voting on or cancelling a proposal, and any method added upstream later. All of them
stay available as post-quantum signed Cosmos transactions. Native QOR the admitted methods move is
charged to the window's value. A contract acting for itself is unaffected.

A stateful precompile reached inside a view call (`STATICCALL`) runs read-only
(`evm-precompile-static-readonly`), and for a window-bound account the guard also takes the same
decision as for any call the account's code makes: a usable key, a live window with a transaction
left, and the scope.

Who the window binds is decided from the transaction's sender and the caller's own EIP-7702
delegation, never from the EVM origin: a delegated account is bound by its own window wherever its
code runs, including inside an EVM run a precompile starts for it, and a sender a module
authorised (a post-quantum signed cross-VM call) is not.

**IBC receivers match exactly.** The channel and the receiver are compared byte for byte, case
included: write the channel as the chain shows it (`channel-0`) and the receiver as its lowercase
bech32 address, exactly as the transfer will carry it.

**Why `--ibc-receivers` rather than refusing IBC.** Bridging QOR out from an EVM wallet is a common
need, so it stays possible, but only to destinations the post-quantum key chose: an exact channel
and receiver, native QOR only (its value counts against the window), and no memo, since a memo can
make the other chain pass the coins on. Listing ICS-20 without `--ibc-receivers` admits no transfer.

**Wallet guidance.**

- Listing a precompile or a contract in `--allowed-targets` is the account's consent to what that
  contract can do with its assets. For ICS-20 that consent is now narrowed to the named receivers;
  for any other listed contract that holds the account's tokens (a vault, a router, a token), the
  window does not bound which tokens move or where.
- To claim staking rewards from MetaMask: list the distribution precompile, keep the withdraw
  address as the account itself.
- To stake from MetaMask: list the staking precompile and name the validators.
- To bridge QOR out: list the ICS-20 precompile and name the exact `channel/receiver` pairs.
- To move an IBC token held as an ERC-20 precompile: list that token's address.
- Anything else (undelegating, voting, governance deposits, changing the withdraw address): send a
  post-quantum signed Cosmos transaction.

## Delegated accounts, signatures and allowances (from v3.2.8)

- **Decided when the account acts** (`evm-delegated-actor-in-frame`). An EIP-7702 delegated account
  other than the sender needs a usable post-quantum key and a live window at the moment its code
  makes a CALL to code or a stateful precompile, creates a contract, sends QOR (a CALL with value, a
  SELFDESTRUCT) or is debited by a precompile. Without one only that call fails, as a failed call
  on Ethereum: the code that called the account sees the failure and carries on, so a payment to a
  delegated account can no longer make the payer's whole transaction revert. A STATICCALL, a
  DELEGATECALL and a CALLCODE are not actions in themselves.
- **What a delegated account is charged** (`evm-delegated-actor-charge-on-cost`). In a transaction it
  did not send, its actions are always checked (key, live window, scope), but it is charged window
  transactions only for what the transaction costs it and it authorised: one per action its code
  takes after its own code had its signature accepted (a wallet executing a signed batch, checking
  the signature itself or through a proxy implementation or library it DELEGATECALLs; through a
  validator module it STATICCALLs, when the window names that module; a wallet that validates and
  executes in separate calls, ERC-4337, when the window names the entry point; an action beyond
  what the window has left fails), and at least one when it lost QOR or granted a native
  allowance, plus the QOR it lost. A stranger calling its code uses nothing, even after replaying
  an old signature of it in a check of its own (by CALL or STATICCALL), in a read-only call the
  account's code makes to the stranger, or in a callback.
- **Validators** (`evm-window-validators`, `--validators a,b,...`, at most 16). The contracts whose
  check of the account's signature counts for its actions: a validator module the wallet
  STATICCALLs, or the ERC-4337 entry point that calls the wallet. Only this list counts; a contract
  listed in `--allowed-targets` (a router, a relay) is not thereby a validator. Validators do not
  narrow where the account's transactions go.
- **What offsets what the account sends.** Only QOR the transaction's sender pays the account
  (the transaction's own value, or a payment by the sender's own code) offsets what it then sends,
  so it can forward what the sender paid it. QOR any contract or other account pays it (a vault
  withdrawal or an unwrap, a refund, an escrow, bridge or withdrawal queue that anyone may
  finalise, a contract that SELFDESTRUCTs to it) and QOR a precompile pays it (a staking reward
  claim, a module payout) do not: a holder of the classical key cannot move the account's
  positions out past the window's value, whoever triggers the payout. Forwarding such a payment in
  the same transaction uses window value.
- **Standard precompiles** (`evm-precompile-method-allowlist`): deny by default per method, see
  [Deny by default, per method](#deny-by-default-per-method-from-v328-evm-precompile-method-allowlist).
  The two items below are the narrower rules it replaces.
- **Lasting authority changes** (`evm-precompile-authority-refused`). Setting the distribution
  withdraw address, creating or editing a validator, undelegating, and governance votes, weighted
  votes and proposal cancellation are refused through the EVM window whatever it lists; send them
  as post-quantum signed Cosmos transactions. Claiming rewards stays available through a window
  that lists the precompile.
- **Staking validators** (`evm-window-staking-validators`, `--staking-validators v1,v2,...`,
  bech32 operator addresses, at most 16). Delegating and redelegating through the window (the
  account's EVM transactions or its EIP-7702 code calling the staking precompile, which the window
  must also list in `--allowed-targets`) go only to a validator this list names; with the list
  empty both are refused through the window and stay available as post-quantum signed Cosmos
  transactions. The account's existing delegations grant nothing, so a classical key cannot make
  a validator "its own" with a dust delegation and then move the whole stake there.
- **Simulations match.** `eth_call` and `eth_estimateGas` (and a module's read-only EVM call) run
  the same in-frame rules and the settlement's decision for every account other than the sender,
  and record nothing, so a preview fails where the transaction would. The sender is taken as
  authorised: a wallet can estimate a transaction before opening the window that will pay for it;
  delivery then needs that window. A module's EVM call made inside a precompile (an ERC-20 balance
  query during an IBC transfer) is part of the transaction that reached the precompile and is
  judged with it, under its sender.

**Known limit: token approvals outlive the window.** An ERC-20 `approve`, a permit, Permit2 or
`setApprovalForAll` granted inside a window (by the account, or by a holder of its classical key
within the window's scope) stays after the window closes, expires or is revoked: the chain cannot
price contract tokens or undo another contract's records. Native QOR allowances are charged as
outflow. When a window ends, review and revoke the approvals it granted (a wallet's approval
manager or a revoke tool), and keep `--allowed-targets` to the contracts you mean to approve.

**Known limit: modular wallets and the transaction count.** An account is charged window
transactions only for actions taken under a signature check the chain recognises as the account's
own (see above). A wallet whose validator module is reached with CALL rather than STATICCALL, or
is not named in `--validators`, produces no such mark: actions through it are checked (live
window, scope, value) but not counted. Wallet integrators: name the validator module in
`--validators`.

**Known limit: verifiers without a nonce.** A wallet whose own public multicall runs its
signature check without a nonce and then a public action, in one call, lets a stranger replay an
old public signature and have that action counted against the window's transactions (never its
value bound). Wallet verifiers must consume a nonce; this is the wallet's responsibility.

**Known limit: what the transaction count does not bound.** The count applies only to the
designs above. Actions taken under a signature checked by a validator module the wallet reaches
with CALL (ERC-7579 modules), by a trusted forwarder (ERC-2771), or by the wallet's own
arithmetic are checked (live window, scope) but not counted; at least one transaction is charged
when native QOR is lost, and the value bound always holds. For such wallets name the validators
you trust (`--validators`), or open scoped (`--allowed-targets`) short windows: the scope, not
the count, is what bounds tokens.

**Known limit: only `ecrecover` is seen.** The window gates signatures checked through the
ecrecover precompile (`0x01`). A wallet or contract that verifies secp256k1 in its own code, or
uses another scheme (P-256 at `0x100`, any other), is outside what the chain can tie to the
account: such signatures are not refused without a window and their actions are not counted.
Whoever chooses such a contract (a token issuer, a wallet) chooses that trust; the precompile
path, which every standard library uses, is covered. No standard or QoreChain precompile verifies
secp256k1 signatures natively, so there is no second precompile path.

**Known limit: forwarder and session-key wallets.** The transaction count bounds a holder of the
classical key only for wallets that verify the account's signature in the account's own frame (or
in a validator it STATICCALLs). A wallet that trusts a forwarder (ERC-2771 meta-transactions:
the forwarder verifies and then calls the account), or acts on an approval or session key stored
earlier, takes actions the chain cannot tie to a signature; they are checked (live window, scope,
value bound) but not counted. For an account delegated to such a wallet, open scoped windows
(`--allowed-targets`) with short expiries.
- **Scope applies to the account's own code** (`evm-delegated-actor-scope`). `--native-only` admits
  no call to code and no creation by the account's delegated code; `--allowed-targets` admits calls
  to the listed addresses only. This also holds for the sender's own delegated code.
- **Signatures** (`evm-ecrecover-window`). When a contract verifies a signature with the ecrecover
  precompile (`0x01`: EIP-2612 permits, Permit2, signed orders, meta-transactions) and the signer
  holds a usable post-quantum key, the signature counts only inside the signer's live window. It is
  never charged to the window (a signature is public once used; a charge would let anyone use the
  window up by replaying it). `--native-only` admits no signature for another contract;
  `--allowed-targets` must list the contract that verifies it, in the account's own transactions
  too. A delegated wallet checking its own signature, the sender's own signature under an unscoped
  window, and every account with no post-quantum record recover as always. An account whose record
  authorises nothing (empty, classical-only, degenerate) is treated as a key without a window: its
  signature is refused until it repairs the record and opens a window. Queries and gas
  estimates are not affected. Open a window before signing a permit, as before sending a
  transaction.
- **Native-QOR allowances** (`evm-native-allowance-window`). If a token pair for native QOR is ever
  registered, an approve that raises an allowance counts the increase against the owner's window
  (an amount no window can hold, such as `approve(max)`, fails the approve). Spending it later with
  `transferFrom` is not charged again, and the allowance stays usable after the owner's window
  expires, as an ERC-20 allowance does.
- **Usable key** (`evm-window-usable-key`). A window works only while the key is usable: not empty,
  classical-only or degenerate, and its algorithm not disabled by governance.


## What is outside the window

The window governs the EVM lane only, that is transactions signed with the classical key alone. These
are not window-governed, because each is a Cosmos-lane transaction carrying its own post-quantum
signature, or is started by a module for an account that such a transaction authorised:

- **`MsgCrossVMCall` from the native lane into the EVM.** The EVM call runs as the signer's own
  `0x` address, and the transaction carries the signer's hybrid signature, so the window is neither
  needed nor charged. The same holds for a CosmWasm contract dispatching `MsgCrossVMCall`.
- abstract-account executions (`MsgExecuteEVM`, authorised by the account's authenticators);
- ERC-20 conversions and IBC callbacks run by the modules;
- every Cosmos-lane message (bank sends, staking, governance) sent directly.

EIP-7702 delegations need a separate consent from the authority's own post-quantum key
(`authorize-evm-delegation`); the sender's window pays for the transaction, it does not speak for the
authorities.

## Refusals

| Message | Code (`pqc`) | Cause |
|---|---|---|
| `no open EVM authorisation window` | 26 | no window, no registered key, or the window expired |
| `EVM authorisation window exhausted` | 27 | a bound is used up, or the transaction is outside the window's scope, including a value-moving precompile the window does not list |
| `invalid EVM authorisation window` | 28 | an open request outside the limits |
| `delegated account has no window covering this action` | 62 | the settlement refused: the sender moved more than its window covers, or a delegated account acted without a window (from v3.2.8 the same rule fails only the offending call, in frame) |
| `<method> is disabled: the circuit breaker has tripped <message>` | revert | a standard precompile method whose message is tripped |
| `<method>: account ... has no open EVM authorisation window that lists 0x...` | revert | a value-moving precompile reached through code, for an account whose window does not list it |
