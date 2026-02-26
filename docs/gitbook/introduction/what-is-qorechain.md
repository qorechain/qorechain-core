# What is QoreChain?

QoreChain is the first Layer 1 blockchain built with post-quantum cryptography at genesis, AI-native transaction processing, and a triple-VM runtime that executes EVM, CosmWasm, and SVM programs on a single chain. Rather than retrofitting quantum resistance onto an existing protocol, QoreChain was designed from the ground up to be secure against both classical and quantum adversaries while delivering the developer experience and interoperability expected of a modern general-purpose blockchain.

## Core Innovations

### 1. Post-Quantum Cryptography

QoreChain uses NIST-standardized ML-DSA-87 (Dilithium-5) for digital signatures and ML-KEM-1024 for key encapsulation, providing security against attacks from both classical and quantum computers. The hybrid signature architecture pairs Ed25519 with ML-DSA-87 so that every transaction can carry dual signatures -- classical wallets continue working unmodified while PQC-enabled wallets gain quantum resistance. Three governance-controlled enforcement modes (disabled, optional, required) allow the network to migrate gradually without disrupting existing users. An algorithm agility framework ensures that signature schemes can be upgraded via governance proposals as cryptographic standards evolve.

### 2. AI-Native Processing

An on-chain reinforcement learning agent (PPO MLP with 73,733 parameters) runs deterministic fixed-point inference directly in the block lifecycle, dynamically tuning consensus parameters such as block time, gas limits, and validator pool weights. Statistical isolation forest anomaly detection and multi-dimensional risk scoring evaluate every transaction in the ante handler chain, flagging fraudulent patterns before execution. Dynamic fee optimization adjusts base fees based on real-time network conditions. All AI inference is fully deterministic across validators -- identical inputs produce identical outputs with no external oracle dependency.

### 3. Triple-VM Runtime

QoreChain is the only Layer 1 that natively runs three virtual machines within one consensus:

- **EVM** -- Full Ethereum compatibility with EIP-1559 gas pricing and JSON-RPC on port 8545. Deploy Solidity contracts using standard tooling (Hardhat, Foundry, Remix).
- **CosmWasm** -- WebAssembly smart contracts written in Rust with full lifecycle support (instantiate, execute, query, migrate).
- **SVM** -- BPF program deployment and execution with a Solana-compatible JSON-RPC server on port 8899. Existing Solana clients and tooling work out of the box.

Cross-VM messaging enables all three runtimes to communicate: EVM contracts call CosmWasm via precompile, CosmWasm contracts call EVM via custom messages, and SVM programs participate through async event-based bridging.

### 4. Deflationary Tokenomics

Ten distinct burn channels (transaction fees, governance penalties, slashing, bridge fees, spam deterrence, epoch excess, manual burns, contract callbacks, cross-VM fees, and rollup creation burns) feed a central burn accounting module. Collected fees are split 40% to validators, 30% permanently burned, 20% to treasury, and 10% to stakers. The xQORE governance staking mechanism lets users lock QOR for doubled governance weight with PvP rebase redistribution -- early exit penalties are redistributed to remaining holders, rewarding conviction. Epoch-based inflation follows a multi-year decay schedule from 17.5% down to 2%, converging toward net-deflationary equilibrium as transaction volume grows.

### 5. 25 Cross-Chain Connections

QoreChain connects to 25 blockchain ecosystems through two complementary protocols:

- **8 IBC channels** -- Cosmos Hub, Osmosis, Noble, Celestia, Stride, Akash, Babylon, and the QoreChain loopback relay. Pre-configured relayer templates with client updates, misbehaviour detection, and automatic packet clearing.
- **17 QCB bridge endpoints** -- Ethereum, BSC, Solana, Avalanche, Polygon, Arbitrum, TON, Sui, Optimism, Base, Aptos, Bitcoin, NEAR, Cardano, Polkadot, Tezos, and TRON. Each endpoint includes per-type address validation, configurable confirmation depth, circuit breaker volume caps, and PQC-signed validator attestations.

Twelve chain types are supported: evm, solana, ton, move, sui_move, cosmos_ibc, aptos_move, utxo, near, cardano, polkadot, and tezos -- covering every major blockchain architecture.

### 6. Rollup Development Kit (v1.3.0)

The `x/rdk` module is a protocol-native framework for deploying application-specific rollups directly on the QoreChain host chain. Four settlement paradigms are supported:

- **Optimistic** -- Fraud proofs with a 7-day challenge window, auto-finalized by EndBlocker.
- **ZK (Zero-Knowledge)** -- SNARK or STARK proofs with instant finality on verification.
- **Based** -- L1-sequenced transactions with finality in approximately 2 host blocks.
- **Sovereign** -- Independent chains using QoreChain exclusively for data availability.

Four preset profiles (DeFi, Gaming, NFT, Enterprise) enable one-click deployment with pre-configured settlement modes, block times, VM choices, DA backends, and gas models. A native DA router provides SHA-256 committed blob storage with configurable retention and automatic pruning. The RL consensus module provides advisory methods for AI-assisted rollup configuration.

### 7. Account and Gas Abstraction

Smart accounts with three programmable types (multisig, social recovery, session-based) support session keys with granular permissions and expiry, per-account spending rules, and denom allowlists. This enables wallet UX patterns impossible with standard accounts: dApp session keys for mobile, social recovery as a first-class account type, and programmable spend limits enforced at consensus. Gas abstraction removes the requirement to hold native QOR for fees -- users can pay in any accepted IBC-transferred token such as USDC or ATOM.

## Ecosystem

QoreChain ships with **18 custom modules** covering security (pqc), AI (ai, reputation, rlconsensus), consensus (qca), virtual machines (vm, svm, crossvm), tokenomics (burn, xqore, inflation), bridges (bridge, babylon, multilayer), governance extensions (abstractaccount, fairblock, gasabstraction), and rollups (rdk). Together with 27 standard framework modules, the chain registers **45 genesis modules** in an open-core architecture -- the protocol layer is fully open source, with optional proprietary extensions for enterprise deployments.
