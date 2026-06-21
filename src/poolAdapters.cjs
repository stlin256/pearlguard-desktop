function pick(source, dottedPath) {
  if (!source || !dottedPath) return undefined;
  return String(dottedPath).split('.').reduce((value, key) => {
    if (value && Object.prototype.hasOwnProperty.call(value, key)) return value[key];
    return undefined;
  }, source);
}

function firstDefined(...values) {
  return values.find((value) => value !== undefined && value !== null && value !== '');
}

function normalizeHashrate(value) {
  if (value === undefined || value === null || value === '') return null;
  if (typeof value === 'string' && /[a-z/]/i.test(value)) return value;
  const units = ['H/s', 'KH/s', 'MH/s', 'GH/s', 'TH/s', 'PH/s', 'EH/s'];
  let num = Number(value);
  if (!Number.isFinite(num)) return String(value);
  let index = 0;
  while (num >= 1000 && index < units.length - 1) {
    num /= 1000;
    index += 1;
  }
  return `${num.toFixed(num >= 100 ? 0 : num >= 10 ? 1 : 2)} ${units[index]}`;
}

function formatPercent(value, multiplier = 1) {
  if (value === undefined || value === null || value === '') return '';
  if (typeof value === 'string' && value.includes('%')) return value;
  const num = Number(value);
  if (!Number.isFinite(num)) return String(value);
  return `${Number((num * multiplier).toFixed(2)).toString()}%`;
}

function coinNode(data, symbol) {
  if (!data) return null;
  const keys = [symbol, String(symbol || '').toUpperCase(), String(symbol || '').toLowerCase(), 'PRL', 'pearl'];
  for (const key of keys) {
    if (key && data[key]) return data[key];
  }
  return data;
}

function normalizeYiimp(pool, data, context) {
  const node = coinNode(data, pool.coinSymbol);
  return {
    miners: firstDefined(node?.workers, node?.miners, node?.workerCount),
    poolHashrate: normalizeHashrate(firstDefined(node?.hashrate, node?.hashrate_shared, node?.pool_hashrate)),
    networkHashrate: normalizeHashrate(firstDefined(node?.network_hashrate, node?.networkHashrate, node?.nethash)),
    blockHeight: firstDefined(node?.height, node?.lastblock, node?.blockHeight),
    estimatedReward: firstDefined(node?.estimate_current, node?.estimate, node?.reward),
    message: context.message || 'Yiimp-compatible pool response normalized.'
  };
}

function normalizeMiningcore(pool, data, context) {
  const poolNode = data?.pool || {};
  const stats = data?.poolStats || poolNode.poolStats || data?.stats || data;
  const network = data?.networkStats || poolNode.networkStats || data?.network || {};
  return {
    miners: firstDefined(stats?.connectedMiners, stats?.miners, stats?.workers, stats?.connectedWorkers),
    poolHashrate: normalizeHashrate(firstDefined(stats?.poolHashRate, stats?.poolHashrate, stats?.hashrate)),
    networkHashrate: normalizeHashrate(firstDefined(network?.networkHashRate, network?.networkHashrate, network?.hashrate)),
    blockHeight: firstDefined(network?.blockHeight, stats?.blockHeight, data?.blockHeight),
    estimatedReward: firstDefined(stats?.estimatedReward, stats?.reward, data?.estimatedReward),
    fee: formatPercent(firstDefined(poolNode.poolFeePercent, data?.poolFeePercent), 1),
    payout: pool.rewardMode || '',
    message: context.message || 'Miningcore-compatible pool response normalized.'
  };
}

function normalizeNomp(_pool, data, context) {
  const stats = data?.pool_stats || data?.poolStats || data?.stats || data;
  return {
    miners: firstDefined(stats?.workers, stats?.miners, data?.workers),
    poolHashrate: normalizeHashrate(firstDefined(stats?.hashrate, stats?.poolHashrate, data?.hashrate)),
    networkHashrate: normalizeHashrate(firstDefined(stats?.networkHashrate, data?.networkHashrate, data?.nethash)),
    blockHeight: firstDefined(stats?.height, data?.height, data?.blockHeight),
    estimatedReward: firstDefined(stats?.reward, data?.reward, data?.estimate),
    message: context.message || 'NOMP-compatible pool response normalized.'
  };
}

