# QoreChain wallet connectors

Drop-in configuration for connecting any wallet, explorer, or dApp to QoreChain —
both networks: **testnet** `qorechain-diana` (EVM 9800) and **mainnet**
`qorechain-vladi` (EVM 9801).

```
connectors/
├── cosmos-chain-registry/        # standard cosmos/chain-registry format (Keplr, Leap, explorers)
│   ├── qorechain-diana/{chain.json, assetlist.json}
│   └── qorechain-vladi/{chain.json, assetlist.json}
└── evm/                          # EIP-3085 wallet_addEthereumChain params (MetaMask & EIP-1193)
    ├── qorechain-diana.json
    └── qorechain-vladi.json
```

## Cosmos wallets (Keplr / Leap / Cosmostation …)

QoreChain requires a FIPS-204 **ML-DSA-87 hybrid signature** on every Cosmos tx,
so a wallet must add the PQC extension before signing. Use the published adapter —
no wallet fork needed:

```js
import { qoreChainInfo, QoreChainSigner, derivePqcKeyFromWallet } from '@qorechain/wallet-adapter';
await window.keplr.experimentalSuggestChain(
  qoreChainInfo({ chainId: 'qorechain-diana', rpc: 'https://rpc.qorechain.xyz', rest: 'https://rest.qorechain.xyz' })
);
// sign+broadcast hybrid txs with QoreChainSigner (see @qorechain/wallet-adapter README)
```

Or use `@qorechain/connect` (`QoreConnect.send(wallet, …)`) which auto-detects
EVM / Cosmos / SVM wallets and layers PQC only where required.

`chain.json` / `assetlist.json` are in the canonical
[cosmos/chain-registry](https://github.com/cosmos/chain-registry) format — submit
them upstream for automatic discovery by Keplr's chain store and major explorers.

## EVM wallets (MetaMask & any EIP-1193)

The EVM lane is classical (secp256k1) — standard Ethereum tooling works as-is.
Native currency is the **18-decimal** `aqor` view of QOR.

```js
import { addQoreEvmToWallet } from '@qorechain/wallet-adapter';
await addQoreEvmToWallet(window.ethereum, { evmChainId: 9800, rpcUrl: 'https://evm.qorechain.xyz', explorerUrl: 'https://explorer.qorechain.xyz' });
// or feed connectors/evm/qorechain-diana.json directly to wallet_addEthereumChain
```

## Reading on-chain state (REST/LCD :1317)

Explorers/dashboards read state over HTTP, e.g.:
- License: `GET {rest}/qorechain/license/v1/check/{grantee}/{feature_id}` → `{active}`
- License list: `GET {rest}/qorechain/license/v1/list/{grantee}`
- PQC key: `GET {rest}/qorechain/pqc/v1/account/{address}`
- Light node: `GET {rest}/qorechain/lightnode/v1/node/{address}`
- Bridge / multilayer / burn / amm: see each module's `/qorechain/<mod>/v1/...` routes.

## Fees

`min_gas_price = 0.1 uqor/gas` (gas-proportional, not a flat floor). Wallet
gas-price step low/avg/high = 0.1 / 0.15 / 0.25 uqor. A ~200k-gas transfer costs
~20,000 uqor ≈ 0.02 QOR.

> Endpoint hostnames (`rpc/rest/grpc/evm/explorer.*`) are the intended production
> URLs; confirm against the actual edge config at deploy.
