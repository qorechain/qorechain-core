# QoreChain Documentation

QoreChain is a quantum-safe, AI-native Layer 1 blockchain engineered with post-quantum cryptography at genesis, a triple-VM runtime (EVM, CosmWasm, SVM), on-chain reinforcement learning consensus optimization, 25 direct cross-chain connections, and a Rollup Development Kit for deploying application-specific chains. Built on QoreChain SDK v0.53 with 18 custom modules and 45 registered genesis modules, QoreChain delivers production-grade security against both classical and quantum adversaries while maintaining full compatibility with existing blockchain tooling.

## Quick Links

- [Getting Started](getting-started/quickstart.md) -- Set up a wallet, connect to the testnet, and send your first transaction.
- [User Guide](user-guide/token-operations.md) -- Token operations, staking, governance, bridging, and rollup deployment.
- [Developer Guide](developer-guide/building-from-source.md) -- Build from source, deploy smart contracts across all three VMs, and run a validator.
- [Architecture](architecture/consensus-mechanism.md) -- Consensus mechanism, RL engine, PQC security, tokenomics, and bridge design.
- [API Reference](api-reference/rest-grpc-endpoints.md) -- REST, gRPC, EVM JSON-RPC, SVM JSON-RPC, and WebSocket endpoints.
- [CLI Reference](cli-reference/node-commands.md) -- Node, transaction, and query commands for the `qorechaind` binary.

## Chain Details

| Parameter | Value |
|-----------|-------|
| Chain ID | `qorechain-diana` |
| Token | QOR (display) / uqor (base denomination) |
| Denomination Exponent | 10^6 (1 QOR = 1,000,000 uqor) |
| Bech32 Prefix | `qor` (addresses: `qor1...`, validators: `qorvaloper...`) |
| Version | v1.3.0 |
| Framework | QoreChain SDK v0.53 |
