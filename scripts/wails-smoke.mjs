import fs from 'node:fs';
import path from 'node:path';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';

const root = process.cwd();
execFileSync(process.execPath, ['scripts/prepare-wails-assets.mjs'], { stdio: 'inherit' });
execFileSync('go', ['test', './...'], { stdio: 'inherit' });

const dist = path.join(root, 'frontend', 'dist');
const renderer = fs.readFileSync(path.join(dist, 'renderer.js'), 'utf8');
const bridge = fs.readFileSync(path.join(dist, 'platform-wails.js'), 'utf8');
const index = fs.readFileSync(path.join(dist, 'index.html'), 'utf8');

assert.ok(renderer.includes('saveSettings'), 'renderer must expose GUI settings save flow');
assert.ok(renderer.includes('toggleMonitor'), 'renderer must expose continuous monitor flow');
assert.ok(bridge.includes('window.go.main.App'), 'Wails bridge must call the bound Go runtime');
assert.ok(index.includes('platform-wails.js'), 'Wails build must load the runtime bridge');
assert.ok(!renderer.includes(['normal', 'local', 'mode'].join(' ')), 'UI copy must not contain obsolete setup wording');
assert.ok(!renderer.includes(['Load', 'demo'].join(' ')), 'primary UI must not expose sample-data controls');

console.log('wails smoke ok: assets generated, Go backend compiled, GUI settings and monitor flow present');
