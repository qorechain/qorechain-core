# Client Libraries

Connect to a QoreChain node and transact on each of its surfaces. All examples
assume a local node; replace hosts/ports for a remote endpoint.

## Cosmos — CosmJS

```ts
import { SigningStargateClient } from "@cosmjs/stargate";

const client = await SigningStargateClient.connectWithSigner(
  "http://localhost:26657", wallet);              // secp256k1, qor-prefixed
await client.sendTokens(from, to,
  [{ denom: "uqor", amount: "1000000" }],
  { amount: [{ denom: "uqor", amount: "25000" }], gas: "200000" });
```

Custom-module messages are built by type URL, e.g.
`/qorechain.amm.v1.MsgCreatePool` or `/qorechain.license.v1.MsgGrantLicense`.
Generate typed clients from [`proto/qorechain/`](https://github.com/qorechain/qorechain-core/tree/main/proto/qorechain)
with `buf generate`.

## EVM — ethers.js / viem / web3.js

```ts
import { JsonRpcProvider, Wallet, ContractFactory } from "ethers";
const provider = new JsonRpcProvider("http://localhost:8545"); // chainId 9800
const signer   = new Wallet(PRIVATE_KEY, provider);
const contract = await new ContractFactory(abi, bytecode, signer).deploy();
await contract.waitForDeployment();
```

MetaMask / Hardhat / Foundry / Remix work unmodified — set the network to
`:8545` with **chain-id 9800**. If `eth_sendRawTransaction` is rejected with
`incorrect chain-id; expected 262144`, the node's `app.toml` `[evm]
evm-chain-id` was left at the cosmos/evm default; it must be `9800` (testnet).

## SVM — @solana/web3.js

```ts
import { Connection, PublicKey } from "@solana/web3.js";
const conn = new Connection("http://localhost:8899");
await conn.getBalance(new PublicKey(addr));   // getAccountInfo / getSlot / sendTransaction
```

Accounts follow Solana **rent-exemption** rules (a data account must hold at
least `(space + overhead) × lamports_per_byte × rent_exemption_multi` lamports).
Programs are BPF/SBF ELFs deployed via `qorechaind tx svm deploy-program <elf>`.

## CosmWasm

```ts
import { SigningCosmWasmClient } from "@cosmjs/cosmwasm-stargate";
const c = await SigningCosmWasmClient.connectWithSigner("http://localhost:26657", wallet);
const { codeId }     = await c.upload(sender, wasmBytes, "auto");
const { contractAddress } = await c.instantiate(sender, codeId, initMsg, "label", "auto");
await c.execute(sender, contractAddress, execMsg, "auto");
await c.queryContractSmart(contractAddress, queryMsg);
```

## CLI as an SDK

The `qorechaind` binary is itself a complete client. Every native and custom
module exposes `tx` and `query` subcommands:

```bash
qorechaind query bank balances <qor1…>
qorechaind query <module> params
qorechaind tx <module> <action> … --from <key> --chain-id qorechain-diana \
  --gas <g> --fees <f>uqor -y
```
