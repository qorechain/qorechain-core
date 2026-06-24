# QoreChain SDK

> How to build **on** QoreChain (client apps and integrations) and **into** it
> (custom modules). Current chain version: **v3.1.70**.

QoreChain is a **Cosmos SDK v0.53** application chain with three embedded virtual
machines (EVM, CosmWasm, SVM). It therefore speaks several SDK dialects at once:
the Cosmos Protobuf/gRPC interfaces for native modules, the Ethereum JSON-RPC for
EVM, and the Solana JSON-RPC for SVM. There is no proprietary client SDK to
install — existing ecosystem tooling connects directly.

---

## 1. Foundation

| Layer | Technology |
|-------|-----------|
| Application framework | Cosmos SDK v0.53.6 (depinject/appconfig) |
| Consensus engine | CometBFT (BFT) |
| Native messages | Protobuf (`gogoproto`), served over gRPC + REST |
| EVM | cosmos/evm — Ethereum JSON-RPC, EIP-1559 |
| SVM | `qoresvm` BPF executor — Solana-compatible JSON-RPC |
| CosmWasm | wasmd / wasmvm |
| Signatures | secp256k1 (classic) + hybrid Dilithium-5 (PQC) |

**Address formats.** A single secp256k1 key yields two *different* addresses:
- Cosmos / native: bech32 `qor1…` (validators `qorvaloper…`).
- EVM: hex `0x…` (keccak-derived — a **different account** from the bech32 one).
Convert with `qorechaind debug addr <hex-or-bech32>`. Fund an EVM account by
bank-sending to its bech32 form.

---

## 2. Endpoints & network identity

| Interface | Default port | Used by |
|-----------|--------------|---------|
| CometBFT RPC | `26657` | CosmJS, block explorers, `qorechaind` |
| gRPC | `9090` | CosmJS gRPC, backend services |
| REST / LCD | `1317` | wallets, browser dashboards |
| EVM JSON-RPC | `8545` (WS `8546`) | ethers/viem/web3.js, MetaMask |
| SVM JSON-RPC | `8899` | @solana/web3.js |

- **Cosmos chain-id:** `qorechain-diana` (testnet) / `qorechain-vladi` (mainnet)
- **EVM chain-id (EIP-155):** `9800` (testnet) / `9801` (mainnet) — set in
  `app.toml` `[evm] evm-chain-id`; clients must sign EVM txs with this id.
- **Base denom:** `uqor` (1 QOR = 10^6 uqor). **Bech32 prefix:** `qor`.

---

## 3. Building ON QoreChain (client SDKs)

### 3.1 Cosmos — CosmJS

```ts
import { SigningStargateClient } from "@cosmjs/stargate";

const client = await SigningStargateClient.connectWithSigner(
  "http://localhost:26657", wallet); // wallet over secp256k1, qor-prefixed
await client.sendTokens(from, to, [{ denom: "uqor", amount: "1000000" }],
  { amount: [{ denom: "uqor", amount: "25000" }], gas: "200000" });
```

The native modules (`bank`, `staking`, `gov`, …) and the custom modules all
register Protobuf types; generate typed clients from the `.proto` files in
[`proto/`](../proto) with `buf generate`, or use `@cosmjs/stargate` generic
message construction with the type URLs (e.g. `/qorechain.amm.v1.MsgCreatePool`).

### 3.2 EVM — ethers / viem / web3.js

```ts
import { JsonRpcProvider, Wallet, ContractFactory } from "ethers";
const provider = new JsonRpcProvider("http://localhost:8545"); // chainId 9800
const signer = new Wallet(PRIVATE_KEY, provider);
const c = await new ContractFactory(abi, bytecode, signer).deploy();
```

Standard Solidity tooling (Hardhat, Foundry, Remix, MetaMask) works unmodified —
just point the network at `:8545` with **chain-id 9800**.

### 3.3 SVM — @solana/web3.js

```ts
import { Connection } from "@solana/web3.js";
const conn = new Connection("http://localhost:8899");
await conn.getBalance(pubkey); // getAccountInfo / getSlot / sendTransaction, …
```

