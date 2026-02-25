# Changelog

All notable changes to QoreChain will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.1.0] - 2026-02-25

### Added
- **PQC Hybrid Signatures**: Dual Ed25519 + ML-DSA-87 signature verification via TX extensions
  - `HybridSignatureMode` enum: Disabled (0) / Optional (1, default) / Required (2)
  - `PQCHybridSignature` TX extension type carrying algorithm ID, PQC signature, and optional public key
  - `PQCHybridVerifyDecorator` ante handler with three-way verification logic:
    - Account with PQC key + extension → verify hybrid (classical + PQC)
    - No PQC key + extension with public key → auto-register + verify (onboarding path)
    - No PQC key + no extension → classical only (or reject if HybridRequired)
  - Auto-registration: wallets can attach PQC public key in the extension to register on first use
  - Events: `pqc_hybrid_verify`, `pqc_hybrid_auto_register`, `pqc_hybrid_classical_only`
- **SHAKE-256 Merkle Hash Foundation**: Post-quantum hash utilities for future IAVL tree replacement
  - `SHAKE256Hash(data, outputLen)` — variable-length XOF output
  - `SHAKE256Hash32(data)` — 32-byte fixed output
  - `SHAKE256ConcatHash(left, right)` — merkle internal node hash
  - `SHAKE256DomainHash(domain, data)` — domain-separated hashing
  - 11 unit tests with known vector verification
- **AI TEE Attestation Interfaces** (`x/ai/types/tee_interface.go`):
  - `TEEAttestation`, `TEEEnclaveStatus`, `TEEExecutionResult` structs
  - `TEEVerifier` and `TEEExecutor` interfaces for SGX/TDX/SEV-SNP/ARM CCA
- **AI Federated Learning Interfaces** (`x/ai/types/federated_interface.go`):
  - `FederatedUpdate`, `FederatedRoundConfig`, `FederatedRoundStatus`, `FederatedGlobalModel` structs
  - `FederatedCoordinator` interface for on-chain FL coordination
  - Support for FedAvg, FedProx, SCAFFOLD aggregation methods
- **qor_ RPC endpoint**: `qor_getHybridSignatureMode` — returns current mode, name, and description
- **CLI command**: `qorechaind query pqc hybrid-mode` — query current hybrid signature enforcement mode
- Unit tests: hybrid type validation, genesis validation, SHAKE-256 vectors, TEE/FL struct marshaling

### Changed
- Ante handler chain extended: `PQCVerify → PQCHybridVerify → AIAnomaly`
- `PQCKeeper` interface extended with `GetHybridSignatureMode()` and `IncrementHybridVerifications()`
- `PQCStats` extended with `TotalHybridVerifications` counter
- `Params` extended with `HybridSignatureMode` field (default: `optional`)
- Existing `pqc_verify` events now include `hybrid_mode` attribute for observability
- Genesis validation checks `HybridSignatureMode` is valid (0, 1, or 2)
- 4 new PQC error codes: `ErrHybridSigRequired`, `ErrHybridSigInvalid`, `ErrHybridModeDisabled`, `ErrInvalidHybridSig`
- `PQCHybridSignature` registered in amino codec for TX extension serialization

---

## [1.0.0] - 2026-02-25

### Added
- **x/burn module**: Central burn accounting with 9 burn mechanisms and EndBlocker fee distribution (40% validators, 30% burned, 20% treasury, 10% stakers)
  - Burn sources: tx_fee, governance_penalty, slashing_burn, bridge_fee, spam_deterrent, epoch_excess, manual_burn, contract_callback, cross_vm_fee
  - Real-time burn statistics with per-source tracking
  - Configurable burn ratio and distribution weights via governance params
- **x/xqore module**: Governance-boosted staking — lock QOR to mint xQORE (1:1)
  - Graduated exit penalties: 50% (<30d), 35% (30-90d), 15% (90-180d), 0% (>180d)
  - PvP rebase: penalties redistributed to remaining xQORE holders
  - Position tracking with lock height and lock time
  - Satisfies `rlconsensus.TokenomicsKeeper` interface for QDRW governance voting power
