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
  if (typeof value === 'string') return value;
  const units = ['H/s', 'KH/s', 'MH/s', 'GH/s', 'TH/s', 'PH/s'];
  let num = Number(value);
  if (!Number.isFinite(num)) return String(value);
  let index = 0;
  while (num >= 1000 && index < units.length - 1) {
    num /= 1000;
    index += 1;
  }
  return `${num.toFixed(num >= 100 ? 0 : num >= 10 ? 1 : 2)} ${units[index]}`;
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

function normalizeMiningcore(_pool, data, context) {
  const stats = data?.poolStats || data?.pool?.poolStats || data?.stats || data;
  const network = data?.networkStats || data?.pool?.networkStats || data?.network || {};
  return {
    miners: firstDefined(stats?.connectedMiners, stats?.miners, stats?.workers),
    poolHashrate: normalizeHashrate(firstDefined(stats?.poolHashRate, stats?.hashrate, stats?.poolHashrate)),
    networkHashrate: normalizeHashrate(firstDefined(network?.networkHashRate, network?.hashrate, network?.networkHashrate)),
    blockHeight: firstDefined(network?.blockHeight, stats?.blockHeight, data?.blockHeight),
    estimatedReward: firstDefined(stats?.estimatedReward, stats?.reward, data?.estimatedReward),
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

function normalizePoolObservation(pool, data, context = {}) {
  let normalized = {};
  if (data) {
    if (pool.adapter === 'miningcore-pool') normalized = normalizeMiningcore(pool, data, context);
    else if (pool.adapter === 'nomp-pool') normalized = normalizeNomp(pool, data, context);
    else if (pool.adapter === 'generic-json') normalized = normalizeGeneric(pool, data, context);
    else normalized = normalizeYiimp(pool, data, context);
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
    message: normalized.message || context.message || 'No live data available.'
  };
}

module.exports = { normalizePoolObservation, normalizeHashrate, pick };
