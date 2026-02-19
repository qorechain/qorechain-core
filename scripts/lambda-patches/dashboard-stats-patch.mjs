// QoreChain Dashboard Stats — Testnet Integration Patch
//
// Add testnet blockchain stats alongside existing DynamoDB stats.
//
// Required env vars:
//   USE_TESTNET=true
//   TESTNET_REST_URL=http://<ec2-ip>:1317

import * as testnet from './testnet-client.mjs';

// Add this to the handler's try block, merge with existing response:
export async function getTestnetStats() {
  if (!testnet.isTestnetEnabled()) return null;

  try {
    const stats = await testnet.getNetworkStats();
    return {
      blockchain: {
        blockHeight: stats.blockHeight,
        activeValidators: stats.activeValidators,
        avgBlockTime: stats.avgBlockTime,
        chainId: 'qorechain-testnet-1',
        source: 'testnet',
      }
    };
  } catch (e) {
    console.warn('Testnet stats unavailable:', e.message);
    return null;
  }
}

// Modified handler pattern:
// In the original handler, add after line 46:
//
//   const testnetStats = await getTestnetStats();
//
// Then merge into response at line 48:
//
//   const response = {
//     stats: { ...originalStats, ...(testnetStats?.blockchain || {}) },
//     latestContracts: contractsData.latest,
//     latestAudits: auditsData.latest,
//     testnet: testnetStats,
//   };
