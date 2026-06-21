import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);
const { normalizePoolObservation, normalizeHashrate } = require('../src/poolAdapters.cjs');
const root = process.cwd();
const localesDir = path.join(root, 'src', 'locales');
const localeFiles = ['en.json', 'ar.json', 'zh-CN.json', 'fr.json', 'ru.json', 'es.json'];
const localeMaps = Object.fromEntries(localeFiles.map((file) => [file, JSON.parse(fs.readFileSync(path.join(localesDir, file), 'utf8'))]));
const enKeys = Object.keys(localeMaps['en.json']).sort();

for (const [file, map] of Object.entries(localeMaps)) {
  assert.deepEqual(Object.keys(map).sort(), enKeys, `${file} must contain exactly the English locale keys`);
}

assert.equal(normalizeHashrate(2420000000000), '2.42 TH/s');
const miningcore = normalizePoolObservation(
  { id: 'pool', name: 'Pool', adapter: 'miningcore-pool', coinSymbol: 'PRL' },
  { poolStats: { connectedMiners: 7, poolHashRate: 4200000000 }, networkStats: { networkHashRate: 81000000000, blockHeight: 100 } },
  { reachable: true, latencyMs: 12 }
);
assert.equal(miningcore.reachable, true);
assert.equal(miningcore.miners, 7);
assert.equal(miningcore.blockHeight, 100);


const alpha = normalizePoolObservation(
  { id: 'alpha', name: 'Alpha', adapter: 'alphapool-prl', coinSymbol: 'PRL', rewardMode: 'PPLNS' },
  { feePercent: 1, chain: { height: 10 }, coins: [{ network_hash: '30 EH/s', ttfLabel: '1h' }], pool: { miners24h: 2, hashrate: '1 EH/s' } },
  { reachable: true, latencyMs: 8 }
);
assert.equal(alpha.reachable, true);
assert.equal(alpha.fee, '1%');
assert.equal(alpha.payout, 'PPLNS');
assert.equal(alpha.blockHeight, 10);

const akoya = normalizePoolObservation(
  { id: 'akoya', name: 'Akoya', adapter: 'akoyapool-prl', coinSymbol: 'PRL', rewardMode: 'PPLTS' },
  { data: { connected_miners: 3, total_hashrate: 220000000000000000, network_hashrate: 31000000000000000000, current_block_height: 20, pool_fee_percent: 2 } },
  { reachable: true }
);
assert.equal(akoya.miners, 3);
assert.equal(akoya.fee, '2%');
assert.equal(akoya.poolHashrate, '220 PH/s');

const nushy = normalizePoolObservation(
  { id: 'nushy', name: 'Nushy', adapter: 'nushypool-v2', coinSymbol: 'PRL', rewardMode: 'FPPS' },
  { result: { pools: [{ ticker: 'PRL', payoutSystem: 'FPPS', poolFee: '1.0', activeMiners: 8, hashrate: { total: 2259030183537978 }, networkBlock: '0x129c7' }] } },
  { reachable: true }
);
assert.equal(nushy.payout, 'FPPS');
assert.equal(nushy.blockHeight, 76231);
const demo = JSON.parse(fs.readFileSync(path.join(root, 'data', 'demo-state.json'), 'utf8'));
assert.equal(demo.wallet.dryRun, true);
assert.ok(demo.snapshots.length >= 6);
assert.ok(demo.addressEvents.every((event) => String(event.txid).startsWith('tx_demo_')));

console.log(`unit tests ok: ${localeFiles.length} locale packs, pool adapters, and demo safety checks passed`);