Programs are BPF/SBF ELFs; accounts follow Solana rent-exemption rules. Deploy
via `qorechaind tx svm deploy-program <elf>` or the Solana-compatible RPC.

### 3.4 CosmWasm

`@cosmjs/cosmwasm-stargate` for clients; `cosmwasm-std` (Rust) for contracts.
Full lifecycle: `store → instantiate → execute → query → migrate`.

---

## 4. The custom-module Protobuf surface

As of v3.1.62–68, all **14 proto-bound custom modules** expose real
proto-generated `Msg` (transactions) and `Query` services with CLI subcommands
under `qorechaind tx <module>` / `qorechaind query <module>`:

`pqc`, `ai`, `amm`, `bridge`, `crossvm`, `license`, `lightnode`, `multilayer`,
`qca`, `rdk`, `reputation`, `rlconsensus`, `svm`, `abstractaccount`.

Type URLs follow `/qorechain.<module>.v1.Msg…` / `…Query…`. Definitions live in
[`proto/qorechain/<module>/v1/`](../proto/qorechain). Regenerate clients with
`cd proto && buf generate`.

Discover the live surface against any node:

```bash
qorechaind tx <module> --help        # transaction subcommands
qorechaind query <module> --help     # query subcommands
qorechaind query <module> params     # most modules expose params
```

---

## 5. Building INTO QoreChain (custom modules & the build model)

QoreChain ships as an **open core**:

- **Community build** (`go build ./cmd/qorechaind`, no build tags): the public
  `qorechain-core` repo. Licensed modules are compiled in as **stub keepers**
  (e.g. `x/pqc`, `x/svm`, `x/license` stubs). Use it to **sync, query, and
  submit transactions** — exchanges and integrators need nothing more.
- **Full build** (`-tags full` + an overlay of the private extensions repo +
  the `libqorepqc` / `libqoresvm` Rust libraries): the real keepers, PQC FFI,
  SVM executor, and on-chain license enforcement.

> ⚠️ **Consensus homogeneity.** The community and full builds are *different
> state machines*. Every validator — and every node that must stay in consensus
> with a feature-active network — MUST run the **full** binary. The community
> build is for read/integration use only. See
> [Building from Source](gitbook/developer-guide/building-from-source.md).

To add a module: write its `proto/qorechain/<mod>/v1/{tx,query}.proto`, run
`buf generate`, implement the keeper + `Msg`/`Query` servers, register it in
`app/app.go` (and `root.go` for non-depinject modules), and wire genesis. The
proto-gen pipeline used for the 14 modules is the reference pattern.

---

## 6. Becoming a validator or light node

Both roles are gated **on-chain** by the `x/license` registry plus a stake floor,
and both require the **full binary**:

| Role | License (granted by the authority) | Stake floor | Register with |
|------|-----------------------------------|-------------|---------------|
| **Validator** | `validator_operator` | ≥ 100,000 QOR self-bond | `qorechaind tx staking create-validator` |
| **Light node** | `lightnode_operator` | ≥ 1,000 QOR delegated | `qorechaind tx lightnode register <sx\|ux> <version>` |

A license is two things: an **off-chain entitlement** (the dashboard purchase)
and an **on-chain grant** — the authority signs `qorechaind tx license grant
<addr> <feature>`. The dashboard backend (holding the authority key) or a
governance proposal turns the purchase into that grant. Verify with
`qorechaind query license check <addr> <feature>`.

See [Running a Validator](gitbook/developer-guide/running-a-validator.md).

---

## 7. Reference

- Protobuf definitions: [`proto/qorechain/`](../proto/qorechain)
- CLI reference: [Transaction Commands](gitbook/cli-reference/transaction-commands.md),
  [Query Commands](gitbook/cli-reference/query-commands.md)
- API reference: [REST/gRPC](gitbook/api-reference/rest-grpc-endpoints.md),
  [EVM JSON-RPC](gitbook/api-reference/json-rpc-evm.md),
  [SVM JSON-RPC](gitbook/api-reference/json-rpc-svm.md),
  [qor\_ namespace](gitbook/api-reference/json-rpc-qor.md)
- Chain parameters: [Chain Parameters](gitbook/appendix/chain-parameters.md)
