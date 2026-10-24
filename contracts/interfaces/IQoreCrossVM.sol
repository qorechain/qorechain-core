// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title IQoreCrossVM - CrossVM Bridge Precompile Interface
/// @notice Interface for QoreChain's CrossVM bridge precompile at address 0x0901.
/// @dev The canonical ABI of this interface is published as contracts/abi/IQoreCrossVM.json.
/// The precompile only answers once governance has added its address to the x/vm
/// active_static_precompiles list AND the target to the x/crossvm bridge_allowed_targets
/// allowlist; check both before relying on it (a call to an inactive precompile
/// returns empty data, which abi.decode refuses).
///
/// Rules the chain enforces on every call:
///  - plain CALL only: STATICCALL, DELEGATECALL and CALLCODE revert;
///  - no value: a call that carries QOR reverts;
///  - targetVM must be 1 (CosmWasm); 0 (EVM) and 2 (SVM) are reserved and revert;
///  - targetContract must be listed in bridge_allowed_targets as "cosmwasm:<address>";
///  - the CosmWasm contract executes with the CALLER of the precompile (the contract
///    or account that called 0x0901) as its sender, converted to its qor1 address;
///  - the callee's gas comes out of the gas this call frame has left, and a revert of
///    the EVM transaction undoes everything the callee did;
///  - nothing reached through the bridge may enter the EVM again.
interface IQoreCrossVM {
    /// @notice Execute a CosmWasm contract and return its response data.
    /// @dev Calls the precompile at address 0x0000000000000000000000000000000000000901.
    /// Gas cost: 50,000 + 4 per input byte, plus whatever the CosmWasm execution uses.
    /// A refused or failed call reverts with an Error(string) reason.
    /// @param targetVM Target VM: 1 = CosmWasm (the only lane the bridge calls).
    /// @param targetContract The CosmWasm contract address (bech32, qor1...).
    /// @param payload The JSON execute message for the contract, as UTF-8 bytes.
    /// @return result The data field of the contract's response (empty if it set none).
    function executeCrossVMCall(
        uint8 targetVM,
        string calldata targetContract,
        bytes calldata payload
    ) external returns (bytes memory result);
}
