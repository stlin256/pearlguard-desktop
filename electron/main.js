const path = require('path');
const fs = require('fs');
const { app, BrowserWindow, ipcMain, dialog, shell } = require('electron');
const { normalizePoolObservation } = require('./poolAdapters');

const repoUrl = 'https://github.com/stlin256/pearlguard-desktop';
const rootDir = path.join(__dirname, '..');
const isSelfTest = process.argv.includes('--self-test') || process.argv.includes('self-test') || app.commandLine.hasSwitch('self-test');
const isDemoMode = process.argv.includes('--demo') || app.commandLine.hasSwitch('demo') || process.env.PEARLGUARD_DEMO === '1';
const forceFixturePools = process.env.PEARLGUARD_POOL_FIXTURE === '1';
const transferDisabled = process.env.PEARLGUARD_DISABLE_TRANSFER !== '0';
let transferRequests = 0;

function readJson(relativePath) {
  const filePath = path.join(rootDir, relativePath);
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function runtimeDir() {
  if (app.isPackaged) return path.dirname(process.execPath);
  return rootDir;
}

function getLocalPaths() {
  const userData = app.getPath('userData');
  return {
    userData,
    runtimeDir: runtimeDir(),
    walletConfig: path.join(userData, 'wallet.config.json'),
    walletConfigPortable: path.join(runtimeDir(), 'wallet.config.json'),
    poolsConfig: path.join(userData, 'pools.local.json'),
    poolsConfigPortable: path.join(runtimeDir(), 'pools.local.json'),
    stateFile: path.join(userData, 'pearlguard-state.json'),
    exampleWalletConfig: path.join(rootDir, 'data', 'wallet.config.example.json'),
    examplePoolsConfig: path.join(rootDir, 'data', 'pools.example.json')
  };
}

function readJsonIfExists(filePath) {
  if (!filePath || !fs.existsSync(filePath)) return null;
  return JSON.parse(fs.readFileSync(filePath, 'utf8'));
}

function firstExisting(paths) {
  return paths.find((filePath) => filePath && fs.existsSync(filePath));
}

function readWalletConfig() {
  const paths = getLocalPaths();
  const configPath = firstExisting([paths.walletConfig, paths.walletConfigPortable]);
  const config = readJsonIfExists(configPath) || {};
  return { config, configPath, configured: Boolean(configPath) };
}

function readPoolConfig() {
  const paths = getLocalPaths();
  const configPath = firstExisting([paths.poolsConfig, paths.poolsConfigPortable]);
  if (configPath) return { ...readJsonIfExists(configPath), configPath };
  return { ...readJson('data/pools.example.json'), configPath: null };
}

function defaultWallet(config = {}) {
  return {
    label: config.walletLabel || 'Local Pearl Wallet',
    network: config.network || 'mainnet',
    configured: false,
    connected: false,
    synced: false,
    blockHeight: null,
    bestPeerHeight: null,
    balancePRL: 0,
    reservePRL: Number(config.reservePRL ?? 0.02),
    thresholdPRL: Number(config.thresholdPRL ?? 1.1),
    mode: 'read-only'
  };
}

function emptyLocalState() {
  const walletConfig = readWalletConfig();
  return {
    source: walletConfig.configured ? 'wallet.config.json' : 'local',
    wallet: { ...defaultWallet(walletConfig.config), configured: walletConfig.configured },
    snapshots: [],
    addressEvents: [],
    auditEvents: []
  };
}

function readRuntimeState() {
  if (isDemoMode) return { ...readJson('data/demo-state.json'), source: 'demo' };
  const paths = getLocalPaths();
  const saved = readJsonIfExists(paths.stateFile);
  if (saved && saved.wallet) return saved;
  return emptyLocalState();
}

function writeRuntimeState(state) {
  const paths = getLocalPaths();
  fs.mkdirSync(path.dirname(paths.stateFile), { recursive: true });
  fs.writeFileSync(paths.stateFile, JSON.stringify(state, null, 2), 'utf8');
}

function nowIso() {
  return new Date().toISOString();
}

function parseCsv(text) {
  const rows = [];
  let row = [];
  let field = '';
  let quoted = false;
  for (let i = 0; i < text.length; i += 1) {
    const char = text[i];
    const next = text[i + 1];
    if (quoted) {
      if (char === '"' && next === '"') {
        field += '"';
        i += 1;
      } else if (char === '"') {
        quoted = false;
      } else {
        field += char;
      }
    } else if (char === '"') {
      quoted = true;
    } else if (char === ',') {
      row.push(field);
      field = '';
    } else if (char === '\n') {
      row.push(field.replace(/\r$/, ''));
      rows.push(row);
      row = [];
      field = '';
    } else {
      field += char;
    }
  }
  if (field.length || row.length) {
    row.push(field.replace(/\r$/, ''));
    rows.push(row);
  }
  const headers = rows.shift() || [];
  return rows.filter((item) => item.length > 1).map((item) => {
    const obj = {};
    headers.forEach((header, index) => { obj[header] = item[index] ?? ''; });
    return obj;
  });
}

function numberOrNull(value) {
  if (value === undefined || value === null || value === '') return null;
  const number = Number(value);
  return Number.isFinite(number) ? number : null;
}

function stateFromAuditRows(rows, filePath) {
  const walletConfig = readWalletConfig();
  const base = emptyLocalState();
  const snapshots = [];
  const addressEvents = [];
  const auditEvents = [];

  for (const row of rows) {
    const timestamp = row.timestamp || row.time || nowIso();
    const balance = numberOrNull(row.balancePRL);
    const amount = numberOrNull(row.amountPRL);
    const event = row.event || 'audit';
    const status = row.status || 'observed';
    auditEvents.push({
      timestamp,
      scope: event,
      event,
      status,
      severity: status === 'sent' ? 'info' : 'info',
      message: row.message || `${event} ${status}`
    });
    if (balance !== null) {
      snapshots.push({
        timestamp,
        balancePRL: balance,
        reservePRL: numberOrNull(row.reservePRL) ?? base.wallet.reservePRL,
        thresholdPRL: numberOrNull(row.minAmountPRL) ?? base.wallet.thresholdPRL,
        blockHeight: numberOrNull(row.blockHeight)
      });
    }
    if (amount !== null && amount !== 0) {
      addressEvents.push({
        timestamp,
        addressLabel: row.exchangeAddress ? 'Configured destination' : 'Local observation',
        direction: amount < 0 || status === 'sent' ? 'out' : 'in',
        amountPRL: Math.abs(amount),
        balanceAfterPRL: balance ?? 0,
        txid: row.txid || '',
        source: path.basename(filePath)
      });
    }
  }

  const lastSnapshot = snapshots[snapshots.length - 1];
  const state = {
    source: path.basename(filePath),
    wallet: {
      ...base.wallet,
      configured: walletConfig.configured,
      balancePRL: lastSnapshot ? lastSnapshot.balancePRL : base.wallet.balancePRL,
      reservePRL: lastSnapshot ? lastSnapshot.reservePRL : base.wallet.reservePRL,
      thresholdPRL: lastSnapshot ? lastSnapshot.thresholdPRL : base.wallet.thresholdPRL,
      blockHeight: lastSnapshot ? lastSnapshot.blockHeight : base.wallet.blockHeight,
      synced: Boolean(lastSnapshot && lastSnapshot.blockHeight)
    },
    snapshots,
    addressEvents,
    auditEvents
  };
  writeRuntimeState(state);
  return state;
}

async function importAuditCsv() {
  const result = await dialog.showOpenDialog({
    title: 'Import PearlGuard audit CSV',
    properties: ['openFile'],
    filters: [{ name: 'CSV files', extensions: ['csv'] }]
  });
  if (result.canceled || !result.filePaths.length) return { canceled: true };
  const filePath = result.filePaths[0];
  const rows = parseCsv(fs.readFileSync(filePath, 'utf8'));
  const state = stateFromAuditRows(rows, filePath);
  return { canceled: false, importedRows: rows.length, state };
}

async function fetchJson(endpoint, timeoutMs = 6500) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  const started = Date.now();
  try {
    const response = await fetch(endpoint, {
      signal: controller.signal,
      headers: { accept: 'application/json', 'user-agent': 'PearlGuard-Desktop/0.2.0' }
    });
    const text = await response.text();
    let data;
    try {
      data = JSON.parse(text);
    } catch (error) {
      throw new Error(`Endpoint did not return JSON (${response.status})`);
    }
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return { data, latencyMs: Date.now() - started };
  } finally {
    clearTimeout(timer);
  }
}

