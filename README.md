# QoreChain - Quantum-Safe, AI-Native Layer 1 Blockchain

[![Build](https://github.com/qorechain/qorechain-core/actions/workflows/build.yml/badge.svg)](https://github.com/qorechain/qorechain-core/actions)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-3.2.3-green.svg)](https://github.com/qorechain/qorechain-core/releases/tag/v3.2.3)

QoreChain is the first Layer 1 blockchain with **post-quantum cryptography at genesis**, **AI-native consensus optimization**, a **triple-VM runtime** executing EVM, CosmWasm, and SVM (Solana Virtual Machine) programs on a single chain, **post-quantum authorisation on every lane** including the EVM lane, a **native on-chain AMM** with constant-product and stable-swap pricing, a **complete tokenomics engine** with burn mechanics, governance-boosted staking, and governance-controlled, capped emission, **44 direct cross-chain connections** spanning IBC (with foundation for next-generation IBC v2), EVM, Move, UTXO, Cairo, UNL, SCP, Hashgraph, and account-model ecosystems, a **Rollup Development Kit (RDK)** enabling one-click deployment of application-specific rollups with four settlement paradigms, a **license-gated multi-chain validator bridge** for cross-chain operations across 37 chain configurations, a **light node network** with a bounded, accounted reward pool, and a **unified account identity** where one key holds a single balance and signs across the EVM, Cosmos, and SVM interfaces (`0x…` / `qor1…` / base58). Built on Cosmos SDK v0.53.8 with 20 custom application modules and 48 registered genesis modules.

## Innovations

### Post-Quantum Authorisation on the EVM Lane (v3.2.0, extended in v3.2.2)

Cosmos-lane transactions carry a mandatory ML-DSA-87 hybrid signature, and from v3.2.0 the same guarantee holds on the EVM lane. Because EVM wallets such as MetaMask sign and broadcast in one step, there is no point at which a second signature could be attached to an Ethereum transaction. Instead, an account authorises its EVM activity in advance with a post-quantum signed Cosmos-lane message:

- **Bounded authorisation window** - `MsgOpenEVMWindow` admits EVM transactions from the sender for a bounded number of blocks (at most 17,280), a bounded number of transactions (at most 1,000) and a bounded total outflow that counts both transferred value and the maximum fee each transaction could pay. Opening replaces any existing window; `MsgCloseEVMWindow` revokes it immediately. Inside the window, EVM wallets work unmodified. An account without a registered post-quantum key cannot use the EVM lane at all, so the classical key alone can never move value.
- **EIP-7702 delegation consent (v3.2.2)** - an EIP-7702 authorization only takes effect if the authority account gave its own post-quantum consent with `MsgAuthorizeEVMDelegation` (single-use, time-bounded, revoked by closing the window). Value leaving a delegated account is charged to that account's own window, and `MsgClearEVMDelegation` removes a delegation.
- **Queries and CLI** - `GET /qorechain/pqc/v1/evm_window/{address}`, `GET /qorechain/pqc/v1/evm_delegation_consent/{address}`, and `qorechaind tx pqc open-evm-window` / `close-evm-window` / `authorize-evm-delegation` / `clear-evm-delegation`.
- **Circuit breaker coverage** - the EVM lane runs through the circuit breaker like every other message, and from v3.2.2 it also runs the same per-sender rate limit, anomaly screening and burn accounting as the Cosmos lane.

### Protocol Feature Switches (v3.2.0)

Consensus-affecting fixes ship behind named **protocol features** recorded in the `x/pqc` store. A release's upgrade handler turns its features on at the governance plan height; a chain that starts on the release turns them on at genesis, exactly as listed in its genesis file. Below the plan height nothing changes, so a new binary replays history byte-identically and new nodes can join by block sync as well as by state sync. v3.2.0 introduced 7 features and v3.2.2 adds 26 (see `x/pqc/types/features.go`).

### Synchronous Cross-VM Calls (v3.2.0)

`MsgCrossVMCall` now executes the call inside the transaction and returns the callee's answer; queuing is opt-in (`async`). The source VM and sender are derived from the execution context rather than taken from the message, nested calls are bounded in depth and re-entry into a frame that is still executing is refused, the EVM leg is bounded by the gas the caller actually provisioned, and calls that attach funds to a lane that cannot carry them are refused instead of silently dropping the funds. CosmWasm contracts can dispatch the message themselves and read the answer from the submessage reply.

### Governance-Controlled Emission and Fee Safeguards (v3.1.94 - v3.2.2)

- **`x/inflation` is governable** - `MsgUpdateParams` (governance authority) controls the emission parameters. Besides the decaying schedule, a fixed per-epoch emission mode is available, with a hard cumulative cap (`max_total_emission`) after which the module stops minting. From v3.2.2 an emission cap of zero or an omitted cap is refused rather than read as unlimited, and the parameters, emission to date and cap are served by the module's query service.
- **Stranded staker share recovered** - the staker share of fees now reaches stakers through the distribution module. The share that accumulated before the fix is recorded at the upgrade height and can only be released by a governance vote (`MsgReleaseStrandedRewards`, `GET /qorechain/burn/v1/stranded_rewards`).
- **Authority checks everywhere** - every administrative message whose signer is its own `authority` field verifies that the authority really is governance (`x/govauth`); a test walks the protobuf definitions so a new admin message cannot ship ungated.

### Validator Admission (v3.2.0, enforced on every path in v3.2.2)

Creating a validator requires an active `validator_operator` license and a self-bond of at least 100,000 QOR, and an operator must keep its self-bond at that floor. From v3.2.2 the staking message server applies this rule itself, so it holds on every dispatch path (including authz and multi-message transactions), and `MsgUnjail` is refused while the operator's self-bond is under the floor. An operator may still exit completely.

### Unified Account Identity - One Key, Three Interfaces (v3.1.83)

A single account key is a first-class citizen on all three of QoreChain's execution interfaces at once. An account created EVM-native - its 20-byte address is the keccak-256 derivation of its secp256k1 public key - is rendered three ways: `qor1…` (Cosmos bech32), `0x…` (EVM), and a base58 SVM address. All three resolve to the **same 20 bytes and the same balance**, so a deposit to any form lands in one account. The same key also **signs** on every interface: EVM transactions as usual, and Cosmos-lane transactions via the `eth_secp256k1` scheme - including the mandatory post-quantum hybrid signature (ML-DSA-87 / FIPS-204) the ante chain requires. This is delivered as a consensus-neutral upgrade (no re-genesis) that registers the EVM key type on the application's interface registry so eth-native accounts can sign standard Cosmos transactions; classic coinType-118 accounts are unaffected. The `@qorechain/wallet-adapter` package generates the three-address identity and produces both the classical and hybrid `eth_secp256k1` signatures.

### Linked-Wallet Authenticators - Scoped, Spend-Limited Delegation (v3.1.85)

An account owner can link an external wallet key (for example an ed25519 key from a Solana-style wallet, or an Ethereum key by its address) to their account under **least-privilege, spend-limited, revocable, time-bounded** terms, and that key can then spend from the account without ever holding the account's own signing key. Both signing schemes a browser wallet can produce are accepted - a raw ed25519 signature over the action, and an Ethereum `personal_sign` (EIP-191) verified by `ecrecover` against the linked address - so a wallet that exposes only an address and `personal_sign` works with no extra tooling. The `x/abstractaccount` module enforces this at execution on all three lanes:

- **Canonical permission taxonomy** - every action maps to a permission string (`send`, `delegate`, `vote`, `evm`, `svm`, `amm`, `ibc`, …). Authorization is **fail-closed**: an unmapped message and any permission the linked key was not granted are rejected, and account-management actions (registering, revoking or rotating keys) can never be delegated. Clients read the taxonomy from the on-chain `PermissionSchema` query - available over gRPC and REST (`GET /qorechain/abstractaccount/v1/permission_schema`) - so scopes are never hardcoded.
- **On-chain spending limits on every lane** - a per-transaction cap and a rolling daily cap (per linked key, per denomination) are enforced and accumulated in state. Value is checked in the handler for the native and EVM lanes and from the realized balance change for the SVM lane, so the same limit applies whichever lane a linked key spends on.
- **Three execution lanes** - `MsgExecuteEVM` performs an authenticator-authorized EVM call or transfer from the account's EVM address; `MsgExecuteCosmos` performs a native transfer from the account; and the SVM lane authorizes an SVM instruction from the same account. A relayer submits and pays the fee, so the account's own key need not be online; each is bound to the chain-id, account, key, action and a nonce so a signature cannot be replayed across chains, accounts, keys or transactions.

This closes on-chain session-key enforcement end-to-end. It is delivered as an additive change (new message types, no re-genesis) coordinated through governance on production networks.

### Same-Algorithm Post-Quantum Key Rotation (v3.1.85)

`MsgRotatePQCKey` lets an account replace its ML-DSA-87 key with a new key of the **same** algorithm - to retire a compromised key or move a key onto the standard derivation - without a full algorithm migration. The message is **dual-signed**: the old key proves ownership and the new key proves control, over domain-separated bytes binding the chain-id, algorithm, account and both keys. Rotation can never be delegated to a linked key, and after it lands the old key no longer matches the registered key, so the same request cannot be replayed. CLI: `qorechaind tx pqc rotate-key`.

### Native AMM Module (v3.0.0)

The `x/amm` module provides on-chain automated market making with two pricing curves:

- **Constant-product** (`x*y=k`) - Uniswap-V2-style pools for the long tail of token pairs
- **Stable-swap** - Curve-style invariant for low-slippage stable-pair swaps, solved via deterministic Newton iteration

Eight messages cover the complete pool lifecycle: `MsgCreatePool`, `MsgAddLiquidity`, `MsgRemoveLiquidity`, `MsgSwapExactIn`, `MsgSwapExactOut`, `MsgPausePool`, `MsgResumePool`, `MsgSetParams`. Per-pool pause via governance, module-wide kill switch, configurable swap fee with LP-accrual + protocol-fee split, slippage cap, and pool creation fee burned through the standard burn engine. A cross-VM hook lets EVM contracts (and SVM via the existing precompile interface) route swap calls into the AMM, while keeping on-chain routing decisions deterministic. All math uses fixed-point integer arithmetic - zero floating-point in any consensus path. From v3.2.2 a swap must name both of the pool's denominations, stable pools are priced with the stable invariant in both swap directions, the protocol fee is collected and routed through `x/burn`, and the `max_swap_impact_bps` cap is enforced.

### IBC Eureka v2 Foundation (v3.0.0)

A `ChainArchitecture` enum on every chain configuration disambiguates classic IBC vs the next-generation IBC v2 stack. New IBC chain onboardings from v3.0.0 forward default to IBC v2 with a configurable client type; existing chains continue on the classic stack and can be migrated by governance proposal. The `x/bridge` module ships public-side packet types and handler-hook interfaces for the next-generation stack and for ICS-27 (Interchain Accounts), ICS-29 (Fee Middleware), and ICS-721 (NFT-IBC) so the validator's cross-network operations can extend cleanly as the upstream modules mature.

### Cross-Network Expansion (v3.0.0)

QoreChain v3.0.0 expands the bridge surface to **37 default chain configurations** across **17 chain architectures** - adding five new architecture families (Cairo VM L2, XRP Ledger UNL, Stellar Consensus Protocol, Hashgraph, Pure Proof-of-Stake) and twenty new chain configurations (zkSync Era, Linea, Scroll, Starknet, Blast, Mantle, Hyperliquid, Berachain, Sonic, Sei, Monad, Plasma, XRPL, Stellar, Hedera, Algorand, Injective, Filecoin FVM, Cronos, Kaia). Each new architecture has a dedicated bridge handler with chain-appropriate confirmation rules; EVM-family chains share the canonical EVM handler with per-chain config injection. The license surface scales accordingly to **74 feature IDs** (1 umbrella + 36 per-chain bridge + 37 per-chain validator) covering all current and previously-onboarded chains.

### Multi-Node Devnet Harness (v3.0.0)

A two-validator local devnet ships with the repo via `docker-compose.devnet.yml` and helper scripts under `scripts/devnet/`. The genesis-creating validator and the joining peer use auto-discovery (genesis fetch + node-ID handshake) so a new operator can boot a working two-node consensus in one command. The included smoke runner exercises block production / finality, liveness slashing, and the bridge sidecar JSON-RPC handshake.

### Light Node Network (v2.0.0)

The `x/lightnode` module enables a decentralized network of light nodes that contribute to chain availability and earn rewards proportional to their uptime and delegated stake. Light node operators register on-chain, send periodic heartbeats to prove liveness, and receive a share of block fees distributed automatically by the EndBlocker.

- **Stake-Weighted Rewards** - Distribution is proportional to `delegated_stake * uptime_factor`, rewarding both commitment and reliability
- **Heartbeat Grace Period** - Configurable tolerance window before marking nodes inactive, accommodating transient network disruptions
- **Auto-Deactivation** - Nodes missing heartbeats beyond the grace period are automatically marked inactive and excluded from rewards until they resume
- **3% Fee Share** - Light node operators receive 3% of block fees, taken from the validator share
- **Accounted Reward Pool (v3.2.0)** - The module balance is split into reserved, outstanding and distributable amounts. Each block hands out only what accrued since the previous settlement, bounded per active node by `max_reward_per_block_per_node` and scaled by uptime; unearned accrual goes to the reserve, which moves only by governance (`MsgReleaseLightNodeReserve`). The split is served at `GET /qorechain/lightnode/v1/reward_pool`.
- **Forward-Accrued Uptime (v3.1.90)** - A governance change to the heartbeat interval only affects blocks after it, so a node's past uptime keeps the rating it earned

### License-Gated Validator Bridge

The `x/license` module provides an on-chain license registry that controls access to extended chain features. Each QoreChain validator can operate as a bridge watcher or full external validator for supported chains - provided they hold the appropriate license.

- **76 Feature IDs** - The `qcb_bridge` umbrella, granular licenses for bridge watching (36 `bridge_*`: `bridge_ethereum`, `bridge_solana`, …) and validator operations (37 `validator_*`: `validator_ethereum`, `validator_solana`, …) across all supported chains, including dedicated licenses for the eight IBC-connected chains, plus the `validator_operator` and `lightnode_operator` operator licenses
- **Expiry Honoured Everywhere** - The license `Check` query and the sidecar both treat a license past its `expires_at` as inactive (v3.2.1)
- **License Lifecycle** - Grant → Active → Suspended/Revoked/Expired with governance-controlled administration
- **Auto-Expiry** - EndBlocker automatically expires licenses past their TTL, ensuring stale grants don't persist
- **Sidecar Orchestration** - Licensed validators can run chain-specific sidecar containers managed by a built-in orchestrator that handles container lifecycle, health monitoring, and credential exchange

### Application-Specific Rollup Deployment (v1.3.0)

The `x/rdk` module is QoreChain's Rollup Development Kit - a protocol-native framework for deploying application-specific rollups directly on the host chain. Unlike generic rollup-as-a-service platforms that require external infrastructure, RDK rollups are first-class citizens of the QoreChain consensus, with settlement, data availability, and lifecycle management handled entirely on-chain.

**Four Settlement Paradigms:**

| Mode | Proof System | Finality | Use Case |
|------|-------------|----------|----------|
| **Optimistic** | Fraud proofs | 7-day challenge window, auto-finalized by EndBlocker | General-purpose rollups with low overhead |
| **ZK (Zero-Knowledge)** | SNARK or STARK | Instant on proof verification | DeFi protocols requiring fast finality |
| **Based** | None (L1-sequenced) | ~2 blocks after submission | Gaming and real-time applications leveraging host chain sequencing |
| **Sovereign** | None | Self-determined | Independent chains using QoreChain only for data availability |

**Four Preset Profiles for One-Click Deployment:**

| Profile | Settlement | Block Time | VM | DA Backend | Gas Model |
|---------|-----------|------------|-----|-----------|-----------|
| **DeFi** | ZK + SNARK | 500ms | EVM | Native | EIP-1559 |
| **Gaming** | Based (L1-sequenced) | 200ms | Custom | Native | Flat fee |
| **NFT** | Optimistic + Fraud | 2,000ms | CosmWasm | Celestia | Standard |
| **Enterprise** | Based | 1,000ms | EVM | Native | Subsidized (zero gas) |

**Key Technical Innovations:**

- **Native DA Router** - SHA-256 committed blob storage in the host chain's KVStore with configurable retention periods and automatic pruning via EndBlocker. Celestia IBC integration stubbed for future activation.
- **Bank Escrow Lifecycle** - Rollup creators bond QOR to a module escrow account on creation; bonds are fully returned when the rollup is stopped. A configurable creation burn rate (default 1%) permanently reduces supply with each new rollup.
- **Settlement Engine** - EndBlocker-driven auto-finalization: optimistic batches finalize after the challenge window expires, based batches finalize after 2 host blocks (L1 finality proxy). ZK batches with valid proofs finalize instantly on submission.
- **AI-Assisted Configuration** - The RL consensus module provides advisory `SuggestRollupProfile` and `OptimizeRollupGas` methods, using on-chain reinforcement learning to recommend optimal rollup parameters based on intended use case.
- **Deep Multilayer Integration** - Every rollup is registered as a layer in `x/multilayer` via `RegisterSidechain`, with state anchored via `AnchorState` on each batch settlement.
- **Configurable Sequencing** - Three sequencer modes: Dedicated (single operator), Shared (multi-operator set), and Based (host chain proposers order rollup transactions with configurable priority fee share).

### 44 Direct Cross-Chain Connections

QoreChain connects to **44 blockchain ecosystems** through two complementary protocols:

- **8 IBC chains** - Cosmos Hub, Osmosis, Noble, Celestia, Stride, Akash, Babylon, Injective. Pre-configured Hermes relayer templates for the first seven (client updates, misbehaviour detection, and packet clearing every 100 blocks); Injective is carried as an IBC-type bridge chain configuration.
- **36 QCB bridge endpoints** - Spanning EVM L2/L1, Cairo L2, UNL ledger, SCP ledger, Hashgraph, Pure-PoS, Move, UTXO, and account-model architectures. Together with Injective these are the bridge's 37 default chain configurations. Each chain has per-type address validation, configurable confirmation depth, circuit breaker volume caps, and PQC-signed validator attestations. Only a route whose status is `active` moves value, and genesis, admin messages and stored state are validated by one set of configuration rules (v3.2.2).
- **17 chain types** - `ibc`, `evm`, `solana`, `ton`, `sui`, `aptos`, `bitcoin`, `near`, `cardano`, `polkadot`, `tezos`, `tron`, `starknet`, `xrpl`, `stellar`, `hedera`, `algorand` - covering every major blockchain architecture.

### BTC Restaking via Babylon Protocol (v1.2.0)

The `x/babylon` module integrates with Babylon Protocol to inherit Bitcoin's proof-of-work finality guarantees. Validators can register BTC staking positions (min 100,000 satoshis), and QoreChain epoch state roots are periodically checkpointed to Bitcoin via IBC-relayed Babylon epochs.

### Smart Account Abstraction (v1.2.0)

The `x/abstractaccount` module enables programmable accounts backed by smart contracts - similar to ERC-4337 but at the protocol layer. Three account types (`multisig`, `social_recovery`, `session_based`) support session keys with granular permissions and expiry, per-account daily and per-transaction spending rules, and scoped denom allowlists. Spending limits are counted per rule rather than per key, an enabled rule must actually set a limit, and ERC-20 token movement by a spend-limited key falls under the rule. From v3.2.2 a rule-bound key gets deny-by-default EVM calls (only allow-listed contract calls), outflow measured from the account's actual balance change, per-token caps (`denom_limits`) and a separate permission for contract creation.

### MEV-Protected Block Space (v1.2.0)

The `x/fairblock` module provides a threshold identity-based encryption (tIBE) framework for encrypted mempools. Transactions are cryptographically opaque to block proposers until after inclusion - eliminating the information asymmetry that enables front-running and sandwich attacks.

### Multi-Token Gas Payment (v1.2.0)

The `x/gasabstraction` module removes the requirement to hold native QOR for transaction fees. From v3.2.2 a fee paid in one accepted non-native token is converted through a **conversion float**: a module account funded in QOR (`MsgFundFloat`) pays the native fee, and the float refills through an `x/amm` pool. Each accepted token is added by governance (`MsgUpdateConfig`) together with its pool, a minimum pool liquidity and a maximum TWAP-to-spot deviation, above which the token is refused as possibly manipulated; a per-block conversion cap and a fee premium (default 3%) apply. The default configuration accepts native `uqor` only, and fees paid in the native denom run the standard fee path unchanged. Queries: `GET /qorechain/gasabstraction/v1/config`, `/conversion_rate`, `/float`.

### 5-Lane Transaction Prioritization (v1.2.0)

Block space is statically partitioned into five priority lanes so that security-critical transactions can never be crowded out by high-volume standard traffic:

| Lane | Priority | Block Space | Purpose |
|------|----------|-------------|---------|
| PQC | 100 | 15% | Post-quantum hybrid-signature transactions |
| MEV | 90 | 20% | FairBlock tIBE-encrypted transactions |
| AI | 80 | 15% | AI-scored and optimized transactions |
| Default | 50 | 40% | Standard transactions |
| Free | 10 | 10% | Gas-abstracted and sponsored transactions |

### Quantum-Safe Hybrid Signatures (v1.1.0)

QoreChain is the only blockchain with **production-ready post-quantum hybrid signatures** - every transaction carries both a classical signature and an ML-DSA-87 (Dilithium-5) signature:

- **Required by default** - Three enforcement modes (Disabled, Optional, Required); new chains start in Required mode with classical-only fallback disabled. Key registration and migration messages are exempt so an account can bootstrap its key.
- **Domain-tagged, chain-bound sign bytes (v3.2.0)** - Hybrid transaction sign bytes, key-migration preimages and bridge attestations use a v2 form that binds a domain tag and the chain-id. On networks that predate v2, the chain verifies the historical form below the upgrade height and only v2 from it; a chain that starts on v3.2.0 or later verifies v2 from its first block. Clients choose the form by asking `applied_plan` for both `v3.1.98` and `v3.2.0`: v2 applies if either has been applied.
- **Validated keys** - Key-encapsulation algorithms are refused as authentication keys, key lengths are checked at registration, and from v3.2.2 a registration's classical public key is validated and must belong to the signer.
- **EVM wallets** - MetaMask-style wallets keep working unmodified inside a post-quantum authorisation window (see above)
- **Seamless onboarding** - PQC public keys auto-register via TX extension on first use
- **Three-way verification** - The `PQCHybridVerifyDecorator` handles all combinations of classical and PQC signatures

### On-Chain Reinforcement Learning (PRISM)

A Go-native fixed-point MLP (~73,733 parameters) runs PPO inference directly in the block lifecycle, dynamically tuning consensus parameters (block time, gas limits, pool weights) without any external oracle or sidecar dependency. Deterministic Taylor series math ensures identical results across all validators.

### Triple-Pool Composite Proof-of-Stake

Validators are automatically classified into RPoS (reputation-weighted), DPoS (delegation-weighted), and PoS (standard) pools every 1,000 blocks. Pool-weighted sortition diversifies block production beyond pure stake dominance.

### Quadratic-Reputation Governance (QDRW)

Voting power uses a square-root function dampened by a sigmoid reputation multiplier, preventing whale capture while rewarding long-term honest participation. xQORE holdings double voting weight via the formula `sqrt(staked + 2 * xQORE) * ReputationMultiplier(r)`.

### Deflationary Burn Engine

Ten distinct burn channels feed a central burn accounting module. Collected fees are split: 37% to validators, 30% permanently burned, 20% to treasury, 10% to stakers, and 3% to light node operators - creating sustained deflationary pressure that increases with network usage.

### xQORE Governance-Boosted Staking

Users lock QOR to mint xQORE at a 1:1 ratio, gaining doubled governance weight in QDRW votes. Early exit penalties (50% under 30 days, graduated down to 0% after 180 days) are redistributed to remaining holders via PvP rebase. The `x/xqore` keeper, parameters and genesis state ship with the chain; lock and unlock are not yet exposed as transaction messages.

### Triple-VM Architecture

The only Layer 1 running three virtual machines (EVM, CosmWasm, SVM) natively within one consensus. Deploy Solidity, Rust/CosmWasm, or BPF programs - all on the same chain, sharing state through cross-VM messaging.

### SVM Runtime with Native Programs & Solana-Compatible RPC

Deploy and execute BPF programs using Solana-compatible tooling. Four native built-in programs (System, SPL Token, ATA, Memo) provide gas-efficient token operations. The JSON-RPC server exposes 20 Solana-compatible methods - existing Solana clients and `@solana/web3.js` work out of the box.

## Key Features

- **PQC-Primary Security** - Dilithium-5 signatures + ML-KEM-1024 key exchange, hybrid classical + ML-DSA-87 via TX extensions with domain-tagged, chain-bound sign bytes, SHAKE-256 as the default hash for QoreChain-controlled commitments, algorithm-agile with governance-controlled migration
- **Hybrid Signature Architecture** - Three enforcement modes (disabled/optional/required, required by default), auto-registration onboarding, three-way ante verification
- **EVM-Lane Post-Quantum Authorisation** - Bounded authorisation windows opened by a post-quantum signed message, EIP-7702 delegations gated on the account's own post-quantum consent
- **Native AMM** - Constant-product + stable-swap pricing curves, protocol fee and swap-impact cap, cross-VM swap routing, deterministic integer math
- **IBC v2 Foundation** - `ChainArchitecture` enum, ICS-27/29/721 handler-hook interfaces, configurable client type per chain
- **Light Node Network** - Stake-weighted rewards from an accounted, per-node-bounded reward pool, heartbeat liveness proofs, automatic deactivation, 3% fee share
- **License-Gated Operations** - On-chain license registry with 76 feature IDs, auto-expiry, sidecar container orchestration for multi-chain validator bridge, licensed validator admission with a 100,000 QOR self-bond floor
- **Rollup Development Kit (RDK)** - 4 settlement modes, 3 DA backends, 4 preset profiles, settlement engine with EndBlocker auto-finalization
- **44 Cross-Chain Connections** - 8 IBC chains + 36 QCB bridge endpoints (37 bridge chain configurations, one of them IBC) across 17 chain types, with PQC-signed attestations
- **BTC Restaking** - Babylon Protocol integration for Bitcoin finality guarantees via IBC-relayed epoch checkpoints
- **Account Abstraction** - Programmable accounts with session keys, spending rules (per-transaction, daily, per-token, allow-listed EVM calls), and social recovery at the protocol layer
- **MEV Protection** - FairBlock tIBE encrypted mempool framework with dedicated block lane
- **Gas Abstraction** - Pay transaction fees in governance-approved IBC tokens through a pool-priced conversion float, without holding native QOR
- **5-Lane Block Prioritization** - PQC, MEV, AI, Default, and Free lanes with static block-space quotas
- **RL-Driven Consensus** - On-chain reinforcement learning agent dynamically tunes block time, gas limits, and pool weights
- **Triple-Pool CPoS** - RPoS/DPoS/PoS validator classification with pool-weighted proposer selection
- **QDRW Governance** - Quadratic delegation with reputation weighting and xQORE boost
- **Tokenomics Engine** - Burn accounting (10 channels), xQORE governance staking, epoch-based emission (decaying schedule or fixed per-epoch amount) with a governance-set cumulative cap
- **EVM Runtime** - Ethereum compatibility with JSON-RPC on port 8545, EIP-1559 gas, EIP-7702, ERC-20 token pairs; replay-protected (EIP-155) transactions only
- **CosmWasm Runtime** - WebAssembly smart contracts with full lifecycle support; nodes refuse code upload and contract instantiation at mempool admission
- **SVM Runtime** - BPF program deployment and execution, 4 native programs, 20 Solana-compatible JSON-RPC methods
- **Cross-VM Bridge** - Synchronous cross-VM calls that return the callee's answer (async queue optional), EVM ↔ CosmWasm (precompile + events) + SVM
- **Progressive Slashing** - Escalating penalties with temporal half-life decay, capped at 33% per infraction
- **Multilayer Architecture** - Main Chain + Sidechains + Paychains + Rollups with cross-layer fee bundling and state anchoring
- **Multi-Node Devnet Harness** - Two-validator `docker-compose.devnet.yml` + smoke-test runner shipped in-repo

## Quick Start

### Docker Compose (Recommended)

```bash
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core
docker compose up -d
```

This starts: QoreChain node (with EVM + CosmWasm + SVM runtimes), AI sidecar, block indexer, Postgres, Prometheus, and Grafana.

RPC, REST, gRPC, EVM JSON-RPC, metrics and sidecar ports are published on `127.0.0.1` only, because these endpoints have no authentication; P2P (`26656`) stays on all interfaces. To serve an endpoint publicly, put a reverse proxy that terminates TLS and rate-limits in front of it.

### Two-Validator Devnet

```bash
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core
docker compose -f docker-compose.devnet.yml up --build -d
./scripts/devnet/smoke.sh
```

See [`scripts/devnet/README.md`](scripts/devnet/README.md) for the operator runbook.

### Build from Source

```bash
# Prerequisites: Go 1.26.1+, CGO enabled, libqorepqc + libqoresvm (see docs/)
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core

# Build the binary (community build) into ./build/qorechaind
make build
# or: CGO_ENABLED=1 go build -o qorechaind ./cmd/qorechaind/

# Initialize a node
./qorechaind init my-node --chain-id qorechain-diana

# Start the node
./qorechaind start
```

### Connect to Testnet

```bash
# Use the network's published genesis (see the network page) and configure peers
curl -fsSL "https://<network-genesis-url>/genesis.json" -o ~/.qorechaind/config/genesis.json
# Edit ~/.qorechaind/config/config.toml to add seeds / persistent peers
./qorechaind start --minimum-gas-prices 0.001uqor
```

`qorechaind init` writes the network's pinned EVM chain ID (9800 for `qorechain-diana`, 9801 for `qorechain-vladi`) to `app.toml`; a node whose `app.toml` value disagrees with its chain reports the mismatch at start-up. At start-up the node also logs the resolved EVM chain ID and the identity of the loaded post-quantum library.

### Run a QoreChain-Only Node (Exchanges & Integrators)

If you only need to **connect to QoreChain** - sync the chain, query
balances/blocks, and submit transactions - you do **not** need any of the
optional or licensed components. A plain full node is everything required.

**Not needed (and not started by this setup):**

- ❌ AI sidecar (licensed image - `x/ai` runs its on-chain heuristic engine by
  default; `UseSidecar` is `false`)
- ❌ Bridge relayers / external-network light clients (the bridge & IBC modules
  are compiled into the binary but stay dormant - with no relayer and no
  per-chain config the node neither validates nor relays for any other network)
- ❌ Block indexer + Postgres, Prometheus/Grafana

Just the node: consensus, the EVM / CosmWasm / SVM runtimes, and the
RPC / REST / gRPC / EVM-JSON-RPC endpoints.

#### Option A - Docker (node-only compose)

The node image is built locally from the public `Dockerfile`; **no private
images are pulled.** Provide the network genesis and peers, then start:

```bash
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core

GENESIS_URL="https://<network-genesis-url>/genesis.json" \
SEEDS="<node_id>@<host>:26656,<node_id>@<host>:26656" \
docker compose -f docker-compose.node.yml up -d --build
```

`docker-compose.node.yml` runs a single `qorechain-node` service and joins the
live network (it does not bootstrap a new chain). Configure it with env vars:

| Variable | Required | Default | Description |
|---|---|---|---|
| `GENESIS_URL` | yes¹ | - | URL of the network's `genesis.json` |
| `SEEDS` | yes² | - | Seed nodes, `id@host:port,…` |
| `PERSISTENT_PEERS` | yes² | - | Persistent peers, `id@host:port,…` |
| `CHAIN_ID` | no | `qorechain-diana` | Network chain ID |
| `MONIKER` | no | `qorechain-node` | Node name |
| `MIN_GAS_PRICE` | no | `0.001uqor` | Minimum gas price |

¹ Or mount the genesis instead of using a URL - uncomment the `./genesis.json`
volume in `docker-compose.node.yml`. ² At least one of `SEEDS` /
`PERSISTENT_PEERS`.

> Get the current `qorechain-diana` genesis URL and seed/peer list from the
> QoreChain network page (or your QoreChain contact).

Check it is syncing:

```bash
curl -s localhost:26657/status | jq '.result.sync_info'   # catching_up: false once synced
```

Exposed endpoints: RPC `:26657`, REST `:1317`, gRPC `:9090`,
EVM JSON-RPC `:8545` (HTTP) / `:8546` (WS).

#### Option B - From source (no Docker)

```bash
# Prerequisites: Go 1.26+, CGO enabled, build-essential
git clone https://github.com/qorechain/qorechain-core.git
cd qorechain-core
CGO_ENABLED=1 go build -tags "netgo ledger" -o qorechaind ./cmd/qorechaind

# Initialize, then join the live network
./qorechaind init <moniker> --chain-id qorechain-diana
curl -fsSL "https://<network-genesis-url>/genesis.json" -o ~/.qorechaind/config/genesis.json
# add seeds / persistent_peers under [p2p] in ~/.qorechaind/config/config.toml
./qorechaind start --minimum-gas-prices 0.001uqor
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                             QoreChain Node                                  │
│                                                                             │
│  ┌──────────────────── Virtual Machines ──────────────────────┐            │
│  │  ┌───────┐    ┌──────────┐    ┌───────┐                   │            │
│  │  │  EVM  │    │ CosmWasm │    │  SVM  │                   │            │
│  │  │(Sol.) │◄──►│ (Wasm)   │◄──►│ (BPF) │                   │            │
│  │  └───┬───┘    └────┬─────┘    └───┬───┘                   │            │
│  │      └─────────┬───┘──────────────┘                       │            │
│  │           x/crossvm (bridge)  ◄──►  x/amm (swap routing)  │            │
│  └────────────────────────────────────────────────────────────┘            │
│                                                                             │
│  ┌──────────────────── On-Chain Markets ─────────────────────┐            │
│  │  ┌───────┐                                                  │            │
│  │  │ x/amm │  CP + StableSwap pools, cross-VM hook,           │            │
│  │  │       │  protocol-fee share routed via x/burn            │            │
│  │  └───────┘                                                  │            │
│  └────────────────────────────────────────────────────────────┘            │
│                                                                             │
│  ┌────────────────────── Tokenomics ─────────────────────────┐            │
│  │  ┌──────┐   ┌───────┐   ┌───────────┐   ┌───────────┐   │            │
│  │  │x/burn│   │x/xqore│   │x/inflation│   │x/lightnode│   │            │
│  │  │10 ch.│   │lock/  │   │epoch decay│   │heartbeat  │   │            │
│  │  │37/30/│   │unlock │   │17.5→2%    │   │rewards    │   │            │
│  │  │20/10 │   │PvP    │   │           │   │3% share   │   │            │
│  │  └──────┘   └───────┘   └───────────┘   └───────────┘   │            │
│  └────────────────────────────────────────────────────────────┘            │
│                                                                             │
│  ┌──────────── IBC / Bridges / License ─────────────────────┐            │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌─────────┐ │            │
│  │  │x/bridge  │  │x/babylon │  │x/abstract │  │x/gas    │ │            │
│  │  │37 QCB +  │  │BTC re-   │  │ account   │  │abstract.│ │            │
│  │  │8 IBC     │  │staking   │  │session key│  │multi-tok│ │            │
│  │  │+ IBC v2  │  │          │  │           │  │         │ │            │
│  │  └──────────┘  └──────────┘  └───────────┘  └─────────┘ │            │
│  │  ┌──────────┐  ┌──────────┐                               │            │
│  │  │x/fair    │  │x/license │  5-Lane: PQC|MEV|AI|Def|Free │            │
│  │  │ block    │  │76 feature│  tIBE encrypted mempool       │            │
│  │  │tIBE      │  │IDs, auto-│  Sidecar orchestration        │            │
│  │  └──────────┘  │expiry    │                               │            │
│  │                 └──────────┘                               │            │
│  └────────────────────────────────────────────────────────────┘            │
│                                                                             │
│  ┌──── Rollup Development Kit (v1.3.0) ──────────────────────┐            │
│  │  ┌──────────┐  ┌──────────┐  ┌───────────┐  ┌──────────┐ │            │
│  │  │ x/rdk    │  │Settlement│  │ DA Router │  │ Profiles │ │            │
│  │  │ 4 modes: │  │Optimistic│  │ Native    │  │ DeFi     │ │            │
│  │  │ opt/zk/  │  │ZK/Based/ │  │ Celestia* │  │ Gaming   │ │            │
│  │  │ based/   │  │Sovereign │  │ Both      │  │ NFT      │ │            │
│  │  │ sovereign│  │          │  │           │  │ Enterpr. │ │            │
│  │  └──────────┘  └──────────┘  └───────────┘  └──────────┘ │            │
│  └────────────────────────────────────────────────────────────┘            │
│                                                                             │
│  ┌──────────────┐ ┌──────┐ ┌────────────┐ ┌─────┐                        │
│  │x/rlconsensus │ │ x/ai │ │x/reputation│ │x/qca│                        │
│  │  RL Agent    │ │      │ │            │ │     │                        │
│  └──────┬───────┘ └──┬───┘ └─────┬──────┘ └──┬──┘                        │
│   PPO MLP         AI Engine   Scoring    CPoS Pools                       │
│   Obs/Action      Fraud Det.  Decay      Bonding                          │
│   Circuit Brk     Fee Opt.    Sigmoid    Slashing                         │
│   Rollup Adv.     TEE/FL                 QDRW Gov                         │
│                                                                             │
│  ┌──────┐ ┌──────────┐ ┌──────┐ ┌───────┐                                │
│  │x/pqc │ │ x/multi  │ │x/svm │ │x/cross│                                │
│  └──┬───┘ └────┬─────┘ └──┬───┘ └───┬───┘                                │
│  Dilithium    Layer       BPF     CrossVM                                  │
│  ML-KEM       Router      Exec    Messaging                                │
│  Hybrid Sig   Rollups                                                      │
│  SHAKE-256                                                                 │
│                                                                             │
└────────┬──────┬─────────────────────────────────┬───────────────────────────┘
         │      │                                  │
   ┌─────┴─────┐│                         ┌───────┴──────┐
   │libqorepqc ││                         │  Indexer     │
   │(Rust PQC) ││                         │  (Postgres)  │
   └───────────┘│                         └──────────────┘
   ┌───────────┐│  ┌──────────┐
   │libqoresvm ││  │AI Sidecar│
   │(Rust BPF) │└──│ (gRPC)   │
   └───────────┘   └──────────┘
```

## Modules

| Module | Description |
|--------|-------------|
| **x/pqc** | Post-quantum cryptography: Dilithium-5, ML-KEM-1024, hybrid classical + ML-DSA-87 signatures (v2 chain-bound sign bytes), key registration / migration / same-algorithm rotation, EVM-lane authorisation windows and EIP-7702 delegation consent, the protocol-feature record, SHAKE-256 default hashing (`qorehash`), algorithm-agile governance |
| **x/ai** | AI engine: transaction routing, anomaly detection, fraud detection, fee optimization, per-sender rate limit (both lanes), governance config updates, TEE attestation interfaces, federated learning coordination |
| **x/rlconsensus** | RL-based dynamic consensus tuning: fixed-point MLP, PPO inference, shadow/conservative/autonomous modes, circuit breaker, rollup advisory; saturating deterministic math and checked policy-architecture validation |
| **x/reputation** | Validator reputation scoring: multi-factor formula with temporal decay; from v3.2.2 the bonded set is rescored per epoch from live stake, slashing-module performance and governance participation settled when voting ends |
| **x/qca** | Consensus Engine Algorithm: triple-pool CPoS, bonding curve, progressive slashing, QDRW governance |
| **x/burn** | Central burn accounting: 10 burn channels (incl. AMM protocol-fee and failed-transaction channels), EndBlocker fee distribution (37% validators / 30% burned / 20% treasury / 10% stakers / 3% light nodes), governance-only release of the recorded stranded staker share |
| **x/xqore** | Governance-boosted staking: lock QOR → mint xQORE (1:1), graduated exit penalties, PvP rebase redistribution (keeper and genesis; no transaction messages yet) |
| **x/inflation** | Epoch-based emission: decaying schedule (Y1 17.5% → Y2 11% → Y3-4 7% → Y5+ 2%) scaled to the epoch length, or a fixed per-epoch amount, with a hard cumulative cap; governance `MsgUpdateParams` and a query service |
| **x/amm** | Native AMM: constant-product + stable-swap pricing curves, 8 messages (create / add / remove / swap-exact-in / swap-exact-out / pause / resume / set-params), cross-VM swap hook, deterministic integer math |
| **x/bridge** | Cross-chain bridge (QCB): 37 default chain configurations across 17 chain types, PQC-signed validator attestations with the configured attestation threshold enforced, its own minting module account, circuit breaker volume caps, IBC v2 foundation (`ChainArchitecture` enum + ICS-27/29/721 handler hooks). Post-deploy per-chain activation via `MsgUpdateChainConfig` + `MsgSetVerifierBootstrap` (a `bridge_admin` key or `qcb_bridge` license - no governance, no upgrade). |
| **x/babylon** | BTC restaking adapter: Babylon Protocol IBC integration, epoch checkpoints to Bitcoin, staking position lifecycle |
| **x/abstractaccount** | Smart account abstraction: multisig/social_recovery/session_based accounts, linked-wallet authenticators on three lanes, spending rules (per-transaction, daily, per-token, allow-listed EVM calls), session keys with expiry |
| **x/fairblock** | MEV protection: threshold IBE encrypted mempool framework, FairBlockDecorator ante handler |
| **x/gasabstraction** | Multi-token gas payment: governance-approved IBC fee tokens converted through a QOR conversion float priced by an `x/amm` pool, fee router in the ante chain |
| **x/rdk** | Rollup Development Kit: 4 settlement paradigms, 3 DA backends, 4 preset profiles, settlement engine with auto-finalization; batches settled only by the rollup's sequencer and in order, per-rollup escrow ledger, enforced creation burn |
| **x/multilayer** | Multi-layer architecture: Sidechains + Paychains (high-frequency microtransactions) + Rollups with cross-layer fee bundling and state anchoring; query service exposed over REST (`/qorechain/multilayer/v1/...` incl. layers, anchors, routing-stats); creator stake locked at registration and returned on decommission, anchors refused without a valid creator signature, governance `MsgUpdateParams` |
| **x/crossvm** | Cross-VM communication: synchronous calls that return the callee's answer (async queue optional), context-derived caller, bounded depth and gas, EVM ↔ CosmWasm (precompile) + SVM, AMM swap routing |
| **x/svm** | SVM runtime: BPF program deployment/execution, native programs (System, SPL Token, ATA, Memo), 20 Solana-compatible JSON-RPC methods; value leaving an account requires that account's signature, single-use signed authorisations, canonical bounded base58 addresses, governance `MsgUpdateParams` |
| **x/lightnode** | Light node network: registration, heartbeat liveness with forward-accrued uptime, reward pool (reserved / outstanding / distributable) with a per-node per-block bound, governance-only reserve release, auto-deactivation |
| **x/license** | On-chain license registry: 76 feature IDs (umbrella `qcb_bridge`, 36 `bridge_*`, 37 `validator_*`, `lightnode_operator`, `validator_operator`), auto-expiry, lifecycle management |
| **x/vm** | EVM precompile definitions (support package, not a separate application module) |
| **x/govauth** | Shared check that an administrative message really comes from governance (support package) |

## Cross-Chain Connectivity

### IBC Chains (8)

| Chain | Prefix | Fee Denom | Notable Integration |
|-------|--------|-----------|---------------------|
| Cosmos Hub | `cosmos` | uatom | Core IBC hub connectivity |
| Osmosis | `osmo` | uosmo | DEX liquidity routing |
| Noble | `noble` | uusdc | Native USDC for gas abstraction |
| Celestia | `celestia` | utia | Data availability layer |
| Stride | `stride` | ustrd | Liquid staking |
| Akash | `akash` | uakt | Decentralized compute |
| Babylon | `bbn` | ubbn | BTC restaking checkpoints |
| Injective | `inj` | inj | DeFi liquidity routing (added v3.0.0; bridge chain configuration of type `ibc`) |

### QCB Bridge Endpoints (36 + Injective)

#### Baseline (17 chains)

| Chain | Type | Confirmations | Supported Tokens |
|-------|------|---------------|------------------|
| Ethereum | evm | 12 | ETH, USDC, USDT, WBTC |
| BSC | evm | 15 | BNB, BUSD |
| Solana | solana | 32 | SOL, USDC |
| Avalanche | evm | 12 | AVAX, USDC |
| Polygon | evm | 128 | MATIC, USDC |
| Arbitrum | evm | 12 | ETH, ARB, USDC |
| TON | ton | 10 | TON |
| Sui | sui | 3 | SUI |
| Optimism | evm | 10 | ETH, USDC, OP |
| Base | evm | 10 | ETH, USDC |
| Aptos | aptos | 6 | APT, USDC |
| Bitcoin | bitcoin | 6 | BTC |
| NEAR | near | 3 | NEAR |
| Cardano | cardano | 15 | ADA |
| Polkadot | polkadot | 12 | DOT |
| Tezos | tezos | 2 | XTZ |
| TRON | tron | 20 | TRX, USDT |

#### Cross-Network Expansion (19 chains below + Injective, v3.0.0)

| Chain | Type | Confirmations | Supported Tokens |
|-------|------|---------------|------------------|
| zkSync Era | evm | 12 | ETH, USDC |
| Linea | evm | 12 | ETH, USDC |
| Scroll | evm | 12 | ETH, USDC |
| Starknet | starknet | 12 | ETH, STRK, USDC |
| Blast | evm | 10 | ETH, USDB |
| Mantle | evm | 10 | MNT, USDC |
| Hyperliquid | evm | 10 | USDC |
| Berachain | evm | 10 | BERA, HONEY |
| Sonic | evm | 10 | S, USDC |
| Sei | evm | 10 | SEI, USDC |
| Monad | evm | 30 | MON, USDC |
| Plasma | evm | 10 | XPL, USDT |
| XRP Ledger | xrpl | 4 | XRP, RLUSD |
| Stellar | stellar | 5 | XLM, USDC |
| Hedera | hedera | 4 | HBAR, USDC |
| Algorand | algorand | 4 | ALGO, USDC |
| Filecoin (FVM) | evm | 10 | FIL, USDC |
| Cronos | evm | 12 | CRO, USDC |
| Kaia | evm | 10 | KAIA, USDC |

(Injective, the 37th default bridge configuration, is listed in the IBC table above.)

## Token Economics

- **Token**: QOR (display) / uqor (base denomination, 1 QOR = 10^6 uqor)
- **Chain ID**: qorechain-vladi (mainnet, EVM chain ID 9801), qorechain-diana (testnet, EVM chain ID 9800)
- **Bech32 Prefix**: qor (addresses: qor1..., validators: qorvaloper...)

### Emission Schedule

| Year | Inflation Rate | Description |
|------|---------------|-------------|
| 1 | 17.5% | Bootstrap phase - aggressive incentives for early validators |
| 2 | 11.0% | Growth phase - reduced emission as network matures |
| 3-4 | 7.0% | Stabilization - converging toward sustainability |
| 5+ | 2.0% | Long-term - minimal new supply, deflationary via burns |

This is the default `schedule` mode, scaled to the epoch length so an epoch mints the share of the year it represents. Governance can instead select the `fixed` mode (a constant amount per epoch) and set `max_total_emission`, a hard cumulative cap after which `x/inflation` stops minting. Both are changed with `MsgUpdateParams`.

### Fee Distribution

Every block, collected fees are split (`x/burn` parameters `validator_share`, `treasury_share`, `staker_share`, `light_node_share`; the remainder is burned):

| Recipient | Share | Purpose |
|-----------|-------|---------|
| Validators | 37% | Block production rewards |
| Burn | 30% | Permanent supply reduction |
| Treasury | 20% | Protocol development fund |
| Stakers | 10% | Passive staking rewards, paid out through the distribution module |
| Light Nodes | 3% | Light node operator rewards |

### xQORE Exit Penalties

| Lock Duration | Penalty | Destination |
|--------------|---------|-------------|
| < 30 days | 50% | Redistributed to remaining xQORE holders |
| 30-90 days | 35% | Redistributed to remaining xQORE holders |
| 90-180 days | 15% | Redistributed to remaining xQORE holders |
| > 180 days | 0% | Full withdrawal |

## JSON-RPC Endpoints

| Port | Protocol | Description |
|------|----------|-------------|
| 8545 | HTTP | EVM JSON-RPC (`eth_`, `web3_`, `net_`, `txpool_`, `qor_` namespaces) |
| 8546 | WebSocket | EVM JSON-RPC (WebSocket) |
| 8899 | HTTP | SVM JSON-RPC (Solana-compatible: 20 methods) |
| 1317 | HTTP | REST API |
| 9090 | gRPC | gRPC query endpoints |
| 26657 | HTTP | RPC (blocks, transactions, consensus) |

## Ante Handler Chain

Cosmos-lane transactions:

```
RejectEthMessages → AuthzLimiter → SetUpContext → WasmMempoolGuard → MultisigDepthGuard →
Wasm (sim gas / tx count / gas register / contracts) → CircuitBreaker → Extension →
ValidateBasic → TxTimeout → Memo → ValidatorLicense → PQCVerify → PQCHybridVerify →
PQCReplayGuard → AIAnomaly → FairBlock → SVMComputeBudget → SVMDeductFee →
FeeRouter (MinGasPrice → ConsumeTxSize → GasAbstraction → DeductFee, or fee conversion) →
SetPubKey → ValidateSigCount → SigGasConsume → SigVerify → IncrementSequence →
BurnTxCount → BurnFailedFee → RedundantRelay → GasWanted
```

EVM-lane transactions:

```
CircuitBreaker → EVMReplayGuard → EVMMono → EVMWindow (post-quantum window + EIP-7702 consent) →
AIEVMAnomaly → BurnTxCount   (the last two from the v3.2.2 plan height)
```

## SDK & Developer Tooling

QoreChain is built on **Cosmos SDK v0.53.8** (ibc-go v10.7, cosmos/evm v0.6.3, wasmd v0.60.9, Go 1.26.1) and exposes the standard Cosmos
interfaces (Protobuf messages, gRPC, REST/LCD, QoreChain Consensus Engine RPC) plus the EVM and
SVM JSON-RPC servers, so existing ecosystem SDKs work out of the box:

| Surface | SDK / tooling | Endpoint |
|---------|---------------|----------|
| Cosmos (bank, staking, gov, and the 20 custom modules) | **CosmJS** (`@cosmjs/stargate`), `cosmpy`, gRPC/Protobuf clients | gRPC `:9090`, REST `:1317`, RPC `:26657` |
| EVM | **ethers.js / viem / web3.js**, Hardhat, Foundry, Remix, MetaMask | JSON-RPC `:8545` (chain-id **9801** mainnet, **9800** testnet) / WS `:8546`; EVM transactions need an open post-quantum authorisation window |
| SVM | **@solana/web3.js**, Anchor clients | Solana-compatible JSON-RPC `:8899` |
| CosmWasm | `@cosmjs/cosmwasm-stargate`, `cosmwasm-std` (Rust) | gRPC/REST |

For building **on** QoreChain (client apps) and building **into** it (custom
modules, the community-vs-full build-overlay model, the Protobuf `Msg`/`Query`
surface of all 17 proto-bound custom modules, and the validator/light-node
license+build flow), see:

- **[SDK Overview & Client Libraries](docs/SDK.md)** - start here
- [Building from Source](https://docs.qorechain.io/developer-guide/building-from-source) - community vs full build
- [Running a Validator](https://docs.qorechain.io/developer-guide/running-a-validator) - license + full-binary requirement

## Documentation

- [SDK Overview & Client Libraries](docs/SDK.md)
- [Architecture Overview](docs/ARCHITECTURE.md)
- [Sidecar Operator Guide](docs/SIDECAR.md)
- [RL Consensus Module](docs/RL_CONSENSUS.md)
- [Consensus Enhancements (CPoS, Bonding, Slashing, QDRW)](docs/CONSENSUS.md)
- [Tokenomics (Burn, xQORE, Inflation)](docs/TOKENOMICS.md)
- [PQC Integration Guide](docs/PQC_INTEGRATION.md)
- [Algorithm Agility](docs/ALGORITHM_AGILITY.md)
- [AI Engine](docs/AI_ENGINE.md)
- [EVM Runtime](docs/EVM.md)
- [EVM Precompiles](docs/EVM_PRECOMPILES.md)
- [Cross-VM Bridge](docs/CROSSVM.md)
- [SVM Runtime](docs/SVM.md)
- [Bridge Documentation](docs/BRIDGE.md)
- [Multilayer Architecture](docs/MULTILAYER.md)
- [Rollup Development Kit (RDK)](docs/RDK.md)
- [Running a Testnet Node](docs/RUNNING_TESTNET.md)
- [Exchange Integration](docs/EXCHANGE_INTEGRATION.md)
- [Two-Validator Devnet](scripts/devnet/README.md)
- [API Reference](docs/API_REFERENCE.md)

## Infrastructure

- 3 separate Go modules: `qorechain-core/`, `sidecar/`, `indexer/`
- 2 Rust crates: `qorepqc` (PQC cryptography), `qoresvm` (BPF executor + native programs)
- 48 registered genesis modules, 20 custom application modules (plus the `x/vm` and `x/govauth` support packages)
- Docker Compose: 6-service deployment stack + 2-validator devnet stack
- GitHub Actions: CI/CD workflows (build, release, docker)
- IBC: pre-configured Hermes relayer templates for 7 counterparty chains (`config/hermes/`)

## License

Apache 2.0 - see [LICENSE](LICENSE)

Core blockchain protocol is open source. PQC cryptographic libraries and BPF execution engine are distributed as pre-compiled binaries under separate licensing terms.
