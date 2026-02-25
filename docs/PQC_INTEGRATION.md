# PQC Integration Guide

## Overview

QoreChain implements NIST-standardized post-quantum cryptographic algorithms at the protocol level. Unlike chains that retrofit PQC, QoreChain uses PQC as its primary signature scheme with a hybrid architecture that bridges classical and quantum-resistant cryptography.

## Supported Algorithms

| Algorithm | Standard | Purpose | Key Size |
|-----------|----------|---------|----------|
| Dilithium-5 (ML-DSA-87) | FIPS 204 | Digital signatures | PK: 2592B, SK: 4896B, Sig: 4627B |
| ML-KEM-1024 | FIPS 203 | Key encapsulation | PK: 1568B, CT: 1568B, SS: 32B |

## Architecture

The PQC implementation uses a Rust FFI bridge:

```
Go (x/pqc) -> CGO -> C FFI -> Rust (libqorepqc)
```

The Rust library (`libqorepqc`) wraps the `pqcrypto` crate family and exposes C-compatible functions.

## Key Registration

Accounts register PQC keys via `MsgRegisterPQCKey`:

```
Key Types:
  - hybrid: Both Dilithium-5 and ECDSA (recommended for transition)
  - pqc_only: Dilithium-5 only (maximum quantum safety)
  - classical_only: ECDSA only (backward compatible)
```

## Hybrid Signatures (v1.1.0)

The hybrid signature system enables dual Ed25519 + ML-DSA-87 verification on every transaction without breaking classical wallet compatibility.

### How It Works

1. **TX Extension**: PQC signatures are attached as `PQCHybridSignature` TX extensions alongside the classical Ed25519/secp256k1 signature. The extension carries:
   - `AlgorithmID`: Identifies the PQC algorithm (e.g., Dilithium-5)
   - `PQCSignature`: The raw ML-DSA-87 signature bytes (4627 bytes for Dilithium-5)
   - `PQCPublicKey` (optional): PQC public key for auto-registration on first use

2. **Ante Handler**: The `PQCHybridVerifyDecorator` runs in the ante handler chain after `PQCVerify` and before `AIAnomaly`:
   ```
   PQCVerify → PQCHybridVerify → AIAnomaly → ... → SigVerify
   ```

3. **Three-Way Verification**:
   - **Account has PQC key + extension present**: Verify both classical + PQC signatures
   - **No PQC key + extension with public key**: Auto-register PQC key, then verify
   - **No PQC key + no extension**: Classical only (or reject if `HybridRequired` mode)

### Hybrid Signature Modes

Governance controls the enforcement level via the `HybridSignatureMode` parameter:

| Mode | Value | Behavior |
|------|-------|----------|
| Disabled | 0 | Classical only; PQC extensions ignored |
| Optional | 1 (default) | PQC verified if present; classical fallback allowed |
| Required | 2 | Both signatures mandatory; classical-only transactions rejected |

### Querying the Mode

```bash
# CLI
qorechaind query pqc hybrid-mode

# JSON-RPC
curl -X POST http://localhost:8545 -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"qor_getHybridSignatureMode","params":[]}'
```

### Events

| Event | Description |
|-------|-------------|
| `pqc_hybrid_verify` | Both classical and PQC signatures verified |
| `pqc_hybrid_auto_register` | PQC key auto-registered from TX extension |
| `pqc_hybrid_classical_only` | Transaction processed with classical signature only |

## SHAKE-256 Hash Foundation (v1.1.0)

QoreChain includes a SHAKE-256 (SHA-3 family extendable-output function) hash utility layer as preparation for future post-quantum Merkle tree replacement:

- `SHAKE256Hash(data, outputLen)` — Variable-length XOF output
- `SHAKE256Hash32(data)` — Fixed 32-byte output
- `SHAKE256ConcatHash(left, right)` — Merkle internal node hash
- `SHAKE256DomainHash(domain, data)` — Domain-separated hashing

Pure Go implementation using `golang.org/x/crypto/sha3`, no FFI dependency.

## Building with PQC

```bash
# The PQC library must be available in lib/{os}_{arch}/
# Download from releases or build from qorechain-proprietary repo

# Build with CGO
CGO_ENABLED=1 go build -o qorechaind ./cmd/qorechaind/

# The library is loaded at runtime
export LD_LIBRARY_PATH=$PWD/lib/linux_amd64  # Linux
export DYLD_LIBRARY_PATH=$PWD/lib/darwin_arm64  # macOS ARM
```

## Bridge PQC Integration

All bridge validator attestations use Dilithium-5 signatures. Bridge operations include ML-KEM commitments for quantum-safe verification.