async function jsonRpc(config, method, params = []) {
  const host = config.rpcHost || '127.0.0.1';
  const port = Number(config.rpcPort || 8335);
  const endpoint = config.rpcUrl || `http://${host}:${port}`;
  const headers = { 'content-type': 'application/json' };
  if (config.rpcUsername || config.rpcPassword) {
    headers.authorization = 'Basic ' + Buffer.from(`${config.rpcUsername || ''}:${config.rpcPassword || ''}`).toString('base64');
  }
  const response = await fetch(endpoint, {
    method: 'POST',
    headers,
    body: JSON.stringify({ jsonrpc: '1.0', id: 'pearlguard', method, params })
  });
  const payload = await response.json();
  if (!response.ok || payload.error) {
    throw new Error(payload.error?.message || `RPC ${method} failed with HTTP ${response.status}`);
  }
  return payload.result;
}

async function readWalletStatus() {
  const { config, configured, configPath } = readWalletConfig();
  const state = readRuntimeState();
  if (!configured) {
    return { ok: false, configured: false, configPath: getLocalPaths().walletConfig, message: 'wallet.config.json was not found.' };
  }
  if (!config.rpcUrl && !config.rpcPort) {
    return { ok: false, configured: true, configPath, message: 'RPC settings are missing in wallet.config.json.' };
  }

  try {
    const [blockchain, balance] = await Promise.allSettled([
      jsonRpc(config, 'getblockchaininfo'),
      jsonRpc(config, 'getbalance')
    ]);
    const block = blockchain.status === 'fulfilled' ? blockchain.value : {};
    const balancePRL = balance.status === 'fulfilled' ? Number(balance.value || 0) : state.wallet.balancePRL;
    const wallet = {
      ...state.wallet,
      label: config.walletLabel || state.wallet.label,
      network: config.network || block.chain || state.wallet.network,
      configured: true,
      connected: true,
      synced: block.blocks !== undefined && block.headers !== undefined ? block.blocks >= block.headers : state.wallet.synced,
      blockHeight: block.blocks ?? state.wallet.blockHeight,
      bestPeerHeight: block.headers ?? state.wallet.bestPeerHeight,
      balancePRL,
      reservePRL: Number(config.reservePRL ?? state.wallet.reservePRL),
      thresholdPRL: Number(config.thresholdPRL ?? state.wallet.thresholdPRL),
      mode: 'read-only'
    };
    const snapshot = {
      timestamp: nowIso(),
      balancePRL: wallet.balancePRL,
      reservePRL: wallet.reservePRL,
      thresholdPRL: wallet.thresholdPRL,
      blockHeight: wallet.blockHeight
    };
    const nextState = {
      ...state,
      source: 'wallet-rpc',
      wallet,
      snapshots: [...(state.snapshots || []), snapshot],
      auditEvents: [...(state.auditEvents || []), {
        timestamp: snapshot.timestamp,
        scope: 'wallet',
        event: 'read-status',
        status: 'ok',
        severity: 'info',
        message: 'Read-only wallet status refreshed.'
      }]
    };
    writeRuntimeState(nextState);
    return { ok: true, state: nextState, configPath };
  } catch (error) {
    return { ok: false, configured: true, configPath, message: error.message || 'Read-only wallet status failed.' };
  }
}

