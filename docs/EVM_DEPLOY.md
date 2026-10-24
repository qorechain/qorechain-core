# Deploying Solidity contracts with Hardhat and Foundry

Contract creation on QoreChain's EVM is open to everyone: no allow-list, no
governance step. What is different from Ethereum is that every account must hold
a post-quantum (ML-DSA-87) key, and the EVM lane only admits an account's
transactions while it has an open **EVM authorisation window**. An Ethereum
transaction cannot carry a post-quantum signature, so the authorisation is given
ahead of time by a Cosmos-lane message that does. Inside the window Hardhat,
Foundry, ethers and MetaMask work unmodified.

The steps, once per account and then once per session:

1. Use one secp256k1 key on both lanes (your Hardhat or Foundry private key).
2. Register a post-quantum key for that account (once).
3. Open a window (each deploy session).
4. Deploy and call contracts as usual.

## Networks

| | Testnet (diana) | Mainnet (vladi) |
|---|---|---|
| EVM JSON-RPC | `https://evm-testnet.qore.host` | `https://evm.qore.host` |
| EVM chain id | `9800` (`0x2648`) | `9801` (`0x2649`) |
| Cosmos chain id | `qorechain-diana` | `qorechain-vladi` |
| RPC (for `qorechaind --node`) | `https://rpc-testnet.qore.host:443` | `https://rpc.qore.host:443` |
| REST | `https://api-testnet.qore.host` | `https://api.qore.host` |

The native currency on the EVM side is QOR with 18 decimals. On the Cosmos side
the same balance is `uqor` with 6 decimals (1 QOR = 1,000,000 uqor). Fees today:
base fee 100 gwei (0.1 uqor per gas) on both networks.

You need the `qorechaind` release binary (the validator build; `gen-key` and
`cosign` are not in a community build). The examples below use the testnet; for
mainnet change the chain id and the endpoints.

```bash
export QNODE=https://rpc-testnet.qore.host:443
export QCHAIN=qorechain-diana
```

## 1. One key on both lanes

Import the private key your Hardhat or Foundry config uses as an `eth_secp256k1`
key. With that key type the account's `qor1...` address and its `0x...` address
are the same account, with one balance.

```bash
qorechaind keys import-hex deployer <private-key-hex-without-0x> --key-type eth_secp256k1
qorechaind keys show deployer -a            # qor1... address
qorechaind debug addr <0x-address-without-0x>  # the same account, both forms
```

Fund the account (testnet faucet, or a transfer to either form of the address).

## 2. Register a post-quantum key (once per account)

```bash
qorechaind tx pqc gen-key deployer-mldsa
# prints public_key_hex and stores the private key under ~/.qorechaind/pqc/deployer-mldsa.dilithium

qorechaind tx pqc register-key-v2 dilithium5 <public_key_hex> hybrid \
  --pqc-key deployer-mldsa \
  --from deployer --chain-id $QCHAIN --node $QNODE \
  --gas auto --gas-adjustment 1.3 --gas-prices 0.2uqor
```

A first registration is the one message an account without a key may send with
its classical signature alone. `--pqc-key` attaches a proof of possession, which
the chain verifies and records. Back up the `.dilithium` file: it is the only copy
of the key unless you derive it from a mnemonic with `tx pqc recover-key`.

Check it:

```bash
qorechaind q pqc account $(qorechaind keys show deployer -a) --node $QNODE
curl -s https://api-testnet.qore.host/qorechain/pqc/v1/account/<qor1-address>
```

From now on every Cosmos-lane transaction of this account needs the post-quantum
signature: build it with `--generate-only`, then sign and send it with
`tx pqc cosign`.

## 3. Open a window

```bash
qorechaind tx pqc open-evm-window --from deployer --chain-id $QCHAIN --node $QNODE \
  --gas 400000 --fees 80000uqor --generate-only > window.json

qorechaind tx pqc cosign window.json --from deployer --pqc-key deployer-mldsa \
  --chain-id $QCHAIN --node $QNODE
```

Without arguments the window takes the defaults sized for a deploy session:

| Bound | Default | Meaning |
|---|---|---|
| blocks | 3600 | how long it stays open (about an hour on the testnet, three on mainnet) |
| max-txs | 10 | how many EVM transactions it admits |
| max-value | 25qor | the total QOR the admitted transactions may move, counting each one's value **plus its maximum fee** (gas limit times fee cap) |

Give them explicitly as `open-evm-window <blocks> <max-txs> <max-value>` (for
example `open-evm-window 7200 30 100qor`). The limits are 17,280 blocks and 1,000
transactions per window. Opening a window replaces the previous one;
`close-evm-window` ends it at once.

