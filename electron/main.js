const path = require('path');
const fs = require('fs');
const { app, BrowserWindow, ipcMain } = require('electron');
const { normalizePoolObservation } = require('./poolAdapters');

const repoUrl = 'https://github.com/stlin256/pearlguard-desktop';
const rootDir = path.join(__dirname, '..');
const isSelfTest = process.argv.includes('--self-test') || process.argv.includes('self-test') || app.commandLine.hasSwitch('self-test');
const forceFixturePools = process.env.PEARLGUARD_POOL_FIXTURE === '1';
const transferDisabled = process.env.PEARLGUARD_DISABLE_TRANSFER !== '0';
let transferRequests = 0;

function readJson(relativePath) {
  const filePath = path.join(rootDir, relativePath);
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function readPoolConfig() {
  const localPath = path.join(app.getPath('userData'), 'pools.local.json');
  if (fs.existsSync(localPath)) {
    return JSON.parse(fs.readFileSync(localPath, 'utf8'));
  }
  return readJson('data/pools.example.json');
}

function nowIso() {
  return new Date().toISOString();
}

async function fetchJson(endpoint, timeoutMs = 6500) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  const started = Date.now();
  try {
    const response = await fetch(endpoint, {
      signal: controller.signal,
      headers: { 'accept': 'application/json', 'user-agent': 'PearlGuard-Desktop/0.1.0' }
    });
    const text = await response.text();
    let data;
    try {
      data = JSON.parse(text);
    } catch (error) {
      throw new Error(`Endpoint did not return JSON (${response.status})`);
    }
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    return { data, latencyMs: Date.now() - started };
  } finally {
    clearTimeout(timer);
  }
}

async function syncPools(options = {}) {
  const fixtureOnly = options.fixtureOnly || forceFixturePools;
  const timestamp = nowIso();
  if (fixtureOnly) {
    const fixture = readJson('data/pool-fixture.json');
    return {
      timestamp,
      mode: 'fixture',
      observations: fixture.observations.map((item) => ({ ...item, timestamp })),
      errors: []
    };
  }

  const config = readPoolConfig();
  const pools = Array.isArray(config.pools) ? config.pools : [];
  const observations = [];
  const errors = [];

  for (const pool of pools) {
    if (!pool.enabled || !pool.endpoint) {
      observations.push(normalizePoolObservation(pool, null, {
        timestamp,
        reachable: false,
        message: 'Pool is present in the catalog but disabled or missing an endpoint.'
      }));
      continue;
    }

    try {
      const { data, latencyMs } = await fetchJson(pool.endpoint);
      observations.push(normalizePoolObservation(pool, data, { timestamp, reachable: true, latencyMs }));
    } catch (error) {
      const message = error && error.message ? error.message : 'Pool sync failed.';
      errors.push({ poolId: pool.id, message });
      observations.push(normalizePoolObservation(pool, null, { timestamp, reachable: false, message }));
    }
  }

  return { timestamp, mode: 'network', observations, errors };
}

function dryRunSweepCheck(input = {}) {
  const balancePRL = Number(input.balancePRL || 0);
  const reservePRL = Number(input.reservePRL || 0);
  const thresholdPRL = Number(input.thresholdPRL || 1.1);
  const sweepablePRL = Number((balancePRL - reservePRL).toFixed(8));
  const thresholdReached = sweepablePRL > thresholdPRL;
  return {
    mode: 'dry-run',
    transferDisabled,
    transferRequests,
    balancePRL,
    reservePRL,
    thresholdPRL,
    sweepablePRL,
    thresholdReached,
    decision: thresholdReached ? 'would-plan-sweep' : 'hold',
    message: thresholdReached
      ? 'Dry-run only: threshold is reached, but no transaction will be broadcast.'
      : 'Dry-run only: threshold is not reached.'
  };
}

function createWindow() {
  const win = new BrowserWindow({
    width: 1260,
    height: 840,
    minWidth: 980,
    minHeight: 680,
    backgroundColor: '#f7fbfb',
    title: 'PearlGuard Desktop',
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true
    }
  });

  win.loadFile(path.join(rootDir, 'src', 'index.html'));

  if (isSelfTest) {
    const selfTestTimer = setTimeout(() => {
      console.error('SELF_TEST_FAIL timeout waiting for renderer');
      app.exit(1);
    }, 20000);
    win.webContents.once('did-fail-load', (_event, code, description) => {
      clearTimeout(selfTestTimer);
      console.error('SELF_TEST_FAIL load failed ' + code + ': ' + description);
      app.exit(1);
    });
    win.webContents.once('did-finish-load', async () => {
      try {
        const result = await win.webContents.executeJavaScript('window.__pearlguardSelfTest && window.__pearlguardSelfTest()', true);
        clearTimeout(selfTestTimer);
        const ok = Boolean(result && result.ok && result.transferRequests === 0);
        console.log(`${ok ? 'SELF_TEST_PASS' : 'SELF_TEST_FAIL'} ${JSON.stringify(result)}`);
        app.exit(ok ? 0 : 1);
      } catch (error) {
        clearTimeout(selfTestTimer);
        console.error(`SELF_TEST_FAIL ${error && error.stack ? error.stack : error}`);
        app.exit(1);
      }
    });
  }

  return win;
}

ipcMain.handle('app:get-bootstrap', async () => {
  return {
    name: 'PearlGuard Desktop',
    version: app.getVersion(),
    repoUrl,
    platform: process.platform,
    locale: app.getLocale(),
    transferDisabled,
    demoState: readJson('data/demo-state.json'),
    poolConfig: readPoolConfig()
  };
});


ipcMain.handle('i18n:get-messages', async (_event, locale) => {
  const allowed = new Set(['en', 'ar', 'zh-CN', 'fr', 'ru', 'es']);
  const safeLocale = allowed.has(locale) ? locale : 'en';
  return readJson(`src/locales/${safeLocale}.json`);
});
ipcMain.handle('pools:sync', async (_event, options) => syncPools(options || {}));
ipcMain.handle('wallet:dry-run-sweep-check', async (_event, input) => dryRunSweepCheck(input || {}));
ipcMain.handle('audit:append-demo', async (_event, entry) => ({ saved: false, mode: 'demo', entry: { ...entry, timestamp: nowIso() } }));

app.whenReady().then(() => {
  createWindow();
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});
