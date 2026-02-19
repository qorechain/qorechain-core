// QoreChain Wallet API — Testnet Integration Patch
//
// Adds real balance queries from testnet node.
//
// Required env vars:
//   USE_TESTNET=true
//   TESTNET_REST_URL=http://<ec2-ip>:1317

import * as testnet from './testnet-client.mjs';

// Replace the getWalletBalances function with this:
export async function getWalletBalances_patched(walletAddress) {
  if (testnet.isTestnetEnabled() && walletAddress) {
    try {
      const balances = await testnet.getBalance(walletAddress);
      return {
        uqor: balances.uqor || 0,
        qor: Math.floor((balances.uqor || 0) / 1_000_000),
        source: 'testnet',
      };
    } catch (e) {
      console.warn('Testnet balance query failed:', e.message);
    }
  }

  // Fallback: return mock balance
  return {
    uqor: 0,
    qor: 0,
    source: 'mock',
  };
}
