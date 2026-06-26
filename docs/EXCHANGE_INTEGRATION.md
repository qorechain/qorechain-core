# QoreChain Exchange / Integrator Runbook — Two Withdrawal Rails

> For exchanges, custodians, and payment processors integrating QoreChain (QOR).
> QoreChain supports **two fully-supported integration rails**, and you can pick
> either one:
>
> - **Rail A — EVM (no PQC):** treat QOR as a standard EVM asset. Easiest if you
>   already run Ethereum/EVM integration tooling. EVM txs are PQC-exempt.
> - **Rail B — Cosmos-native (PQC-secured):** quantum-secure withdrawals using the
>   chain's FIPS-204 ML-DSA-87 hybrid signature. **Recommended for the highest
>   security posture.**
>
> Both rails move the *same* QOR token over the *same* shared chain state — they
> are just two lanes onto it. Deposits and balances reconcile across both.

---

## 0. Token & network facts (both rails)

| Fact | Value |
|------|-------|
| Cosmos chain-id | `qorechain-diana` (testnet) / `qorechain-vladi` (mainnet) |
| EVM chain-id (EIP-155) | `9800` testnet (`0x2648`) / `9801` mainnet (`0x2649`) |
| Base denom (Cosmos) | `uqor` — **6 decimals** (1 QOR = 1,000,000 uqor) |
| EVM native currency | QOR — **18 decimals** (the EVM lane scales `uqor` by 1e12) |
| Bech32 prefix | `qor` (accounts `qor1…`, validators `qorvaloper…`) |
| Native RPC / gRPC / REST | `26657` / `9090` / `1317` |
| EVM JSON-RPC / WS | `8545` / `8546` |

> ⚠️ **Decimals trap:** a balance is `uqor`×10⁶ on the Cosmos rail and ×10¹⁸ on
> the EVM rail. `eth_getBalance` returns `(uqor balance) × 10¹²`. Never mix the
> two scales in your accounting.

---

## Rail A — EVM (no PQC)

Treat QOR as an ordinary EVM coin. EVM transactions route through
`ExtensionOptionsEthereumTx` and **bypass the PQC ante chain entirely** — so you
sign and broadcast them exactly like Ethereum, with no QoreChain-specific crypto.

### A.1 Addresses & the funding gotcha

A single secp256k1 key yields **two different addresses**:
- Cosmos/native: bech32 `qor1…`
- EVM: hex `0x…` (keccak-derived — a *different account* from the bech32 one).

**You fund an EVM `0x…` account by bank-sending to its bech32 form**, because the
EVM and Cosmos views share one account store. Convert between them:

```bash
qorechaind debug addr 0xD6A292ECF6FF6DE9FDC6BC9C6DA2C6D81B52FB8D
#   -> prints the matching qor1… bech32 for that hex account
qorechaind debug addr qor1...           # also works in reverse
```

So to fund an exchange EVM hot wallet `0xHOT`:
1. `qorechaind debug addr 0xHOT` → `qor1hot…`
2. bank-send QOR to `qor1hot…` (any funded account; if that sender is a normal
   Cosmos account it must itself be PQC-hybrid-signed — see Rail B).
3. The balance is now visible via `eth_getBalance(0xHOT)` (×10¹² scaled).

### A.2 Withdrawals — standard `eth_sendRawTransaction`

Sign an EIP-1559 (or legacy) tx with chainId **9800**, then:

```bash
curl -s -X POST -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":["0x<signed-rlp>"],"id":1}' \
  http://127.0.0.1:8545
```

With ethers.js v6:

```javascript
import { JsonRpcProvider, Wallet, parseEther } from "ethers";
const provider = new JsonRpcProvider("http://127.0.0.1:8545", { name: "qorechain", chainId: 9800 });
const hot = new Wallet(process.env.HOT_PRIVKEY, provider);   // 0x… key
const tx = await hot.sendTransaction({ to: "0xRecipient", value: parseEther("1.0") }); // 1 QOR (18-dec)
await tx.wait();
```

Gas/fees behave like Ethereum (EIP-1559). The native gas token is QOR.

