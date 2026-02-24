# QoreChain EVM Precompiles

QoreChain extends the EVM runtime with 6 custom precompiled contracts that expose PQC cryptography, AI risk analysis, RL consensus parameters, and cross-VM communication to Solidity smart contracts.

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
function pqcKeyStatus(address account) view returns (bool registered, uint8 algorithmId, bytes pubkey)
```

**Parameters:**
- `account` (address): EVM address to query (converted to bech32 internally)

**Returns:**
- `registered`: Whether the address has a PQC key
- `algorithmId`: Algorithm ID (1 = Dilithium-5, 2 = ML-KEM-1024)
- `pubkey`: The registered PQC public key bytes

**Gas:** 2,500 flat

## AI Precompiles

> **Determinism Note:** All AI precompiles use the on-chain deterministic heuristic engine (Z-score + isolation forest). They never call the QCAI Backend sidecar, which is non-deterministic and unsuitable for consensus-critical code paths.

### aiRiskScore (0x0B01)

Analyzes contract bytecode or transaction data for security risks using deterministic heuristics.

**Solidity Signature:**
```solidity
function aiRiskScore(bytes txData) view returns (uint256 score, uint8 level)
```

**Parameters:**
- `txData` (bytes): Contract bytecode or transaction data to analyze

**Returns:**
- `score`: Risk score in basis points (0 = safe, 10000 = critical)
- `level`: Severity enum: 0=SAFE, 1=LOW, 2=MEDIUM, 3=HIGH, 4=CRITICAL

**Gas:** 50,000 flat

**Example:**
```solidity
address constant AI_RISK = address(0x0000000000000000000000000000000000000B01);

(bool success, bytes memory result) = AI_RISK.staticcall(
    abi.encodeWithSignature("aiRiskScore(bytes)", contractBytecode)
);
(uint256 score, uint8 level) = abi.decode(result, (uint256, uint8));
require(level < 3, "Contract risk too high");
```

### aiAnomalyCheck (0x0B02)

Checks whether a transaction pattern is anomalous based on historical behavior.

**Solidity Signature:**
```solidity
function aiAnomalyCheck(address sender, uint256 amount) view returns (uint256 anomalyScore, bool flagged)
```

**Parameters:**
- `sender` (address): The transaction sender
- `amount` (uint256): The transaction amount in base denomination

**Returns:**
- `anomalyScore`: Anomaly score in basis points (0 = normal, 10000 = highly anomalous)
- `flagged`: Whether the transaction is flagged as anomalous

**Gas:** 40,000 flat

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

## Community vs Proprietary Build

In the **community build** (default), all 6 precompiles return descriptive errors indicating the feature is not available. The EVM itself functions normally with default geth precompiles.

In the **proprietary build** (`-tags proprietary`), all 6 precompiles are fully functional and connected to the PQC, AI, and CrossVM keeper modules.

## Solidity Interfaces

Solidity interface files are provided at:
- `contracts/interfaces/IQorePQC.sol`
- `contracts/interfaces/IQoreAI.sol`
- `contracts/interfaces/IQoreConsensus.sol`
