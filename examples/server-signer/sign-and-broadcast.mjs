#!/usr/bin/env node
// QoreChain headless server-side signer.
//
// Builds a bank MsgSend, layers the FIPS-204 ML-DSA-87 hybrid signature that the
// chain's PQC ante chain requires, signs the classical secp256k1 part from a
// mnemonic (no browser, no Keplr), and broadcasts the TxRaw over the native RPC.
//
// This mirrors `qorechaind tx bank send --generate-only | qorechaind tx pqc cosign`,
// but in pure Node.js — suitable for an exchange hot-wallet withdrawal worker.
//
// Uses the published packages:
//   @qorechain/wallet-adapter  — QoreChainSigner (the hybrid framing/signing)
//   @qorechain/pqc             — mldsa (ML-DSA-87), shake256
//   @cosmjs/proto-signing      — secp256k1 wallet from mnemonic (peer dep)
//   @cosmjs/stargate           — account number/sequence lookup + broadcast
//
// Run:  node sign-and-broadcast.mjs   (see README.md for env vars)

import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { StargateClient } from '@cosmjs/stargate';
import { MsgSend } from 'cosmjs-types/cosmos/bank/v1beta1/tx.js';
import { QoreChainSigner } from '@qorechain/wallet-adapter';
import { mldsa, shake256 } from '@qorechain/pqc';

// ---------------------------------------------------------------------------
// Configuration (env-driven so the same script works against any QoreChain net)
// ---------------------------------------------------------------------------
const RPC          = process.env.QORE_RPC      || 'http://127.0.0.1:26657';
const CHAIN_ID     = process.env.QORE_CHAIN_ID || 'qorechain-diana';
const MNEMONIC     = process.env.QORE_MNEMONIC || '';        // hot-wallet seed (24 words)
const TO           = process.env.QORE_TO       || 'qor1recipientaddresshere0000000000000000000';
const AMOUNT_UQOR  = process.env.QORE_AMOUNT   || '1000000'; // 1 QOR = 1_000_000 uqor (6 dec)
const MEMO         = process.env.QORE_MEMO     || '';
// The ML-DSA-87 secret key registered on-chain for this hot wallet (hex).
// Generate + register once with the bootstrap flow in README.md, then keep it
// in your secrets manager. If unset we DERIVE one deterministically below.
const PQC_SK_HEX   = process.env.QORE_PQC_SK_HEX || '';

if (!MNEMONIC) {
  console.error('Set QORE_MNEMONIC to the hot-wallet 24-word seed. Aborting.');
  process.exit(1);
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function hexToBytes(hex) {
  const clean = hex.startsWith('0x') ? hex.slice(2) : hex;
  const out = new Uint8Array(clean.length / 2);
  for (let i = 0; i < out.length; i++) out[i] = parseInt(clean.substr(i * 2, 2), 16);
  return out;
}
function bytesToHex(bytes) {
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
}

// Resolve the ML-DSA-87 keypair for the hot wallet.
//   - If QORE_PQC_SK_HEX is set, use that exact registered secret key.
//   - Otherwise derive one deterministically from the mnemonic via SHAKE-256,
//     so re-runs are reproducible. The matching public key MUST already be
//     registered on-chain (`tx pqc register-key <pub-hex> hybrid`).
function resolvePqcKeypair(mnemonic) {
  if (PQC_SK_HEX) {
    const secretKey = hexToBytes(PQC_SK_HEX);
    // Derive the public key by re-keygen is not possible from sk alone; the
    // adapter only needs secretKey to SIGN. publicKey is informational here.
    return { secretKey, publicKey: new Uint8Array(0) };
  }
  // Deterministic seed: SHAKE-256(mnemonic) -> 32-byte ML-DSA xi.
  const seed = shake256(new TextEncoder().encode(mnemonic), 32);
  return mldsa.keygen(seed); // { publicKey, secretKey }
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------
async function main() {
  // 1. Classical secp256k1 wallet from the mnemonic (qor-prefixed addresses).
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(MNEMONIC, { prefix: 'qor' });
  const [account] = await wallet.getAccounts();
  const fromAddress = account.address;
  const pubkeySecp256k1 = account.pubkey; // compressed 33-byte secp256k1 pubkey
  console.log('hot wallet:', fromAddress);

  // 2. ML-DSA-87 keypair (the PQC half of the hybrid signature).
  const pqc = resolvePqcKeypair(MNEMONIC);
  if (pqc.publicKey.length) {
    console.log('ML-DSA-87 pubkey (register on-chain if not already):');
    console.log('  ' + bytesToHex(pqc.publicKey));
    console.log('  qorechaind tx pqc register-key ' + bytesToHex(pqc.publicKey) + ' hybrid --from <key>');
  }

  // 3. On-chain account number + sequence (needed for the SignDoc).
  const client = await StargateClient.connect(RPC);
  const acct = await client.getAccount(fromAddress);
  if (!acct) {
    throw new Error(`account ${fromAddress} not found on-chain — fund it first`);
  }
  const { accountNumber, sequence } = acct;

  // 4. Build the bank MsgSend (uqor, 6-decimal base denom).
  const msgSend = {
    typeUrl: '/cosmos.bank.v1beta1.MsgSend',
    value: MsgSend.fromPartial({
      fromAddress,
      toAddress: TO,
      amount: [{ denom: 'uqor', amount: AMOUNT_UQOR }],
    }),
  };

  // 5. Fee (gasPriceStep low = 0.001uqor; 200000 gas -> 200 uqor fee).
  const fee = {
    amount: [{ denom: 'uqor', amount: '5000' }],
    gasLimit: 200000n,
  };

  // 6. The adapter exposes a wallet-like signDirect; @cosmjs's offline signer
  //    provides exactly that. QoreChainSigner does the ML-DSA-87 framing and
  //    asks this signer for the classical secp256k1 signature.
  const signer = new QoreChainSigner({
    wallet: {
      // QoreChainSigner calls wallet.signDirect(chainId, address, signDoc).
      signDirect: (chainId, address, signDoc) => wallet.signDirect(address, signDoc),
    },
    chainId: CHAIN_ID,
    address: fromAddress,
    pubkeySecp256k1,
    accountNumber,
    pqc,
  });

  const txBytes = await signer.signHybrid({
    messages: [msgSend],
    fee,
    memo: MEMO,
    sequence,
  });

  // 7. Broadcast the assembled TxRaw.
  console.log(`broadcasting ${AMOUNT_UQOR}uqor -> ${TO} ...`);
  const result = await client.broadcastTx(txBytes);
  console.log('txhash:', result.transactionHash);
  console.log('code  :', result.code, result.code === 0 ? '(success)' : '(FAILED)');
  if (result.code !== 0) {
    console.error('rawLog:', result.rawLog);
    process.exit(1);
  }
  client.disconnect();
}

main().catch((err) => {
  console.error('signer error:', err.message || err);
  process.exit(1);
});
