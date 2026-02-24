// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title IQoreAI - AI Engine Precompile Interface
/// @notice Interface for QoreChain's AI precompiles at addresses 0x0B01 and 0x0B02.
/// @dev These precompiles provide deterministic AI risk assessment and anomaly detection
/// using the on-chain heuristic engine (Z-score + isolation forest).
/// IMPORTANT: These precompiles use ONLY deterministic on-chain algorithms.
/// They never call the QCAI Backend sidecar to preserve consensus determinism.
interface IQoreAI {
    /// @notice Score the risk level of contract bytecode or transaction data.
    /// @dev Calls the precompile at address 0x0000000000000000000000000000000000000B01.
    /// Gas cost: 50,000 flat.
    /// @param txData The contract bytecode or transaction data to analyze.
    /// @return score Risk score in basis points (0 = safe, 10000 = critical risk).
    /// @return level Severity level: 0=SAFE, 1=LOW, 2=MEDIUM, 3=HIGH, 4=CRITICAL.
    function aiRiskScore(
        bytes calldata txData
    ) external view returns (uint256 score, uint8 level);

    /// @notice Check if a transaction pattern is anomalous.
    /// @dev Calls the precompile at address 0x0000000000000000000000000000000000000B02.
    /// Gas cost: 40,000 flat.
    /// @param sender The sender address to check.
    /// @param amount The transaction amount (in base denomination).
    /// @return anomalyScore Anomaly score in basis points (0 = normal, 10000 = highly anomalous).
    /// @return flagged True if the transaction is flagged as anomalous.
    function aiAnomalyCheck(
        address sender,
        uint256 amount
    ) external view returns (uint256 anomalyScore, bool flagged);
}
