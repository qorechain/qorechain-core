# EVM Precompiles

QoreChain extends the EVM with **six custom precompiled contracts** that expose protocol-level features directly to Solidity. These precompiles provide on-chain access to post-quantum cryptography, AI risk scoring, cross-VM messaging, and RL consensus parameters.

---

## Precompile Address Map

| Precompile | Address | Base Gas | Description |
|-----------|---------|----------|-------------|
| **CrossVM Bridge** | `0x0000000000000000000000000000000000000901` | 50,000 | Synchronous cross-VM calls (EVM to CosmWasm) |
| **PQC Verify** | `0x0000000000000000000000000000000000000A01` | 25,000 + 8/byte | Verify ML-DSA-87 post-quantum signatures |
| **PQC Key Status** | `0x0000000000000000000000000000000000000A02` | 2,500 | Check if an account has a registered PQC key |
| **AI Risk Score** | `0x0000000000000000000000000000000000000B01` | 50,000 | Get AI-generated risk score for transaction data |
| **AI Anomaly Check** | `0x0000000000000000000000000000000000000B02` | 40,000 | Check if a transfer is flagged as anomalous |
| **RL Consensus Params** | `0x0000000000000000000000000000000000000C01` | 1,500 | Read current RL-tuned consensus parameters |

---

## Solidity Interfaces

### IQorePQC.sol

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IQorePQC {
    /// @notice Verify an ML-DSA-87 (Dilithium-5) signature
    /// @param pubkey The 2592-byte ML-DSA-87 public key
    /// @param signature The 4627-byte ML-DSA-87 signature
    /// @param message The original message bytes
    /// @return valid True if the signature is valid
    function pqcVerify(
        bytes calldata pubkey,
        bytes calldata signature,
        bytes calldata message
    ) external view returns (bool valid);

    /// @notice Check if an account has a registered PQC public key
    /// @param account The Ethereum-style address to check
    /// @return registered True if a PQC key is registered
    /// @return algorithmId The PQC algorithm identifier (1 = ML-DSA-87)
    /// @return pubkey The registered public key bytes (empty if not registered)
    function pqcKeyStatus(address account)
        external
        view
        returns (bool registered, uint8 algorithmId, bytes memory pubkey);
}
```

### IQoreAI.sol

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IQoreAI {
    /// @notice Get the AI-generated risk score for transaction data
    /// @param txData The transaction data to evaluate
    /// @return score Risk score in basis points (0-10000)
    /// @return level Risk level: 0=SAFE, 1=LOW, 2=MEDIUM, 3=HIGH, 4=CRITICAL
    function aiRiskScore(bytes calldata txData)
        external
        view
        returns (uint256 score, uint8 level);

    /// @notice Check if a transfer is flagged as anomalous
    /// @param sender The sender address
    /// @param amount The transfer amount in wei
    /// @return anomalyScore The anomaly score (higher = more anomalous)
    /// @return flagged True if the transfer exceeds the anomaly threshold
    function aiAnomalyCheck(address sender, uint256 amount)
        external
        view
        returns (uint256 anomalyScore, bool flagged);
}
```

### IQoreConsensus.sol

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IQoreConsensus {
    /// @notice Get current RL-tuned consensus parameters
    /// @return blockTime Current target block time in milliseconds
    /// @return baseGasPrice Current base gas price in wei
    /// @return validatorSetSize Current active validator set size
    /// @return epoch Current RL epoch number
    function rlConsensusParams()
        external
        view
        returns (
            uint256 blockTime,
            uint256 baseGasPrice,
            uint256 validatorSetSize,
            uint256 epoch
        );
}
```

---

## Usage Examples

### PQC Verify -- Verify a Post-Quantum Signature

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "./interfaces/IQorePQC.sol";

contract PQCVerifier {
    IQorePQC constant PQC = IQorePQC(0x0000000000000000000000000000000000000A01);

    /// @notice Verify a PQC signature and revert if invalid
    function verifyOrRevert(
        bytes calldata pubkey,
        bytes calldata signature,
        bytes calldata message
    ) external view {
        bool valid = PQC.pqcVerify(pubkey, signature, message);
        require(valid, "PQC signature verification failed");
    }
}
```

