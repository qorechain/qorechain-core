# PQC Algorithm Agility Framework (v0.6.0)

## Overview

The Algorithm Agility Framework enables QoreChain to dynamically manage multiple post-quantum cryptographic (PQC) algorithms. This provides:

- **Governance-controlled algorithm registry** — Add, deprecate, or emergency-disable algorithms via on-chain governance.
- **Dual-signature key migration** — Securely migrate accounts between algorithms with cryptographic proof of ownership.
- **Multi-algorithm dispatch** — The ante handler verifies transactions using the algorithm registered to each account.
- **Modular FFI layer** — New algorithms can be added in Rust and exposed to the Go runtime via the C FFI bridge.

## Architecture

```
                    ┌──────────────────────────────────────┐
                    │          Governance Module            │
                    │  (MsgAddAlgorithm, MsgDeprecate,     │
                    │   MsgDisableAlgorithm)               │
                    └──────────┬───────────────────────────┘
                               │
                    ┌──────────▼───────────────────────────┐
                    │       x/pqc Keeper (Algorithm Registry)│
                    │  - RegisterAlgorithm()               │
                    │  - UpdateAlgorithmStatus()            │
                    │  - GetMigration() / SetMigration()   │
                    └──────────┬───────────────────────────┘
                               │
          ┌────────────────────┼─────────────────────────┐
          │                    │                         │
┌─────────▼────────┐ ┌────────▼──────────┐ ┌────────────▼────┐
│  Ante Decorator   │ │  Msg Server       │ │  CLI Commands   │
│ (Multi-algo       │ │ (RegisterKey V2,  │ │ (query/tx for   │
│  dispatch)        │ │  MigratePQCKey)   │ │  algorithms)    │
└─────────┬────────┘ └────────┬──────────┘ └─────────────────┘
          │                    │
          └────────┬───────────┘
                   │
          ┌────────▼──────────────────────────────────────┐
          │              FFI Bridge (cgo)                   │
          │  Keygen / Sign / Verify / AlgorithmInfo        │
          └────────┬──────────────────────────────────────┘
                   │
          ┌────────▼──────────────────────────────────────┐
          │           libqorepqc (Rust)                    │
          │  PQCSignatureScheme / PQCKEMScheme traits      │
          │  AlgorithmRegistry (static dispatch)           │
          │  Dilithium-5, ML-KEM-1024                      │
          └───────────────────────────────────────────────┘
```

## Algorithm Lifecycle

Each algorithm has a lifecycle status:

| Status      | Description                                     | New Keys? | Verify? |
|-------------|------------------------------------------------|-----------|---------|
| `active`    | Fully operational                              | Yes       | Yes     |
| `migrating` | Dual-signature period active                   | No        | Yes     |
| `deprecated`| Still verifiable, no new key registrations     | No        | Yes     |
| `disabled`  | Emergency kill switch — cannot verify           | No        | No      |

### State transitions:

```
active → migrating → deprecated → (governance may disable)
active → disabled (emergency, skip migration)
```

## Built-in Algorithms

### Dilithium-5 (ID: 1)
- **Standard:** NIST FIPS 204
- **Category:** Digital Signature
- **NIST Level:** 5
- **Key sizes:** Public key 2,592 bytes, Private key 4,896 bytes
- **Signature size:** 4,627 bytes

### ML-KEM-1024 (ID: 2)
- **Standard:** NIST FIPS 203
- **Category:** Key Encapsulation Mechanism
- **NIST Level:** 5
- **Key sizes:** Public key 1,568 bytes, Private key 3,168 bytes
- **Ciphertext size:** 1,568 bytes, Shared secret: 32 bytes

## CLI Commands

### Query Commands

```bash
# List all registered algorithms
qorechaind query pqc algorithms

# Query a specific algorithm
qorechaind query pqc algorithm dilithium5
qorechaind query pqc algorithm 1

# Query PQC account info
qorechaind query pqc account qor1...

# Query module statistics
qorechaind query pqc stats

# Query module parameters
qorechaind query pqc params

# Query active migration
qorechaind query pqc migration 1
```

### Transaction Commands

```bash
# Register a PQC key (legacy, defaults to Dilithium-5)
qorechaind tx pqc register-key <pubkey-hex> hybrid --from mykey

# Register with explicit algorithm selection (v0.6.0)
qorechaind tx pqc register-key-v2 dilithium5 <pubkey-hex> hybrid --from mykey

# Migrate to a new algorithm (requires active migration + dual signatures)
qorechaind tx pqc migrate-key <new-algo> <old-pubkey> <new-pubkey> <old-sig> <new-sig> --from mykey
```

## Governance Operations

Algorithm management is controlled through governance proposals:

### Add a New Algorithm
Submit a `MsgAddAlgorithm` through `MsgSubmitProposal`. The algorithm is registered with `active` status.

### Deprecate an Algorithm
Submit a `MsgDeprecateAlgorithm` to start a migration period. The default migration period is 1,000,000 blocks (~69 days at 6s/block). During this period, accounts must use dual signatures.

### Emergency Disable
Submit a `MsgDisableAlgorithm` to immediately disable an algorithm (e.g., if a vulnerability is discovered). Accounts using the disabled algorithm fall back to classical ECDSA if hybrid mode is enabled, otherwise transactions are rejected.

## Migration Process

When an algorithm is deprecated:

1. **Governance creates migration** — `MsgDeprecateAlgorithm` sets the source algorithm to `migrating` status and creates a `MigrationInfo` record.

2. **Dual-signature period** — For `DefaultMigrationBlocks` (1,000,000 blocks), accounts must provide signatures from both their old and new keys.

3. **Account migration** — Users submit `MsgMigratePQCKey` with:
   - Their old public key
   - Their new public key (for the target algorithm)
   - Old key signature over `"migrate:<address>"`
   - New key signature over `"migrate:<address>"`

4. **Verification** — The keeper verifies both signatures via the FFI layer, then updates the account's algorithm and public key.

5. **Completion** — After the migration period, the old algorithm status changes to `deprecated`.

## Extending with New Algorithms

To add a new PQC algorithm:

1. **Rust layer** — Implement `PQCSignatureScheme` or `PQCKEMScheme` trait in `qorechain-proprietary/pqc/src/`
2. **Registry** — Add the algorithm to `AlgorithmRegistry::new()` with a new ID
3. **Go types** — Add the algorithm ID constant in `types/algorithm.go`
4. **Governance** — Submit `MsgAddAlgorithm` proposal on-chain
5. **Compile & deploy** — Build new `libqorepqc` and deploy to validators

## Security Considerations

- All PQC operations run through the Rust FFI layer — no pure-Go cryptographic implementations
- Dual-signature migration ensures both old and new keys are proven valid
- Algorithm disable is immediate (no migration period) for emergency response
- The ante decorator checks algorithm status on every transaction
- Backward compatibility: pre-v0.6.0 accounts default to Dilithium-5
