# Glossary

Alphabetical reference of terms, abbreviations, and acronyms used throughout the QoreChain documentation.

---

**BPF** -- Berkeley Packet Filter. The bytecode format used by the SVM runtime to execute on-chain programs. Programs are compiled to BPF before deployment.

**CLFB** -- Cross-Layer Fee Balancing. A mechanism within the multilayer architecture that dynamically adjusts fees across sidechains and paychains to maintain equilibrium and prevent congestion on any single layer.

**CPI** -- Cross-Program Invocation. A mechanism in the SVM runtime that allows one deployed program to call another program within the same transaction context.

**CPoS** -- Classified Proof-of-Stake. QoreChain's consensus classification system that groups validators into three pools (Emerald, Sapphire, Ruby) based on reputation scores. Each pool has distinct weights in the proposer selection algorithm.

**DA** -- Data Availability. The guarantee that transaction data published to the chain can be retrieved by any node. The RDK module provides native DA for rollups, storing blobs on-chain with configurable retention periods.

**DPoS** -- Delegated Proof-of-Stake. A consensus mechanism where token holders delegate their stake to validators who produce blocks on their behalf. QoreChain extends DPoS with reputation-weighted classification (CPoS).

**EIP-1559** -- Ethereum Improvement Proposal 1559. A transaction fee model that uses a base fee (burned) plus a priority fee (paid to validators). QoreChain implements EIP-1559-style fee mechanics in its EVM runtime.

**HCS** -- Hybrid Cryptographic Signatures. QoreChain's dual-signature system where transactions carry both a classical signature (Ed25519) and a post-quantum signature (ML-DSA-87), providing cryptographic security against both classical and quantum adversaries.

**IBC** -- Inter-Blockchain Communication. A protocol for authenticated message passing between independent blockchains. QoreChain supports IBC channels for cross-chain token transfers and data relay.

**MLP** -- Multi-Layer Perceptron. A class of neural network used in the AI sidecar for pattern recognition in fraud detection and anomaly scoring.

**ML-DSA-87** -- Module-Lattice Digital Signature Algorithm (security level 5). The NIST-standardized post-quantum digital signature scheme used by QoreChain for quantum-resistant transaction signing. Produces signatures of 4,627 bytes with public keys of 2,592 bytes.

**ML-KEM-1024** -- Module-Lattice Key Encapsulation Mechanism (security level 5). A NIST-standardized post-quantum key encapsulation scheme available in QoreChain's PQC algorithm registry for future encrypted communication channels.

**MEV** -- Maximal Extractable Value. The profit that can be obtained by reordering, inserting, or censoring transactions within a block. QoreChain's FairBlock module (tIBE encryption) and 5-lane TX prioritization mitigate MEV extraction.

**PQC** -- Post-Quantum Cryptography. Cryptographic algorithms designed to be secure against attacks from both classical and quantum computers. QoreChain implements PQC through its `x/pqc` module with ML-DSA-87 as the primary signature scheme.

**PPO** -- Proximal Policy Optimization. A reinforcement learning algorithm used by the RL Consensus module to optimize chain parameters (block size, gas limits, validator set size) based on observed network conditions.

**PvP Rebase** -- Player versus Player Rebase. A mechanism in the xQORE module where penalties from early unlock are redistributed proportionally to remaining locked stakers, rewarding long-term commitment.

**QCB** -- QoreChain Bridge. QoreChain's native bridge protocol for connecting to non-IBC chains (e.g., Ethereum, Bitcoin, Solana, Avalanche). QCB uses a federated validator set for cross-chain attestation.

**QCAI** -- QoreChain Artificial Intelligence. The umbrella term for QoreChain's AI subsystem, including the on-chain heuristic engine (`x/ai` module) and the off-chain AI sidecar service that provides advanced inference capabilities.

**QDRW** -- QoreChain Dynamic Reward Weighting. A governance mechanism that allows the RL agent (under governance approval) to dynamically adjust reward distribution weights across validator pools, optimizing for network health metrics.

**RDK** -- Rollup Development Kit. QoreChain's native framework for deploying and managing rollups with four settlement paradigms (optimistic, zk, pessimistic, sovereign), five preset profiles, and integrated data availability.

**RL** -- Reinforcement Learning. A machine learning approach where an agent learns optimal actions through trial and reward. QoreChain's RL Consensus module uses RL to dynamically tune chain parameters.

**RPoS** -- Reputation-based Proof-of-Stake. The overarching consensus model combining DPoS delegation with reputation scoring. Validators earn reputation through uptime, participation, and community contributions, which influences their block proposal frequency.

**SHAKE-256** -- A variable-output-length hash function from the SHA-3 family. QoreChain uses SHAKE-256 as its foundational hash function for PQC-related operations, including key derivation and address computation.

**SNARK** -- Succinct Non-interactive Argument of Knowledge. A type of zero-knowledge proof that can be verified quickly with a small proof size. Supported as a settlement paradigm in the RDK module for zk-rollups.

**STARK** -- Scalable Transparent Argument of Knowledge. A zero-knowledge proof system that requires no trusted setup and is quantum-resistant. Available as an alternative proof system for zk-rollup settlement in the RDK.

**SVM** -- Solana Virtual Machine. A high-performance execution environment for BPF programs. QoreChain integrates the SVM as one of three supported runtimes alongside the EVM and CosmWasm.

**TEE** -- Trusted Execution Environment. A secure area of a processor that ensures code and data are protected from external access. QoreChain's PQC module supports TEE attestation for key generation proofs.

**tIBE** -- Threshold Identity-Based Encryption. A cryptographic scheme where a message can only be decrypted when a threshold of parties collaborate. Used by the FairBlock module to encrypt transaction contents until block finalization, preventing MEV extraction.

**uqor** -- The base denomination of the QOR token. 1 QOR = 1,000,000 uqor (10^6). All on-chain amounts, fees, and staking values are denominated in uqor.

**xQORE** -- The governance staking derivative of QOR. Users lock QOR to receive xQORE, which grants enhanced governance voting power and earns PvP rebase rewards from early-unlock penalties.
