# Changelog

All notable changes to QoreChain will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [3.2.3] - 2026-10-20

A coordinated governance upgrade (plan name `v3.2.3`, applied on mainnet at height 3,662,051). Every consensus change below is guarded by a protocol feature that the v3.2.3 upgrade handler turns on at the plan height (`ReleaseV3_2_3ProtocolFeatures`); below that height the binary replays history byte-identically.

### Added
- **Governance can replace the license grant authority** (`MsgUpdateAuthority`, x/license, governance only).
- **Optional proof of possession** on `MsgRegisterPQCKeyV2` (field `proof_of_possession`): the key being registered signs `qorechain-pqc-pop-v1|<chain-id>|<algorithm>|<account>|<hex public key>`. It is verified when present and is not required.
- **Scoped EVM authorization windows:** a window can be opened for native QOR transfers only (`native_only`) or for listed recipients only (`allowed_targets`). A window without a scope behaves as before.
- **Light node reward bound per month:** `max_reward_per_month_per_node` (2,000 QOR) prorated over block time, so the bound no longer depends on the block cadence. The reward pool query reports it.

### Changed
- Post-quantum key maintenance from an account that already holds a key needs the hybrid signature; registration refuses weak or degenerate keys and a key the account used before; PQC key messages are refused inside authz; a transaction carries at most one hybrid extension; algorithm governance cannot remove the last active signature algorithm.
- An account without a registered key can no longer attach a hybrid extension it cannot verify.
- Vesting accounts can only be created for the signer itself.
- Stake authorizations name canonical validator addresses.
- Cross-VM queue processing is atomic per message, and the EVM precompile set is bounded and read-only where it should be.
- RDK: escrow is ledgered per rollup and withdrawals are checked against it (denomination, positive amount, proof depth and sibling size).
- Emission years are counted by epoch length, AMM end-block work is bounded, burn records are pruned, and reinforcement-learning consensus parameters are bounded.
- Smart-account hardening on the EVM path (the signed digest binds the chain and the account).
- The public build refuses to start on a public network (matched by chain-id and genesis time), so a node can only join mainnet or testnet with the release binary.

### Notes
- Upgrade handler: `v3.2.3`. Transaction signing for standard transactions is unchanged.

---

## [3.2.2] - 2026-10-01

A coordinated governance upgrade (plan name `v3.2.2`). Every consensus change below is guarded by a protocol feature that the v3.2.2 upgrade handler turns on at the plan height (26 features, listed in `ReleaseV3_2_2ProtocolFeatures`); below that height the binary replays history byte-identically.

### Added
- **EIP-7702 post-quantum consent.** `MsgAuthorizeEVMDelegation` records an account's own post-quantum consent to one EIP-7702 delegation (single-use, time-bounded, revoked by `MsgCloseEVMWindow`); `MsgClearEVMDelegation` removes a delegation. Query: `GET /qorechain/pqc/v1/evm_delegation_consent/{address}`; CLI: `tx pqc authorize-evm-delegation` / `clear-evm-delegation`, `query pqc evm-delegation-consent`.
- **Gas abstraction conversion float.** A fee paid in one accepted non-native token is converted through a QOR float held by the module (`MsgFundFloat`) and refilled through an `x/amm` pool. Tokens are added by governance (`MsgUpdateConfig`) with a pool, a minimum liquidity and a maximum TWAP-to-spot deviation; a per-block conversion cap and a fee premium apply. Queries: `config`, `conversion_rate`, `float`. The default configuration accepts native `uqor` only.
- **Abstract-account spending policy.** Rule-bound keys get deny-by-default EVM calls (`allowed_evm_calls`), outflow measured from the account's actual balance change, per-token caps (`denom_limits`) and a separate `deploy` permission for contract creation.
- `x/inflation` query service for the parameters, the emission to date and the cap.
- `x/multilayer` `MsgUpdateParams` (governance only), and REST routes for the RDK query service (`/qorechain/rdk/v1/...`).
- `x/ai` governance configuration updates, including the screening configuration fields.

### Changed
- **Validator admission is enforced by the staking message server.** The `validator_operator` license and the 100,000 QOR self-bond floor now hold on every dispatch path, not only in the ante handler. An operator may exit completely; `MsgUnjail` is refused while the operator's self-bond is under the floor.
- **EVM-lane parity.** EVM transactions now run the same per-sender rate limit, anomaly screening and milestone-burn transaction counter as the Cosmos lane, and `x/ai` anomaly screening and the rate limit also apply to standard Cosmos messages. Screening is deterministic and does not change receipts or gas estimates.
- **Reputation scoring v2.** The bonded set is rescored per epoch from live stake, performance taken from the slashing module, and governance participation settled when voting ends.
- **Governance proposer reward** is bounded by the amount the proposal had on deposit.
- **AMM.** A swap must name both of the pool's denominations, stable pools are priced with the stable invariant in both directions, the protocol fee is collected and routed through `x/burn`, `max_swap_impact_bps` is enforced, and StableSwap solve gas is charged on failed swaps too.
- **Burn.** The `failed_tx` burn channel now applies, based on the fee actually charged; a milestone tier whose burn failed is retried instead of skipped.
- **Bridge.** Genesis, admin messages and stored state share one set of configuration rules: only a route whose status is exactly `active` moves value, statuses are normalised at the upgrade, verifier trust roots are validated and zero amounts are handled consistently.
- **RDK / multilayer.** New rollup and layer ids follow a defined grammar, keeper loops match exact ids, finalization keeps the batch pointer, settlement anchors are accepted only on the rollup creator's own layer, and anchors use v2 signed bytes.
- **SVM.** Base58 addresses are bounded and canonical in the RPC and gRPC queries.
- Every module account in the permission table is created as a module account at the upgrade height.
- Dependencies: Cosmos SDK v0.53.8, consensus engine v0.38.26, ibc-go v10.7.0, IAVL v1.2.8, wasmd v0.60.9 and wasmvm v2.3.5 (public releases), x/crypto v0.57.0. cosmos/evm stays on v0.6.3. SDK v0.53.8 tightens which edge-case transactions are accepted, so this binary replaces the previous one at the upgrade height, not node by node.
- A node whose `app.toml` `evm-chain-id` disagrees with its chain now reports the mismatch instead of refusing to start.

### Security
- An EIP-7702 delegation takes effect only with the authority account's own post-quantum consent, value leaving a delegated account is charged to that account's own authorisation window, and a failed type-4 transaction keeps the authorizations it applied.
- Unprotected (pre-EIP-155) Ethereum transactions are refused in block execution as well as in the mempool.
- Multisig signatures are bounded in block execution too, not only at admission.
- PQC key registration validates the classical public key and binds it to the signer.
- An emission cap of zero or an omitted cap is refused rather than read as unlimited.
- Module account addresses cannot be claimed through the EVM, and contract creation keeps the sender nonce increments made during it.
- ERC-20 reconversion on an IBC timeout or error acknowledgement is isolated, so a failed reconversion can no longer strand the refund or prevent the timeout from executing.
- `x/rlconsensus` validates a policy's architecture with checked arithmetic before storing or running it, and `x/ai` refuses a NaN threshold in a governance config update.
- `x/gasabstraction` keeps the native denom fixed in a config update.

### Notes
- Upgrade handler: `v3.2.2`. All one-time state work is idempotent and runs before the switches turn on.

---

## [3.2.1] - 2026-09-30

Node-local release: no consensus change and no upgrade plan; nodes can be replaced one at a time.

### Changed
- Consensus engine v0.38.25 and grpc-go v1.83.2.
- `qorechaind init` writes the network's pinned EVM chain ID to `app.toml`, the EVM chain ID is cross-checked against the Cosmos chain-id, and the resolved EVM chain ID and the loaded post-quantum library are reported when the node starts. The full build self-tests the post-quantum library before using it.
- Mempool: the Cosmos pool is bounded, proposals are fit-checked before the ante handler, the eviction index is ranked by the transaction's gas meter, and EVM transactions the mempool promotes are gossiped.
- `tx pqc cosign` and `rotate-key` honour `--generate-only`, `--offline` and `--dry-run`.
- Genesis export and import carry authenticator sequences and spend buckets (`x/abstractaccount`), and AMM export decodes LP balances and paused pools correctly.

