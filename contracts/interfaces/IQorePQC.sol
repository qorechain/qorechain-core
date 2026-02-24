// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @title IQorePQC - Post-Quantum Cryptography Precompile Interface
/// @notice Interface for QoreChain's PQC precompiles at addresses 0x0A01 and 0x0A02.
/// @dev These precompiles provide access to ML-DSA (Dilithium-5) signature verification
/// and PQC key registration status queries from Solidity contracts.
interface IQorePQC {
    /// @notice Verify a post-quantum signature (ML-DSA / Dilithium-5).
    /// @dev Calls the precompile at address 0x0000000000000000000000000000000000000A01.
    /// Gas cost: 25,000 base + 8 per input byte.
    /// @param pubkey The ML-DSA public key (2592 bytes for Dilithium-5).
    /// @param signature The ML-DSA signature (4627 bytes for Dilithium-5).
    /// @param message The message that was signed.
    /// @return valid True if the signature is valid, false otherwise.
    function pqcVerify(
        bytes calldata pubkey,
        bytes calldata signature,
        bytes calldata message
    ) external view returns (bool valid);

    /// @notice Check PQC key registration status for an address.
    /// @dev Calls the precompile at address 0x0000000000000000000000000000000000000A02.
    /// Gas cost: 2,500 flat.
    /// @param account The address to check (EVM format, converted to bech32 internally).
    /// @return registered True if the address has a PQC key registered.
    /// @return algorithmId The PQC algorithm ID (1 = Dilithium-5, 2 = ML-KEM-1024).
    /// @return pubkey The registered PQC public key bytes.
    function pqcKeyStatus(
        address account
    ) external view returns (bool registered, uint8 algorithmId, bytes memory pubkey);
}
