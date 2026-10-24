# QoreChain EVM Precompiles

QoreChain extends the EVM runtime with 6 custom precompiled contracts that expose PQC cryptography, AI risk analysis, RL consensus parameters, and cross-VM communication to Solidity smart contracts.

> **Status:** none of these precompiles is active on mainnet or testnet today. They are compiled into the full build only, and each one runs only after governance lists its address in the EVM parameter `active_static_precompiles`, which is empty on both networks. A call to any address below currently succeeds with empty return data. See [Availability](#availability-build-and-network-status) before using one.

## Precompile Address Table

| Address | Name | Gas Cost | Category |
|---------|------|----------|----------|
| `0x0000...0901` | CrossVM Bridge | 50,000 flat | Cross-VM |
| `0x0000...0A01` | PQC Verify | 25,000 + 8/byte | PQC |
| `0x0000...0A02` | PQC Key Status | 2,500 flat | PQC |
| `0x0000...0B01` | AI Risk Score | 50,000 flat | AI |
| `0x0000...0B02` | AI Anomaly Check | 40,000 flat | AI |
| `0x0000...0C01` | RL Consensus Params | 1,500 flat | Consensus |

## PQC Precompiles

### pqcVerify (0x0A01)

Verifies ML-DSA (Dilithium-5) post-quantum signatures on-chain.

**Solidity Signature:**
```solidity
function pqcVerify(bytes pubkey, bytes signature, bytes message) view returns (bool valid)
```

**Parameters:**
- `pubkey` (bytes): ML-DSA public key (2592 bytes for Dilithium-5)
- `signature` (bytes): ML-DSA signature (4627 bytes for Dilithium-5)
- `message` (bytes): The original signed message

**Returns:** `true` if the signature is valid

**Gas:** 25,000 base + 8 per input byte

**Example:**
```solidity
address constant PQC_VERIFY = address(0x0000000000000000000000000000000000000A01);

(bool success, bytes memory result) = PQC_VERIFY.staticcall(
    abi.encodeWithSignature("pqcVerify(bytes,bytes,bytes)", pubkey, sig, msg)
);
bool valid = abi.decode(result, (bool));
```

### pqcKeyStatus (0x0A02)

Queries whether an address has a PQC key registered on-chain.

**Solidity Signature:**
```solidity
function pqcKeyStatus(address account) view returns (bool registered, uint8 algorithmId, bytes pubkey, bool possessionProven)
```

From v3.2.4 (`pqc-possession-flag`) the output carries a fourth value,
`possessionProven`, appended after the dynamic `pubkey` field. The first three
values keep their encoding, so a caller that decodes `(bool, uint8, bytes)`
still reads them unchanged; below the switch the output is the three-value
form.

**Parameters:**
- `account` (address): EVM address to query (converted to bech32 internally)

**Returns:**
- `registered`: Whether the address has a PQC key
- `algorithmId`: Algorithm ID (1 = Dilithium-5, 2 = ML-KEM-1024)
- `pubkey`: The registered PQC public key bytes
- `possessionProven` (v3.2.4): Whether the current key was proven by a proof of possession (a registration with a valid `proof_of_possession`, or a rotation or migration)

> **Security:** a registered key is not proof that the account holds its secret. A key registered without a proof of possession may have been copied from someone else. `pqcKeyStatus` and `pqcVerify` must not be activated for contracts unless those contracts check `possessionProven`; a contract must never accept a `pqcVerify` result as the account's signature for a key whose `possessionProven` is false.

**Gas:** 2,500 flat

## AI Precompiles

> **Determinism Note:** Both AI precompiles compute their answer on-chain from fixed rules, with the same result on every node. They never call the QCAI Backend sidecar, which is non-deterministic and unsuitable for consensus-critical code paths. Neither one writes any state, whether it is reached by `CALL` or `STATICCALL`.

### aiRiskScore (0x0B01)

A minimal, length-only heuristic. It is not a security analysis of the bytes it is given, and a contract must not treat its answer as one.

**Solidity Signature:**
```solidity
function aiRiskScore(bytes txData) view returns (uint256 score, uint8 level)
```

**Parameters:**
- `txData` (bytes): any bytes. Only their length is used. The precompile does not look at opcodes, selectors or patterns, does not read chain state, and does not resolve an address to its deployed code.

**What it returns (this is the whole rule set):**

| Input | `score` | `level` |
|-------|---------|---------|
| empty bytes | 8000 | 3 (HIGH) |
| more than 500,000 bytes | 5000 | 2 (MEDIUM) |
| any other length | 0 | 1 (LOW) |
| calldata that does not ABI-decode as `(bytes)` | 10000 | 4 (CRITICAL, fail closed) |

- `score`: basis points (0 to 10000).
- `level`: 0=SAFE, 1=LOW, 2=MEDIUM, 3=HIGH, 4=CRITICAL. Level 0 is never returned, and level 4 only for undecodable calldata.

**Gas:** 50,000 flat

**Example:**
```solidity
address constant AI_RISK = address(0x0000000000000000000000000000000000000B01);

(bool success, bytes memory result) = AI_RISK.staticcall(
    abi.encodeWithSignature("aiRiskScore(bytes)", data)
);
require(success && result.length == 64, "aiRiskScore not available");
(uint256 score, uint8 level) = abi.decode(result, (uint256, uint8));
```

### aiAnomalyCheck (0x0B02)

Returns the verdict the chain's transaction screening (x/ai) would give a transfer of `amount` from `sender` in the current block.

**Solidity Signature:**
```solidity
function aiAnomalyCheck(address sender, uint256 amount) view returns (uint256 anomalyScore, bool flagged)
```

**Parameters:**
- `sender` (address): the account to check (the same account as its `qor1` form).
- `amount` (uint256): the amount in `uqor` (6 decimals). Values above 2^64-1 are clamped to 2^64-1.

**How it is computed (from v3.2.5):** the same deterministic rules the ante handler applies to every transaction, over the account's stored screening state, read-only:
- the account's last 32 screened transactions and its transaction count in the 30-block rate window (plus this one);
- the EVM-lane rate limit (`max_evm_tx_per_minute`, or `max_tx_per_minute` when that is 0);
- rules: half the rate limit or more in the window scores 0.6; an amount more than three standard deviations from the account's mean (at least three earlier amounts) scores deviation/5, at most 1.0; a transfer above 1,000 QOR from an account with fewer than three screened transactions scores 0.5. The highest rule wins. The rapid-repeat-receiver rule never fires here, since the call names no receiver.

**Returns:**
- `anomalyScore`: the score in basis points (0 = normal, 10000 = highly anomalous).
- `flagged`: true when the score is above 0.5 (5000).

Malformed calldata returns `(10000, true)` (fail closed). Before v3.2.5 the precompile asked an older heuristic with no history.

**Gas:** 40,000 flat (the bounded reads are not metered separately).

## Consensus Precompile

### rlConsensusParams (0x0C01)

Returns current consensus parameters, tuned by the reinforcement learning module when active.

**Solidity Signature:**
```solidity
function rlConsensusParams() view returns (uint256 blockTime, uint256 baseGasPrice, uint256 validatorSetSize, uint256 epoch)
```

**Returns:**
- `blockTime`: Target block time in milliseconds (default: 5000)
- `baseGasPrice`: Base gas price in uqor (default: 100)
- `validatorSetSize`: Active validator set size (default: 100)
- `epoch`: RL training epoch (0 if RL module not active)

**Gas:** 1,500 flat

> **Forward Compatibility:** The `epoch` field returns 0 until the RL module (Chapter 2, Section 2.1) is implemented. The `RLConsensusParamsProvider` interface allows drop-in replacement of the static provider with the real RL module.

## CrossVM Bridge Precompile

### executeCrossVMCall (0x0901)

Enables synchronous calls from EVM contracts to CosmWasm contracts.

**Solidity Signature:**
```solidity
function executeCrossVMCall(uint8 targetVM, string targetContract, bytes payload) returns (bytes result)
```

**Parameters:**
- `targetVM` (uint8): Target VM type (0 = EVM, 1 = CosmWasm)
- `targetContract` (string): Target contract address (bech32 for CosmWasm)
- `payload` (bytes): ABI-encoded call data for the target contract

**Returns:** Raw response bytes from the target contract

**Gas:** 50,000 flat (actual execution gas varies)

## Availability: build and network status

**None of these precompiles is active on qorechain-vladi (mainnet) or qorechain-diana (testnet), in any build.** On both networks the EVM module parameter `active_static_precompiles` is empty (check it with `GET /cosmos/evm/vm/v1/params`), so the chain dispatches no call to any of the addresses above.

- **Community build** (the default `go build`, no tag): `registerEVMPrecompiles` is a no-op (`app/precompiles_stub.go`), so none of the QoreChain precompiles exists in the binary. The EVM runs with the standard Ethereum precompiles only (ecrecover, sha256, ripemd160, identity, modexp, the bn256 pairing set and blake2f).
- **Full build** (`-tags full`, what the network validators run): the implementations are compiled in and connected to the PQC, AI and CrossVM keepers, but a precompile only runs once governance adds its address to `active_static_precompiles`. Until then the full build behaves exactly like the community build for these addresses.

**A call to an address that is not an active precompile succeeds and returns empty data.** That is ordinary EVM behaviour for a call to an address with no code, and it is the case for every address in the table today. A contract must therefore never read empty returndata as an answer: `abi.decode` of empty bytes reverts, but a hand-written decoder or a `success` check alone would read a call to `0x0A01` as a verification result it never was. Integrators should check that the address is listed in `active_static_precompiles` before relying on a precompile, and treat a call that returns no data as "not available".

## Solidity Interfaces

Solidity interface files are provided at:
- `contracts/interfaces/IQorePQC.sol`
- `contracts/interfaces/IQoreAI.sol`
- `contracts/interfaces/IQoreConsensus.sol`
