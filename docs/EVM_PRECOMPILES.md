# QoreChain EVM Precompiles

QoreChain extends the EVM runtime with six QoreChain precompiled contracts (post-quantum signature
checks, AI screening, consensus parameters and the cross-VM bridge) and carries the standard
cosmos/evm precompiles (staking, distribution, IBC transfer, bank, governance, slashing, bech32,
P-256).

> **Status (2026-10-25):** no precompile beyond the standard Ethereum ones is active on mainnet
> (qorechain-vladi) or on the testnet (qorechain-diana). Each one runs only after governance lists
> its address in the x/vm parameter `active_static_precompiles`, which is empty on both networks
> (`GET /cosmos/evm/vm/v1/params`). On the testnet, governance is expected to add `pqcVerify`,
> `pqcKeyStatus` and `rlConsensusParams` right after the v3.2.5 upgrade. A call to an inactive
> address succeeds with empty return data; see [Availability](#availability-build-and-network-status).

## Address table

| Address (as listed in `active_static_precompiles`) | Name | Gas (precompile itself) | Kind |
|---|---|---|---|
| `0x0000000000000000000000000000000000000901` | CrossVM Bridge | 50,000 + 4/byte + callee | QoreChain |
| `0x0000000000000000000000000000000000000a01` | pqcVerify | 25,000 + 8/byte of input | QoreChain |
| `0x0000000000000000000000000000000000000a02` | pqcKeyStatus | 2,500 flat | QoreChain |
| `0x0000000000000000000000000000000000000B01` | aiRiskScore | 50,000 flat | QoreChain |
| `0x0000000000000000000000000000000000000B02` | aiAnomalyCheck | 40,000 flat | QoreChain |
| `0x0000000000000000000000000000000000000C01` | rlConsensusParams | 1,500 flat | QoreChain |
| `0x0000000000000000000000000000000000000800` | staking | cosmos/evm | standard, moves funds |
| `0x0000000000000000000000000000000000000801` | distribution | cosmos/evm | standard, moves funds |
| `0x0000000000000000000000000000000000000802` | ics20 (IBC transfer) | cosmos/evm | standard, moves funds |
| `0x0000000000000000000000000000000000000804` | bank | cosmos/evm | standard, read-only |
| `0x0000000000000000000000000000000000000805` | gov | cosmos/evm | standard, moves funds |
| `0x0000000000000000000000000000000000000806` | slashing | cosmos/evm | standard |
| `0x0000000000000000000000000000000000000100` | p256 | cosmos/evm | stateless |
| `0x0000000000000000000000000000000000000400` | bech32 | 6,000 base | stateless |

The address column is the exact string x/vm matches (see [Activation](#activation-by-governance)):
note the lower-case `a` in `0a01`/`0a02` and the upper-case `B` and `C` in `0B01`, `0B02`, `0C01`.
Solidity accepts any of these as `address(0x0A01)` and so on; only the governance parameter is
case-sensitive.

## PQC precompiles

### pqcVerify (0x0A01)

Verifies an ML-DSA-87 (FIPS 204, also called Dilithium-5) signature.

```solidity
function pqcVerify(bytes pubkey, bytes signature, bytes message) view returns (bool valid)
```

- `pubkey`: an ML-DSA-87 public key, 2,592 bytes.
- `signature`: an ML-DSA-87 signature, 4,627 bytes.
- `message`: the signed bytes, verified as given (no prehash, empty context string). Hedged and
  deterministic signatures both verify; deterministic signing is a client default, not a rule.

**Returns** `true` only for a valid signature. Every other case returns `false` and does not revert:
an invalid signature, the wrong key, the wrong message, a key or signature of another size, and
calldata that does not decode as `(bytes, bytes, bytes)`. The 4-byte selector is not checked, so
the same arguments behind another selector are verified the same way; call it through the interface
and do not rely on a wrong selector failing.

**Gas:** `25,000 + 8 * len(input)`, where input is the whole calldata including the selector. With a
2,592-byte key, a 4,627-byte signature and a short message the input is about 7,500 bytes, so the
precompile charges about 85,000 gas. Measured on a v3.2.5 devnet (42-byte message):

| Path | Gas |
|---|---|
| precompile charge (`RequiredGas`, 7,492-byte input) | 84,936 |
| from a contract, including ABI encoding and the copy of key and signature | 89,900 to 91,750 |
| a whole transaction to a contract that verifies once (`eth_estimateGas`) | about 313,000 |
| a transaction sent straight to `0x0A01` (`eth_estimateGas`) | about 312,700 |

Most of a transaction's gas is the calldata of the key and the signature, not the verification. A
contract that verifies against a key it already stores or reads from `pqcKeyStatus` sends only the
signature.

**Call types:** the precompile reads nothing and writes nothing, so `STATICCALL`, `CALL` and
`DELEGATECALL` all answer the same, and a revert in the caller after the call leaves nothing behind.
A call given less gas than the charge fails (`success == false`) and uses the gas it was given; the
calling frame continues.

**Replay:** the precompile checks a signature over bytes, nothing more. A contract must bind what it
accepts to the context it accepts it in (chain id, its own address, a nonce or deadline), or a
signature valid once is valid everywhere.

### pqcKeyStatus (0x0A02)

```solidity
function pqcKeyStatus(address account) view returns (bool registered, uint8 algorithmId, bytes pubkey, bool possessionProven)
```

- `account`: an EVM address; the precompile looks up the same account in its `qor1` form.
- `registered`: whether the account has a post-quantum signature key.
- `algorithmId`: `1` for ML-DSA-87.
- `pubkey`: the registered public key (2,592 bytes for ML-DSA-87); empty when not registered.
- `possessionProven` (from v3.2.4, `pqc-possession-flag`): `true` when the current key was proven
  by a proof of possession, that is a registration carrying a valid `proof_of_possession` (the CLI
  `register-key-v2 ... --pqc-key <name>` adds one), or a key rotation or migration, whose
  signatures by the new key are a proof of possession. Keys registered before v3.2.4, or without a
  proof, report `false` until the account rotates its key.

An unregistered account returns `(false, 0, "", false)`. The flag is appended after the dynamic
`pubkey` field, so a caller that decodes `(bool, uint8, bytes)` still reads the first three values
unchanged.

**Gas:** 2,500 flat. Measured: a transaction straight to `0x0A02` about 23,900; inside a contract,
including decoding the returned key, about 7,100.

> **Security: what a contract may rely on.** A registered key is not proof that the account holds
> its secret. Anyone can register someone else's public key for their own account without a proof
> of possession; the chain allows it because a copied key only harms the account that registers
> it. A contract that verifies "a signature by account X" against `pqcKeyStatus(X).pubkey` must
> therefore require `possessionProven`, or a copied key lets one address pass as another. The
> devnet test registered a victim's public key on a second account: `pqcVerify` with the second
> account's registered key accepted the victim's signature, and the pattern below refused it.

```solidity
interface IQorePQC {
    function pqcVerify(bytes calldata pubkey, bytes calldata signature, bytes calldata message) external view returns (bool);
    function pqcKeyStatus(address account) external view returns (bool, uint8, bytes memory, bool);
}

IQorePQC constant PQC_VERIFY = IQorePQC(address(0x0A01));
IQorePQC constant PQC_KEYSTATUS = IQorePQC(address(0x0A02));

/// True only for a valid ML-DSA-87 signature by `who` over `message`, with the key the chain
/// holds for `who`, and only if `who` proved possession of that key.
function verifyFor(address who, bytes calldata signature, bytes calldata message) view returns (bool) {
    (bool registered, uint8 algo, bytes memory pub, bool proven) = PQC_KEYSTATUS.pqcKeyStatus(who);
    require(registered, "no PQC key");
    require(algo == 1, "not ML-DSA-87");
    require(proven, "key possession not proven");
    return PQC_VERIFY.pqcVerify(pub, signature, message);
}
```

A high-level call to an inactive precompile reverts (Solidity checks that the target has code), which
is the safe outcome. A low-level `staticcall` to an inactive one succeeds with empty data; decode it
with `abi.decode`, which reverts on empty data, never by hand.

## AI precompiles

> **Determinism:** both AI precompiles compute their answer on chain from fixed rules, with the same
> result on every node. They never call the AI sidecar, and neither writes any state, whether it is
> reached by `CALL` or `STATICCALL`.

### aiRiskScore (0x0B01)

A minimal, length-only heuristic. It is not a security analysis of the bytes it is given, and a
contract must not treat its answer as one.

```solidity
function aiRiskScore(bytes txData) view returns (uint256 score, uint8 level)
```

| Input | `score` | `level` |
|-------|---------|---------|
| empty bytes | 8000 | 3 (HIGH) |
| more than 500,000 bytes | 5000 | 2 (MEDIUM) |
| any other length | 0 | 1 (LOW) |
| calldata that does not ABI-decode as `(bytes)` | 10000 | 4 (CRITICAL, fail closed) |

`score` is in basis points (0 to 10000); `level` 0=SAFE, 1=LOW, 2=MEDIUM, 3=HIGH, 4=CRITICAL.
Level 0 is never returned, level 4 only for undecodable calldata. **Gas:** 50,000 flat.

### aiAnomalyCheck (0x0B02)

Returns the verdict the chain's transaction screening (x/ai) would give a transfer of `amount` from
`sender` in the current block.

```solidity
function aiAnomalyCheck(address sender, uint256 amount) view returns (uint256 anomalyScore, bool flagged)
```

- `amount` is in `uqor` (6 decimals); values above 2^64-1 are clamped.
- From v3.2.5 (`evm-ai-anomaly-check-v2`) it applies the same deterministic rules the ante handler
  applies, over the account's stored screening state, read-only: the last 32 screened transactions
  and the count in the 30-block rate window; half the EVM-lane rate limit or more in the window
  scores 0.6; an amount more than three standard deviations from the account's mean (at least three
  earlier amounts) scores deviation/5, at most 1.0; a transfer above 1,000 QOR from an account with
  fewer than three screened transactions scores 0.5. The highest rule wins.
- `flagged` is true above 5000. Malformed calldata returns `(10000, true)`.

**Gas:** 40,000 flat.

## rlConsensusParams (0x0C01)

```solidity
function rlConsensusParams() view returns (uint256 blockTimeMs, uint256 baseGasPrice, uint256 validatorSetSize, uint256 epoch)
```

Today the full build wires the static provider, so the answer is fixed: `(5000, 100, 100, 0)`
whatever the chain's real block time, gas price and validator set. It is a placeholder for the RL
module's tuned values and must not be used as a measurement of the network. **Gas:** 1,500 flat
(a transaction straight to it: about 22,600).

## CrossVM bridge (0x0901)

Synchronous calls from EVM contracts to reviewed CosmWasm contracts. Canonical ABI:
`contracts/abi/IQoreCrossVM.json`; Solidity interface: `contracts/interfaces/IQoreCrossVM.sol`.

```solidity
function executeCrossVMCall(uint8 targetVM, string targetContract, bytes payload) returns (bytes result)
```

- Selector `0x3c11078c`; the two-argument form `executeCrossVMCall(string,bytes)` is refused.
- `targetVM` must be `1` (CosmWasm); `targetContract` a `qor1...` contract listed in the x/crossvm
  parameter `bridge_allowed_targets` as `cosmwasm:<address>`; `payload` the JSON execute message.
- Returns the contract response's `data`, ABI-encoded as `bytes`. Refusals revert with `Error(string)`.
- Plain `CALL` only, no value; the caller of `0x0901` is the CosmWasm sender; a call reached through
  the bridge cannot enter the EVM again; the call honours the `MsgCrossVMCall` circuit-breaker entry.
- **Gas:** 50,000 + 4 per input byte, plus the CosmWasm execution, metered against the calling frame.

These rules apply from the v3.2.5 switches `crossvm-bridge-abi` and `crossvm-lane-bounds`.

## Standard precompiles (staking, distribution, ics20, bank, gov, slashing)

These are the cosmos/evm v0.6 precompiles with their upstream ABIs (`precompiles/<name>/abi.json`
in cosmos/evm). They are compiled into the full build and inactive on both networks. They act for
`msg.sender`: the account whose funds move is the caller of the precompile (an EOA calling it
directly, a contract, or an EIP-7702 delegated account whose code calls it). Amounts are in `uqor`
(6 decimals), not in the 18-decimal unit of `msg.value`.

They execute Cosmos messages by calling the module message servers directly, past the ante handler
and the message router where the circuit breaker is checked. From v3.2.5 two protocol features close
what that bypassed:

- **`evm-precompile-circuit`:** every transaction method refuses, with a revert reason, while the
  circuit-breaker entry of the Cosmos message it executes is tripped (`delegate` honours
  `/cosmos.staking.v1beta1.MsgDelegate`, ics20 `transfer` honours
  `/ibc.applications.transfer.v1.MsgTransfer`, `claimRewards` honours `MsgWithdrawDelegatorReward`,
  and so on for every method). Queries are unaffected.
- **`evm-precompile-window-scope`:** the value-moving ones (staking, distribution, ics20, gov) run
  for an account that the EVM authorisation window authorises only when that account's window lists
  the precompile in `allowed_targets`. See [EVM_WINDOW.md](./EVM_WINDOW.md#value-moving-precompiles).

The bank precompile is read-only (`balances`, `totalSupply`, `supplyOf`) and slashing can only
unjail the caller's own validator, so neither needs a window listing.

## Activation by governance

A precompile is activated by a governance proposal carrying x/vm `MsgUpdateParams`
(`/cosmos.evm.vm.v1.MsgUpdateParams`, authority = the gov module account). Three things to get right:

1. **`MsgUpdateParams` replaces every x/vm parameter.** Copy all current parameters from the chain
   and change only `active_static_precompiles`; keep the precompiles already active.
2. **Exact checksum spelling.** x/vm activates a precompile by exact string comparison with the
   EIP-55 form of its address, the column in the table above. `0x...0A01` passes validation and the
   vote, and `0x...0a01` then stays inactive (shown on the devnet with `0x...0b01`).
3. **Byte order.** The list must be sorted by byte value, so upper-case hex sorts before lower-case:
   `0x...0901`, `0x...0B01`, `0x...0C01`, `0x...0a01`, `0x...0a02`. A case-insensitive sort fails
   validation when the proposal is submitted.

The script used for the testnet does all three from the live parameters and refuses to build a
proposal for a precompile whose protocol features are not enabled yet.

## Availability: build and network status

- **Community build** (default `go build`, no tag): the QoreChain and standard stateful precompiles
  are not in the binary; the EVM runs with the standard Ethereum precompiles only.
- **Full build** (`-tags full`, what the validators run): the implementations are compiled in, and a
  precompile runs once governance adds its address to `active_static_precompiles`.

**A call to an address that is not an active precompile succeeds and returns empty data**, as for any
address with no code. Never read empty return data as an answer: check that the address is listed in
`active_static_precompiles`, and treat a call that returns no data as "not available".

**Gas billing on QoreChain:** the fee market's `min_gas_multiplier` is 0.5 on both networks, so a
transaction pays for at least half of its gas limit whatever it uses. Set gas limits close to the
estimate.

## Solidity interfaces

- `contracts/interfaces/IQorePQC.sol`
- `contracts/interfaces/IQoreAI.sol`
- `contracts/interfaces/IQoreConsensus.sol`
- `contracts/interfaces/IQoreCrossVM.sol` (ABI: `contracts/abi/IQoreCrossVM.json`)

See also [EVM_WINDOW.md](./EVM_WINDOW.md) for the EVM authorisation window,
[PQC_KEM_REGISTRY.md](./PQC_KEM_REGISTRY.md) for published encryption keys, and
[CROSSVM.md](./CROSSVM.md) for the cross-VM model.
