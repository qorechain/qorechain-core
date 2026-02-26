# CosmWasm Development

QoreChain supports **CosmWasm** smart contracts, enabling developers to write secure, sandboxed programs in Rust that compile to WebAssembly. CosmWasm contracts run alongside EVM and SVM programs within the QoreChain triple-VM architecture.

---

## Prerequisites

| Dependency | Version | Purpose |
|------------|---------|---------|
| **Rust** | Latest stable | Contract compilation |
| **wasm32-unknown-unknown** | target | WebAssembly compilation target |
| **cargo-generate** | Latest | Project scaffolding |
| **cosmwasm-std** | 1.5+ | CosmWasm standard library |

Install the Wasm target:

```bash
rustup target add wasm32-unknown-unknown
```

---

## Contract Lifecycle

CosmWasm contracts follow a five-step lifecycle: **Build**, **Store**, **Instantiate**, **Execute**, and **Query**.

### 1. Build

Compile your contract to optimized WebAssembly:

```bash
cd my-contract

# Standard build
cargo build --release --target wasm32-unknown-unknown

# Optimized build (recommended for deployment)
docker run --rm -v "$(pwd)":/code \
  cosmwasm/rust-optimizer:0.15.0
```

The optimized `.wasm` file will be in the `artifacts/` directory.

### 2. Store

Upload the compiled contract to the chain:

```bash
qorechaind tx wasm store contract.wasm \
  --from mykey \
  --gas auto \
  --gas-adjustment 1.3 \
  -y
```

After the transaction confirms, query the stored code ID:

```bash
qorechaind query wasm list-code
```

### 3. Instantiate

Create a new contract instance from a stored code ID:

```bash
qorechaind tx wasm instantiate <code-id> \
  '{"count": 0, "owner": "qor1..."}' \
  --label "my-counter-contract" \
  --from mykey \
  --no-admin \
  --gas auto \
  --gas-adjustment 1.3 \
  -y
```

| Flag | Description |
|------|-------------|
| `<code-id>` | Numeric ID returned from the store transaction |
| `--label` | Human-readable label for this instance |
| `--no-admin` | No admin address (contract is immutable) |
| `--admin <address>` | Set an admin who can migrate the contract |

Retrieve the contract address:

```bash
qorechaind query wasm list-contracts-by-code <code-id>
```

### 4. Execute

Call a contract's execute entry point to change state:

```bash
qorechaind tx wasm execute <contract-addr> \
  '{"increment": {}}' \
  --from mykey \
  --gas auto \
  -y
```

Execute with funds attached:

```bash
qorechaind tx wasm execute <contract-addr> \
  '{"deposit": {}}' \
  --amount 1000000uqor \
  --from mykey \
  -y
```

### 5. Query

Read contract state without submitting a transaction:

```bash
qorechaind query wasm contract-state smart <contract-addr> \
  '{"get_count": {}}'
```

Query responses are returned as JSON.

---

## Useful Queries

```bash
# List all stored code
qorechaind query wasm list-code

# List all instances of a specific code ID
qorechaind query wasm list-contracts-by-code <code-id>

# Get contract info (code ID, admin, label)
qorechaind query wasm contract <contract-addr>

# Get raw contract state by key
qorechaind query wasm contract-state raw <contract-addr> <key-hex>

# Get full contract state (all keys)
qorechaind query wasm contract-state all <contract-addr>

# Get contract history (instantiate/migrate events)
qorechaind query wasm contract-history <contract-addr>
```

---

## Contract Structure

A typical CosmWasm contract has three entry points:

```rust
use cosmwasm_std::{
    entry_point, Binary, Deps, DepsMut, Env, MessageInfo,
    Response, StdResult,
};

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    // Initialize contract state
    Ok(Response::new().add_attribute("method", "instantiate"))
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response> {
    // Handle state-changing operations
    Ok(Response::new().add_attribute("method", "execute"))
}

#[entry_point]
pub fn query(
    deps: Deps,
    _env: Env,
    msg: QueryMsg,
) -> StdResult<Binary> {
    // Handle read-only queries
    Ok(Binary::default())
}
```

---

## Cross-VM Calls

CosmWasm contracts can interact with contracts deployed on the EVM and SVM through the `x/crossvm` module. Cross-VM calls from CosmWasm use the **asynchronous** message path:

```rust
use cosmwasm_std::{CosmosMsg, CustomMsg};

// Call an EVM contract from CosmWasm
let cross_vm_msg = CosmosMsg::Custom(QoreChainMsg::CrossVMCall {
    target_vm: "evm".to_string(),
    target_contract: "0x1234...abcd".to_string(),
    payload: hex::encode(abi_encoded_data),
    funds: vec![],
});

Ok(Response::new().add_message(cross_vm_msg))
```

The message is submitted to a queue and processed by the EndBlocker in the next block. See [Cross-VM Interoperability](cross-vm-interop.md) for the full message lifecycle.

---

## Module Integration

CosmWasm contracts can interact with QoreChain SDK modules through standard message passing:

```rust
// Send native tokens via the bank module
let send_msg = BankMsg::Send {
    to_address: "qor1recipient...".to_string(),
    amount: vec![Coin {
        denom: "uqor".to_string(),
        amount: Uint128::new(1_000_000),
    }],
};

// Delegate to a validator via the staking module
let delegate_msg = StakingMsg::Delegate {
    validator: "qorvaloper1...".to_string(),
    amount: Coin {
        denom: "uqor".to_string(),
        amount: Uint128::new(1_000_000),
    },
};
```

---

## Contract Migration

If the contract was instantiated with an `--admin` address, the admin can migrate it to a new code ID:

```bash
qorechaind tx wasm migrate <contract-addr> <new-code-id> \
  '{"migrate_msg": {}}' \
  --from admin-key \
  -y
```

This calls the `migrate` entry point on the new code with the existing contract state.

---

## Next Steps

- [Cross-VM Interoperability](cross-vm-interop.md) -- Call EVM and SVM contracts from CosmWasm
- [SVM Development](svm-development.md) -- Deploy BPF programs on QoreChain
- [EVM Precompiles](evm-precompiles.md) -- Access PQC and AI features from Solidity