### A.3 Deposit detection

Poll new blocks and scan transactions to your deposit addresses:

```javascript
const latest = await provider.getBlockNumber();
const block = await provider.send("eth_getBlockByNumber", ["0x" + latest.toString(16), true]);
for (const tx of block.transactions) {
  if (depositAddrs.has(tx.to?.toLowerCase())) credit(tx.from, tx.value); // value is 18-dec wei
}
```

For ERC-20 token-pair deposits, subscribe to logs:

```javascript
const logs = await provider.send("eth_getLogs", [{ fromBlock, toBlock, address: tokenAddr,
  topics: [TRANSFER_TOPIC, null, padTo32(depositAddr)] }]);
```

Use confirmations = a few blocks (QoreChain has instant finality at the consensus
layer; a 1–2 block buffer guards against RPC lag).

### A.4 Rail A pros / cons

| | |
|---|---|
| ✅ Reuse existing Ethereum integration (ethers/web3/viem, MetaMask, hardware signers). |
| ✅ No QoreChain-specific cryptography; PQC-exempt by design. |
| ✅ `eth_*` deposit detection identical to any EVM chain. |
| ⚠️ The funding gotcha: fund `0x…` accounts via their bech32 form. |
| ⚠️ **Not quantum-secure** — secp256k1/ECDSA only. Acceptable per your risk model, but not the hardened option. |
| ⚠️ Watch the 18-dec vs 6-dec scaling in accounting. |

---

## Rail B — Cosmos-native (PQC-secured) — **recommended**

The native rail enforces QoreChain's defining security feature: every Cosmos tx
carries a FIPS-204 **ML-DSA-87** hybrid signature (in a tx-body extension) in
addition to the classical secp256k1 signature. This makes withdrawals
**quantum-secure**. It is the recommended rail for a custodial hot wallet.

### B.1 One-time bootstrap of the hot wallet

The PQC-key **registration tx is classical-exempt**, so you can bootstrap a fresh
account before it has a PQC key. Do this once per hot wallet:

```bash
# 1. Generate + store an ML-DSA-87 (Dilithium-5) key; prints its pubkey hex.
qorechaind tx pqc gen-key hotwallet --from hotwallet
#   stored Dilithium-5 private key: ~/.qorechaind/pqc/hotwallet.dilithium
#   public_key_hex: <PUB_HEX>

# 2. Register that pubkey on-chain as a HYBRID key (this tx is classical-exempt).
qorechaind tx pqc register-key <PUB_HEX> hybrid --from hotwallet \
  --chain-id qorechain-diana --yes
```

`register-key` takes `[pubkey-hex] [key-type]` where key-type is one of
`hybrid` (recommended), `pqc_only`, or `classical_only`. After this, every
further tx from `hotwallet` **must** be hybrid-signed.

### B.2 Withdrawals — generate-only + cosign

Every withdrawal is two steps: build an unsigned tx, then hybrid-sign + broadcast
it with `tx pqc cosign`. **These are the real, verified CLI flags:**

```bash
# 1. Build the unsigned bank transfer (1 QOR = 1000000 uqor).
qorechaind tx bank send <from> <to> 1000000uqor \
  --generate-only > tx.json

# 2. PQC + classical co-sign and broadcast.
qorechaind tx pqc cosign tx.json \
  --from <from> \
  --pqc-key hotwallet \
  --chain-id qorechain-diana \
  --yes
```

- `cosign` takes the unsigned-tx file as its **single positional arg**.
- `--pqc-key <name>` (required) names the key created by `gen-key`
  (stored at `<home>/pqc/<name>.dilithium`).
- `--from`, `--chain-id`, and the rest are standard Cosmos tx flags.

Under the hood `cosign` re-derives `B0 = TxBody{messages, memo, timeoutHeight}`
(no extension), ML-DSA-87-signs `BE32(len B0)‖B0‖BE32(len authInfo)‖authInfo`,
bakes the `PQCHybridSignature` extension into the body, then adds the classical
secp256k1 signature over the final `SignDoc` and broadcasts the `TxRaw`.