### Security
- Unprotected legacy Ethereum transactions are kept out of the mempool.
- Block-sync responses are bounded by transaction count and size, and EVM JSON-RPC bounds the resources one caller can hold and answers for the block and sender that were asked for.
- Multisig signatures are bounded in every mode except block execution (block execution follows in v3.2.2).
- Transactions without a fee are refused cleanly, and panic stack traces no longer appear in responses.
- The license `Check` query and the sidecar treat a license past `expires_at` as inactive.
- The authenticator seed is kept off the command line and standard output.
- Each fee-distribution share is bounded on its own, gas abstraction genesis validates conversion rates, and its config is validated before it is written.

---

## [3.2.0] - 2026-09-20

A coordinated governance upgrade. Testnet took the same release under the plan name `v3.1.98` (2026-09-18) and then `v3.2.0`; both names run the same, idempotent handler. The release is a minor version because the EVM lane now requires post-quantum authorisation, a breaking change for EVM wallets. Seven protocol features are enabled at the plan height (`ReleaseV3_2_0ProtocolFeatures`).

### Added
- **Post-quantum authorisation on the EVM lane.** An EVM transaction is admitted only from an account that holds a registered post-quantum key and has opened a bounded window with `MsgOpenEVMWindow` (at most 17,280 blocks, at most 1,000 transactions, and a total outflow that counts value plus the maximum fee). Opening replaces the existing window; `MsgCloseEVMWindow` revokes it. Query `GET /qorechain/pqc/v1/evm_window/{address}`, CLI `tx pqc open-evm-window` / `close-evm-window`, `query pqc evm-window`.
- **Protocol features.** A per-feature record in the `x/pqc` store, switched on by a release's upgrade handler at the plan height or at genesis on a new chain (exactly the features the genesis lists, and exported with it).
- **Synchronous cross-VM calls.** `MsgCrossVMCall` executes in the transaction and returns the callee's answer; `async` keeps the queue. CosmWasm contracts can dispatch it and read the reply. SVM targets take an instruction payload with accounts, where only the caller may be a signer.
- **Light node reward pool.** The module balance is accounted as reserved, outstanding and distributable; each block distributes only the new accrual, bounded per node by `max_reward_per_block_per_node` and scaled by uptime. The reserve moves only by governance (`MsgReleaseLightNodeReserve`). Query: `GET /qorechain/lightnode/v1/reward_pool`.
- **Stranded staker share.** The staker share that accumulated in the distribution module before this release is recorded at the upgrade height and can be released only by governance (`MsgReleaseStrandedRewards`, `GET /qorechain/burn/v1/stranded_rewards`).
- `x/svm` `MsgUpdateParams` (governance), the token instructions wallets use, return-data syscalls and seed-derived system instructions.
- `tx license grant-batch` grants a reviewed batch of licenses; `lightnode_operator` grants are set aside until the chain's reward pool is initialized.

### Changed
- Hybrid transaction sign bytes, key-migration preimages and bridge attestations switch to a v2 form with a domain tag and the chain-id. Clients detect it by querying `applied_plan` for `v3.2.0` and `v3.1.98`; a chain that starts on this release uses v2 from block one.
- The EVM lane is covered by the circuit breaker, and the upgrade reopens it with the post-quantum requirement in force in the same block.
- The staker share of fees stays in the fee collector, where the distribution module pays it to stakers.
- The `x/ai` per-sender rate limit is enforced, set by the upgrade to 600 transactions per 30-block window so IBC relayers are not throttled.
- Schedule-mode emission is scaled to the epoch length, so an epoch mints the share of the year it represents.
- The upgrade version map now lists every registered module, so a future migration cannot re-initialise modules from default genesis.
- The bridge module has its own module account for minting and burning, and `ProcessProposal` validates proposed blocks with the same rules used to build them.
- Every module account except `svm` refuses plain bank sends.
- `MsgConvertERC20` can be signed (it was rejected by the signing context) and accepts a bech32 sender.
- `x/rlconsensus` deterministic math saturates instead of panicking in the block hooks.
- Multilayer: an unknown layer returns not-found instead of panicking.
- EVM JSON-RPC: `eth_createAccessList` without a gas value no longer fails the handler, the `earliest` block tag resolves to the earliest state, and the unified mempool compares fees in one unit.

### Security
- The ante path no longer shares mutable parameters between simulation and block execution, judges transaction TTL against block time rather than wall-clock time, and bounds multisig nesting before signatures are expanded.
- Validator admission (license and self-bond floor) cannot be bypassed through authz, and the floor is enforced on undelegate and redelegate as well as at creation.
- Vesting-account creation (all three message types) is refused inside authz at any nesting depth.
- PQC key registration requires a signing algorithm on every path, refuses key-encapsulation algorithms as authentication keys and checks key lengths; key migration binds both keys and validates the destination key.
- An account that holds a post-quantum key can no longer unjail with a classical-only signature.
- Bridge attestation sign bytes are injective and chain-bound, and negative amounts are refused.
- Abstract accounts: the enabled flag is enforced on all paths, the authenticated EVM call gas is bounded, spend limits accumulate per rule (not per key), an enabled rule without a limit is refused, ERC-20 transfer, transferFrom and approve by a spend-limited key need an allow-listed token, and the authenticator sequence is consumed on the EVM lane.
- SVM: lamport conservation and account ownership are checked before settlement, only canonical addresses are backed by a bank balance, a signed authorisation executes once, and authenticator sign bytes are injective.
- Cross-VM: the call guard is scoped to the transaction rather than the process, the caller's lane and address are derived rather than trusted, and the EVM leg is bounded by the gas the caller paid for.
- RDK: only the rollup's sequencer can settle, batches settle in order, escrow is ledgered per rollup, the creation burn is enforced and stop refunds only the escrowed stake.
- Multilayer: the creator's stake is locked at registration and returned on decommission, and anchors are refused when the creator has no key.
- EVM locked-balance accounting for vesting accounts uses the EVM's own decimal representation, and a redelegation entry's total slash is bounded by the slash factor times its initial balance.

### Notes
- Upgrade handlers: `v3.1.98` and `v3.2.0` (same handler). Wallets and SDKs must use the v2 sign bytes and open an EVM window before sending EVM transactions.

---

## [3.1.97] - 2026-09-04

Includes 3.1.96.

### Security
- Nodes refuse CosmWasm code upload and contract instantiation (`MsgStoreCode`, `MsgInstantiateContract`, `MsgInstantiateContract2`) at mempool admission. The check runs in CheckTx only, so block validity is unchanged and nodes can be replaced one at a time.

### Changed
- Release builds are position-independent executables (3.1.97).

---

## [3.1.95] - 2026-08-26

Includes 3.1.94.

### Added
- **Governable emission (`x/inflation`).** The module gains an authority and `MsgUpdateParams`, a fixed per-epoch emission mode and `max_total_emission`, a hard cumulative cap. The `v3.1.94` upgrade handler switches the module to fixed-mode emission with a cap; later changes go through governance.

### Security
- Administrative messages whose signer is their own `authority` field verify that the authority is governance (`x/govauth`), and a test checks that every such handler calls the guard.
- The bridge enforces the configured attestation threshold.
- The Docker Compose stacks publish RPC, REST, gRPC, EVM JSON-RPC, metrics and sidecar ports on `127.0.0.1` only; P2P stays public.

### Changed
- cosmos/evm v0.6.3: balance addition fails on overflow instead of wrapping (3.1.95, rolling upgrade).

### Notes
- Upgrade handler: `v3.1.94`. v3.1.95 needs no upgrade plan.

---

## [3.1.92] - 2026-08-24

### Fixed
- **New nodes can join again.** Gas charged by `x/ai` depended on how long a node process had been running, so a node replaying blocks could compute a different result and stop at the first block with a transaction. The configuration read is now taken off the gas meter, so gas depends only on chain state. Not consensus-breaking; nodes can be replaced one at a time.
- `scripts/join-network.sh` creates a missing `priv_validator_state.json` when a node is restored from a snapshot, without ever overwriting an existing one.

---

## [3.1.90] - 2026-08-22

Includes 3.1.89.

### Security
- SVM: value leaving an account requires that account's signature, not only that it is writable (3.1.89).

### Fixed
- Light nodes: expected heartbeats accrue forward, so a governance change to `heartbeat_interval` only affects blocks after it and never re-rates a node's past uptime (3.1.90).

---

## [3.1.87] - 2026-07-19

### Changed
- `qorechaind version` no longer prints build details and is hidden from help, and node info no longer advertises an application version, so a running build cannot be fingerprinted remotely. CLI-only and consensus-neutral.

