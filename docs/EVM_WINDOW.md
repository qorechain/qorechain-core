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
  takes after its own code had its signature accepted in that frame (a wallet executing a signed
  batch, checking the signature itself, through a library it DELEGATECALLs or through a validator
  it STATICCALLs, in a frame that did not revert; an action beyond what the window has left
  fails), and at least one when it lost QOR or granted a native allowance, plus the QOR it lost.
  A stranger calling its code uses nothing, even after replaying an old signature of it in another
  call or in a callback the account's code made.

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

**Known limit.** The chain sees signature checks made through the ecrecover precompile only. A
contract that verifies secp256k1 signatures with its own arithmetic (bytecode elliptic-curve code,
no precompile) is outside what the chain can see, and a classical signature it accepts is not
bounded by the window. Whoever chooses such a contract (a token issuer, a wallet) chooses that
trust; the precompile path, which every standard library uses, is covered. No standard or
QoreChain precompile verifies secp256k1 signatures natively, so there is no second precompile path.

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
