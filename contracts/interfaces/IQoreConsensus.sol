// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title IQoreConsensus - RL Consensus Parameters Precompile Interface
/// @notice Interface for QoreChain's RL consensus precompile at address 0x0C01.
/// @dev Returns current consensus parameters that are tuned by the reinforcement
/// learning module (when active). Falls back to static genesis parameters when
/// the RL module is not active.
interface IQoreConsensus {
    /// @notice Get current RL-tuned consensus parameters.
    /// @dev Calls the precompile at address 0x0000000000000000000000000000000000000C01.
    /// Gas cost: 1,500 flat.
    /// @return blockTime Current target block time in milliseconds.
    /// @return baseGasPrice Current base gas price in base denomination (uqor).
    /// @return validatorSetSize Current active validator set size.
    /// @return epoch Current RL training epoch (0 if RL not active).
    function rlConsensusParams() external view returns (
        uint256 blockTime,
        uint256 baseGasPrice,
        uint256 validatorSetSize,
        uint256 epoch
    );
}
