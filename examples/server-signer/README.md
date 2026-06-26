# QoreChain headless server-side signer (PQC rail)

A standalone Node.js script that performs a **Cosmos-native, PQC-secured**
QOR withdrawal end-to-end with no browser and no Keplr:

1. derives the classical secp256k1 hot-wallet key from a mnemonic,
2. resolves / derives the FIPS-204 **ML-DSA-87** keypair (the PQC half),
3. builds a `bank.MsgSend`,
4. layers the ML-DSA-87 **hybrid signature** into the tx-body extension (the
   exact framing the chain's PQC ante chain re-derives), then adds the classical
   secp256k1 signature,
5. broadcasts the assembled `TxRaw` over the native RPC.

It is the programmatic equivalent of:

```bash
qorechaind tx bank send <from> <to> 1000000uqor --generate-only > tx.json
qorechaind tx pqc cosign tx.json --from <from> --pqc-key <name> --chain-id qorechain-diana
```

…but built on the published npm packages `@qorechain/wallet-adapter` and
`@qorechain/pqc`, so it drops straight into an exchange withdrawal worker.

## Install

```bash
cd qorechain-core/examples/server-signer
npm install
```

This pulls the published `@qorechain/wallet-adapter` and `@qorechain/pqc`
together with the `@cosmjs/*` peer deps.

## One-time on-chain bootstrap (per hot wallet)

The chain requires every Cosmos tx to carry an ML-DSA-87 hybrid signature, but
the **registration tx itself is classical-exempt** so you can bootstrap. Do this
once for the hot wallet:

```bash
# (a) generate + store a Dilithium-5 / ML-DSA-87 key and print its pubkey hex
qorechaind tx pqc gen-key hotwallet --from hotwallet
#   -> stored Dilithium-5 private key: ~/.qorechaind/pqc/hotwallet.dilithium
#   -> public_key_hex: <PUB_HEX>

# (b) register that pubkey on-chain as a hybrid key (classical-signed, exempt)
qorechaind tx pqc register-key <PUB_HEX> hybrid --from hotwallet --chain-id qorechain-diana
```

After registration, every further tx from that account MUST be hybrid-signed.

> This Node script can **derive** an ML-DSA-87 key deterministically from the
> mnemonic (via SHAKE-256) instead of reading the CLI's `.dilithium` file. If you
> let it derive, register the public key it prints on first run. To instead reuse
> the exact key the CLI generated, export its hex as `QORE_PQC_SK_HEX`.

## Run

```bash
export QORE_RPC=http://127.0.0.1:26657
export QORE_CHAIN_ID=qorechain-diana
export QORE_MNEMONIC="word1 word2 ... word24"   # hot-wallet seed
export QORE_TO=qor1recipient...
export QORE_AMOUNT=1000000                       # uqor (6 dec) -> 1 QOR
# optional: export QORE_PQC_SK_HEX=<registered ml-dsa-87 secret key hex>
node sign-and-broadcast.mjs
```

Expected output ends with `code: 0 (success)` and a `txhash`.

## Syntax check (no node required)

```bash
node --check sign-and-broadcast.mjs   # prints nothing, exit 0 if valid
```

## Notes

- **Denominations:** the Cosmos/bank denom is `uqor` (6 decimals; 1 QOR =
  1,000,000 uqor). The EVM lane uses an 18-decimal view — irrelevant here, this
  is the native rail.
- **Hybrid framing** is identical to the chain's `tx pqc cosign`:
  `B0 = TxBody{messages, memo, timeoutHeight}` (no extension);
  `sigP = ML-DSA-87.sign( BE32(len B0)‖B0‖BE32(len authInfo)‖authInfo )`;
  the extension `PQCHybridSignature{algorithmId=1, sigP}` is baked into the body
  BEFORE the classical secp256k1 signature is taken over the final `SignDoc`.
  All of that lives in `@qorechain/wallet-adapter`'s `QoreChainSigner`.
- **Security:** keep `QORE_MNEMONIC` (and `QORE_PQC_SK_HEX` if used) in a secrets
  manager / HSM, never on disk in plaintext. This example reads env vars only.

See `../../docs/EXCHANGE_INTEGRATION.md` for the full two-rail integration
runbook (EVM no-PQC vs. Cosmos-native PQC).
