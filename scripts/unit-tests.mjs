import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';

const require = createRequire(import.meta.url);
const { normalizePoolObservation, normalizeHashrate } = require('../electron/poolAdapters.js');
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

const demo = JSON.parse(fs.readFileSync(path.join(root, 'data', 'demo-state.json'), 'utf8'));
assert.equal(demo.wallet.dryRun, true);
assert.ok(demo.snapshots.length >= 6);
assert.ok(demo.addressEvents.every((event) => String(event.txid).startsWith('tx_demo_')));

console.log(`unit tests ok: ${localeFiles.length} locale packs, pool adapters, and demo safety checks passed`);