### B.3 Programmatic / headless signing (no CLI)

For a withdrawal worker, use the published packages instead of the CLI. A
complete, runnable example lives at
[`../examples/server-signer/`](../examples/server-signer/): it builds a
`bank.MsgSend`, layers the ML-DSA-87 hybrid sig via
`@qorechain/wallet-adapter`'s `QoreChainSigner`, and broadcasts over RPC — no
browser, no Keplr. The JS hybrid framing is byte-identical to `tx pqc cosign`.

### B.4 Deposit detection

Use the native RPC `tx_search` or REST, filtering on transfer events to your
deposit addresses:

```bash
# All transfers crediting a deposit address (paged).
curl -s "http://127.0.0.1:26657/tx_search?query=\"transfer.recipient='qor1deposit...'\"&page=1&per_page=50" \
  | jq '.result.txs[].hash'

# Or REST/LCD for a balance snapshot.
curl -s http://127.0.0.1:1317/cosmos/bank/v1beta1/balances/qor1deposit...
```

For a live feed, subscribe to the native WebSocket (`ws://host:26657/websocket`,
`tm.event='Tx'`) or run the repo's block indexer
(`qorechain-core/indexer/`) and read its Postgres `transactions`/`events` tables.

### B.5 Rail B pros / cons

| | |
|---|---|
| ✅ **Quantum-secure** — FIPS-204 ML-DSA-87 hybrid signature on every withdrawal. |
| ✅ Native Cosmos semantics: precise `uqor` (6-dec) amounts, event-based deposit detection, no decimal scaling. |
| ✅ Bootstrap is one-time and classical-exempt; CLI + npm SDK both supported. |
| ⚠️ Requires PQC key management (generate, register, protect the ML-DSA-87 secret). |
| ⚠️ Slightly larger txs (ML-DSA-87 signature ≈ 4.6 KB) and a two-step sign flow. |

---

## Decision table — which rail?

| Dimension | Rail A — EVM (no PQC) | Rail B — Cosmos-native (PQC) |
|-----------|-----------------------|------------------------------|
| Security posture | secp256k1/ECDSA only — **not** quantum-resistant | **Quantum-secure** (ML-DSA-87 hybrid) — recommended |
| Signing | Standard EVM signer (ethers/web3/HSM) | `gen-key`+`register-key` once, then `cosign` per tx (or npm SDK) |
| Deposit detection | `eth_getBlockByNumber` / `eth_getLogs` | `tx_search` / REST / indexer events |
| Decimals | 18 (EVM wei view) | 6 (`uqor`) |
| Ops complexity | Low — reuse Ethereum tooling | Moderate — manage one extra PQC key per hot wallet |
| Latency / finality | Instant finality; 1–2 block confirm buffer | Instant finality; same |
| Funding gotcha | Fund `0x…` via its `qor1…` bech32 (`qorechaind debug addr`) | None — native addresses throughout |
| Best for | Fast onboarding with existing EVM stack | Custodial wallets wanting the hardened, future-proof path |

**Recommendation:** integrate **Rail B (PQC)** for the security guarantee that
distinguishes QoreChain — it is the recommended secure option. **Rail A (EVM) is
fully supported** and is the fastest path if you already operate EVM
infrastructure; both rails settle the same QOR on the same chain.

---

## See also

- [`../examples/server-signer/`](../examples/server-signer/) — headless Node.js
  PQC signer (Rail B), runnable example with README.
- [`PQC_INTEGRATION.md`](./PQC_INTEGRATION.md) — the hybrid-signature ante chain
  internals, plus the per-language client coordinate table.
- [`SDK.md`](./SDK.md) — node endpoint/port reference (RPC, gRPC, REST, EVM RPC/WS)
  for wiring deposit watchers and dashboards.
- [`EVM.md`](./EVM.md) — EVM lane details and precompiles.
- `@qorechain/wallet-adapter` (npm) — `QoreChainSigner`, `qoreChainInfo()`,
  `qoreEvmChainParams()`, `addQoreEvmToWallet()`.