How much a deploy counts: Hardhat (ethers) sets the fee cap to twice the base fee
plus the tip, about 212.5 gwei today, so a 3,000,000 gas deploy counts about
0.64 QOR against max-value even though it pays less. Foundry adds a 30% margin to
the gas limit. The default 25 QOR covers several large deploys, and a base fee
several times higher.

**Do not narrow a deploy window.** `--native-only` and `--allowed-targets` refuse
contract creation. They are for later sessions: once the contract exists,
`--allowed-targets <contract-address>` limits a window to calls to it, which also
bounds token transfers (the value bound counts native QOR only, and a token
transfer carries value zero).

`--dry-run` on `cosign` prints the gas estimate. From v3.2.5 a node's estimate
for a post-quantum signed transaction is what the block charges (before it, the
estimate was about 35,000 gas short, so add that margin on an older node).

Check the window:

```bash
qorechaind q pqc evm-window <qor1-address> --node $QNODE
curl -s https://api-testnet.qore.host/qorechain/pqc/v1/evm_window/<qor1-address>
```

`live: true` with `remaining_blocks`, `remaining_txs` and `remaining_value`
(uqor) is what you want to see.

## 4a. Deploy with Hardhat

```js
// hardhat.config.js
require("@nomicfoundation/hardhat-toolbox");

module.exports = {
  solidity: "0.8.24",
  networks: {
    diana: {
      url: "https://evm-testnet.qore.host",
      chainId: 9800,
      accounts: [process.env.DEPLOYER_KEY],
    },
    vladi: {
      url: "https://evm.qore.host",
      chainId: 9801,
      accounts: [process.env.DEPLOYER_KEY],
    },
  },
};
```

```js
// scripts/deploy.js
const hre = require("hardhat");

async function main() {
  const counter = await hre.ethers.deployContract("Counter");
  await counter.waitForDeployment();
  console.log("Counter at", await counter.getAddress());

  const tx = await counter.increment();   // a second window transaction
  await tx.wait();
  console.log("count =", (await counter.number()).toString()); // a read: no window needed
}

main().catch((e) => { console.error(e); process.exit(1); });
```

```bash
DEPLOYER_KEY=0x... npx hardhat run scripts/deploy.js --network diana
```

## 4b. Deploy with Foundry

```bash
export ETH_RPC_URL=https://evm-testnet.qore.host

forge create src/Counter.sol:Counter --private-key $DEPLOYER_KEY --broadcast
# Deployed to: 0x...

cast send <contract> "increment()" --private-key $DEPLOYER_KEY
cast call <contract> "number()(uint256)"
```

Scripts work the same way: `forge script script/Counter.s.sol --broadcast
--private-key $DEPLOYER_KEY`. Each broadcast transaction uses one of the window's
`max-txs`.

## What counts against the window

- Every EVM transaction the account sends, deploy or call, when it is admitted,
  whether it then succeeds or reverts.
- Not reads: `eth_call`, `cast call`, view functions and gas estimation need no
  window and use none of it.
- Each admitted transaction adds its value plus gas limit times fee cap to the used
  value, rounded up to whole uqor.
- QOR the transaction moves beyond its value (through a precompile, for example)
  is charged after execution. From v3.2.5 the staking, distribution, ics20 and gov
  precompiles are reachable only through a window that lists them in
  `--allowed-targets`. The full policy is in [EVM_WINDOW.md](./EVM_WINDOW.md).

## Errors

| Message (from `eth_sendRawTransaction`) | Cause | Fix |
|---|---|---|
| `account ... has no registered post-quantum key` | step 2 not done for this account | register the key |
| `no open EVM authorisation window` | no window, or it expired | open a window |
| `EVM authorisation window exhausted` | max-txs or max-value used up | open a new window with larger bounds |
| `... does not admit this transaction: ... not contract creation` | the window has `--native-only` or `--allowed-targets` | open a window without a scope to deploy |
| `only replay-protected (EIP-155) Ethereum transactions are accepted` | a transaction signed without a chain id | set `chainId` in the network config |

## MetaMask and ethers

The same window covers transactions from MetaMask or any ethers or viem client:
add the network with the chain id and RPC URL above (currency QOR, 18 decimals),
import the same key, and send as usual while the window is live.
`@qorechain/wallet-adapter` emits the right EIP-3085 parameters with
`qoreEvmChainParams` and `addQoreEvmToWallet`.

See also [EVM.md](./EVM.md) for the runtime, [EVM_WINDOW.md](./EVM_WINDOW.md) for the window
policy and [EVM_PRECOMPILES.md](./EVM_PRECOMPILES.md) for the precompiles.