function normalizeGeneric(pool, data, context) {
  const mapping = pool.mapping || {};
  return {
    miners: firstDefined(pick(data, mapping.miners), data?.miners, data?.workers),
    poolHashrate: normalizeHashrate(firstDefined(pick(data, mapping.poolHashrate), data?.poolHashrate, data?.hashrate)),
    networkHashrate: normalizeHashrate(firstDefined(pick(data, mapping.networkHashrate), data?.networkHashrate, data?.nethash)),
    blockHeight: firstDefined(pick(data, mapping.blockHeight), data?.blockHeight, data?.height),
    estimatedReward: firstDefined(pick(data, mapping.estimatedReward), data?.estimatedReward, data?.reward),
    message: context.message || 'Generic JSON pool response normalized.'
  };
}

function normalizeKnownPool(pool, data, context) {
  if (pool.adapter === 'alphapool-prl') {
    const coin = data?.coins?.[0] || {};
    return { miners: firstDefined(data?.pool?.miners24h, data?.pool?.miners, data?.pool?.workers), poolHashrate: firstDefined(data?.pool?.hashrate, data?.pool?.hashrate1h), networkHashrate: firstDefined(coin.network_hash, coin.networkHashrate), blockHeight: firstDefined(data?.chain?.height, coin.block_height), estimatedReward: firstDefined(coin.ttfLabel, data?.pool?.ttfLabel), fee: formatPercent(firstDefined(data?.feePercent, 1)), payout: pool.rewardMode || 'PPLNS', message: context.message || 'Pool stats endpoint normalized.' };
  }
  if (pool.adapter === 'akoyapool-prl') {
    const stats = data?.data || {};
    return { miners: firstDefined(stats.connected_miners, stats.registered_miners, stats.active_workers), poolHashrate: normalizeHashrate(stats.total_hashrate), networkHashrate: normalizeHashrate(stats.network_hashrate), blockHeight: stats.current_block_height, estimatedReward: firstDefined(stats.expected_block_time_seconds, stats.total_paid24_h), fee: formatPercent(stats.pool_fee_percent), payout: pool.rewardMode || 'PPLTS', message: context.message || 'Akoya public pool stats endpoint normalized.' };
  }
  if (pool.adapter === 'nushypool-v2') {
    const mode = String(pool.rewardMode || '').toUpperCase();
    const node = (data?.result?.pools || []).find((item) => String(item.ticker).toUpperCase() === String(pool.coinSymbol || 'PRL').toUpperCase() && (!mode || String(item.payoutSystem).toUpperCase() === mode)) || {};
    const height = String(node.networkBlock || '').startsWith('0x') ? parseInt(node.networkBlock, 16) : node.networkBlock;
    return { miners: firstDefined(node.activeMiners, node.activeWorkers), poolHashrate: normalizeHashrate(node.hashrate?.total), blockHeight: height, estimatedReward: firstDefined(node.dailyRewardPerHashrateUnit, node.baseBlockReward), fee: formatPercent(node.poolFee), payout: firstDefined(node.payoutSystem, pool.rewardMode, 'POOL'), message: context.message || 'NushyPool V2 public pool stats endpoint normalized.' };
  }
  return null;
}

function normalizePoolObservation(pool, data, context = {}) {
  let normalized = {};
  if (data) {
    normalized = normalizeKnownPool(pool, data, context) || {};
    if (!Object.keys(normalized).length) {
      if (pool.adapter === 'miningcore-pool' || pool.adapter === 'himpool-miningcore') normalized = normalizeMiningcore(pool, data, context);
      else if (pool.adapter === 'nomp-pool') normalized = normalizeNomp(pool, data, context);
      else if (pool.adapter === 'generic-json') normalized = normalizeGeneric(pool, data, context);
      else normalized = normalizeYiimp(pool, data, context);
    }
  }

  return {
    timestamp: context.timestamp || new Date().toISOString(),
    poolId: pool.id,
    poolName: pool.name,
    adapter: pool.adapter,
    reachable: Boolean(context.reachable && data),
    miners: normalized.miners ?? null,
    poolHashrate: normalized.poolHashrate ?? null,
    networkHashrate: normalized.networkHashrate ?? null,
    blockHeight: normalized.blockHeight ?? null,
    estimatedReward: normalized.estimatedReward ?? null,
    latencyMs: context.latencyMs ?? null,
    homepage: pool.homepage || '',
    fee: normalized.fee || '',
    payout: normalized.payout || '',
    share: normalized.share || '',
    message: normalized.message || context.message || 'No live data available.'
  };
}

module.exports = { normalizePoolObservation, normalizeHashrate, pick, formatPercent };