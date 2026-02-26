# JSON-RPC -- eth\_ Namespace

QoreChain implements a fully EVM-compatible JSON-RPC interface, enabling standard Ethereum tooling (MetaMask, Hardhat, Foundry, ethers.js, web3.js) to interact with the chain without modification.

## Connection

| Transport | Default Address         |
| --------- | ----------------------- |
| HTTP      | `http://localhost:8545` |
| WebSocket | `ws://localhost:8546`   |

---

## Supported Namespaces

| Namespace | Description                                 |
| --------- | ------------------------------------------- |
| `eth_`    | Core Ethereum JSON-RPC methods              |
| `web3_`   | Utility methods (client version, hashing)   |
| `net_`    | Network status methods                      |
| `txpool_` | Transaction pool inspection                 |
| `qor_`    | QoreChain-specific extensions (see [qor\_ Namespace](json-rpc-qor.md)) |

---

## eth\_ Methods

| Method                        | Parameters                                              | Description                                           |
| ----------------------------- | ------------------------------------------------------- | ----------------------------------------------------- |
| `eth_blockNumber`             | none                                                    | Returns the latest block number                       |
| `eth_getBalance`              | `address`, `blockNumber`                                | Returns the balance of an address in wei              |
| `eth_getTransactionCount`     | `address`, `blockNumber`                                | Returns the nonce (transaction count) for an address  |
| `eth_sendRawTransaction`      | `signedTxData`                                          | Submits a signed transaction for broadcast            |
| `eth_call`                    | `callObject`, `blockNumber`                             | Executes a read-only call against the EVM             |
| `eth_estimateGas`             | `callObject`                                            | Estimates the gas required for a transaction          |
| `eth_getBlockByNumber`        | `blockNumber`, `fullTx` (bool)                          | Returns block data by number                          |
| `eth_getTransactionByHash`    | `txHash`                                                | Returns transaction data by hash                      |
| `eth_getTransactionReceipt`   | `txHash`                                                | Returns the receipt for a mined transaction           |
| `eth_getLogs`                 | `filterObject`                                          | Returns logs matching a filter                        |
| `eth_chainId`                 | none                                                    | Returns the chain ID (hex-encoded)                    |
| `eth_gasPrice`                | none                                                    | Returns the current gas price in wei                  |
| `eth_feeHistory`              | `blockCount`, `newestBlock`, `rewardPercentiles`        | Returns historical fee data (EIP-1559)                |

---

## web3\_ Methods

| Method              | Parameters       | Description                                |
| ------------------- | ---------------- | ------------------------------------------ |
| `web3_clientVersion` | none            | Returns the client version string          |
| `web3_sha3`          | `data` (hex)    | Returns the Keccak-256 hash of the input   |

---

## net\_ Methods

| Method          | Parameters | Description                                    |
| --------------- | ---------- | ---------------------------------------------- |
| `net_version`   | none       | Returns the network ID                         |
| `net_listening` | none       | Returns `true` if the node is listening        |
| `net_peerCount` | none       | Returns the number of connected peers (hex)    |

---

## Configuration

Enable and configure the JSON-RPC server in `app.toml`:

```toml
[json-rpc]
# Enable the JSON-RPC server
enable = true

# HTTP server address
address = "0.0.0.0:8545"

# WebSocket server address
ws-address = "0.0.0.0:8546"

# Enabled API namespaces
api = "eth,web3,net,txpool,qor"

# Maximum number of logs returned by eth_getLogs
filter-cap = 10000

# Maximum gas for eth_call and eth_estimateGas
gas-cap = 25000000

# EVM execution timeout
evm-timeout = "5s"

# Transaction fee cap (in QOR)
txfee-cap = 1

# Maximum open WebSocket connections
max-open-connections = 0
```

---

## Examples

### eth\_blockNumber

**Request:**

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "eth_blockNumber",
    "params": [],
    "id": 1
  }'
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": "0x53b35"
}
```

### eth\_getBalance

**Request:**

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "eth_getBalance",
    "params": ["0x742d35Cc6634C0532925a3b844Bc9e7595f2bD28", "latest"],
    "id": 2
  }'
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": "0x56bc75e2d63100000"
}
```

---

## Connecting with ethers.js

```javascript
import { ethers } from "ethers";

const provider = new ethers.JsonRpcProvider("http://localhost:8545");

// Get latest block
const block = await provider.getBlockNumber();
console.log("Latest block:", block);

// Get balance
const balance = await provider.getBalance("0x742d35Cc6634C0532925a3b844Bc9e7595f2bD28");
console.log("Balance:", ethers.formatEther(balance), "QOR");
```

---

## Notes

- **Chain ID** is returned as a hex string. Convert to decimal for wallet configuration.
- **Gas pricing** follows the EIP-1559 model. Use `eth_feeHistory` for base fee and priority fee estimation.
- **Block tags** accepted: `"latest"`, `"earliest"`, `"pending"`, or a hex block number.
- **Filter limitations**: `eth_getLogs` is capped at `filter-cap` results per query (default 10,000). Use narrower block ranges for large datasets.
