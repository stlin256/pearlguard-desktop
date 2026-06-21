const supportedLocales = ['en', 'ar', 'zh-CN', 'fr', 'ru', 'es'];
const localeNames = { en: 'English', ar: 'العربية', 'zh-CN': '简体中文', fr: 'Français', ru: 'Русский', es: 'Español' };
const adapters = ['zpool-status', 'yiimp-status', 'miningcore-pool', 'nomp-pool', 'generic-json'];

const appState = {
  activePage: 'dashboard',
  locale: 'en',
  messages: {},
  bootstrap: null,
  settings: {},
  poolConfig: { version: 1, pollSeconds: 120, pools: [] },
  state: null,
  poolSync: null,
  dryRun: null,
  historyFilter: '',
  transferRequests: 0,
  notice: '',
  busy: false,
  monitorTimer: null
};

const navItems = [
  ['dashboard', 'nav.dashboard', '◇'],
  ['monitor', 'nav.monitor', '◉'],
  ['history', 'nav.history', '↕'],
  ['curves', 'nav.curves', '⌁'],
  ['pools', 'nav.pools', '⌬'],
  ['settings', 'nav.settings', '⚙']
];

function escapeHtml(value) {
  return String(value ?? '').replace(/[&<>'"]/g, (char) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' })[char]);
}
function chooseLocale(systemLocale, navigatorLocale, configuredLocale) {
  const stored = localStorage.getItem('pearlguard.locale');
  if (supportedLocales.includes(configuredLocale)) return configuredLocale;
  if (supportedLocales.includes(stored)) return stored;
  const raw = `${systemLocale || navigatorLocale || 'en'}`.toLowerCase();
  if (raw.startsWith('ar')) return 'ar';
  if (raw.startsWith('zh')) return 'zh-CN';
  if (raw.startsWith('fr')) return 'fr';
  if (raw.startsWith('ru')) return 'ru';
  if (raw.startsWith('es')) return 'es';
  return 'en';
}
async function loadMessages(locale) { return window.pearlGuard.getMessages(locale); }
function t(key, params = {}) { return Object.entries(params).reduce((text, [name, value]) => text.replaceAll(`{${name}}`, value), appState.messages[key] || key); }
function fmtPrl(value) { return `${Number(value || 0).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 8 })} PRL`; }
function fmtTime(value) { if (!value) return '—'; const date = new Date(value); return Number.isNaN(date.getTime()) ? value : date.toLocaleString(appState.locale, { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit' }); }
function latest(items) { return items[items.length - 1]; }
function getState() { return appState.state || { wallet: {}, snapshots: [], addressEvents: [], auditEvents: [] }; }
function getSnapshots() { return getState().snapshots || []; }
function getWallet() { return getState().wallet || {}; }
function getAuditEvents() { return getState().auditEvents || []; }
function getPoolObservations() { return appState.poolSync?.observations || []; }
function hasRealData() { return Boolean(getWallet().configured || getSnapshots().length || getAuditEvents().length); }
function statusPill(label, tone = 'neutral') { return `<span class="status-pill ${tone}">${escapeHtml(label)}</span>`; }
function metricCard(label, value, detail, tone = '') { return `<article class="metric-card ${tone}"><span>${escapeHtml(label)}</span><strong>${escapeHtml(value)}</strong><small>${escapeHtml(detail || '')}</small></article>`; }
function notice(message) { appState.notice = message; render(); }
function configuredPools() { return appState.poolConfig?.pools || []; }

function getSyncGap(wallet) {
  if (wallet.blockHeight == null || wallet.bestPeerHeight == null) return null;
  return Math.max(0, Number(wallet.bestPeerHeight) - Number(wallet.blockHeight));
}
function decisionTone(dry) {
  if (!dry) return 'neutral';
  return dry.thresholdReached ? 'warn' : 'safe';
}
function nextAction() {
  const wallet = getWallet();
  if (!wallet.configured) return t('dashboard.nextConfigure');
  if (!wallet.connected) return t('dashboard.nextRefresh');
  if (!wallet.synced) return t('dashboard.nextSync');
  if (appState.dryRun?.thresholdReached) return t('dashboard.nextReview');
  return t('dashboard.nextObserve');
}

function chartSvg(points, options = {}) {
  const width = 820;
  const height = options.height || 250;
  const pad = 28;
  if (!points.length) return `<div class="empty-chart">${escapeHtml(t('chart.empty'))}</div>`;
  const values = points.map((p) => Number(p.balancePRL || 0));
  const maxValue = Math.max(...values, Number(options.threshold || 0), 1.2);
  const minValue = Math.min(0, ...values);
  const range = Math.max(0.0001, maxValue - minValue);
  const x = (index) => pad + (index / Math.max(1, points.length - 1)) * (width - pad * 2);
  const y = (value) => height - pad - ((Number(value) - minValue) / range) * (height - pad * 2);
  const path = points.map((point, index) => `${index === 0 ? 'M' : 'L'} ${x(index).toFixed(2)} ${y(point.balancePRL).toFixed(2)}`).join(' ');
  const dots = points.map((point, index) => `<circle cx="${x(index).toFixed(2)}" cy="${y(point.balancePRL).toFixed(2)}" r="5" class="chart-dot"><title>${escapeHtml(fmtTime(point.timestamp))}: ${escapeHtml(fmtPrl(point.balancePRL))}</title></circle>`).join('');
  const thresholdY = y(options.threshold || 1.1).toFixed(2);
  const reserveY = y(options.reserve || 0.02).toFixed(2);
  return `<svg class="chart" viewBox="0 0 ${width} ${height}" role="img" aria-label="${escapeHtml(t('chart.balanceCurve'))}">
    <line x1="${pad}" x2="${width - pad}" y1="${height - pad}" y2="${height - pad}" class="chart-axis" />
    <line x1="${pad}" x2="${pad}" y1="${pad}" y2="${height - pad}" class="chart-axis" />
    <line x1="${pad}" x2="${width - pad}" y1="${thresholdY}" y2="${thresholdY}" class="chart-threshold" />
    <line x1="${pad}" x2="${width - pad}" y1="${reserveY}" y2="${reserveY}" class="chart-reserve" />
    <path d="${path}" class="chart-line" />${dots}
    <text x="${width - pad - 120}" y="${Number(thresholdY) - 8}" class="chart-label">${escapeHtml(t('chart.threshold'))}</text>
    <text x="${width - pad - 100}" y="${Number(reserveY) - 8}" class="chart-label muted">${escapeHtml(t('chart.reserve'))}</text>
  </svg>`;
}

function renderShell(pageHtml) {
  const direction = appState.locale === 'ar' ? 'rtl' : 'ltr';
  document.documentElement.lang = appState.locale;
  document.documentElement.dir = direction;
  const nav = navItems.map(([id, key, icon]) => `<button class="nav-button ${appState.activePage === id ? 'active' : ''}" data-page="${id}" title="${escapeHtml(t(key))}"><span class="nav-icon">${icon}</span><span>${escapeHtml(t(key))}</span></button>`).join('');
  document.getElementById('app').className = 'app-shell';
  document.getElementById('app').innerHTML = `<aside class="sidebar">
    <div class="brand-lockup"><img src="assets/app-icon.svg" alt="" /><div><strong>PearlGuard</strong><small>${escapeHtml(t('app.subtitle'))}</small></div></div>
    <nav class="nav-list" aria-label="${escapeHtml(t('nav.label'))}">${nav}</nav>
    <div class="side-status">${statusPill(getWallet().configured ? t('status.configured') : t('status.setupRequired'), getWallet().configured ? 'safe' : 'warn')}<span>${escapeHtml(nextAction())}</span></div>
  </aside>
  <section class="workspace">
    <header class="topbar"><div><p class="eyebrow">${escapeHtml(t('app.eyebrow'))}</p><h1>${escapeHtml(t(`page.${appState.activePage}`))}</h1></div>
      <div class="top-actions"><button class="ghost-button" id="refreshWalletTop">${escapeHtml(t('action.refreshWallet'))}</button><button class="ghost-button" id="syncPoolsTop">${escapeHtml(t('action.syncPools'))}</button><button class="primary-button" id="openSettingsTop">${escapeHtml(t('action.configure'))}</button></div></header>
    <main class="page-view">${appState.notice ? `<section class="notice">${escapeHtml(appState.notice)}</section>` : ''}${pageHtml}</main>
  </section>`;
  document.querySelectorAll('[data-page]').forEach((button) => button.addEventListener('click', () => { appState.activePage = button.dataset.page; render(); }));
  document.getElementById('syncPoolsTop').addEventListener('click', syncPoolsAndRender);
  document.getElementById('refreshWalletTop').addEventListener('click', refreshWallet);
  document.getElementById('openSettingsTop').addEventListener('click', () => { appState.activePage = 'settings'; render(); });
}

function setupPanel() {
  if (hasRealData()) return '';
  return `<section class="panel action-panel setup-panel"><div><h2>${escapeHtml(t('dashboard.setupTitle'))}</h2><p>${escapeHtml(t('dashboard.setupCopy'))}</p></div><div class="button-row"><button class="primary-button" id="goSettingsTop">${escapeHtml(t('action.configure'))}</button><button class="ghost-button" id="importCsvTop">${escapeHtml(t('action.importCsv'))}</button></div></section>`;
}

function renderDashboard() {
  const wallet = getWallet();
  const snapshots = getSnapshots();
  const last = latest(snapshots) || wallet;
  const pools = getPoolObservations();
  const reachable = pools.filter((pool) => pool.reachable).length;
  const totalMiners = pools.reduce((sum, pool) => sum + Number(pool.miners || 0), 0);
  const syncGap = getSyncGap(wallet);
  const recentAudit = getAuditEvents().slice(-5).reverse();
  return `${setupPanel()}<section class="hero-band"><div><p class="eyebrow">${escapeHtml(t('dashboard.operatorPreview'))}</p><h2>${escapeHtml(t('dashboard.title'))}</h2><p>${escapeHtml(t('dashboard.copy'))}</p></div><div class="hero-signal" aria-hidden="true"><span></span><span></span><span></span></div></section>
  <section class="metric-grid">${metricCard(t('metric.balance'), fmtPrl(last.balancePRL), wallet.blockHeight ? t('metric.syncedAt', { height: wallet.blockHeight }) : t('wallet.configMissing'), 'pearl')}${metricCard(t('metric.sweepable'), fmtPrl(Math.max(0, Number(wallet.balancePRL || 0) - Number(wallet.reservePRL || 0))), t('metric.threshold', { value: fmtPrl(wallet.thresholdPRL || 1.1) }), decisionTone(appState.dryRun))}${metricCard(t('metric.pools'), `${reachable}/${pools.length}`, t('metric.miners', { count: totalMiners }), 'violet')}${metricCard(t('metric.syncGap'), syncGap == null ? '—' : String(syncGap), wallet.synced ? t('status.synced') : t('status.notSynced'), wallet.synced ? 'safe' : 'amber')}</section>
  <section class="split-layout"><article class="panel wide-panel"><div class="panel-header"><h3>${escapeHtml(t('curves.balanceCurve'))}</h3>${statusPill(hasRealData() ? t('status.verified') : t('status.setupRequired'), hasRealData() ? 'safe' : 'warn')}</div>${chartSvg(snapshots, { threshold: wallet.thresholdPRL || 1.1, reserve: wallet.reservePRL || 0.02, height: 270 })}</article><article class="panel"><div class="panel-header"><h3>${escapeHtml(t('dashboard.auditTrail'))}</h3>${statusPill(`${recentAudit.length}`, 'neutral')}</div><div class="event-list">${recentAudit.length ? recentAudit.map(renderAuditEvent).join('') : `<p class="empty-note">${escapeHtml(t('dashboard.noAudit'))}</p>`}</div></article></section>
  <section class="panel"><div class="panel-header"><h3>${escapeHtml(t('pools.intelligence'))}</h3>${statusPill(appState.poolSync?.mode || t('status.catalog'), 'neutral')}</div>${renderPoolRows(pools.slice(0, 4))}</section>`;
}

function renderAuditEvent(event) { return `<div class="event-row"><span class="event-dot ${escapeHtml(event.severity || 'info')}"></span><div><strong>${escapeHtml(event.scope)} · ${escapeHtml(event.event)}</strong><small>${escapeHtml(fmtTime(event.timestamp))} - ${escapeHtml(event.message)}</small></div></div>`; }

function renderMonitor() {
  const wallet = getWallet();
  const dry = appState.dryRun;
  const active = Boolean(appState.monitorTimer);
  return `<section class="panel action-panel"><div><h2>${escapeHtml(t('monitor.title'))}</h2><p>${escapeHtml(t('monitor.copy'))}</p></div><div class="button-row"><button class="primary-button" id="toggleMonitor">${escapeHtml(active ? t('monitor.stop') : t('monitor.start'))}</button><button class="ghost-button" id="dryRunCheck">${escapeHtml(t('monitor.runDryCheck'))}</button></div></section>
  <section class="metric-grid">${metricCard(t('metric.balance'), fmtPrl(wallet.balancePRL), wallet.connected ? t('status.synced') : t('wallet.configMissing'))}${metricCard(t('metric.reserve'), fmtPrl(wallet.reservePRL || 0.02), t('monitor.reserveHint'))}${metricCard(t('metric.thresholdPlain'), fmtPrl(wallet.thresholdPRL || 1.1), t('monitor.thresholdHint'))}${metricCard(t('metric.broadcasts'), String(appState.transferRequests), t('monitor.broadcastHint'), 'safe')}</section>
  <section class="panel decision-panel"><div class="panel-header"><h3>${escapeHtml(t('monitor.latestDecision'))}</h3>${statusPill(dry?.thresholdReached ? t('monitor.ready') : t('monitor.hold'), decisionTone(dry))}</div><div class="decision-grid"><div><span>${escapeHtml(t('metric.sweepable'))}</span><strong>${escapeHtml(fmtPrl(dry?.sweepablePRL || 0))}</strong></div><div><span>${escapeHtml(t('metric.thresholdPlain'))}</span><strong>${escapeHtml(fmtPrl(dry?.thresholdPRL || wallet.thresholdPRL || 1.1))}</strong></div><div><span>${escapeHtml(t('monitor.decision'))}</span><strong>${escapeHtml(dry?.decision || t('monitor.noDecision'))}</strong></div></div><p class="decision-copy">${escapeHtml(dry?.message || t('monitor.noDecision'))}</p></section>`;
}

function renderHistory() {
  const events = getState().addressEvents || [];
  const query = appState.historyFilter.trim().toLowerCase();
  const filtered = events.filter((item) => !query || JSON.stringify(item).toLowerCase().includes(query));
  return `<section class="panel action-panel"><div><h2>${escapeHtml(t('history.title'))}</h2><p>${escapeHtml(t('history.copy'))}</p></div><div class="button-row"><input id="historyFilter" class="search-input" value="${escapeHtml(appState.historyFilter)}" placeholder="${escapeHtml(t('history.search'))}" /><button class="primary-button" id="importCsvPage">${escapeHtml(t('action.importCsv'))}</button></div></section>
  <section class="panel table-panel"><table><thead><tr><th>${escapeHtml(t('table.time'))}</th><th>${escapeHtml(t('table.address'))}</th><th>${escapeHtml(t('table.direction'))}</th><th>${escapeHtml(t('table.amount'))}</th><th>${escapeHtml(t('table.balanceAfter'))}</th><th>${escapeHtml(t('table.txid'))}</th></tr></thead><tbody>${filtered.length ? filtered.map((event) => `<tr><td>${escapeHtml(fmtTime(event.timestamp))}</td><td>${escapeHtml(event.addressLabel)}</td><td>${escapeHtml(t(`direction.${event.direction}`))}</td><td>${escapeHtml(fmtPrl(event.amountPRL))}</td><td>${escapeHtml(fmtPrl(event.balanceAfterPRL))}</td><td><code>${escapeHtml(event.txid || '—')}</code></td></tr>`).join('') : `<tr><td colspan="6">${escapeHtml(t('history.noRows'))}</td></tr>`}</tbody></table></section>`;
}

function verifyCurve(snapshots, wallet) {
  const peaks = snapshots.filter((point) => Number(point.balancePRL) - Number(point.reservePRL || wallet.reservePRL || 0.02) > Number(wallet.thresholdPRL || 1.1));
  const returnsToReserve = snapshots.filter((point) => Math.abs(Number(point.balancePRL) - Number(wallet.reservePRL || 0.02)) < 0.05).length;
  const sampleCount = snapshots.length;
  return [
    { tone: peaks.length ? 'safe' : 'warn', title: t('verify.thresholdTitle'), message: t('verify.thresholdMessage', { count: peaks.length }) },
    { tone: returnsToReserve ? 'safe' : 'warn', title: t('verify.reserveTitle'), message: t('verify.reserveMessage', { count: returnsToReserve }) },
    { tone: sampleCount > 2 ? 'safe' : 'warn', title: t('verify.sampleTitle'), message: t('verify.sampleMessage', { count: sampleCount }) }
  ];
}

function renderCurves() {
  const wallet = getWallet();
  const snapshots = getSnapshots();
  const checks = verifyCurve(snapshots, wallet);
  return `<section class="panel wide-panel"><div class="panel-header"><h3>${escapeHtml(t('curves.title'))}</h3>${statusPill(hasRealData() ? t('status.verified') : t('status.setupRequired'), hasRealData() ? 'safe' : 'warn')}</div>${chartSvg(snapshots, { threshold: wallet.thresholdPRL || 1.1, reserve: wallet.reservePRL || 0.02, height: 330 })}</section><section class="verification-grid">${checks.map((check) => `<article class="verify-card ${check.tone}"><strong>${escapeHtml(check.title)}</strong><p>${escapeHtml(check.message)}</p></article>`).join('')}</section>`;
}

function renderPoolsPage() {
  const pools = getPoolObservations();
  const catalog = configuredPools();
  return `<section class="panel action-panel"><div><h2>${escapeHtml(t('pools.title'))}</h2><p>${escapeHtml(t('pools.copy'))}</p></div><div class="button-row"><button class="ghost-button" id="goPoolSettings">${escapeHtml(t('pools.configure'))}</button><button class="primary-button" id="syncPoolsPage">${escapeHtml(t('action.syncPools'))}</button></div></section><section class="metric-grid">${metricCard(t('pools.catalog'), String(catalog.length), t('pools.catalogHint'))}${metricCard(t('pools.reachable'), String(pools.filter((p) => p.reachable).length), t('pools.reachableHint'))}${metricCard(t('pools.lastSync'), fmtTime(appState.poolSync?.timestamp), appState.poolSync?.mode || t('status.catalog'))}${metricCard(t('pools.adapters'), String(adapters.length), adapters.join(', '))}</section><section class="panel">${renderPoolRows(pools)}</section>`;
}

function renderPoolRows(pools) {
  if (!pools.length) return `<p class="empty-note">${escapeHtml(t('pools.empty'))}</p>`;
  return `<div class="pool-grid">${pools.map((pool) => `<article class="pool-card"><div class="pool-title"><strong>${escapeHtml(pool.poolName)}</strong>${statusPill(pool.reachable ? t('status.online') : t('status.catalog'), pool.reachable ? 'safe' : 'neutral')}</div><dl><div><dt>${escapeHtml(t('pools.miners'))}</dt><dd>${escapeHtml(pool.miners ?? '—')}</dd></div><div><dt>${escapeHtml(t('pools.poolHashrate'))}</dt><dd>${escapeHtml(pool.poolHashrate ?? '—')}</dd></div><div><dt>${escapeHtml(t('pools.networkHashrate'))}</dt><dd>${escapeHtml(pool.networkHashrate ?? '—')}</dd></div><div><dt>${escapeHtml(t('pools.height'))}</dt><dd>${escapeHtml(pool.blockHeight ?? '—')}</dd></div></dl><small>${escapeHtml(pool.message)}</small></article>`).join('')}</div>`;
}

function inputField(id, label, value, options = {}) {
  const type = options.type || 'text';
  const step = options.step ? ` step="${escapeHtml(options.step)}"` : '';
  const min = options.min !== undefined ? ` min="${escapeHtml(options.min)}"` : '';
  const placeholder = options.placeholder ? ` placeholder="${escapeHtml(options.placeholder)}"` : '';
  return `<label class="field"><span>${escapeHtml(label)}</span><input id="${id}" type="${type}" value="${escapeHtml(value ?? '')}"${step}${min}${placeholder} /></label>`;
}
function selectField(id, label, value, values) {
  return `<label class="field"><span>${escapeHtml(label)}</span><select id="${id}">${values.map((item) => `<option value="${escapeHtml(item)}" ${item === value ? 'selected' : ''}>${escapeHtml(localeNames[item] || item)}</option>`).join('')}</select></label>`;
}
function checkboxField(id, label, checked) {
  return `<label class="field toggle-field"><span>${escapeHtml(label)}</span><input id="${id}" type="checkbox" ${checked ? 'checked' : ''} /></label>`;
}
function adapterOptions(value) { return adapters.map((adapter) => `<option value="${escapeHtml(adapter)}" ${adapter === value ? 'selected' : ''}>${escapeHtml(adapter)}</option>`).join(''); }

function renderSettings() {
  const settings = appState.settings || {};
  const pools = configuredPools();
  return `<section class="settings-layout">
    <article class="panel settings-card"><div class="panel-header"><h3>${escapeHtml(t('settings.walletConnection'))}</h3>${statusPill(getWallet().configured ? t('status.configured') : t('status.setupRequired'), getWallet().configured ? 'safe' : 'warn')}</div><div class="form-grid">${inputField('walletLabel', t('settings.walletLabel'), settings.walletLabel || '')}${inputField('network', t('settings.network'), settings.network || 'mainnet')}${inputField('rpcUrl', t('settings.rpcUrl'), settings.rpcUrl || '', { placeholder: 'http://127.0.0.1:8335' })}${inputField('rpcHost', t('settings.rpcHost'), settings.rpcHost || '127.0.0.1')}${inputField('rpcPort', t('settings.rpcPort'), settings.rpcPort || 8335, { type: 'number', min: 1 })}${inputField('rpcUsername', t('settings.rpcUsername'), settings.rpcUsername || '')}${inputField('rpcPassword', t('settings.rpcPassword'), settings.rpcPassword || '', { type: 'password' })}</div></article>
    <article class="panel settings-card"><div class="panel-header"><h3>${escapeHtml(t('settings.guardPolicy'))}</h3>${statusPill(t('status.readOnly'), 'safe')}</div><div class="form-grid">${inputField('reservePRL', t('settings.reservePRL'), settings.reservePRL ?? 0.02, { type: 'number', min: 0, step: '0.00000001' })}${inputField('thresholdPRL', t('settings.thresholdPRL'), settings.thresholdPRL ?? 1.1, { type: 'number', min: 0, step: '0.00000001' })}${inputField('destinationAddress', t('settings.destinationAddress'), settings.destinationAddress || '')}${inputField('refreshSeconds', t('settings.refreshSeconds'), settings.refreshSeconds || 30, { type: 'number', min: 10 })}${inputField('poolSyncSeconds', t('settings.poolSyncSeconds'), settings.poolSyncSeconds || 120, { type: 'number', min: 30 })}${selectField('uiLanguage', t('settings.language'), appState.locale, supportedLocales)}${checkboxField('autoRefresh', t('settings.autoRefresh'), Boolean(settings.autoRefresh))}</div></article>
  </section>
  <section class="panel action-panel"><div><h2>${escapeHtml(t('settings.saveTitle'))}</h2><p>${escapeHtml(t('settings.saveCopy'))}</p></div><div class="button-row"><button class="ghost-button" id="testRpcButton">${escapeHtml(t('settings.testRpc'))}</button><button class="primary-button" id="saveSettingsButton">${escapeHtml(t('settings.save'))}</button></div></section>
  <section class="panel"><div class="panel-header"><h3>${escapeHtml(t('settings.poolEndpoints'))}</h3><div class="button-row"><button class="ghost-button" id="addPoolButton">${escapeHtml(t('settings.addPool'))}</button><button class="primary-button" id="savePoolsButton">${escapeHtml(t('settings.savePools'))}</button></div></div><div class="pool-editor">${pools.map(renderPoolEditor).join('')}</div></section>`;
}

function renderPoolEditor(pool, index) {
  return `<article class="pool-editor-row" data-pool-index="${index}"><div class="pool-editor-top"><label class="mini-check"><input class="pool-enabled" type="checkbox" ${pool.enabled ? 'checked' : ''} /> ${escapeHtml(t('settings.enabled'))}</label><button class="ghost-button danger-button" data-remove-pool="${index}">${escapeHtml(t('settings.remove'))}</button></div><div class="form-grid compact"><label class="field"><span>${escapeHtml(t('settings.poolName'))}</span><input class="pool-name" value="${escapeHtml(pool.name || '')}" /></label><label class="field"><span>${escapeHtml(t('settings.adapter'))}</span><select class="pool-adapter">${adapterOptions(pool.adapter || 'generic-json')}</select></label><label class="field"><span>${escapeHtml(t('settings.coinSymbol'))}</span><input class="pool-coin" value="${escapeHtml(pool.coinSymbol || 'PRL')}" /></label><label class="field wide-field"><span>${escapeHtml(t('settings.endpoint'))}</span><input class="pool-endpoint" value="${escapeHtml(pool.endpoint || '')}" /></label></div></article>`;
}

function readSettingsForm() {
  return {
    version: 1,
    walletLabel: document.getElementById('walletLabel')?.value.trim() || 'Local Pearl Wallet',
    network: document.getElementById('network')?.value.trim() || 'mainnet',
    rpcUrl: document.getElementById('rpcUrl')?.value.trim() || '',
    rpcHost: document.getElementById('rpcHost')?.value.trim() || '127.0.0.1',
    rpcPort: Number(document.getElementById('rpcPort')?.value || 8335),
    rpcUsername: document.getElementById('rpcUsername')?.value.trim() || '',
    rpcPassword: document.getElementById('rpcPassword')?.value || '',
    reservePRL: Number(document.getElementById('reservePRL')?.value || 0),
    thresholdPRL: Number(document.getElementById('thresholdPRL')?.value || 1.1),
    destinationAddress: document.getElementById('destinationAddress')?.value.trim() || '',
    refreshSeconds: Number(document.getElementById('refreshSeconds')?.value || 30),
    poolSyncSeconds: Number(document.getElementById('poolSyncSeconds')?.value || 120),
    uiLanguage: document.getElementById('uiLanguage')?.value || appState.locale,
    autoRefresh: Boolean(document.getElementById('autoRefresh')?.checked),
    readOnly: true
  };
}

function readPoolEditor() {
  return {
    version: 1,
    pollSeconds: Number(appState.settings?.poolSyncSeconds || appState.poolConfig?.pollSeconds || 120),
    pools: Array.from(document.querySelectorAll('.pool-editor-row')).map((row, index) => {
      const existing = configuredPools()[index] || {};
      return {
        ...existing,
        id: existing.id || `pool-${index + 1}`,
        name: row.querySelector('.pool-name')?.value.trim() || `Pool ${index + 1}`,
        adapter: row.querySelector('.pool-adapter')?.value || 'generic-json',
        enabled: Boolean(row.querySelector('.pool-enabled')?.checked),
        endpoint: row.querySelector('.pool-endpoint')?.value.trim() || '',
        coinSymbol: row.querySelector('.pool-coin')?.value.trim() || 'PRL'
      };
    })
  };
}

function bindSettingsHandlers() {
  document.getElementById('saveSettingsButton')?.addEventListener('click', saveSettings);
  document.getElementById('testRpcButton')?.addEventListener('click', testRpcConnection);
  document.getElementById('savePoolsButton')?.addEventListener('click', savePools);
  document.getElementById('addPoolButton')?.addEventListener('click', () => {
    appState.poolConfig = { ...(appState.poolConfig || {}), pools: [...configuredPools(), { id: '', name: 'Pearl Pool', adapter: 'generic-json', enabled: false, endpoint: '', coinSymbol: 'PRL' }] };
    render();
  });
  document.querySelectorAll('[data-remove-pool]').forEach((button) => button.addEventListener('click', () => {
    const removeIndex = Number(button.dataset.removePool);
    appState.poolConfig = { ...(appState.poolConfig || {}), pools: configuredPools().filter((_, index) => index !== removeIndex) };
    render();
  }));
}

function render() {
  const pageHtml = { dashboard: renderDashboard, monitor: renderMonitor, history: renderHistory, curves: renderCurves, pools: renderPoolsPage, settings: renderSettings }[appState.activePage]();
  renderShell(pageHtml);
  document.querySelectorAll('#dryRunCheck').forEach((button) => button.addEventListener('click', runReadOnlyCheck));
  document.querySelectorAll('#refreshWalletPage').forEach((button) => button.addEventListener('click', refreshWallet));
  document.querySelectorAll('#importCsvTop,#importCsvPage').forEach((button) => button.addEventListener('click', importCsv));
  document.querySelectorAll('#goSettingsTop,#goPoolSettings').forEach((button) => button.addEventListener('click', () => { appState.activePage = 'settings'; render(); }));
  document.getElementById('toggleMonitor')?.addEventListener('click', toggleMonitor);
  const historyInput = document.getElementById('historyFilter');
  if (historyInput) historyInput.addEventListener('input', (event) => { appState.historyFilter = event.target.value; render(); });
  document.getElementById('syncPoolsPage')?.addEventListener('click', syncPoolsAndRender);
  if (appState.activePage === 'settings') bindSettingsHandlers();
}

async function refreshWallet() {
  if (appState.busy) return;
  appState.busy = true;
  try {
    const result = await window.pearlGuard.readWalletStatus();
    if (result.state) appState.state = result.state;
    appState.notice = result.ok ? t('wallet.statusOk') : `${t('wallet.statusFailed')} ${result.message || ''}`;
    appState.dryRun = await window.pearlGuard.dryRunSweepCheck(getWallet());
    appState.transferRequests = appState.dryRun.transferRequests || 0;
  } finally {
    appState.busy = false;
    render();
  }
}

async function importCsv() {
  const result = await window.pearlGuard.importAuditCsv();
  if (!result.canceled && result.state) {
    appState.state = result.state;
    appState.notice = t('history.importedRows', { count: result.importedRows });
  }
  render();
}

async function runReadOnlyCheck() {
  appState.dryRun = await window.pearlGuard.dryRunSweepCheck(getWallet());
  appState.transferRequests = appState.dryRun.transferRequests || 0;
  render();
}

async function syncPoolsAndRender() {
  appState.poolSync = await window.pearlGuard.syncPools({});
  render();
}

async function saveSettings() {
  const settings = readSettingsForm();
  const result = await window.pearlGuard.saveSettings(settings);
  appState.settings = result.settings || settings;
  if (result.state) appState.state = result.state;
  const requestedLocale = appState.settings.uiLanguage;
  if (supportedLocales.includes(requestedLocale) && requestedLocale !== appState.locale) {
    appState.locale = requestedLocale;
    localStorage.setItem('pearlguard.locale', appState.locale);
    appState.messages = await loadMessages(appState.locale);
  }
  appState.notice = t('settings.saved');
  render();
}

async function testRpcConnection() {
  const result = await window.pearlGuard.testRpcConnection(readSettingsForm());
  appState.notice = result.ok ? `${t('settings.rpcOk')} ${result.blocks ?? ''}` : `${t('settings.rpcFailed')} ${result.message || ''}`;
  render();
}

async function savePools() {
  const result = await window.pearlGuard.savePoolConfig(readPoolEditor());
  if (result.poolConfig) appState.poolConfig = result.poolConfig;
  appState.notice = t('settings.poolsSaved');
  render();
}

async function monitorTick() {
  await refreshWallet();
  const seconds = Number(appState.settings?.poolSyncSeconds || 120);
  if (Date.now() % Math.max(30000, seconds * 1000) < 1500) await syncPoolsAndRender();
}

function toggleMonitor() {
  if (appState.monitorTimer) {
    clearInterval(appState.monitorTimer);
    appState.monitorTimer = null;
    notice(t('monitor.stopped'));
    return;
  }
  const intervalMs = Math.max(10, Number(appState.settings?.refreshSeconds || 30)) * 1000;
  appState.monitorTimer = setInterval(monitorTick, intervalMs);
  notice(t('monitor.started'));
  monitorTick();
}

async function boot() {
  appState.bootstrap = await window.pearlGuard.getBootstrap();
  appState.settings = appState.bootstrap.settings || {};
  appState.poolConfig = appState.bootstrap.poolConfig || { version: 1, pollSeconds: 120, pools: [] };
  appState.state = appState.bootstrap.state;
  appState.locale = chooseLocale(appState.bootstrap.locale, navigator.language, appState.settings.uiLanguage);
  appState.messages = await loadMessages(appState.locale);
  appState.poolSync = await window.pearlGuard.syncPools({});
  appState.dryRun = await window.pearlGuard.dryRunSweepCheck(getWallet());
  appState.transferRequests = appState.dryRun.transferRequests || 0;
  if (appState.settings.autoRefresh) toggleMonitor();
  render();
}

window.__pearlguardReady = boot();
window.__pearlguardSelfTest = async () => {
  await window.__pearlguardReady;
  const poolSync = await window.pearlGuard.syncPools({});
  const dryRun = await window.pearlGuard.dryRunSweepCheck(getWallet());
  const labels = Array.from(document.querySelectorAll('.nav-button')).map((node) => node.textContent.trim()).join('|');
  return {
    ok: Boolean(document.querySelector('.app-shell') && appState.bootstrap.mode === 'local' && poolSync.observations.length >= 1 && labels.includes(t('nav.dashboard')) && document.getElementById('openSettingsTop')),
    locale: appState.locale,
    mode: appState.bootstrap.mode,
    poolCount: poolSync.observations.length,
    transferRequests: dryRun.transferRequests,
    decision: dryRun.decision
  };
};
