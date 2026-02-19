// QoreChain Explorer API — Testnet Integration Patch
//
// Add these imports and replace the mock functions in qorechain-explorer-api/index.mjs
// with the testnet-aware versions below.
//
// Required env vars:
//   USE_TESTNET=true
//   TESTNET_REST_URL=http://<ec2-ip>:1317
//   TESTNET_RPC_URL=http://<ec2-ip>:26657

import * as testnet from './testnet-client.mjs';

// Replace: getRecentBlocks(limit) — line ~379
export async function getRecentBlocks_patched(limit) {
  if (testnet.isTestnetEnabled()) {
    try {
      const { blocks, currentHeight } = await testnet.getRecentBlocks(limit);
      return {
        statusCode: 200,
        headers: corsHeaders,
        body: JSON.stringify({ success: true, blocks, currentHeight, source: 'testnet' })
      };
    } catch (e) {
      console.warn('Testnet block fetch failed, falling back to mock:', e.message);
    }
  }
  // Original mock fallback
  return getRecentBlocks_original(limit);
}

// Replace: getBlockDetails(height) — line ~395
export async function getBlockDetails_patched(height) {
  if (testnet.isTestnetEnabled()) {
    try {
      const block = await testnet.getBlock(height);
      return {
        statusCode: 200,
        headers: corsHeaders,
        body: JSON.stringify({ success: true, block, source: 'testnet' })
      };
    } catch (e) {
      console.warn('Testnet block detail failed:', e.message);
    }
  }
  return getBlockDetails_original(height);
}

// Replace: getTransactionDetails(hash) — line ~122
export async function getTransactionDetails_patched(hash) {
  if (testnet.isTestnetEnabled()) {
    try {
      const transaction = await testnet.getTransaction(hash);
      return {
        statusCode: 200,
        headers: corsHeaders,
        body: JSON.stringify({ success: true, transaction, source: 'testnet' })
      };
    } catch (e) {
      console.warn('Testnet tx lookup failed, trying DynamoDB:', e.message);
    }
  }
  // Fall through to existing DynamoDB + mock logic
  return getTransactionDetails_original(hash);
}

// Replace: getRecentTransactions(limit) — line ~295
export async function getRecentTransactions_patched(limit) {
  if (testnet.isTestnetEnabled()) {
    try {
      const transactions = await testnet.getRecentTransactions(limit);
      if (transactions.length > 0) {
        return {
          statusCode: 200,
          headers: corsHeaders,
          body: JSON.stringify({ success: true, transactions, total: transactions.length, source: 'testnet' })
        };
      }
    } catch (e) {
      console.warn('Testnet tx list failed:', e.message);
    }
  }
  return getRecentTransactions_original(limit);
}

// Replace: getAddressInfo(address) — line ~413
export async function getAddressInfo_patched(address) {
  if (testnet.isTestnetEnabled()) {
    try {
      const account = await testnet.getAccount(address);
      return {
        statusCode: 200,
        headers: corsHeaders,
        body: JSON.stringify({
          success: true,
          address: account,
          source: 'testnet'
        })
      };
    } catch (e) {
      console.warn('Testnet address lookup failed:', e.message);
    }
  }
  return getAddressInfo_original(address);
}

// Replace: getNetworkStats() — line ~429
export async function getNetworkStats_patched() {
  if (testnet.isTestnetEnabled()) {
    try {
      const stats = await testnet.getNetworkStats();
      return {
        statusCode: 200,
        headers: corsHeaders,
        body: JSON.stringify({ success: true, stats, source: 'testnet' })
      };
    } catch (e) {
      console.warn('Testnet stats failed:', e.message);
    }
  }
  return getNetworkStats_original();
}