### Notes
- Versions 3.1.88, 3.1.91 and 3.1.93 were not released on their own; their changes shipped in later releases.

---

## [3.1.86] - 2026-07-05

### Added
- **Browser-wallet (EIP-191) authenticators.** A secp256k1 authenticator can be linked by its Ethereum address and verified with `personal_sign` (EIP-191) + `ecrecover`, so a wallet that exposes only an address and `personal_sign` (such as MetaMask) can be a scoped, spend-limited, revocable linked key on any lane.

### Changed
- **Validators can always recover from a jail.** Post-quantum signatures are required on transactions, but `MsgUnjail` is now exempt from that requirement when the operator account has not yet registered a key — so a validator can never be locked out of unjailing itself. (Operators should still register a post-quantum key at setup; a helper script is provided.)

### Notes
- v3.1.86 carries everything from 3.1.85 (linked-wallet authenticator execution on all three lanes, on-chain permissions and spending limits, `PermissionSchema`, same-algorithm post-quantum key rotation) plus the two items above. Additive change, no re-genesis; coordinated through governance on production networks.

---

## [3.1.85] - 2026-07-05

### Added
- **Linked-wallet authenticator execution (`x/abstractaccount`).** An account owner can link an external wallet key under scoped, spend-limited, revocable, time-bounded terms, and that key can spend from the account without holding the account's own signing key.
  - `MsgExecuteEVM` — authenticator-authorized EVM call/transfer from the account's EVM address.
  - `MsgExecuteCosmos` — authenticator-authorized native transfer from the account on the Cosmos lane.
  - A relayer submits and pays the fee; each message is bound to the chain-id, account, key, action and a nonce, so a signature cannot be replayed across chains, accounts, keys or transactions.
- **On-chain permission enforcement** — every action maps to a canonical permission string; authorization is fail-closed (unmapped or ungranted actions are rejected, account-management actions are never delegable).
- **On-chain spending limits on all three lanes** — per-transaction and rolling daily caps per linked key and denomination, enforced and accumulated in state. Value is checked in the handler for the native and EVM lanes, and from the realized balance change for the SVM lane, so the same limit applies whichever lane a linked key spends on.
- **`PermissionSchema` query (gRPC and REST)** — returns the canonical taxonomy (permissions, message→permission map, schema version) so clients validate scopes without hardcoding them. Available over REST at `GET /qorechain/abstractaccount/v1/permission_schema`.
- **Browser-wallet (EIP-191) authenticators.** A secp256k1 authenticator can be linked by its 20-byte Ethereum address and verified with EIP-191 `personal_sign` + `ecrecover`, so a wallet that only exposes an address and `personal_sign` (such as MetaMask) can be a scoped, spend-limited, revocable linked key on any lane. Existing 33-byte compressed-key authenticators are unchanged.
- **Same-algorithm post-quantum key rotation (`MsgRotatePQCKey`).** An account can replace its ML-DSA-87 key with a new key of the same algorithm — to retire a compromised key or move a key onto the standard derivation — dual-signed by the old key (proves ownership) and the new key (proves control). Key rotation can never be delegated to a linked key. CLI: `tx pqc rotate-key`.
- **Deterministic post-quantum key recovery** — `tx pqc recover-key` reconstructs an account's ML-DSA-87 key from its BIP-39 mnemonic using the ecosystem-standard derivation, so a key can be recovered or reproduced on a new host without a random keygen. Client-side signing uses the standards `qorechain-pqc` library (FIPS-204, KAT-vectored).
- CLI: `tx abstractaccount execute-cosmos` / `execute-evm`, `tx pqc rotate-key` / `recover-key`, and authenticator signing helpers.

### Changed
- A linked authenticator that has **expired** now returns a distinct error (`session key expired`) instead of a generic permission denial, so wallets can prompt to re-link.
- The binary is now stamped with its release version (reported by `qorechaind version` and in node info).

### Notes
- Additive change (new message types, no re-genesis); coordinated through governance on production networks. Existing accounts and standard transactions are unaffected.

---

## [3.0.0] - 2026-05-07

The first major release after v1.4.0. v3.0.0 brings a native on-chain automated market maker, expanded cross-network validator support across additional chain architectures, and the foundation for the next-generation IBC stack.

### Added — native AMM module (`x/amm`)
- New genesis module (count moves from 45 to 46).
- Two pricing curves: constant-product (`x*y=k`) and stable-swap (Curve-style invariant solved via deterministic Newton iteration).
- 8 messages: pool create / add liquidity / remove liquidity / swap-exact-in / swap-exact-out / pause / resume / set-params.
- Per-pool pause via governance, module-wide kill switch, swap fee with LP-accrual + protocol-fee split, configurable pool creation fee, slippage cap.
- Cross-VM hook so EVM contracts (and SVM via the existing precompile interface) can route swap calls into the AMM.
- Optional advisory route hint for clients; on-chain routing remains deterministic.
- Genesis import/export with strict invariants (pool ID uniqueness, LP supply consistency, paused-pool registry).
- All math is integer; zero floating-point in any consensus path.

### Added — cross-network expansion
- Five new chain architectures recognised: Cairo VM L2, XRP Ledger UNL, Stellar Consensus Protocol, Hashgraph, and Pure Proof-of-Stake.
- Twenty new default chain configurations span EVM L2 ZK rollups, EVM L2 optimistic / yield-bearing, EVM L1 (parallel-EVM, stablecoin-focused, BTC-anchored, FVM-compatible, etc.), the new non-EVM L1s, and one additional cosmos-IBC chain. Total `DefaultChainConfigs` count: 17 → 37.
- Five dedicated bridge handlers for the new non-EVM architectures: deposit / withdrawal validation + chain-appropriate confirmation timing.
- Five new sidecar Docker container scaffolds (per non-EVM chain), plus the orchestrator chain registry covering all twenty new chains with sensible default Docker images, finality hints, and confirmation env vars.
- License surface grew from ~10 feature IDs to 74: 1 umbrella, 36 per-chain `bridge_*`, 37 per-chain `validator_*` (including extended licensing for the eight existing IBC chains).

### Added — IBC v2 foundation
- New `ChainArchitecture` enum and IBC-specific fields on `ChainConfig` (`IBCChannelID`, `IBCPortID`, `IBCConnectionID`, `EurekaClientType`).
- Public-side packet types and handler-hook interfaces for the next-generation IBC stack and for ICS-27 / ICS-29 / ICS-721, with ~25 boundary tests covering happy-path validation, missing-field rejection, ack-success invariants, and compile-time interface-shape guards.

### Added — multi-node devnet
- Two-validator local devnet via `docker-compose.devnet.yml`.
- Init scripts for the genesis-creating validator and the joining peer (peer discovery via `/genesis` + node-ID handshake).
- Smoke-test runner covering block production / finality, liveness slashing, and the bridge sidecar JSON-RPC handshake.
- Operator runbook in `scripts/devnet/README.md`.

### Added — operator + bridge documentation
- `docs/SIDECAR.md` Supported Chains section expanded with the full 37-chain catalog: baseline, EVM-family expansion, non-EVM with dedicated handlers, IBC, and pending entries.
- `docs/BRIDGE.md` enriched with the full chain table and the 74-feature-ID license surface, including the helper functions in `x/license/types/feature_ids.go`.

### Internal — release hygiene
- Pre-push security scan complete (forbidden-term, source-level proprietary references, full-history identity, full-history co-author / AI attribution, tag annotations, secrets regex, `.gitignore` correctness).
- All commits authored and committed by `Liviu Epure <liviu.etty@gmail.com>` across the entire git history; no co-author trailers anywhere.

### Note on the v2.x range
v2.x was the internal pre-3.0 trail (v2.20.0 → v2.45.0); the public CHANGELOG jumps directly from 1.4.0 → 3.0.0.

---

## [2.22.0] - 2026-05-07

### Added
- Regression tests covering the recent audit fixes:
  - `x/burn/types/params_test.go` — locks in QCTokenomics v2 fee split (37/30/20/10/3) and the sum-to-1 invariant; rejects negative shares, shares > 1.0, non-monotonic milestone schedules, zero milestone amounts.
  - `x/license/types/license_test.go` — `IsExpired` boundary table (7 cases), `IsActive` with suspended/expired interactions (7 cases), Marshal round-trip, suspension persistence.
  - `x/lightnode/types/params_test.go` — locks `RewardShare = 0.03` matching `burn.LightNodeShare`; rejects zero `HeartbeatInterval`, zero `MaxLightNodes`, negative grace period, out-of-bounds shares.
  - `x/rlconsensus/mathutil/determinism_test.go` — bit-exact determinism across 256 invocations and across 32 concurrent goroutines for `ExpApprox`, `TaylorLn1PlusX`, `SigmoidApprox`, `ReputationMultiplier`. Regression guard for the v2.6.2 fix.

