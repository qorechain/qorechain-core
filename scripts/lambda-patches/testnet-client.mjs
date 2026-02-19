// QoreChain Testnet Client — Shared module for Lambda ↔ Testnet integration
// Drop this file into each Lambda that needs testnet access.

const TESTNET_REST = process.env.TESTNET_REST_URL || '';
const TESTNET_RPC = process.env.TESTNET_RPC_URL || '';
const USE_TESTNET = process.env.USE_TESTNET === 'true';

export function isTestnetEnabled() {
  return USE_TESTNET && TESTNET_REST;
}

async function fetchJSON(url) {
  const resp = await fetch(url);
  if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${url}`);
  return resp.json();
}

// ────────────────────── Block queries ──────────────────────

export async function getLatestBlock() {
  const data = await fetchJSON(`${TESTNET_REST}/cosmos/base/tendermint/v1beta1/blocks/latest`);
  return {
    height: parseInt(data.block.header.height),
    hash: data.block_id.hash,
    timestamp: data.block.header.time,
    proposer: data.block.header.proposer_address,
    txCount: (data.block.data.txs || []).length,
  };
}

export async function getBlock(height) {
  const data = await fetchJSON(`${TESTNET_REST}/cosmos/base/tendermint/v1beta1/blocks/${height}`);
  return {
    height: parseInt(data.block.header.height),
    hash: data.block_id.hash,
    timestamp: data.block.header.time,
    proposer: data.block.header.proposer_address,
    txCount: (data.block.data.txs || []).length,
  };
}

export async function getRecentBlocks(limit = 20) {
  const latest = await getLatestBlock();
  const blocks = [latest];
  for (let i = 1; i < limit && latest.height - i > 0; i++) {
    try {
      blocks.push(await getBlock(latest.height - i));
    } catch { break; }
  }
  return { blocks, currentHeight: latest.height };
}

// ────────────────────── Transaction queries ──────────────────────

export async function getTransaction(hash) {
  const data = await fetchJSON(`${TESTNET_REST}/cosmos/tx/v1beta1/txs/${hash}`);
  const resp = data.tx_response;
  return {
    hash: resp.txhash,
    type: extractTxType(resp.events),
    status: resp.code === 0 ? 'confirmed' : 'failed',
    blockHeight: parseInt(resp.height),
    timestamp: resp.timestamp,
    from: extractSender(resp.events),
    to: extractReceiver(resp.events),
    amount: extractAmount(resp.events),
    token: 'QOR',
    fee: extractFee(data.tx),
    pqcVerified: hasPQCEvent(resp.events),
    aiScore: extractAIScore(resp.events),
  };
}

export async function getRecentTransactions(limit = 20) {
  const latest = await getLatestBlock();
  const txs = [];
  // Search recent blocks for transactions
  for (let h = latest.height; h > latest.height - 50 && txs.length < limit; h--) {
    try {
      const data = await fetchJSON(
        `${TESTNET_REST}/cosmos/tx/v1beta1/txs?events=tx.height=${h}`
      );
      for (const resp of (data.tx_responses || [])) {
        txs.push({
          hash: resp.txhash,
          type: extractTxType(resp.events),
          from: extractSender(resp.events),
          to: extractReceiver(resp.events),
          amount: extractAmount(resp.events),
          token: 'QOR',
          timestamp: resp.timestamp,
          status: resp.code === 0 ? 'confirmed' : 'failed',
          blockHeight: parseInt(resp.height),
        });
        if (txs.length >= limit) break;
      }
    } catch { continue; }
  }
  return txs;
}

// ────────────────────── Account / Balance ──────────────────────

export async function getBalance(address) {
  const data = await fetchJSON(
    `${TESTNET_REST}/cosmos/bank/v1beta1/balances/${address}`
  );
  const balances = {};
  for (const b of (data.balances || [])) {
    balances[b.denom] = parseInt(b.amount);
  }
  return balances;
}

export async function getAccount(address) {
  const balances = await getBalance(address);
  return {
    address,
    balance: {
      qor: Math.floor((balances.uqor || 0) / 1_000_000),
      uqor: balances.uqor || 0,
    },
    pqcKeyRegistered: false, // TODO: query from indexer
  };
}

// ────────────────────── Network stats ──────────────────────

export async function getNetworkStats() {
  const [latest, validators] = await Promise.all([
    getLatestBlock(),
    fetchJSON(`${TESTNET_REST}/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED`),
  ]);
  return {
    blockHeight: latest.height,
    totalTransactions: 0, // TODO: query from indexer
    activeValidators: (validators.validators || []).length,
    tps: 0,
    avgBlockTime: 5.0, // 5s configured timeout_commit
  };
}

// ────────────────────── Faucet (send tokens) ──────────────────────

export async function sendTokens(fromMnemonic, toAddress, amountUqor) {
  // For MVP: use the node's tx broadcast endpoint
  // This requires signing on the Lambda side — complex setup.
  // Alternative: call a faucet endpoint on the node directly.
  throw new Error('sendTokens requires CosmJS setup — see faucet-api integration');
}

// ────────────────────── Event parsing helpers ──────────────────────

function extractTxType(events) {
  for (const e of (events || [])) {
    if (e.type === 'message') {
      for (const a of e.attributes) {
        if (a.key === 'action') return a.value;
      }
    }
  }
  return 'unknown';
}

function extractSender(events) {
  for (const e of (events || [])) {
    if (e.type === 'transfer' || e.type === 'message') {
      for (const a of e.attributes) {
        if (a.key === 'sender') return a.value;
      }
    }
  }
  return '';
}

function extractReceiver(events) {
  for (const e of (events || [])) {
    if (e.type === 'transfer') {
      for (const a of e.attributes) {
        if (a.key === 'recipient') return a.value;
      }
    }
  }
  return '';
}

function extractAmount(events) {
  for (const e of (events || [])) {
    if (e.type === 'transfer') {
      for (const a of e.attributes) {
        if (a.key === 'amount') {
          const match = a.value.match(/^(\d+)/);
          return match ? parseInt(match[1]) : 0;
        }
      }
    }
  }
  return 0;
}

function extractFee(tx) {
  const fees = tx?.auth_info?.fee?.amount || [];
  if (fees.length > 0) return parseInt(fees[0].amount) / 1_000_000;
  return 0;
}

function hasPQCEvent(events) {
  return (events || []).some(e => e.type === 'pqc_verify');
}

function extractAIScore(events) {
  for (const e of (events || [])) {
    if (e.type === 'ai_anomaly') {
      for (const a of e.attributes) {
        if (a.key === 'score') return parseFloat(a.value);
      }
    }
  }
  return 0;
}