- **x/inflation module**: Epoch-based emission with year-over-year decay
  - Emission schedule: Y1: 17.5%, Y2: 11%, Y3-4: 7%, Y5+: 2%
  - Configurable epoch length (default: 100 blocks) and blocks-per-year
  - Epoch info tracking: current epoch, current year, total minted
- **qor_ RPC endpoints**: 4 new JSON-RPC methods
  - `qor_getBurnStats` — burn totals and per-source breakdown
  - `qor_getXQOREPosition(address)` — xQORE position lookup
  - `qor_getInflationRate` — current rate and epoch info
  - `qor_getTokenomicsOverview` — combined tokenomics dashboard
- Replaced `NilTokenomicsKeeper` with real xQORE adapter in RL consensus module
  - Compile-time assertions prove both stub and proprietary keepers satisfy `TokenomicsKeeper`
- Module account permissions: burn (Burner), xqore (Minter+Burner), inflation (Minter)
- Factory pattern wiring for all three modules (keeper, AppModule, ModuleBasic)

### Changed
- Module lifecycle ordering extended: burn → xqore → inflation → rlconsensus (BeginBlockers, EndBlockers, InitGenesis, ExportGenesis)
- Total registered genesis modules increased from 37 to 40

---

## [0.9.0] - 2026-02-25

### Added
- **RL Consensus Module** (x/rlconsensus): Reinforcement learning-based dynamic consensus parameter tuning
  - Go-native fixed-point MLP (25->256->256->5 architecture, ~73,733 parameters)
  - PPO inference engine with shadow/conservative/autonomous/paused agent modes
  - 25-dimension observation vector capturing chain state every 10 blocks
  - 5-dimension action space: block time, gas limit, gas price floor, pool weights
  - Multi-objective reward function (throughput, finality, decentralization, MEV, failed txs)
  - Circuit breaker: auto-reverts parameters if <50% blocks produced on time
  - Deterministic math utilities: Taylor series exp/ln, Newton sqrt, sigmoid approximation
  - CLI: query agent-status, observation, reward, params, policy; tx set-mode, resume, update-policy
- **Triple-Pool CPoS** (x/qca): Composite Proof-of-Stake with RPoS/DPoS/PoS validator pools
  - Reputation-weighted pool classification every 1000 blocks
  - Pool-weighted proposer selection with deterministic sortition
  - Configurable pool weights (default: RPoS 40%, DPoS 35%, PoS 25%)
- **Custom Bonding Curve** (x/qca): R(v,t) = beta * S_v * (1 + alpha * log(1+L_v)) * Q(r_v) * P(t)
  - Loyalty duration bonus via deterministic logarithm
  - Reputation quality factor Q clamped to [0.75, 1.25]
  - Configurable protocol phase multiplier (genesis=1.5, growth=1.0, mature=0.8)
- **Progressive Slashing** (x/qca): Escalating penalties with temporal decay
  - Formula: base_rate * 1.5^effective_count * severity_factor, capped at 33%
  - Half-life decay: 0.5^(blocks_since/100000) for each past infraction
  - Persistent slashing records with KV store iteration
- **QDRW Governance** (x/qca): Quadratic Delegation with Reputation Weighting
  - VP(v) = sqrt(staked + 2 * xQORE) * ReputationMultiplier(r)
  - Sigmoid reputation multiplier maps [0,1] to [0.5, 2.0]
  - TokenomicsKeeper stub interface for future xQORE integration
  - Starts disabled; governance-activatable
- **qor_ RPC Extensions**: 4 new JSON-RPC endpoints
  - qor_getRLAgentStatus, qor_getRLObservation, qor_getRLReward, qor_getPoolClassification

### Changed
- x/qca module extended with pool config, bonding curve config, slashing config, QDRW config
- QCA genesis state expanded with pool classifications and slashing records
- QCA keeper adds optional staking and RL consensus reader dependencies
- Total registered genesis modules remains at 37

---

## [0.8.0] - 2026-02-25

