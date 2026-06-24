# Building from Source

This guide walks you through building the `qorechaind` binary from source, covering both the community (open-core) build and the full internal build.

---

## Prerequisites

| Dependency | Minimum Version | Notes |
|------------|----------------|-------|
| **Go** | 1.26+ | Required for all builds |
| **CGO** | Enabled (`CGO_ENABLED=1`) | Required for PQC and SVM FFI bridges |
| **Rust toolchain** | Latest stable | Required to compile `libqorepqc` and `libqoresvm` |
| **Make** | 3.81+ | Build automation |
| **Git** | 2.x | Source checkout |

Verify your environment:

```bash
go version        # go1.26.x or later
rustc --version   # stable toolchain
cargo --version
echo $CGO_ENABLED # must be 1
```

> **Important:** Every `go build`, `go test`, and `go run` invocation **must** have `CGO_ENABLED=1` set. The PQC and SVM modules use FFI bridges that require cgo.

---

## Native Libraries

QoreChain depends on two Rust-built native libraries that are loaded at runtime.

### libqorepqc (Post-Quantum Cryptography)

The PQC library provides ML-DSA-87 (Dilithium-5) key generation, signing, and verification through a C-compatible FFI interface.

```bash
cd rust/qorepqc
cargo build --release
```

The compiled library is placed in `lib/{os}_{arch}/`:

| Platform | Library File | Directory |
|----------|-------------|-----------|
| macOS arm64 | `libqorepqc.dylib` | `lib/darwin_arm64/` |
| Linux amd64 | `libqorepqc.so` | `lib/linux_amd64/` |
| Linux arm64 | `libqorepqc.so` | `lib/linux_arm64/` |

### libqoresvm (SVM Runtime)

The SVM library provides the BPF program execution environment for the x/svm module.

```bash
cd rust/qoresvm
cargo build --release
```

Output follows the same `lib/{os}_{arch}/` convention as above (`libqoresvm.dylib` on macOS, `libqoresvm.so` on Linux).

### Setting the Library Path

The native libraries must be discoverable at runtime. Set the appropriate environment variable for your platform:

**macOS:**

```bash
export DYLD_LIBRARY_PATH=$(pwd)/lib/darwin_arm64:$DYLD_LIBRARY_PATH
```

**Linux:**

```bash
export LD_LIBRARY_PATH=$(pwd)/lib/linux_amd64:$LD_LIBRARY_PATH
```

> **Tip:** Add the export to your shell profile (`~/.bashrc`, `~/.zshrc`) so it persists across sessions.

---

## Open-Core Architecture

QoreChain follows an **open-core** model:

- **Community build** -- Contains the full module interfaces, CLI commands, protobuf definitions, and message types for every QoreChain module (x/pqc, x/ai, x/reputation, x/qca, x/svm, x/crossvm, etc.). Keepers for internal modules use **stub implementations** that return safe defaults or no-op responses. This allows third-party tooling, wallets, and indexers to integrate with all QoreChain APIs without requiring internal code.

- **Full (internal) build** -- Enables the complete keeper implementations behind the `internal` build tag. This includes the real AI anomaly detection logic, reinforcement-learning consensus parameter tuning, advanced reputation scoring, and all production-grade features.

Both builds produce the same `qorechaind` binary name and expose identical CLI commands and gRPC/REST endpoints. The difference is in the runtime behavior of the keeper logic behind those interfaces.

---

## Community Build

```bash
CGO_ENABLED=1 go build -o qorechaind ./cmd/qorechaind/
```

This compiles all public module interfaces with stub keepers for internal features. The resulting binary is suitable for:

- Submitting and querying transactions
- Interacting with EVM, CosmWasm, and SVM VMs
- Building third-party integrations and tooling
- Local development and testing

> ⚠️ **Do not validate with the community build.** The stub keepers make it a
> *different state machine* from the full build — the same transaction can yield
> a different app-hash, so a community-build node will fork or halt the moment a
> feature-active network processes a licensed-feature transaction. Validators,
> and any node that must follow consensus on a feature-active network, MUST run
> the **full** binary (below). The community build is for read/integration use
> (exchanges, indexers, wallets).

---

## Full Build

```bash
# from the private extensions repo working tree:
bash generate-overlay.sh                       # writes overlay.json
cd qorechain-core
CGO_ENABLED=1 go build -tags full \
  -overlay=../<extensions>/overlay.json \
  -o qorechaind ./cmd/qorechaind/
```

The `-tags full` build, applied over the private extensions overlay, activates
the real keeper implementations (PQC FFI, SVM executor, on-chain license
enforcement). It links `libqorepqc` and `libqoresvm`; build `libqorepqc` from
the PQC Rust crate (`cargo build --release`) and place it under
`lib/<os_arch>/`. This build requires access to the private extensions
repository. Every validator on a licensed network runs this binary.

---

## Configuring `app.toml` (EVM chain-id)

A fresh `qorechaind init` writes the EVM (EIP-155) chain ID into `app.toml`
`[evm] evm-chain-id`. It must match the network's chain ID — **9800** for the
`qorechain-diana` testnet, **9801** for `qorechain-vladi` mainnet — or the
JSON-RPC backend rejects every `eth_sendRawTransaction` with `incorrect
chain-id; expected 262144` (the cosmos/evm default). v3.1.69+ defaults this
correctly; on older nodes or hand-edited configs, set it explicitly:

```toml
[evm]
evm-chain-id = 9800
```

---

## Running Tests

```bash
CGO_ENABLED=1 go test ./... -count=1
```

The `-count=1` flag disables test caching, ensuring a clean run every time. Individual package tests can be run with:

```bash
CGO_ENABLED=1 go test ./x/pqc/... -count=1 -v
CGO_ENABLED=1 go test ./x/ai/... -count=1 -v
CGO_ENABLED=1 go test ./x/svm/... -count=1 -v
```

Run the Rust library tests separately:

```bash
cd rust/qorepqc && cargo test
cd rust/qoresvm && cargo test
```

---

## Build Verification

After a successful build, verify the binary:

```bash
./qorechaind version
./qorechaind init test-node --chain-id qorechain-diana
```

The `init` command should create a genesis file and node configuration in `~/.qorechaind/` without errors.

---

## Docker Build

For containerized builds, a Dockerfile is provided at the repository root:

```bash
docker build -t qorechaind:latest .
```

The Docker image handles all native library compilation and path configuration automatically. See the [Quickstart](../getting-started/quickstart.md) guide for running a node with Docker Compose.

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `cgo: C compiler not found` | Install Xcode CLI tools (macOS) or `build-essential` (Linux) |
| `cannot find -lqorepqc` | Build the Rust libraries first and set `LD_LIBRARY_PATH` / `DYLD_LIBRARY_PATH` |
| `undefined: sonic.*` | Ensure `go.sum` is up to date: `go mod tidy` |
| `signal: killed` during build | Increase available memory (common in Docker with low limits) |
| PQC tests fail with size mismatch | Verify you are using `pqcrypto v0.5.0+` (ML-DSA-87: pubkey=2592, privkey=4896, sig=4627 bytes) |
