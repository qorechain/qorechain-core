// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title IQoreAI - AI Engine Precompile Interface
/// @notice Interface for QoreChain's AI precompiles at addresses 0x0B01 and 0x0B02.
/// @dev Both precompiles compute their answer on-chain from fixed rules and write no state.
/// They never call the QCAI Backend sidecar. See docs/EVM_PRECOMPILES.md for the exact rules.
interface IQoreAI {
    /// @notice Length-only heuristic, not a security analysis: empty bytes score 8000 (level 3),
    /// more than 500,000 bytes score 5000 (level 2), anything else scores 0 (level 1).
    /// @dev Calls the precompile at address 0x0000000000000000000000000000000000000B01.
    /// Gas cost: 50,000 flat. Undecodable calldata returns (10000, 4).
    /// @param txData Any bytes; only their length is used.
    /// @return score Score in basis points (0 to 10000).
    /// @return level Severity level: 0=SAFE, 1=LOW, 2=MEDIUM, 3=HIGH, 4=CRITICAL.
    function aiRiskScore(
        bytes calldata txData
    ) external view returns (uint256 score, uint8 level);

    /// @notice The x/ai screening verdict for a transfer of amount from sender in this block,
    /// over the sender's stored history and rate window (read-only).
    /// @dev Calls the precompile at address 0x0000000000000000000000000000000000000B02.
    /// Gas cost: 40,000 flat.
    /// @param sender The sender address to check.
    /// @param amount The transfer amount in uqor.
    /// @return anomalyScore Anomaly score in basis points (0 = normal, 10000 = highly anomalous).
    /// @return flagged True if the transaction is flagged as anomalous.
    function aiAnomalyCheck(
        address sender,
        uint256 amount
    ) external view returns (uint256 anomalyScore, bool flagged);
}