### Added
- **SVM Runtime** (x/svm): Full Solana Virtual Machine as the third execution environment
  - BPF program deployment and execution via Rust-backed executor
  - Account model: 32-byte addresses, lamports, data, owner, rent epoch
  - Program lifecycle: deploy, execute, with deterministic address derivation
  - Rent collection system with configurable exemption thresholds
  - SVM-specific ante decorators: compute budget validation, deployment size limits
  - Solana-compatible JSON-RPC server: `getAccountInfo`, `getBalance`, `getSlot`, `getMinimumBalanceForRentExemption`, `getVersion`, `getHealth`
  - CLI commands: `deploy-program`, `execute`, `create-account` (tx); `account`, `program`, `params`, `slot` (query)
  - Base58 address encoding/decoding for Solana compatibility
  - Optional PQC key registration for SVM accounts (`MsgRegisterSVMPQCKey`)
- **Rust qoresvm crate**: Native BPF execution engine
  - ELF loader with validation (magic, class, endianness, machine type, size limits)
  - BPF executor with configurable compute budget and instruction metering
  - Syscall stubs: `sol_log`, `sol_log_64`, `sol_sha256`, `sol_keccak256`, `create_program_address`
  - SPL program stubs: Token, Associated Token Account (ATA), Memo
  - Memory management with heap allocation and region mapping
  - Account serialization/deserialization for FFI boundary
  - 79 Rust unit tests (all passing)
- **Go FFI bridge** (x/svm/ffi): CGO bridge to Rust qoresvm library
  - `qore_svm_init`, `qore_svm_execute`, `qore_svm_validate_elf`, `qore_svm_free`, `qore_svm_version`
  - Platform-specific linker flags (macOS ARM64/AMD64, Linux AMD64/ARM64)
  - JSON-encoded execution result exchange between Go and Rust
- **CrossVM SVM extensions**: SVM as third target in cross-VM messaging
  - `VMTypeSVM` message type for EVM/CosmWasm → SVM calls
  - Async event-based bridge with callback injection pattern

### Changed
- Ante handler upgraded to triple routing: EVM path, QoreChain SDK path (with SVM decorators), SVM-aware compute budget validation
- x/crossvm module updated with SVM call handler and `ErrSVMExecution` error code
- SVMKeeper interface: 16 methods including `GetCurrentSlot`, `GetMinimumBalance`, `CollectRent`
- Factory pattern extended with SVM keeper, module, and ante decorator factory variables
- Total registered genesis modules increased from 35 to 36

---

## [0.7.0] - 2026-02-24

### Added
- **EVM Precompiles**: 6 custom precompiles exposing QoreChain SDK modules to Solidity contracts
  - PQC precompile: verify Dilithium-5 signatures from Solidity
  - AI precompile: query risk scores and AI verdicts on-chain
  - Reputation precompile: read validator reputation scores
  - Bridge precompile: initiate cross-chain transfers from EVM
  - Multilayer precompile: query layer status and route transactions
  - CrossVM precompile: call CosmWasm contracts from EVM
- Precompile address constants and ABI helpers (`x/vm/precompiles/`)
- Solidity interface files for all 6 custom precompiles
- Stub precompiles for community (non-proprietary) build
- Unit tests for stub precompiles
- Documentation: `docs/EVM_PRECOMPILES.md`

---

## [0.6.5] - 2026-02-24

### Fixed
- Moved AI sidecar and block indexer source code to private distribution channel
- Removed hardcoded AI model identifiers from Docker Compose; replaced with environment variable references
- Removed generated protobuf Go files from public sidecar directory (interface `.proto` retained)
- Removed SQL migration scripts from public indexer directory

### Changed
- AI sidecar and block indexer Docker services now reference pre-built container images (`ghcr.io/qorechain/ai-sidecar`, `ghcr.io/qorechain/block-indexer`)
- PostgreSQL migration volume mount removed from public compose (migrations bundled in indexer image)
- AI model configuration externalized to `AI_MODEL_ID` environment variable

### Security
- Proprietary AI inference prompts, fraud detection heuristics, and contract analysis logic are no longer exposed in public repository
- QCAI Backend integration details removed from public configuration files

---

## [0.6.0] - 2026-02-23

