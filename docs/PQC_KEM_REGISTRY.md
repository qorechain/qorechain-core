# Published encryption keys (the KEM registry)

From v3.2.5 (protocol feature `pqc-kem-registry`) an account can publish one post-quantum
**key-encapsulation** public key on chain, so that anyone can encrypt to it: a sender reads the key,
encapsulates a shared secret to it with ML-KEM (FIPS 203), and only the holder of the matching secret
key can recover that secret.

The published key is **not** an authentication key. The account keeps signing with its ML-DSA-87 key,
and that signature is what authenticates the registration, so the KEM key needs no proof of
possession of its own: a KEM cannot sign, and an account that publishes a key it cannot decapsulate
with only stops itself from receiving.

## Algorithm

| ID | Algorithm | Public key |
|---|---|---|
| 2 | ML-KEM-1024 (FIPS 203) | 1,568 bytes: `ByteEncode12(t_hat) || rho` |

A key-encapsulation algorithm added by governance to the on-chain algorithm registry later is
accepted at the size the registry records. A signature algorithm is refused.

An ML-KEM-1024 key must pass the FIPS 203 encapsulation key check (every 12-bit coefficient of
`t_hat` below q = 3329), so a conforming sender can always encrypt to it, and a key whose `t_hat` is
all zero is refused, because anything encrypted to it is readable by anyone.

## Messages

| Message | Effect |
|---|---|
| `/qorechain.pqc.v1.MsgRegisterKEMKey` `{sender, algorithm_id, public_key}` | publishes the key; sending it again replaces the published key (the response says `replaced`) |
| `/qorechain.pqc.v1.MsgRemoveKEMKey` `{sender}` | withdraws the published key |

Rules for both:

- the sender must already hold a registered post-quantum signature key;
- the transaction needs the account's hybrid signature (classical plus ML-DSA-87); the messages are
  not part of the registration-only set that may be signed classically;
- they are refused inside an authz grant: only the account itself maintains its key.

Events: `pqc_kem_key_registered` (`address`, `algorithm_id`, `replaced`) and `pqc_kem_key_removed`
(`address`).

Withdrawing a key does not make data already encrypted to it unreadable to the holder of the secret
key; it only stops new senders from finding it.

## Reading a key

All three answer at any height, before the registry opens too (`found` is then false). The address is
bech32 (`qor1...`) or `0x` hex.

| Interface | Request |
|---|---|
| REST | `GET /qorechain/pqc/v1/kem/{address}` |
| gRPC | `qorechain.pqc.v1.Query/KEMKey` |
| CLI | `qorechaind q pqc kem-key <address>` |
| JSON-RPC (EVM endpoint) | `qor_getKEMKey(address)` |

REST and gRPC answer `{found, key: {address, algorithm_id, algorithm_name, public_key, registered_height}}`
(`public_key` base64 in REST). `registered_height` is the height of the registration that published
the current key; a replacement resets it.

`qor_getKEMKey` answers on the EVM JSON-RPC endpoint, with the key as `0x` hex the way Ethereum
tooling expects bytes:

```bash
curl -s -X POST $EVM_RPC -H 'content-type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"qor_getKEMKey","params":["qor1..."]}'
```

```json
{"jsonrpc":"2.0","id":1,"result":{"address":"qor1...","found":true,"algorithm_id":2,
 "algorithm_name":"mlkem1024","public_key":"0x...","registered_height":123}}
```

The `qor` namespace is served only where the node operator lists `qor` in `app.toml`
`[json-rpc] api`. It is read-only.

## CLI

```bash
# generate a key pair: secret in <home>/pqc/<name>.mlkem1024 (mode 0600), public key next to it
qorechaind tx pqc gen-kem-key inbox

# publish it (hybrid-signed like every other key maintenance)
qorechaind tx pqc register-kem-key --kem-key inbox --from me --generate-only > kem.json
qorechaind tx pqc cosign kem.json --from me --pqc-key me-mldsa --chain-id $QCHAIN --node $QNODE

# withdraw it
qorechaind tx pqc remove-kem-key --from me --generate-only > rm.json
qorechaind tx pqc cosign rm.json --from me --pqc-key me-mldsa --chain-id $QCHAIN --node $QNODE
```

`register-kem-key` also takes the public key as hex instead of `--kem-key`.

## Error codes (codespace `pqc`)

| Code | Error | When |
|---|---|---|
| 250 | `the KEM key registry is not enabled on this chain yet` | before the `pqc-kem-registry` feature |
| 251 | `invalid KEM public key` | wrong size, empty, over 4,096 bytes, or failing the encapsulation key check |
| 252 | `no KEM key published for this account` | `MsgRemoveKEMKey` with nothing published |
| 253 | `this KEM key is already published for this account` | publishing the same key again |
| 254 | `publishing a KEM key needs a registered post-quantum signature key` | the sender has no signature key |

An `algorithm_id` that names a signature algorithm, or an algorithm that is not active in the
registry, is refused with the module's algorithm errors.
