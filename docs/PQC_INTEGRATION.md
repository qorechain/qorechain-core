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

The Rust library (`libqorepqc`) wraps the audited FIPS-standard crates
(`fips204` for ML-DSA-87, `fips203` for ML-KEM-1024) and exposes C-compatible
functions. SHAKE-256 (FIPS-202) is provided by `golang.org/x/crypto/sha3`. These
are the same standard implementations used by the public `qorechain-pqc` library,
so on-chain signatures are **byte-compatible** with any standard ML-DSA-87
verifier (Cloudflare CIRCL, @noble/post-quantum, Bouncy Castle, liboqs).

## Interoperability & client libraries

Because the chain uses the final NIST standards (not a custom variant), standard
tooling interoperates directly:

| Where | Package | Install |
|-------|---------|---------|
| JavaScript/TypeScript | `@qorechain/pqc` | `npm i @qorechain/pqc` |
| Python | `qorechain-pqc` | `pip install qorechain-pqc` (then `import qorpqc`) |
| Rust | `qorechain-pqc` | `cargo add qorechain-pqc` |
| Go | `…/qorechain-pqc/go` | `go get github.com/qorechain/qorechain-pqc/go@v0.1.0` |
| Java | `io.github.qorechain:qorechain-pqc` | Maven Central — add the dependency (`<version>0.1.0</version>`) |
| C | source bindings | see the [qorechain-pqc](https://github.com/qorechain/qorechain-pqc) repo |

**Universal wallet adapter.** `@qorechain/wallet-adapter` lets any Cosmos wallet
(Keplr, Leap, Cosmostation) add QoreChain and sign its PQC-required transactions
with no wallet-side changes: the wallet produces an ordinary `SIGN_MODE_DIRECT`
signature over the final body, into which the adapter has already layered the
standard ML-DSA-87 hybrid extension. MetaMask works natively via the EVM path
(structurally PQC-exempt).

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

## SHAKE-256 — Default QoreChain Hash

SHAKE-256 (the SHA-3 / Keccak family extendable-output function, FIPS 202) is the
**default application-level hash** for every QoreChain-controlled commitment and
identifier. Together with Dilithium-5 (signatures) and ML-KEM-1024 (key
encapsulation) it completes the chain's post-quantum baseline: the three
documented PQC algorithms are all implemented *and* default.

The canonical implementation is the `qorehash` package (pure Go,
`golang.org/x/crypto/sha3`, no FFI, no build tags — byte-for-byte reproducible in
both community and full builds):

- `qorehash.Sum256(data)` / `qorehash.Sum(data)` — 32-byte digest (drop-in for `sha256.Sum256`)
- `qorehash.New()` — streaming `hash.Hash` with 32-byte output (drop-in for `sha256.New()`)
- `qorehash.SumN(data, n)` — variable-length XOF output
- `qorehash.ConcatHash(left, right)` — second-preimage-resistant Merkle node hash (length-prefixed)
- `qorehash.DomainHash(domain, data)` — domain-separated hash

The `x/pqc` module additionally exposes the original `SHAKE256*` helpers, which
produce identical output.

**Where SHAKE-256 is the default (QoreChain-controlled, producer = verifier):**
multilayer state-anchor `canonicalTransitionRoot`; rdk withdrawal Merkle roots,
DA content-addressing and optimistic `batchTransitionRoot`; the in-tree STARK
prover/verifier transcript and Merkle commitments; cross-VM message IDs; SVM
address/program/account derivation; QCA reputation-weighted proposer selection;
abstract-account address derivation.

**Where native hashes are deliberately retained (hybrid only at network egress):**
external-chain verification keeps each foreign chain's own format — Bitcoin
`sha256d`, Ethereum MPT `keccak256`, Ethereum Beacon SSZ `sha256`, BLS/Pedersen
(bridge light-clients). Framework hashing owned by Cosmos SDK / QoreChain Consensus Engine / IAVL,
EVM ABI selectors (keccak256), the SVM tx-signature that mirrors the QoreChain Consensus Engine's tx
hash, and Solana SVM syscalls (`sol_sha256`/`sol_keccak256`) are unchanged so
existing bytecode and tooling keep working.

## Building with PQC

```bash
# The PQC library must be available in lib/{os}_{arch}/
# Download from releases or build from qorechain-internal repo

# Build with CGO
CGO_ENABLED=1 go build -o qorechaind ./cmd/qorechaind/

# The library is loaded at runtime
export LD_LIBRARY_PATH=$PWD/lib/linux_amd64  # Linux
export DYLD_LIBRARY_PATH=$PWD/lib/darwin_arm64  # macOS ARM
```

## Bridge PQC Integration

All bridge validator attestations use Dilithium-5 signatures. Bridge operations include ML-KEM commitments for quantum-safe verification.