### Added
- **Algorithm Agility Framework**: Governance-controlled multi-algorithm PQC management
  - `AlgorithmID` type with lifecycle states: active, migrating, deprecated, disabled
  - `AlgorithmInfo` struct: ID, name, category, NIST level, key/sig sizes, status
  - `AlgorithmRegistry` in Rust: trait-based dispatch (`PQCSignatureScheme`, `PQCKEMScheme`)
  - Static dispatch via `LazyLock` singleton for zero-overhead FFI calls
- **Algorithm-aware FFI exports** (5 new C functions in libqorepqc):
  - `qore_pqc_keygen()`, `qore_pqc_sign()`, `qore_pqc_verify()`
  - `qore_pqc_algorithm_info()`, `qore_pqc_list_algorithms()`
- **Algorithm-aware key registration** (`MsgRegisterPQCKeyV2`): Explicit algorithm selection
- **Dual-signature key migration** (`MsgMigratePQCKey`): Proves ownership of both old and new keys
- **Governance messages** for algorithm lifecycle management:
  - `MsgAddAlgorithm`: Add new PQC algorithm via governance
  - `MsgDeprecateAlgorithm`: Start migration period (default: 1,000,000 blocks / ~69 days)
  - `MsgDisableAlgorithm`: Emergency disable with reason
- **Multi-algorithm ante decorator**: Dispatches verification by account's registered AlgorithmID
  - Handles active, migrating, deprecated, and disabled algorithm states
  - Backward-compatible with pre-v0.6.0 accounts (defaults to Dilithium-5)
- **CLI commands** for PQC module:
  - Query: `algorithms`, `algorithm`, `account`, `stats`, `params`, `migration`
  - Tx: `register-key` (legacy), `register-key-v2` (algorithm-aware), `migrate-key`
- **Rust PQC library** (qorepqc v0.6.0): Complete rewrite with algorithm abstraction
  - `PQCSignatureScheme` trait: keygen, sign, verify with algorithm-specific dispatch
  - `PQCKEMScheme` trait: keygen, encapsulate, decapsulate
  - `AlgorithmRegistry` with `AlgorithmMeta` repr(C) metadata
  - 32 Rust unit tests (all passing)
  - Optimized release: 385KB dylib, LTO fat, codegen-units=1
- Unit tests: 28 Go tests for types (algorithm, genesis, messages)
- Documentation: `docs/ALGORITHM_AGILITY.md`

### Changed
- `PQCAccountInfo` struct redesigned: `DilithiumPubkey` replaced with `PublicKey` + `AlgorithmID`
  - Added `MigrationPublicKey` and `MigrationAlgorithmID` for dual-key mode
- `PQCStats` extended with `TotalDualSigVerifies` and `TotalKeyMigrations` counters
- `GenesisState` extended with `Algorithms` and `Migrations` fields
  - Default genesis registers Dilithium-5 (ID=1) and ML-KEM-1024 (ID=2)
- `Params` extended with `DefaultMigrationBlocks` and `DefaultSignatureAlgo`
- `PQCClient` interface extended with algorithm-aware methods
- `PQCKeeper` interface extended with algorithm registry and migration methods
- FFI bridge updated with algorithm-aware Go functions
- 10 new error codes (ErrInvalidAlgorithm through ErrUnauthorizedGovAction)

---

## [0.5.0] - 2026-02-23

### Added
- **EVM Runtime**: Full Ethereum Virtual Machine compatibility
  - x/vm (EVM execution engine), x/feemarket (EIP-1559 gas pricing), x/erc20 (token pairs), x/precisebank (decimal precision)
  - JSON-RPC server on port 8545 (HTTP) and 8546 (WebSocket)
  - Standard `eth_`, `web3_`, `net_`, `txpool_` namespaces
  - Dual ante handler routing: EVM path and QoreChain SDK path (PQC + AI + CosmWasm decorators)
  - EVM precompile registration for SDK modules (bank, staking, distribution, gov, IBC transfer, ERC-20)
  - EVM start command with JSON-RPC configuration flags