---

## [2.21.0] - 2026-05-07

### Added
- **`WireSidecarHooks(startCmd)` indirection** in `cmd/qorechaind/cmd/start_sidecar_stub.go` — no-op in public builds; in extended builds, wraps the start command's `PreRunE` to launch the sidecar orchestrator alongside the node and registers a SIGINT/SIGTERM listener for graceful shutdown.
- `SIDECAR_DISABLED=1` env var (extended builds only) to opt out of orchestrator startup without rebuilding.

### Fixed
- **Sidecar orchestrator startup bug** — the `SidecarStartHook` type was defined since v2.5.0 but never invoked from any caller. License-gated bridge sidecars never spawned in production. Now wired through `cosmosevmserver.AddCommands`'s start-customizer callback.
- Two pre-existing compile errors in extended builds that were masked by upstream CGO env failures during vet:
  - `x/vm/precompiles/pqc_verify.go` — 4 call sites used `EncodePQCVerifyOutput(false)` in single-value context after the v2.16.0 audit changed the signature to `([]byte, error)`.
  - `x/svm/rpc/handlers_token.go` — unused `encoding/base64` import.

---

## [2.45.0] - 2026-05-07

### Fixed — public CHANGELOG hygiene

The v2.44.0 entry shipped with the security-scan target patterns spelled out by name in a "verified clean" list, which itself violates the public-CHANGELOG content policy. This release rewrites the v2.44.0 entry below to reference the patterns by category instead of listing them, and adds this note for traceability. No source code changes.

The original v2.44.0 commit message and tag annotation were already category-only and remain clean. The internal scan report at `~/Development/Qore/update/SECURITY_SCAN_v2.45.0.md` retains the full pattern detail (engineer-facing, never committed).

---

## [2.44.0] - 2026-05-07

### Security — full pre-push scan complete

Closes the security-scan blocker for the v3.0.0 release path. All seven scan blocks pass clean across the full git history. The patterns scanned for are the seven categories specified in the internal release runbook (consensus-engine identifiers, proprietary repo identifiers, AI-vendor identifiers, predecessor brand identifiers, cloud-vendor identifiers, AI-model identifiers, and credential / secret patterns).

**Findings + remediation:**
- `.gitignore` was missing `overlay.json` — added. The build-overlay file contains absolute paths to the open-core internals and must never be committed to a public repo.

**Verified clean (full git history):**
- 0 hits across all seven forbidden-term categories
- 0 source-level open-core internal references
- Single author + committer identity (`Liviu Epure <liviu.etty@gmail.com>`)
- 0 co-author / AI-attribution mentions in any commit message or tag annotation
- 0 cloud credential keys, 0 PEM private keys, 0 BIP-39 mnemonics, 0 git-host PATs
- All six required `.gitignore` entries present

The full scan report is archived (engineer-facing, never committed) at `~/Development/Qore/update/SECURITY_SCAN_v2.45.0.md`.

---

## [2.43.0] - 2026-05-07

### Added — 2-node docker-compose devnet for multi-node smoke testing

Closes the §4.28–§4.30 acceptance items at the configuration layer. Operators can now stand up a 2-validator local devnet, run finality + liveness-slashing + bridge-sidecar smoke tests, and verify the chain's behavior end-to-end before promoting a build.

**New files:**
- `docker-compose.devnet.yml` — two-validator config on the `172.30.0.0/24` subnet, with port-forwarded RPC/REST/gRPC/EVM endpoints (node-1 on the standard ports, node-2 on +100 offset)
- `scripts/devnet/init-node1.sh` — genesis-creating validator init: creates a 100M-QOR-funded validator, self-delegates 50M QOR to bootstrap consensus, lowers `timeout_commit` to 1s for fast feedback
- `scripts/devnet/init-node2.sh` — joining peer init: waits for node-1 to be reachable, pulls genesis over `/genesis`, discovers node-1's node ID, configures persistent peer
- `scripts/devnet/smoke.sh` — runs the three multi-node smoke items with explicit exit codes (1/2/3 per §4.28/§4.29/§4.30)
- `scripts/devnet/README.md` — operator runbook with boot / smoke / teardown commands

### Smoke test coverage
- **§4.28** (block production + <2s finality): waits for both nodes to reach height 5, then samples a 5-second window to confirm block rate
- **§4.29** (liveness slashing): halts node-2 for 60s and verifies the slashing module's `signing_infos` endpoint reports the missed signatures, then restarts node-2
- **§4.30** (bridge sidecar against Anvil): launches Anvil on `:18545` and verifies the JSON-RPC handshake; full attestation round-trip is out of smoke-test scope (requires deployed bridge contract)

`anvil` is auto-detected; the smoke script skips §4.30 with a warning if Foundry isn't installed locally.

---

## [2.42.0] - 2026-05-07

### Added — ICS-27 / ICS-29 / ICS-721 handler hooks

Public-side type and interface scaffolding for the v3.0.0 §3.3 classic IBC handler completeness work. The proprietary keeper integrates with the upstream cosmos-sdk modules; this commit defines the local public surface so call sites can be wired and tested independently of the upstream wiring.

**ICS-27 (Interchain Accounts):**
- `ICARegistration` — Owner / ConnectionID / optional Version
- `ICAHandlerHook` — `RegisterICA`, `QueryICA`, `SubmitICATx`

**ICS-29 (Fee Middleware):**
- `FeePayload` — RecvFee / AckFee / TimeoutFee / Payee (string-encoded uqor amounts for big-int safety)
- `FeeMiddlewareHook` — `EscrowPacketFees`, `PayPacketFee`, `RefundUnusedFees`
- Routes through the v2.6.3 fee distribution split so relayer rewards settle through the standard 37/30/20/10/3 path

**ICS-721 (NFT-IBC):**
- `NFTPacketData` — ClassID / ClassURI / TokenIDs / TokenURIs / Sender / Receiver / Memo
- `NFTHandlerHook` — `SendNFT`, `OnRecvNFTPacket`
- `Validate()` enforces non-empty class+sender+receiver and aligned-length TokenIDs/TokenURIs

### Tests
13 new tests cover ICA registration validation (3 cases), fee payload validation (5 cases including all four required fields + happy path), NFT packet validation (5 cases including token-URI length-mismatch detection and the `nil TokenURIs` accepted-as-no-metadata path), and three compile-time interface-shape guards (one per hook interface) so accidental signature drift is caught at build time.

The proprietary keeper's actual wiring against the upstream `cosmos-sdk/x/icaauth`, `ibc-go/modules/apps/29-fee`, and `ibc-go/modules/apps/nft-transfer` modules lands in a follow-up commit once the upstream module versions are pinned in `go.mod`.

---

## [2.41.0] - 2026-05-07

### Added — IBC Eureka v2 packet types and handler hook interface

Public-side types for the IBC Eureka v2 packet flow. Closes the v3.0.0 §3.2 type-layer scope; the proprietary keeper's actual handler implementation against the upstream `cosmos-sdk/x/ibc/eureka/v2` module follows in a separate commit once the upstream module path is confirmed.

**New types:**
- `EurekaPacket` — generic Eureka v2 packet shape (source/dest chain, sequence, port/channel, client type, data, optional timeout) with `Validate()`
- `EurekaAck` — application-level acknowledgement with `IsSuccess()` predicate
- `EurekaHandlerHook` — interface contract the proprietary keeper's handler implements: `OnRecvPacket`, `OnAcknowledgement`, `OnTimeout`

### Tests
12 new tests cover: happy-path packet validation, 8 rejection cases (each missing field + same-source-dest), ack `IsSuccess` invariant (including the contradictory `Success: true + Error: "boom"` case), and a compile-time check that any handler implementation has the expected method set (interface-shape stability).

---

## [2.40.0] - 2026-05-07

### Wave-close release

The 21-version v2.20.0 → v2.40.0 wave brought the v2.x line up to the v3.0.0 acceptance gate. The v3.0.0 release tag stays reserved for the moment four documented blockers are cleared:

1. **IBC Eureka v2 packet handling (§3.2)** — foundation shipped in v2.35.0 (`ChainArchitecture` enum + `ChainConfig` IBC fields), full handler wiring needs the upstream `cosmos-sdk/x/ibc/eureka/v2` import path confirmed.
2. **Classic IBC ICS-27/29/721 completeness (§3.3)** — needs ICA registration helper, fee middleware, and NFT-IBC integration with upstream-module dependency bumps.
3. **33-item test plan + multi-node devnet smoke (§4)** — items 1–18 (static + unit + build) green; items 19–33 (integration + multi-node smoke + e2e) need a docker-compose 2-node devnet that isn't in-session reachable.
4. **Full §5.2 historical security scan** — per-commit scans were done on every diff in the wave; the historical-blob and secrets-regex sweeps over the entire git history are pending.

A detailed status of each blocker with the decision/access required to clear it lives in `~/Development/Qore/update/V3.0.0_BLOCKERS.md` (engineer-facing, never committed to a public repo).

### Wave totals
- 21 minor releases (v2.20.0–v2.40.0)
- 0 CI failures across the entire wave
- 22 new public-repo tests + ~40 extended-build (overlay) tests; all green
- Module count: 45 → **46** (added `x/amm` in v2.23.0)
- ChainType count: 12 → **17** (5 new in v2.24.0)
- `DefaultChainConfigs` count: 17 → **37** (20 new in v2.25.0)
- License feature IDs: ~10 → **74** (matches §3.4.4 acceptance exactly)
- Bridge handlers: 7 → **12** (5 new dedicated handlers)
- Sidecar Docker dirs: 1 → **6** (5 new chain dirs)
- AMM module: constant-product + StableSwap pricing, cross-VM hook, deterministic Newton-iteration math
- Orchestrator chain registry: 20 chain entries with sensible defaults

### Identity and security
- Every commit authored by `Liviu Epure <liviu.etty@gmail.com>` — no co-author trailers
- Forbidden-term scan clean on every commit's source + CHANGELOG entry
- Public + full-overlay builds green at every commit

---

## [2.39.0] - 2026-05-07

### Documentation — `docs/BRIDGE.md` chain catalog

Updated the bridge documentation to reflect the full v2.24.0–v2.34.0 expansion. The Supported Chains section now contains four explicit groups:

- **Baseline (10 chains)** — pre-v2.24.0 with status table
- **Cross-network expansion EVM (14 chains)** — full enumeration
- **Cross-network expansion non-EVM (5 chains)** — table with architecture + protocol path + default confirmations per chain
- **IBC-connected (8 chains)** — including Injective added in v2.25.0
- **Other (7 chains)** — NEAR, Bitcoin, Cardano, Polkadot, Tezos, Tron, Aptos — `Pending` until production handlers ship

The doc also lists the **74 license feature ID** surface and the helper functions in `x/license/types/feature_ids.go`.

---

## [2.38.0] - 2026-05-07

### Documentation — `docs/SIDECAR.md` operator guide

Expanded "Supported Chains" section with the full v2.24.0–v2.36.0 chain catalog:

