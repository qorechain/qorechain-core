# PQC Integration Guide

## Overview

QoreChain implements NIST-standardized post-quantum cryptographic algorithms at the protocol level. Unlike chains that retrofit PQC, QoreChain uses PQC as its primary signature scheme.

## Supported Algorithms

| Algorithm | Standard | Purpose | Key Size |
|-----------|----------|---------|----------|
| Dilithium-5 | FIPS 204 | Digital signatures | PK: 2592B, SK: 4896B, Sig: 4627B |
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