- **CosmWasm Runtime**: WebAssembly smart contract support
  - x/wasm module with full upload/instantiate/execute/migrate lifecycle
  - CosmWasm ante decorators: LimitSimulationGas, CountTX, GasRegister, TxContracts
  - Configurable upload permissions (default: Everybody)
- **IBC v2**: Inter-Blockchain Communication
  - IBC core + ICS-20 token transfers
  - Foundation for cross-chain interoperability
- **x/crossvm module**: Cross-VM communication between EVM and CosmWasm
  - Synchronous path: EVM precompile at `0x...0901` calls CosmWasm contracts directly
  - Asynchronous path: Event-based message queue processed in EndBlocker
  - Message lifecycle: pending, executed, failed, timed_out
  - Configurable parameters: max message size (64KB), queue size (1000), timeout (100 blocks)
- **`qor_` JSON-RPC namespace**: QoreChain-specific RPC methods
  - `qor_getPQCKeyStatus(address)` — PQC key registration status
  - `qor_getAIStats()` — AI module statistics and configuration
  - `qor_getCrossVMMessage(msgId)` — Cross-VM message status
  - `qor_getReputationScore(validator)` — Validator reputation breakdown
  - `qor_getLayerInfo(layerId)` — Multilayer chain info
  - `qor_getBridgeStatus(chainId)` — Bridge connection status
- Documentation: `docs/EVM.md`, `docs/CROSSVM.md`
- Updated `docs/API_REFERENCE.md` with JSON-RPC, cross-VM, and multilayer endpoints
- Unit tests for x/crossvm types (params, messages, genesis, msg validation)

### Changed
- Ante handler rewritten to dual routing architecture (EVM + QoreChain SDK paths)
- Total registered genesis modules increased from 26 to ~35
- Encoding config updated to EVM-compatible encoding (supports MsgEthereumTx signing)
- Docker Compose: added EVM JSON-RPC ports (8545, 8546)

### Fixed
- Dockerfile updated for public build
- Go version updated to 1.26 in all CI workflows

---

## [0.3.9] - 2026-02-22

### Added
- **x/multilayer module**: Multi-layer architecture support (Main Chain + Sidechains + Paychains)
  - Sidechain registration and lifecycle management (max 10 active, 1000 QOR min stake)
  - Paychain registration for high-frequency microtransactions (max 50 active, 100 QOR min stake, 500ms target block time)
  - Hierarchical Commitment Schemes (HCS) for PQC-signed state anchoring to Main Chain
  - QCAI-powered heuristic transaction router with 4-factor scoring (congestion 0.3, capability 0.4, cost 0.2, latency 0.1)
  - Cross-Layer Fee Bundling (CLFB) — single fee covers execution across all layers in TX path
  - Fraud proof challenge mechanism for state anchors (24-hour challenge period)
  - PQC-signed (Dilithium-5) aggregate signatures on all state anchors
  - Layer lifecycle state machine (Proposed, Active, Suspended, Decommissioned)
  - Configurable routing confidence threshold (default 0.6)
- Proto definitions: `layer.proto`, `tx.proto`, `query.proto`, `genesis.proto`
- 7 transaction types: RegisterSidechain, RegisterPaychain, AnchorState, RouteTransaction, UpdateLayerStatus, ChallengeAnchor, UpdateParams
- 7 query types: Layer, Layers, Anchor, Anchors, RoutingStats, SimulateRoute, Params
- CLI commands for all multilayer operations
- REST/gRPC API endpoints for all multilayer queries
- Genesis configuration with 10 tunable parameters
- Documentation: `docs/MULTILAYER.md`

### Changed
- Total registered genesis modules increased from 25 to 26

---

## [0.2.1] - 2026-02-20

### Added
- **Polygon PoS bridge** (`polygon_bridge.go`): EVM-compatible, 128 block confirmations (~256s), native asset POL, supports USDC/USDT/WETH
- **Arbitrum One bridge** (`arbitrum_bridge.go`): L2, 64 block confirmations (~16s at 0.25s/block), native asset ETH, supports USDC/ARB/USDT
- **Sui bridge** (`sui_bridge.go`): Move VM chain, 3 checkpoint confirmations (~9s at 3s/checkpoint), native asset SUI, supports USDC
- New `ChainTypeSui` constant for Move VM address validation (0x + 64 hex chars, 32 bytes)
- Total supported bridge chains expanded from 6 to 9