**Gas cost:** 25,000 base + 8 gas per byte of input data. For a typical ML-DSA-87 verification (2592 + 4627 + message bytes), expect approximately 80,000-90,000 gas.

### PQC Key Status -- Check Account Registration

```solidity
contract PQCChecker {
    IQorePQC constant PQC_STATUS = IQorePQC(0x0000000000000000000000000000000000000A02);

    function requirePQCKey(address account) external view {
        (bool registered, , ) = PQC_STATUS.pqcKeyStatus(account);
        require(registered, "Account must have a registered PQC key");
    }
}
```

**Gas cost:** 2,500 flat.

### AI Risk Score -- Evaluate Transaction Risk

```solidity
import "./interfaces/IQoreAI.sol";

contract RiskGate {
    IQoreAI constant AI = IQoreAI(0x0000000000000000000000000000000000000B01);

    uint8 constant MAX_ALLOWED_RISK = 2; // Allow up to MEDIUM

    function executeIfSafe(bytes calldata txData) external {
        (uint256 score, uint8 level) = AI.aiRiskScore(txData);
        require(level <= MAX_ALLOWED_RISK, "Risk level too high");

        // Proceed with execution...
    }
}
```

**Risk levels:**

| Level | Value | Score Range (bps) |
|-------|-------|-------------------|
| SAFE | 0 | 0 - 1000 |
| LOW | 1 | 1001 - 3000 |
| MEDIUM | 2 | 3001 - 6000 |
| HIGH | 3 | 6001 - 8500 |
| CRITICAL | 4 | 8501 - 10000 |

**Gas cost:** 50,000 flat.

### AI Anomaly Check -- Flag Suspicious Transfers

```solidity
contract AnomalyGuard {
    IQoreAI constant AI = IQoreAI(0x0000000000000000000000000000000000000B02);

    function safeTransfer(address sender, uint256 amount) external view {
        (uint256 anomalyScore, bool flagged) = AI.aiAnomalyCheck(sender, amount);

        if (flagged) {
            revert("Transfer flagged as anomalous");
        }
    }
}
```

**Gas cost:** 40,000 flat.

### RL Consensus Params -- Read Consensus State

```solidity
contract ConsensusReader {
    IQoreConsensus constant RL = IQoreConsensus(
        0x0000000000000000000000000000000000000C01
    );

    function getBlockTime() external view returns (uint256) {
        (uint256 blockTime, , , ) = RL.rlConsensusParams();
        return blockTime;
    }

    function getCurrentEpoch() external view returns (uint256) {
        (, , , uint256 epoch) = RL.rlConsensusParams();
        return epoch;
    }
}
```

**Gas cost:** 1,500 flat.

### CrossVM Bridge -- Call CosmWasm from EVM

```solidity
interface ICrossVM {
    function call(bytes calldata payload) external returns (bytes memory);
}

contract CrossVMExample {
    ICrossVM constant CROSSVM = ICrossVM(
        0x0000000000000000000000000000000000000901
    );

    function callCosmWasm(
        string memory contractAddr,
        string memory executeMsg
    ) external returns (bytes memory) {
        bytes memory payload = abi.encode(contractAddr, executeMsg, uint256(0));
        return CROSSVM.call(payload);
    }
}
```

**Gas cost:** 50,000 base + target contract execution cost. See [Cross-VM Interoperability](cross-vm-interop.md) for details.

---

## Interface File Locations

The Solidity interface files are available in the repository for direct import:

```
contracts/
  interfaces/
    IQorePQC.sol
    IQoreAI.sol
    IQoreConsensus.sol
    ICrossVM.sol
```

Install them in your Hardhat or Foundry project:

```bash
# Copy interfaces to your project
cp -r contracts/interfaces/ my-project/contracts/interfaces/
```

Or reference them via import path in your Solidity files:

```solidity
import "./interfaces/IQorePQC.sol";
import "./interfaces/IQoreAI.sol";
import "./interfaces/IQoreConsensus.sol";
```

---

## Next Steps

- [Cross-VM Interoperability](cross-vm-interop.md) -- Full cross-VM messaging documentation
- [EVM Development](evm-development.md) -- Deploying Solidity contracts
- [Account Abstraction](account-abstraction.md) -- Programmable accounts with session keys
