# Module Development

QoreChain ships as an **open core**. Understanding the build model is the first
prerequisite for developing modules or operating a node.

## Community build vs full build

| | Community build | Full build |
|---|---|---|
| Source | public `qorechain-core` | public + private extensions overlay |
| Command | `go build ./cmd/qorechaind` | `CGO_ENABLED=1 go build -tags full -overlay=<overlay.json> ./cmd/qorechaind` |
| Licensed modules (`pqc`, `svm`, `license`, …) | **stub keepers** | **real keepers** + Rust FFI |
| Native libs | none required | `libqorepqc`, `libqoresvm` |
| License enforcement | permissive (stubs return "allowed") | enforced on-chain |
| Use case | sync / query / submit (exchanges, integrators) | **validators**, licensed operators |

> ⚠️ **Consensus homogeneity.** The community and full builds are *different
> state machines* — the same transaction can produce a different app-hash. Every
> validator, and every node that must stay in consensus with a feature-active
> network, MUST run the **full** binary. The community build is for
> read/integration use only.

## Anatomy of a custom module

The 14 proto-bound custom modules follow the standard Cosmos SDK module layout
plus the proto-gen pipeline:

1. **Proto** — define `proto/qorechain/<mod>/v1/{tx,query}.proto` (messages,
   `Msg` service, `Query` service). Use `gogoproto` options
   (`casttype`/`customtype`) to keep Go field names and typed amounts.
2. **Generate** — `cd proto && buf generate`, then copy the generated
   `*.pb.go` into `x/<mod>/types/`.
3. **Keeper** — implement state access over the module store key.
4. **Msg / Query servers** — implement the generated `MsgServer` /
   `QueryServer` interfaces; wrap keeper methods.
5. **Wire** — `RegisterServices` in `module.go`; register the module in
   `app/app.go` (and `root.go`'s basic-manager for non-depinject modules so
   `init` sees its genesis).
6. **CLI** — real `tx`/`query` subcommands under `x/<mod>/client/cli`.

For interface-keeper modules (e.g. `amm`, `license`) the query server lives in
the public `x/<mod>`; for full-build-only modules (e.g. `rdk`, `bridge`,
`multilayer`, `abstractaccount`, `ai`) it lives in the private keeper as
`grpc_query.go`.

## State & genesis notes

- Persist anything that must survive a restart in the **module KV store**, not in
  keeper struct fields. `InitGenesis` runs only at height 0 — a value set there
  but not stored is lost on restart. (This is why the `x/license` grant authority
  is now store-backed.)
- Bech32 prefixes (`qor`) must be set before depinject runs.
- All consensus math must be deterministic — no floating point in any consensus
  path (fixed-point integer arithmetic throughout).

## Build the full binary

```bash
# from the private extensions repo working tree:
bash generate-overlay.sh                      # → overlay.json
cd qorechain-core
CGO_ENABLED=1 go build -tags full \
  -overlay=../<extensions>/overlay.json \
  -o qorechaind ./cmd/qorechaind
```

`libqorepqc` is built from the PQC Rust crate (`cargo build --release`) and
placed in `lib/<os_arch>/`; `libqoresvm` is committed. See
[Building from Source](../developer-guide/building-from-source.md).