### Changed
- Updated `DefaultChainConfigs()` with Polygon, Arbitrum, Sui entries
- Updated `EstimateConfirmationTime()` with polygon (256s) and arbitrum (16s) cases
- Unified bridge protocol documentation to "QCB Native + IBC"
- Path optimizer automatically routes through new chains via KVStore chain configs

### Fixed
- Public repo CI workflows: build configuration corrected
- Public repo Dockerfile: removed unnecessary library references
- Go version pinned to 1.26 in all CI workflows

### Security
- All new chain bridges use PQC-signed attestations (Dilithium-5)
- Circuit breaker protection extended to Polygon, Arbitrum, Sui
- Sui address validation enforces 32-byte Move VM format

---

## [0.1.0] - 2026-02-19

### Added

#### Core Blockchain
- Initialized QoreChain testnet project structure
- Go module: `github.com/qorechain/qorechain-core`
- Chain ID: `qorechain-diana`
- Token: QOR (display) / uqor (base, 10^6)
- Bech32 prefixes: `qor` (accounts), `qorvaloper` (validators)
- Binary build and chain initialization verified

#### PQC Rust Library
- **Rust crate `qorepqc`**: Post-quantum cryptographic primitives
  - Dilithium-5 (NIST FIPS 204): keygen, sign, verify
  - ML-KEM-1024 (NIST FIPS 203): keygen, encapsulate, decapsulate
  - Quantum random beacon (ChaCha20-based CSPRNG seeded from OS entropy)
  - C FFI bridge via `cbindgen` for Go interop
- Compiled `libqorepqc` for 4 platforms: macOS ARM64/AMD64, Linux AMD64/ARM64
- 20/20 Rust unit tests passing
- Dilithium-5 actual sizes: PUBKEY=2592, PRIVKEY=4896, SIG=4627 bytes

#### x/pqc Module
- **Go-to-Rust FFI bridge** (`x/pqc/ffi/bridge.go`):
  - `DilithiumKeygen()`, `DilithiumSign()`, `DilithiumVerify()`
  - `MLKEMKeygen()`, `MLKEMEncapsulate()`, `MLKEMDecapsulate()`
  - `QuantumRandom()`
  - CGO directives with platform-specific linker flags
- **PQC Keeper**: Key registration, verification delegation
- **PQC AnteDecorator**: Signature verification in AnteHandler chain
  - PQC-primary: Dilithium-5 verification via Rust FFI
  - Classical fallback: ECDSA verification via Go native
  - Emits `classical_fallback_used` event when fallback path taken
- Message types: `MsgRegisterPQCKey`, `MsgStorePQCKey`

#### x/ai Module
- **AI-Native Transaction Processing Engine**
- **Heuristic Engine**: Fast-path rule-based AI
- **Smart Router**: Transaction routing optimization
- **Anomaly Detector**: Z-score analysis on amount, gas, frequency patterns with sliding-window tracking
- **Risk Scorer**: Multi-dimensional risk assessment
- **Fraud Detector**: Statistical isolation forest implementation detecting amount anomalies, gas manipulation, rapid-fire transactions, time-of-day patterns
- **Fee Optimizer**: Dynamic fee prediction
- **Network Optimizer**: Network parameter tuning
- **Resource Allocator**: Compute resource management
- **AI AnteDecorator**: Integrates AI verdicts into AnteHandler chain (ALLOW, FLAG, REJECT with confidence scores)
- ~2,050 lines of Go across 11 keeper files

#### x/reputation Module
- **Multi-Dimensional Validator Scoring**
- Formula: `R_i = alpha*S_i + beta*P_i + gamma*C_i + delta*T_i`
  - S_i: Stake weight, P_i: Performance, C_i: Community trust, T_i: Transaction validation accuracy
- Temporal decay: scores decay over time without active participation
- Configurable weights: `alpha=0.3, beta=0.3, gamma=0.2, delta=0.2`