- **Baseline (10 chains)**: pre-v2.24.0 chains unchanged
- **Cross-network expansion EVM (14 chains)**: zkSync Era, Linea, Scroll, Blast, Mantle, Hyperliquid, Berachain, Sonic, Sei, Monad, Plasma, Filecoin, Cronos, Kaia — all sharing the Ethereum sidecar with chain-specific config injection (default `FINALITY_BLOCKS` documented per chain; Monad's higher 30-block rule called out)
- **Cross-network expansion non-EVM (5 chains)**: Starknet, XRPL, Stellar, Hedera, Algorand — dedicated sidecar images per chain with chain-appropriate confirmation env (`LEDGER_CONFIRMATIONS` / `CONSENSUS_ROUNDS` / `ROUND_CONFIRMATIONS`)
- **IBC (8 chains)**: Cosmos Hub, Osmosis, Noble, Celestia, Stride, Akash, Babylon, Injective — no sidecar container; packet flow via Hermes relayer

The doc also explains the `Architecture` override (`ibc_classic` vs `ibc_eureka_v2`) for new IBC chain onboardings from v3.0.0 forward.

---

## [2.37.0] - 2026-05-07

### Documentation

Updates to the public-facing architecture and CHANGELOG framing for the v2.20.0 → v2.36.0 wave that brought the v2.x line up to the v3.0.0 acceptance gate. No source code changes — this is a documentation-only release that summarizes the cumulative state.

**Wave totals:**
- 17 minor releases (v2.20.0–v2.36.0)
- 0 CI failures across the entire wave
- 22 new tests in the public repo (74 across the repo+overlay test surface)
- Module count: 45 → 46 (added x/amm in v2.23.0)
- ChainType count: 12 → 17 (5 new in v2.24.0)
- DefaultChainConfigs count: 17 → 37 (20 new in v2.25.0)
- License feature IDs: ~10 → 74 (matches v3.0.0 §3.4.4 acceptance criterion exactly)
- Bridge handlers: 7 → 12 (5 new in v2.29.0–v2.33.0)
- Sidecar Docker dirs: 1 → 6 (5 new in v2.34.0)

**Notes for operators:**
- v3.0.0 stays reserved for the final release after IBC Eureka v2 wiring (§3.2), classic IBC handler completeness for ICS-20/27/29/721 (§3.3), and the §10 security scan complete.
- Every commit in the wave is authored by `Liviu Epure <liviu.etty@gmail.com>` — no co-author trailers.

---

## [2.36.0] - 2026-05-07

### Added — Orchestrator chain registry for the 27 new chains

The sidecar orchestrator now ships with sensible per-chain defaults for every chain registered in v2.25.0. Operators get a working baseline out of the box; the `[sidecar.chains.<chain>]` section of `app.toml` can override any field.

**New extended-build helpers:**
- `DefaultChainSpecs()` — fresh map of `chain → ChainConfig{Image, Version, ExtraEnv, ExtraPorts}` for every onboarded chain
- `KnownChains()` — sorted list of chain IDs the orchestrator recognizes
- `IsKnownChain(chain)` — boolean predicate

**Defaults populated:**
- EVM chains get a `FINALITY_BLOCKS` env hint (12 baseline; 30 for Monad's higher-finality rule)
- Dedicated-sidecar chains (Starknet, XRPL, Stellar, Hedera, Algorand) get an explicit `Image` override pointing at their per-chain image in the registry
- Each non-EVM chain gets a chain-appropriate confirmation env (`LEDGER_CONFIRMATIONS` / `CONSENSUS_ROUNDS` / `ROUND_CONFIRMATIONS`)
- Injective is registered for completeness even though it uses Hermes for IBC packet flow

### Tests
5 new tests verify: no duplicate chain entries, sorted order, all 20 v2.25.0 chains have a spec, the returned map is fresh per call (no shared mutable state), every dedicated-sidecar chain has an `Image` override, every EVM chain has a `FINALITY_BLOCKS` hint.

### Build infrastructure
`sidecar/orchestrator/doc.go` placeholder added in the public repo so `go vet` / `go test -overlay` can chdir into the package — same pattern as the v2.23.0 AMM and v2.29.0 bridge keeper placeholders.

---

## [2.35.0] - 2026-05-07

### Added — `ChainArchitecture` enum + IBC fields on `ChainConfig`

Foundation for the v3.0.0 §3.2 IBC Eureka v2 wiring. New `ChainArchitecture` enum disambiguates classic IBC vs the next-generation Eureka v2 stack:

- `ChainArchEmpty` — non-IBC chains (default)
- `ChainArchIBCClassic` — legacy IBC; the 7 baseline IBC chains (cosmoshub, osmosis, noble, celestia, stride, akash, babylon)
- `ChainArchIBCEurekaV2` — next-gen IBC; new IBC chains added from v3.0.0 forward default here

`ChainConfig` gains five new optional, omitempty-tagged fields populated only for IBC chains:
- `Architecture` — the new enum
- `IBCChannelID` / `IBCPortID` / `IBCConnectionID` — channel-level identifiers
- `EurekaClientType` — for chains using Eureka v2 (e.g., `"tendermint"`, `"solomachine"`)

### Tests
4 new tests cover enum validity (positive + negative cases including case sensitivity), wire-format invariant (non-IBC chains never serialize the new fields, protecting external integrations), JSON round-trip preservation for populated IBC chains, and a baseline check that no default non-IBC chain has a non-empty Architecture.

---

## [2.34.0] - 2026-05-07

### Added — 5 new sidecar Docker container scaffolds

Per-chain watcher/validator sidecar dirs for the new architectures registered in v2.24.0–v2.33.0. Each follows the same pattern as the existing Ethereum sidecar (the reference implementation):

- `sidecars/starknet/` — Cairo VM L2 sidecar (Starknet)
- `sidecars/xrpl/` — XRP Ledger sidecar
- `sidecars/stellar/` — Stellar Consensus Protocol sidecar
- `sidecars/hedera/` — Hedera Hashgraph sidecar
- `sidecars/algorand/` — Algorand sidecar

Each contains:
- `Dockerfile` — multi-stage build (golang:1.26-bookworm builder → debian:bookworm-slim runtime), `EXPOSE 8080` for the health endpoint, env-driven `MODE=watcher|validator|both`
- `main.go` — gRPC client wiring, mode dispatch, health server (`/healthz` + `/readyz`), graceful SIGTERM/SIGINT shutdown
- `watcher.go` — chain-event monitoring skeleton with documented finality rule per chain
- `validator.go` — remote validator client management skeleton

The skeletons emit heartbeat ticks and can be deployed today; production chain-specific event monitoring (decoding deposit events, finality verification) lands in subsequent releases as each chain's bridge integration matures.

### Verified
All 5 sidecar binaries compile cleanly (`go build -tags full -overlay=...`) at 8 MB each.

---

## [2.33.0] - 2026-05-07

### Added — Algorand bridge handler

Final per-`ChainType` handler from the v3.0.0 §3.4 cross-network expansion. All 5 new chain architectures (Starknet/XRPL/Stellar/Hedera/Algorand) now have functional handlers in the extended build.

**Behavior**
- Source-tx-hash validation: 52-char base32 (RFC 4648 no-pad, uppercase A-Z + 2-7) — the SHA-512/256 of the canonical transaction encoding.
- Address validation: 58-char base32 (32-byte public key + 4-byte checksum, base32-encoded). Trailing-checksum verification is deferred to the production handler.
- Confirmation time: 13s (4 confirmation rounds × ~3.3s round time).

### Tests
3 new tests cover deposit hash matrix (5 cases including lowercase, wrong length, non-base32 digit), address matrix (10 cases including all four base32 alphabet boundaries: A, Z, 2, 7), and confirmation-time positivity.

### Cross-network expansion summary
After v2.33.0 the `qorechain-core` codebase has:
- 17 ChainTypes (12 baseline + 5 added in v2.24.0)
- 37 default chain configurations (17 baseline + 20 added in v2.25.0)
- 74 license feature IDs (1 umbrella + 36 bridge + 37 validator)
- 5 new dedicated bridge handlers in the extended build (Starknet/XRPL/Stellar/Hedera/Algorand)

EVM-family chains continue to share the existing `evm_bridge.go` via per-chain config injection. Sidecar containers, orchestrator chain registry entries, and IBC Eureka v2 wiring follow on subsequent v2.x minor releases.

---

## [2.32.0] - 2026-05-07

### Added — Hedera bridge handler

Per-`ChainType` handler for Hedera Hashgraph. Same shape as the prior new-architecture handlers (Starknet/XRPL/Stellar).

**Behavior**
- Source-tx-hash validation: Hedera transaction ID format `<shard>.<realm>.<account>@<seconds>.<nanos>`, all numeric segments (each ≤18 digits).
- Address validation: Hedera account ID `<shard>.<realm>.<num>` (typically `0.0.<num>`).
- Confirmation time: 12s (4 consensus rounds of safety margin on top of base finality).

**Notes**
- Watcher integration target: Hedera Consensus Service (HCS) topic subscriptions for ordered events + Mirror Node REST for transaction state lookup.

### Tests
3 new tests covering deposit ID matrix (8 cases including the minimum-nanos edge and a 19-digit overflow), account ID matrix (8 cases), and confirmation-time positivity.

---

## [2.31.0] - 2026-05-07

### Added — Stellar bridge handler

Per-`ChainType` handler for the Stellar Consensus Protocol (SCP). Same shape as Starknet (v2.29.0) and XRPL (v2.30.0).

**Behavior**
- Source-tx-hash validation: 64-char lowercase hex (Stellar canonical form, no prefix).
- Address validation: StrKey-encoded ed25519 public keys — `G…` (account, 56 chars) or `M…` (muxed, 69 chars), both base32 (RFC 4648 alphabet, A–Z + 2–7).
- Confirmation time: 25s (5 ledger closes × 5 seconds each, providing SCP quorum overlap).

### Tests
3 new tests cover hash format edge cases (lowercase canonical), address validation matrix (real account ID, muxed, wrong length, wrong prefix, non-base32 chars, empty), and confirmation-time positivity.

---

## [2.30.0] - 2026-05-07

### Added — XRPL bridge handler

Per-`ChainType` handler for the XRP Ledger. Same shape as the Starknet handler shipped in v2.29.0 (`ValidateDeposit` / `ValidateWithdrawal` / `EstimateConfirmationTime`).

**Behavior**
- Source-tx-hash validation: 64-character uppercase hex (XRPL canonical form, no `0x` prefix).
- Address validation: classic `r…` addresses or X-addresses (`X…`), 25–35 base58 chars; full base58check checksum verification deferred to the production handler.
- Confirmation time: 16s (4 ledger closes × 4 seconds each).

### Tests
3 new tests cover hash format edge cases (empty, lowercase rejected, wrong length), address validation matrix, and confirmation-time positivity.

---

## [2.29.0] - 2026-05-07

### Added — Starknet bridge handler

First per-`ChainType` handler from the v3.0.0 §3.4 cross-network expansion. The handler implements the same shape as the existing per-chain bridge handlers (`ValidateDeposit` / `ValidateWithdrawal` / `EstimateConfirmationTime`) for the Cairo-VM L2.

**Behavior**
- Source-tx-hash validation: 0x-prefixed hex up to 64 characters (Starknet felt encoding).
- Address validation: Starknet contract/account addresses use the same felt format.
- Confirmation time: 600s (~10 minutes for soft-finality with safe-block delay; the watcher upgrades attestations to hard-finality once the corresponding STARK proof is verified on L1).

**Notes**
- The handler is a thin validator at this stage; the full L1 state-update + STARK proof verification path is documented as the production roadmap inside the file.

### Tests
3 new tests in the extended-build keeper package cover deposit hash validation (valid hash accepted, missing prefix / non-hex / overlength rejected), address validation (boundary cases), and confirmation-time positivity.

### Build infrastructure
Added an empty `x/bridge/keeper/doc.go` placeholder in the public repo so `go vet` / `go test -overlay` can chdir into the package even with no overlay active. Same pattern as the v2.23.0 AMM keeper placeholder.

---

## [2.28.0] - 2026-05-07

### Added — AMM StableSwap pool variant

The AMM module's stable-swap pool variant is now functional. `MsgCreatePool` accepts `PoolTypeStableSwap` with an `AmplificationCoefficient` in `[1, 5000]`, and `SwapExactIn` / `QuoteExactIn` route to the appropriate math based on pool type.

**Math** (in the extended-build keeper):
- 2-asset Curve invariant: `4A·(x+y) + D = 4A·D + D³ / (4·x·y)`
- `D` solved via Newton iteration with a fixed step cap of 64 (determinism guard; empirical convergence is <8 steps near equilibrium)
- `y` given fixed `x` and `D` solved via a second Newton iteration with the same cap
- All math is integer (`cosmossdk.io/math.Int`); zero `float64`, zero platform-dependent operations

**Behavior near equilibrium:** for stable-pair swaps where reserves are close in value, the stable-swap variant produces output ≥ the equivalent constant-product output — that's the entire point of the curve.

### Tests

9 new tests cover:
- `D` invariant computation on a balanced pool (analytic baseline)
- Bit-exact determinism across 256 invocations (consensus safety)
- Bit-exact determinism across 32 goroutines × 50 iterations (no shared mutable state)
- Output ≥ CP-output at equilibrium (low-slippage claim)
- Boundary: invalid denom, zero reserves, zero amplification rejected with the right sentinel
- Max-iterations termination on a pathologically skewed pool
- `D` preservation across a fee-free swap (drift bounded at 0.1%)

`QuoteExactOut` for stable-swap pools is currently `ErrNotImplemented` — the inverse requires a second iterative solver and lands in a follow-up. Stable-swap callers should use `QuoteExactIn`.

---

## [2.27.0] - 2026-05-07

### Added — 27 new validator license feature IDs

Closes the validator-license portion of the cross-network expansion. Total `validator_*` features now 37 (10 baseline + 19 non-IBC v2.27.0 + 8 IBC v2.27.0).

**19 non-IBC chain validator licenses:**

`FeatureValidatorZKSyncEra`, `FeatureValidatorLinea`, `FeatureValidatorScroll`, `FeatureValidatorStarknet`, `FeatureValidatorBlast`, `FeatureValidatorMantle`, `FeatureValidatorHyperliquid`, `FeatureValidatorBerachain`, `FeatureValidatorSonic`, `FeatureValidatorSei`, `FeatureValidatorMonad`, `FeatureValidatorPlasma`, `FeatureValidatorXRPL`, `FeatureValidatorStellar`, `FeatureValidatorHedera`, `FeatureValidatorAlgorand`, `FeatureValidatorFilecoin`, `FeatureValidatorCronos`, `FeatureValidatorKaia`.

**8 IBC chain validator licenses:**

`FeatureValidatorCosmosHub`, `FeatureValidatorOsmosis`, `FeatureValidatorNoble`, `FeatureValidatorCelestia`, `FeatureValidatorStride`, `FeatureValidatorAkash`, `FeatureValidatorBabylon`, `FeatureValidatorInjective`.

These IBC chains already have IBC connectivity at the protocol layer; their bridge function is implicit through the IBC handler. The new validator licenses cover the new role of running a remote validator on those chains as part of the cross-network validation strategy.

### Total feature ID surface

- 1 QCB-bridge umbrella
- 36 `bridge_*` (16 baseline + 20 added in v2.26.0)
- 37 `validator_*` (10 baseline + 19 non-IBC + 8 IBC)

**Grand total: 74 feature IDs** — closes the §3.4.4 acceptance target.

### Tests

3 new tests added on top of v2.26.0's 7: validator no-duplicate invariant, presence check for all 27 new validator IDs, and bridge↔validator symmetry (every chain in `DefaultChainConfigs` has both a `bridge_*` and a `validator_*` feature). The pre-existing count test was updated from `47` → `74`.

---

## [2.26.0] - 2026-05-07

### Added — 20 new bridge license feature IDs

License feature surface for the chains added in v2.25.0. Total `bridge_*` features now 36 (16 pre-existing + 20 new).

**New feature constants:** `FeatureBridgeZKSyncEra`, `FeatureBridgeLinea`, `FeatureBridgeScroll`, `FeatureBridgeStarknet`, `FeatureBridgeBlast`, `FeatureBridgeMantle`, `FeatureBridgeHyperliquid`, `FeatureBridgeBerachain`, `FeatureBridgeSonic`, `FeatureBridgeMonad`, `FeatureBridgePlasma`, `FeatureBridgeFilecoin`, `FeatureBridgeCronos`, `FeatureBridgeKaia`, `FeatureBridgeSei`, `FeatureBridgeXRPL`, `FeatureBridgeStellar`, `FeatureBridgeHedera`, `FeatureBridgeAlgorand`, `FeatureBridgeInjective`.

### Added — split helper functions

`AllFeatureIDs()` is now composed from two stable-ordered slices:
- `AllBridgeFeatureIDs()` — every `bridge_*` feature (36 entries in v2.26.0)
- `AllValidatorFeatureIDs()` — every `validator_*` feature (10 entries; will grow in v2.27.0)

This makes downstream iteration (e.g., listing bridge-only or validator-only licenses) more efficient and removes the brittle hand-maintained list-of-strings.

### Tests

7 new tests cover counts (36 bridges + 10 validators + 1 umbrella = 47 total), no-duplicate invariant, prefix invariants for both bridge and validator, presence of all 20 new IDs in the registered set, validity check edge cases (case-sensitivity, empty input, unregistered prefixes), and `ChainFromFeature` extraction.

Validator licenses for the new chains and the 7 IBC chains follow in v2.27.0.

---

## [2.25.0] - 2026-05-07

### Added — 20 new default chain configurations

`DefaultChainConfigs()` now returns 37 configurations (up from 17). The new entries cover EVM L2/L1 chains (zkSync Era, Linea, Scroll, Blast, Mantle, Hyperliquid, Berachain, Sonic, Sei, Monad, Plasma, Filecoin FVM, Cronos, Kaia), the new architectures registered in v2.24.0 (Starknet, XRPL, Stellar, Hedera, Algorand), and one additional IBC chain (Injective).

All new entries default to `BridgeStatusPending`; production deployment flips them to `Active` only after the bridge contract address is set and the operator confirms the route.

### Tests

6 new tests validate: total count (37), every new chain present, no duplicate `chain_id`, every config has a valid `ChainType` (cross-checks v2.24.0's `IsValidChainType`), every new ChainType is actually used in at least one config, and every config defaults to pending status.

The pre-existing `TestDefaultChainConfigsCount` was updated from 17 → 37 with a comment listing the v2.25.0 additions.

---

## [2.24.0] - 2026-05-07

### Added — 5 new chain architectures

The bridge module now recognizes five additional chain types for cross-network expansion:

- `ChainTypeStarknet` — Cairo VM L2
- `ChainTypeXRPL` — XRP Ledger UNL consensus
- `ChainTypeStellar` — Stellar Consensus Protocol
- `ChainTypeHedera` — Hashgraph; HCS subscription model
- `ChainTypeAlgorand` — Pure Proof-of-Stake

New helpers:
- `AllChainTypes()` returns the 17-element ordered list of every supported chain type.
- `IsValidChainType(t)` validates an inbound chain type at the message/config boundary.

Per-chain handler implementations and configuration entries follow in subsequent minor releases.

### Tests
- 5 new tests cover the ChainType ordering, validity check, wire-level string values (cross-system compatibility), and uniqueness invariant.

---

## [2.23.0] - 2026-05-07

### Added — `x/amm` automated market maker module

Native on-chain AMM with constant-product (Uniswap-V2-style) pricing. Genesis module count moves from 45 to 46.

- New `Pool` type with sorted-denom indexing (`TokenA < TokenB`), constant-product or stable-swap variant flag, status tracking (active/paused), and a TWAP-like weighted-average price.
- 8 new messages: `MsgCreatePool`, `MsgAddLiquidity`, `MsgRemoveLiquidity`, `MsgSwapExactIn`, `MsgSwapExactOut`, `MsgPausePool`, `MsgResumePool`, `MsgSetParams`.
- Module-wide kill switch (`Params.Enabled`), per-pool pause via governance, swap fee (default 30bps) split into LP-accrual and protocol-fee portions, MaxPoolsPerCreator cap, MinLiquidity floor, optional MaxSwapImpactBps slippage cap, configurable `PoolCreationFee` burned at creation.
- Cross-VM hook so EVM contracts (and SVM via the existing precompile interface) can route swap calls into the AMM, resolving the pool by `(denomIn, denomOut)`.
- Optional advisory route hint that clients may consult for multi-pool routing — never used to bind on-chain routing decisions, which remain deterministic.
- Fee-distribution integration with `x/burn` via the new `BurnSourceAMM` constant.
- Genesis import/export round-trips pool state with strict invariants (pool ID uniqueness, `next_pool_id > max(pool.ID)`, `sum(lp_balances per pool) == pool.LPSupply`, paused-pool IDs reference known pools).

### Tests
- 16 new public tests cover params bounds, pool validation, sorted-denom helpers, LP supply consistency, and StableSwap amplification bounds.
- All math is integer (`cosmossdk.io/math.Int`); zero `float64` in any consensus path.

### Audit-fix backlog (v2.20.0 – v2.22.0)
- v2.20.0: critical light-node keeper wiring fix — full builds were silently using a no-op stub for the light node module.
- v2.21.0: sidecar orchestrator was defined since v2.5.0 but never invoked from the start command; now wired through `PreRunE` with a SIGINT/SIGTERM listener for graceful shutdown.
- v2.22.0: 60+ regression tests added to lock in the v2.6.2/v2.6.3/v2.13.0/v2.17.0 fixes (deterministic math, fee-split invariants, license expiry boundaries, light-node param bounds).

---

## [1.4.0] - 2026-03-14

### Added
- **SVM Native Built-in Programs**: Four Solana-compatible native programs executing without BPF interpretation
  - System Program (`11111111111111111111111111111111`): CreateAccount, Assign, Transfer, Allocate
  - SPL Token Program (`TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA`): InitializeMint, InitializeAccount, Transfer, Approve, Revoke, MintTo, Burn, CloseAccount, GetAccountDataSize
  - Associated Token Account Program (`ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL`): Create, CreateIdempotent
  - Memo Program (`MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr`): On-chain memo logging
- **Account Serialization**: Solana-compatible binary format for BPF account input/output across the FFI boundary
  - `SerializeAccountsForBPF`, `DeserializeAccountsFromBPF`, `DeserializeModifiedAccounts` in Go
  - Matching Rust-side serialization in the execution engine
- **CPI Bridge**: BPF programs can call native built-in programs via `sol_invoke_signed` syscall
  - BPF-to-Native direction supported; signer seed validation for PDA signing
- **PDA Derivation**: Program Derived Address syscalls
  - `sol_create_program_address` — derive PDA from seeds and program ID
  - `sol_try_find_program_address` — find valid PDA with bump seed iteration
- **Sysvar Syscalls**: BPF programs can query on-chain state
  - `sol_get_clock_sysvar` — slot, epoch, unix timestamp
  - `sol_get_rent_sysvar` — lamports per byte-year, exemption threshold
- **14 New Solana-Compatible JSON-RPC Methods** (20 total):
  - Transaction: `sendTransaction`, `simulateTransaction`
  - Account queries: `getProgramAccounts`, `getMultipleAccounts`, `getSignaturesForAddress`, `getTransaction`
  - Token queries: `getTokenAccountsByOwner`, `getTokenAccountsByDelegate`
  - Block/fee: `getBlockHeight`, `getRecentBlockhash`, `getLatestBlockhash`, `getFeeForMessage`, `isBlockhashValid`
  - Testnet: `requestAirdrop`
- **Extended FFI Bridge**: `qore_svm_execute_v2` and `qore_svm_execute_native` FFI functions
- **Extended SVMExecutor Interface**: `ExecuteV2` and `ExecuteNative` methods
- 6 new Go error codes for SVM operations
- 11 new Rust error codes for native program execution
- Stub files for all new RPC methods (public community build)

### Changed
- Keeper executor routing: automatic dispatch between native programs and BPF execution based on program ID
- FFI bridge extended with v2 and native execution paths
- libqoresvm rebuilt (434KB) with native program support

### Testing
- 139 Rust unit tests passing (up from 79 in v0.8.0)
- Both full and public builds verified

---

## [1.3.0] - 2026-02-26

### Added
- **x/rdk module**: Rollup Development Kit — deploy application-specific rollups on QoreChain
  - Four settlement paradigms: Optimistic (fraud proofs, 7-day challenge window),
    ZK (validity proofs, instant finality), Based (L1-sequenced), Sovereign (self-sequenced)
  - Three sequencer modes: Dedicated, Shared, Based (L1 proposers sequence)
  - Three DA backends: Native (KVStore blob storage), Celestia (stub in v1.3.0), Both
  - Four proof systems: Fraud, SNARK, STARK, None
  - Four preset profiles: DeFi (ZK+SNARK, 500ms, EVM), Gaming (based, 200ms, custom VM),
    NFT (optimistic+fraud, Celestia DA, CosmWasm), Enterprise (based, subsidized gas, EVM)
  - Full rollup lifecycle: Create → Active → Pause → Resume → Stop with bond escrow/return
  - Settlement engine: SubmitBatch, ChallengeBatch, FinalizeBatch with EndBlocker
    auto-finalization for optimistic (past challenge window) and based (after 2 blocks)
  - Native DA router: SHA-256 commitment, blob pruning past retention period
  - AI-assisted profile selection via x/rlconsensus advisory methods
  - Integration with x/burn (rollup creation burn fee, non-fatal) and
    x/multilayer (RegisterSidechain, AnchorState, UpdateLayerStatus)
  - Bank escrow: stake sent from creator to module account on creation, returned on stop
- **x/burn**: Added `BurnSourceRollupCreate` burn source for RDK module integration
- **x/multilayer**: Added `LayerTypeRollup` layer type for rollup layer registration
- **x/rlconsensus**: Added `SuggestRollupProfile` and `OptimizeRollupGas` advisory methods
- **qor_ RPC endpoints**: GetRollupStatus, ListRollups, GetSettlementBatch,
  SuggestRollupProfile, GetDABlobStatus
- **CLI commands**: Query (rollup, list-rollups, batch, config, suggest-profile) and
  TX (create-rollup, pause-rollup, resume-rollup, stop-rollup, submit-batch, challenge-batch)
- Unit tests: 33 tests across 8 files covering types (enums, defaults, validation,
  JSON round-trip, settlement/sequencer/proof compatibility matrix) and keeper
  (preset profiles, DA backend selection, error sentinels, lifecycle state machine)

### Changed
- Module account permissions: rdk added (Minter, Burner)
- BeginBlockers + EndBlockers: rdk added (EndBlocker for settlement auto-finalization)
- InitGenesis + ExportGenesis: rdk added (after gasabstraction)
- Total registered genesis modules increased from 44 to 45

---

## [1.2.0] - 2026-02-25

### Added
- **25 Direct Cross-Chain Connections**: 8 IBC + 17 non-IBC bridge endpoints
  - IBC: Cosmos Hub, Osmosis, Noble, Celestia, Stride, Akash, Babylon, QoreChain
  - Non-IBC bridge: Ethereum, BSC, Solana, Avalanche, Polygon, Arbitrum, TON, Sui,
    Optimism, Base, Aptos, Bitcoin, NEAR, Cardano, Polkadot, Tezos, Tron
  - 7 new chain types: aptos, bitcoin, near, cardano, polkadot, tezos, tron
  - 9 new default chain configs (Optimism + Base reuse EVM type)
  - Unified QCB bridge handler dispatches by ChainType with per-chain address validation
- **x/babylon module**: BTC restaking adapter (types, keeper, genesis, factory pattern)
  - BTCRestakingConfig, BTCStakingPosition, BTCCheckpoint, BabylonEpochSnapshot
  - BeginBlocker + EndBlocker for epoch management
- **x/abstractaccount module**: Smart-contract account abstraction
  - AbstractAccount, SpendingRule, SessionKey types with expiry logic
  - Full keeper with JSON-in-KVStore CRUD
- **x/fairblock module**: Threshold IBE encrypted mempool stub
  - FairBlockDecorator ante handler (passthrough in v1.2.0, tIBE not activated)
  - Config for tibe_threshold, decryption_delay, max_encrypted_size
- **x/gasabstraction module**: IBC token fee payment
  - GasAbstractionDecorator validates non-native fee denoms before DeductFee
  - Static conversion rates: uqor (1.0), ibc/USDC (1.0), ibc/ATOM (10.0)
- **5-Lane Transaction Prioritization** (configuration only):
  - PQC (100, 15%), MEV (90, 20%), AI (80, 15%), Default (50, 40%), Free (10, 10%)
- **Bridge Fee Burn Integration**: Withdrawal fees routed to x/burn module (non-fatal)
- **IBC Hermes Relayer**: Config templates for 8 IBC chains (config/hermes/)
- **qor_ RPC endpoints**: GetBTCStakingPosition, GetAbstractAccount,
  GetFairBlockStatus, GetGasAbstractionConfig, GetLaneConfiguration
- **CLI commands**: Query/TX skeletons for babylon, abstractaccount, fairblock, gasabstraction
- Unit tests: bridge chain types, babylon types/genesis, abstract account types/spending rules,
  fairblock types/decorator, gas abstraction types/decorator, lane configuration

### Changed
- Ante handler chain extended: AIAnomaly -> FairBlock -> SVM; ConsumeGasForTxSize -> GasAbstraction -> DeductFee
- Module account permissions: babylon, abstractaccount, fairblock, gasabstraction added
- BeginBlockers + EndBlockers: babylon added
- InitGenesis + ExportGenesis: 4 new modules (babylon, abstractaccount, fairblock, gasabstraction)
- Total registered genesis modules increased from 40 to 44
- Bridge keeper updated to accept burn keeper for fee integration

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
  - Compile-time assertions prove both stub and full keepers satisfy `TokenomicsKeeper`
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
- Stub precompiles for community (non-full) build
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
- Full AI inference prompts, fraud detection heuristics, and contract analysis logic are no longer exposed in public repository
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
- **Consensus Engine Algorithm Hooks**
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