async function syncPools(options = {}) {
  const fixtureOnly = options.fixtureOnly || forceFixturePools || isDemoMode;
  const timestamp = nowIso();
  if (fixtureOnly) {
    const fixture = readJson('data/pool-fixture.json');
    return { timestamp, mode: 'demo', observations: fixture.observations.map((item) => ({ ...item, timestamp })), errors: [] };
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
        message: 'Pool is in the local catalog but disabled or missing an endpoint.'
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

  return { timestamp, mode: 'local', observations, errors, configPath: config.configPath };
}

function dryRunSweepCheck(input = {}) {
  const balancePRL = Number(input.balancePRL || 0);
  const reservePRL = Number(input.reservePRL || 0);
  const thresholdPRL = Number(input.thresholdPRL || 1.1);
  const sweepablePRL = Number((balancePRL - reservePRL).toFixed(8));
  const thresholdReached = sweepablePRL > thresholdPRL;
  return {
    mode: 'read-only-check',
    transferDisabled,
    transferRequests,
    balancePRL,
    reservePRL,
    thresholdPRL,
    sweepablePRL,
    thresholdReached,
    decision: thresholdReached ? 'would-plan-sweep' : 'hold',
    message: thresholdReached
      ? 'Read-only check: threshold is reached, but this desktop build will not broadcast a transaction.'
      : 'Read-only check: threshold is not reached.'
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
        const ok = Boolean(result && result.ok && result.transferRequests === 0 && result.mode === 'local');
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
    mode: isDemoMode ? 'demo' : 'local',
    transferDisabled,
    paths: getLocalPaths(),
    state: readRuntimeState(),
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
ipcMain.handle('wallet:read-status', async () => readWalletStatus());
ipcMain.handle('audit:import-csv', async () => importAuditCsv());
ipcMain.handle('app:load-demo-state', async () => ({ state: { ...readJson('data/demo-state.json'), source: 'demo' } }));
ipcMain.handle('app:open-config-folder', async () => shell.openPath(app.getPath('userData')));

app.whenReady().then(() => {
  createWindow();
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow();
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') app.quit();
});
