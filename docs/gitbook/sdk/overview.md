# SDK Overview

QoreChain is a **Cosmos SDK v0.53** application chain with three embedded virtual
machines, so it speaks several SDK dialects at once. There is no proprietary
client SDK to install — existing ecosystem tooling connects directly to the
standard interfaces.

| Surface | SDK / tooling | Endpoint |
|---------|---------------|----------|
| Cosmos (bank, staking, gov + 21 custom modules) | CosmJS, cosmpy, gRPC/Protobuf | gRPC `:9090`, REST `:1317`, RPC `:26657` |
| EVM | ethers.js, viem, web3.js, Hardhat, Foundry, MetaMask | JSON-RPC `:8545` / WS `:8546` |
| SVM | @solana/web3.js, Anchor | JSON-RPC `:8899` |
| CosmWasm | @cosmjs/cosmwasm-stargate, cosmwasm-std (Rust) | gRPC / REST |

## Network identity

- **Cosmos chain-id:** `qorechain-diana` (testnet) / `qorechain-vladi` (mainnet)
- **EVM chain-id (EIP-155):** **9800** testnet / **9801** mainnet — clients MUST
  sign EVM transactions with this id (it is set in `app.toml` `[evm]
  evm-chain-id`)
- **Base denom:** `uqor` (1 QOR = 10^6 uqor); **bech32 prefix:** `qor`

## Two addresses, one key

A secp256k1 key produces a bech32 `qor1…` Cosmos address *and* a hex `0x…` EVM
address — these are **different accounts**. Convert and fund with:

```bash
qorechaind debug addr <hex-no-0x | bech32>     # convert between formats
qorechaind keys export <name> --unarmored-hex --unsafe   # raw key for EVM tooling
```

To fund an EVM account, bank-send `uqor` to its bech32 form.

## Where to go next

- [Client Libraries](client-libraries.md) — connect and transact on each surface
- [Module Development](module-development.md) — build custom modules; the
  community-vs-full build model
- Full reference: [`docs/SDK.md`](https://github.com/qorechain/qorechain-core/blob/main/docs/SDK.md)
