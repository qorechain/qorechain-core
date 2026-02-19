// QoreChain Faucet API — Testnet Integration Patch
//
// For testnet faucet, we need to broadcast real transactions.
// This requires a funded faucet account key on the Lambda side.
//
// Strategy: Lambda calls the node's REST API to broadcast a signed tx.
// The faucet mnemonic is stored in QoreChain Secrets Manager or env var.
//
// Required env vars:
//   USE_TESTNET=true
//   TESTNET_REST_URL=http://<ec2-ip>:1317
//   FAUCET_MNEMONIC=<mnemonic from init-testnet.sh>

import * as testnet from './testnet-client.mjs';

// The faucet needs CosmJS for signing. Install as a Lambda layer:
//   npm install @cosmjs/stargate @cosmjs/proto-signing @cosmjs/crypto

export async function distributeFaucetTokens_patched(recipientAddress, amountUqor = 1_000_000_000) {
  if (!testnet.isTestnetEnabled()) {
    return { success: false, error: 'Testnet not enabled' };
  }

  // For MVP, the faucet can use `qorechaind tx bank send` via exec,
  // or we can set up a simple HTTP endpoint on the node container
  // that accepts faucet requests.
  //
  // Full CosmJS integration:
  //
  // import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
  // import { SigningStargateClient } from '@cosmjs/stargate';
  //
  // const mnemonic = process.env.FAUCET_MNEMONIC;
  // const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, { prefix: 'qor' });
  // const [account] = await wallet.getAccounts();
  //
  // const client = await SigningStargateClient.connectWithSigner(
  //   process.env.TESTNET_RPC_URL, wallet
  // );
  //
  // const result = await client.sendTokens(
  //   account.address, recipientAddress,
  //   [{ denom: 'uqor', amount: String(amountUqor) }],
  //   { amount: [{ denom: 'uqor', amount: '500' }], gas: '200000' }
  // );
  //
  // return { success: true, txHash: result.transactionHash };

  return {
    success: false,
    error: 'CosmJS integration pending — use init-testnet.sh faucet account for manual distribution',
    hint: `qorechaind tx bank send faucet ${recipientAddress} ${amountUqor}uqor --keyring-backend test --chain-id qorechain-testnet-1`,
  };
}