#### x/qca Module
- **QoreChain Consensus Algorithm Hooks**
- Reputation-weighted random proposer selection via PrepareProposal/ProcessProposal ABCI hooks
- Heuristic selector with weighted random selection using cumulative distribution function
- Falls back to uniform random if no reputation data available

#### AI Sidecar Service
- **Separate Go module** (`sidecar/`) — independent deployment
- **gRPC server** on port 50051 with 7 RPC endpoints:
  - `AnalyzeTransaction`, `DeepAnalyzeContract`, `GenerateContract`, `AnalyzeFraud`, `OptimizeNetwork`, `PredictFees`, `HealthCheck`
- **QCAI Backend client**: Multi-tier inference with fast and balanced analysis paths
- **Fraud analyzer**: Deep pattern analysis
- **Contract auditor**: Security vulnerability scanning
- **Contract generator**: AI contract generation
- **Network advisor**: Optimization recommendations
- **Fee predictor**: Historical fee analysis
- **Embedded heuristics**: Z-score anomaly detection, risk scoring, request routing
- Proto definition: `sidecar/proto/ai_sidecar/v1/service.proto` (252 lines)
- ~1,500 lines of Go server code

#### Docker Compose + Genesis
- **Dockerfile**: Multi-stage build for chain binary
- **docker-compose.yml**: Full deployment stack
  - `qorechain-node`: Chain binary (RPC :26657, P2P :26656, REST :1317, gRPC :9090)
  - `ai-sidecar`: AI service (gRPC :50051)
  - `block-indexer`: Event indexer
  - `postgres`: Database for indexer (:5432)
  - `prometheus`: Metrics collection (:9090)
  - `grafana`: Dashboard visualization (:3000)
- **`scripts/init-testnet.sh`**: Genesis initialization script
- **`.env.example`**: Environment variable template

#### Block Indexer
- **Separate Go module** (`indexer/`) — independent deployment
- **WebSocket listener**: Subscribes to new blocks, REST API fallback, reconnection with exponential backoff
- **Transaction processor**: Parses blocks, extracts events, tracks PQC signatures, bridge operations, AI verdicts
- **Database layer**: PostgreSQL with blocks, transactions, events tables
- SQL migrations for initial schema

#### CI/CD and Documentation
- **GitHub Actions workflows**: build + test, binary releases, Docker image build + push to GHCR
- **Documentation**: README, ARCHITECTURE, AI_ENGINE, API_REFERENCE, BRIDGE, PQC_INTEGRATION, RUNNING_TESTNET
- **Community files**: CONTRIBUTING, SECURITY, LICENSE (Apache 2.0), issue templates, PR template

### AnteHandler Chain
```
SetUpContext -> CircuitBreaker -> PQCVerify -> AIAnomaly -> Extension ->
ValidateBasic -> TxTimeout -> UnorderedTx -> Memo -> DeductFee ->
SetPubKey -> ValidateSigCount -> SigGasConsume -> SigVerify -> IncrementSequence
```

### Security
- All bridge validator attestations signed with Dilithium-5
- ML-KEM-1024 commitments for bridge operations
- Circuit breaker protection for bridge transfers (rate limiting, max amount)
- 24-hour challenge period for large withdrawals
- Real-time fraud detection in AnteHandler chain
- AI-powered anomaly detection (Z-score, isolation forest)
- PQC-primary signature verification (classical ECDSA fallback)

### Infrastructure
- 3 separate Go modules: `qorechain-core/`, `sidecar/`, `indexer/`
- 108 Go source files, ~14,000 lines of Go
- PQC Rust library: 4-platform cross-compilation
- Docker Compose: 6-service deployment stack
- GitHub Actions: 3 CI/CD workflows (build, release, docker)
- Bech32: `qor1...` (accounts), `qorvaloper...` (validators)
- 25 registered modules in genesis

---

## [0.0.0] - 2026-02-19

### Added
- `QORECHAIN_TESTNET_V1_ARCHITECTURE.md` — Complete build specification (14 sections)
- Decision matrix: 11 key architecture decisions documented
- System topology diagram and data flow specifications
- Module specifications for all 5 custom modules
- Testing strategy and upgrade path documentation
