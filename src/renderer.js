const supportedLocales = ['en', 'ar', 'zh-CN', 'fr', 'ru', 'es'];
const localeNames = {
  en: 'English',
  ar: 'العربية',
  'zh-CN': '简体中文',
  fr: 'Français',
  ru: 'Русский',
  es: 'Español'
};

const appState = {
  activePage: 'dashboard',
  locale: 'en',
  messages: {},
  bootstrap: null,
  poolSync: null,
  dryRun: null,
  historyFilter: '',
  transferRequests: 0
};

const navItems = [
  ['dashboard', 'nav.dashboard', '◆'],
  ['monitor', 'nav.monitor', '◉'],
  ['history', 'nav.history', '↕'],
  ['curves', 'nav.curves', '⌁'],
  ['pools', 'nav.pools', '⌬'],
  ['settings', 'nav.settings', '⚙']
];

function escapeHtml(value) {
  return String(value ?? '').replace(/[&<>'"]/g, (char) => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    "'": '&#39;',
    '"': '&quot;'
  })[char]);
}

function chooseLocale(systemLocale, navigatorLocale) {
  const stored = localStorage.getItem('pearlguard.locale');
  if (supportedLocales.includes(stored)) return stored;
  const raw = `${systemLocale || navigatorLocale || 'en'}`.toLowerCase();
  if (raw.startsWith('ar')) return 'ar';
  if (raw.startsWith('zh')) return 'zh-CN';
  if (raw.startsWith('fr')) return 'fr';
  if (raw.startsWith('ru')) return 'ru';
  if (raw.startsWith('es')) return 'es';
  return 'en';
}

async function loadMessages(locale) {
  return window.pearlGuard.getMessages(locale);
}

function t(key, params = {}) {
  const template = appState.messages[key] || key;
  return Object.entries(params).reduce((text, [name, value]) => text.replaceAll(`{${name}}`, value), template);
}

function fmtPrl(value) {
  const number = Number(value || 0);
  return `${number.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 8 })} PRL`;
}

function fmtTime(value) {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(appState.locale, { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit' });
}

function latest(items) {
  return items[items.length - 1];
}

function getSnapshots() {
  return appState.bootstrap.demoState.snapshots || [];
}

function getWallet() {
  return appState.bootstrap.demoState.wallet || {};
}

function getPoolObservations() {
  return appState.poolSync?.observations || [];
}

function statusPill(label, tone = 'neutral') {
  return `<span class="status-pill ${tone}">${escapeHtml(label)}</span>`;
}

function metricCard(label, value, detail, tone = '') {
  return `<article class="metric-card ${tone}"><span>${escapeHtml(label)}</span><strong>${escapeHtml(value)}</strong><small>${escapeHtml(detail || '')}</small></article>`;
}

function chartSvg(points, options = {}) {
  const width = 820;
  const height = options.height || 250;
  const pad = 28;
  if (!points.length) return `<svg class="chart" viewBox="0 0 ${width} ${height}" aria-label="${escapeHtml(t('chart.empty'))}"></svg>`;
  const values = points.map((p) => Number(p.balancePRL));
  const maxValue = Math.max(...values, Number(options.threshold || 0), 1.2);
  const minValue = Math.min(0, ...values);
  const range = Math.max(0.0001, maxValue - minValue);
  const x = (index) => pad + (index / Math.max(1, points.length - 1)) * (width - pad * 2);
  const y = (value) => height - pad - ((Number(value) - minValue) / range) * (height - pad * 2);
  const path = points.map((point, index) => `${index === 0 ? 'M' : 'L'} ${x(index).toFixed(2)} ${y(point.balancePRL).toFixed(2)}`).join(' ');
  const dots = points.map((point, index) => `<circle cx="${x(index).toFixed(2)}" cy="${y(point.balancePRL).toFixed(2)}" r="5" class="chart-dot"><title>${escapeHtml(fmtTime(point.timestamp))}: ${escapeHtml(fmtPrl(point.balancePRL))}</title></circle>`).join('');
  const thresholdY = y(options.threshold || 1.1).toFixed(2);
  const reserveY = y(options.reserve || 0.02).toFixed(2);
  return `
    <svg class="chart" viewBox="0 0 ${width} ${height}" role="img" aria-label="${escapeHtml(t('chart.balanceCurve'))}">
      <line x1="${pad}" x2="${width - pad}" y1="${height - pad}" y2="${height - pad}" class="chart-axis" />
      <line x1="${pad}" x2="${pad}" y1="${pad}" y2="${height - pad}" class="chart-axis" />
      <line x1="${pad}" x2="${width - pad}" y1="${thresholdY}" y2="${thresholdY}" class="chart-threshold" />
      <line x1="${pad}" x2="${width - pad}" y1="${reserveY}" y2="${reserveY}" class="chart-reserve" />
      <path d="${path}" class="chart-line" />
      ${dots}
      <text x="${width - pad - 120}" y="${Number(thresholdY) - 8}" class="chart-label">${escapeHtml(t('chart.threshold'))}</text>
      <text x="${width - pad - 100}" y="${Number(reserveY) - 8}" class="chart-label muted">${escapeHtml(t('chart.reserve'))}</text>
    </svg>`;
}

function renderShell(pageHtml) {
  const direction = appState.locale === 'ar' ? 'rtl' : 'ltr';
  document.documentElement.lang = appState.locale;
  document.documentElement.dir = direction;
  const nav = navItems.map(([id, key, icon]) => `
    <button class="nav-button ${appState.activePage === id ? 'active' : ''}" data-page="${id}" title="${escapeHtml(t(key))}">
      <span class="nav-icon">${icon}</span><span>${escapeHtml(t(key))}</span>
    </button>`).join('');

  document.getElementById('app').className = 'app-shell';
  document.getElementById('app').innerHTML = `
    <aside class="sidebar">
      <div class="brand-lockup">
        <img src="../assets/app-icon.svg" alt="" />
        <div><strong>PearlGuard</strong><small>${escapeHtml(t('app.subtitle'))}</small></div>
      </div>
      <nav class="nav-list" aria-label="${escapeHtml(t('nav.label'))}">${nav}</nav>
      <div class="side-status">
        ${statusPill(t('status.dryRun'), 'safe')}
        <span>${escapeHtml(t('status.noBroadcast'))}</span>
      </div>
    </aside>
    <section class="workspace">
      <header class="topbar">
        <div>
          <p class="eyebrow">${escapeHtml(t('app.eyebrow'))}</p>
          <h1>${escapeHtml(t(`page.${appState.activePage}`))}</h1>
        </div>
        <div class="top-actions">
          <select id="localeSelect" aria-label="${escapeHtml(t('settings.language'))}">
            ${supportedLocales.map((locale) => `<option value="${locale}" ${locale === appState.locale ? 'selected' : ''}>${localeNames[locale]}</option>`).join('')}
          </select>
          <button class="ghost-button" id="syncPoolsTop">${escapeHtml(t('action.syncPools'))}</button>
        </div>
      </header>
      <main class="page-view">${pageHtml}</main>
    </section>`;

  document.querySelectorAll('[data-page]').forEach((button) => {
    button.addEventListener('click', () => {
      appState.activePage = button.dataset.page;
      render();
    });
  });
  document.getElementById('localeSelect').addEventListener('change', async (event) => {
    appState.locale = event.target.value;
    localStorage.setItem('pearlguard.locale', appState.locale);
    appState.messages = await loadMessages(appState.locale);
    render();
  });
  document.getElementById('syncPoolsTop').addEventListener('click', syncPoolsAndRender);
}

function renderDashboard() {
  const wallet = getWallet();
  const snapshots = getSnapshots();
  const last = latest(snapshots) || wallet;
  const pools = getPoolObservations();
  const reachable = pools.filter((pool) => pool.reachable).length;
  const totalMiners = pools.reduce((sum, pool) => sum + Number(pool.miners || 0), 0);
  const recentAudit = (appState.bootstrap.demoState.auditEvents || []).slice(-4).reverse();

  return `
    <section class="hero-band">
      <div>
        <p class="eyebrow">${escapeHtml(t('dashboard.operatorPreview'))}</p>
        <h2>${escapeHtml(t('dashboard.title'))}</h2>
        <p>${escapeHtml(t('dashboard.copy'))}</p>
      </div>
      <div class="hero-signal" aria-hidden="true"><span></span><span></span><span></span></div>
    </section>
    <section class="metric-grid">
      ${metricCard(t('metric.balance'), fmtPrl(last.balancePRL), t('metric.syncedAt', { height: wallet.blockHeight }), 'pearl')}
      ${metricCard(t('metric.sweepable'), fmtPrl(Math.max(0, wallet.balancePRL - wallet.reservePRL)), t('metric.threshold', { value: fmtPrl(wallet.thresholdPRL) }), 'teal')}
      ${metricCard(t('metric.pools'), `${reachable}/${pools.length}`, t('metric.miners', { count: totalMiners }), 'violet')}
      ${metricCard(t('metric.mode'), t('status.dryRun'), t('status.noBroadcast'), 'amber')}
    </section>
    <section class="split-layout">
      <article class="panel wide-panel">
        <div class="panel-header"><h3>${escapeHtml(t('curves.balanceCurve'))}</h3>${statusPill(t('status.verified'), 'safe')}</div>
        ${chartSvg(snapshots, { threshold: wallet.thresholdPRL, reserve: wallet.reservePRL, height: 270 })}
      </article>
      <article class="panel">
        <div class="panel-header"><h3>${escapeHtml(t('dashboard.auditTrail'))}</h3>${statusPill(`${recentAudit.length}`, 'neutral')}</div>
        <div class="event-list">${recentAudit.map(renderAuditEvent).join('')}</div>
      </article>
    </section>
    <section class="panel">
      <div class="panel-header"><h3>${escapeHtml(t('pools.intelligence'))}</h3>${statusPill(appState.poolSync?.mode || 'fixture', 'neutral')}</div>
      ${renderPoolRows(pools.slice(0, 4))}
    </section>`;
}

function renderAuditEvent(event) {
  return `<div class="event-row"><span class="event-dot ${escapeHtml(event.severity || 'info')}"></span><div><strong>${escapeHtml(event.scope)} · ${escapeHtml(event.event)}</strong><small>${escapeHtml(fmtTime(event.timestamp))} — ${escapeHtml(event.message)}</small></div></div>`;
}

function renderMonitor() {
  const wallet = getWallet();
  const dry = appState.dryRun;
  return `
    <section class="panel action-panel">
      <div>
        <h2>${escapeHtml(t('monitor.title'))}</h2>
        <p>${escapeHtml(t('monitor.copy'))}</p>
      </div>
      <button class="primary-button" id="dryRunCheck">${escapeHtml(t('monitor.runDryCheck'))}</button>
    </section>
    <section class="metric-grid">
      ${metricCard(t('metric.balance'), fmtPrl(wallet.balancePRL), wallet.synced ? t('status.synced') : t('status.notSynced'))}
      ${metricCard(t('metric.reserve'), fmtPrl(wallet.reservePRL), t('monitor.reserveHint'))}
      ${metricCard(t('metric.thresholdPlain'), fmtPrl(wallet.thresholdPRL), t('monitor.thresholdHint'))}
      ${metricCard(t('metric.broadcasts'), String(appState.transferRequests), t('monitor.broadcastHint'), 'safe')}
    </section>
    <section class="panel">
      <div class="panel-header"><h3>${escapeHtml(t('monitor.latestDecision'))}</h3>${statusPill(t('status.dryRun'), 'safe')}</div>
      <pre class="decision-box">${escapeHtml(JSON.stringify(dry || { message: t('monitor.noDecision') }, null, 2))}</pre>
    </section>`;
}

function renderHistory() {
  const events = appState.bootstrap.demoState.addressEvents || [];
  const query = appState.historyFilter.trim().toLowerCase();
  const filtered = events.filter((item) => !query || JSON.stringify(item).toLowerCase().includes(query));
  return `
    <section class="panel action-panel">
      <div>
        <h2>${escapeHtml(t('history.title'))}</h2>
        <p>${escapeHtml(t('history.copy'))}</p>
      </div>
      <input id="historyFilter" class="search-input" value="${escapeHtml(appState.historyFilter)}" placeholder="${escapeHtml(t('history.search'))}" />
    </section>
    <section class="panel table-panel">
      <table>
        <thead><tr><th>${escapeHtml(t('table.time'))}</th><th>${escapeHtml(t('table.address'))}</th><th>${escapeHtml(t('table.direction'))}</th><th>${escapeHtml(t('table.amount'))}</th><th>${escapeHtml(t('table.balanceAfter'))}</th><th>${escapeHtml(t('table.txid'))}</th></tr></thead>
        <tbody>${filtered.map((event) => `<tr><td>${escapeHtml(fmtTime(event.timestamp))}</td><td>${escapeHtml(event.addressLabel)}</td><td>${escapeHtml(t(`direction.${event.direction}`))}</td><td>${escapeHtml(fmtPrl(event.amountPRL))}</td><td>${escapeHtml(fmtPrl(event.balanceAfterPRL))}</td><td><code>${escapeHtml(event.txid)}</code></td></tr>`).join('')}</tbody>
      </table>
    </section>`;
}

function renderCurves() {
  const wallet = getWallet();
  const snapshots = getSnapshots();
  const checks = verifyCurve(snapshots, wallet);
  return `
    <section class="panel wide-panel">
      <div class="panel-header"><h3>${escapeHtml(t('curves.title'))}</h3>${statusPill(t('status.fixture'), 'neutral')}</div>
      ${chartSvg(snapshots, { threshold: wallet.thresholdPRL, reserve: wallet.reservePRL, height: 330 })}
    </section>
    <section class="verification-grid">
      ${checks.map((check) => `<article class="verify-card ${check.tone}"><strong>${escapeHtml(check.title)}</strong><p>${escapeHtml(check.message)}</p></article>`).join('')}
    </section>`;
}

function verifyCurve(snapshots, wallet) {
  const peaks = snapshots.filter((point) => Number(point.balancePRL) - Number(point.reservePRL) > Number(wallet.thresholdPRL));
  const returnsToReserve = snapshots.filter((point) => Math.abs(Number(point.balancePRL) - Number(wallet.reservePRL)) < 0.01).length;
  return [
    { tone: 'safe', title: t('verify.thresholdTitle'), message: t('verify.thresholdMessage', { count: peaks.length }) },
    { tone: 'safe', title: t('verify.reserveTitle'), message: t('verify.reserveMessage', { count: returnsToReserve }) },
    { tone: 'warn', title: t('verify.feeTitle'), message: t('verify.feeMessage') }
  ];
}

function renderPoolsPage() {
  const pools = getPoolObservations();
  const catalog = appState.bootstrap.poolConfig.pools || [];
  return `
    <section class="panel action-panel">
      <div>
        <h2>${escapeHtml(t('pools.title'))}</h2>
        <p>${escapeHtml(t('pools.copy'))}</p>
      </div>
      <button class="primary-button" id="syncPoolsPage">${escapeHtml(t('action.syncPools'))}</button>
    </section>
    <section class="metric-grid">
      ${metricCard(t('pools.catalog'), String(catalog.length), t('pools.catalogHint'))}
      ${metricCard(t('pools.reachable'), String(pools.filter((p) => p.reachable).length), t('pools.reachableHint'))}
      ${metricCard(t('pools.lastSync'), fmtTime(appState.poolSync?.timestamp), appState.poolSync?.mode || 'fixture')}
      ${metricCard(t('pools.adapters'), '5', 'zpool, yiimp, miningcore, nomp, generic')}
    </section>
    <section class="panel">${renderPoolRows(pools)}</section>`;
}

function renderPoolRows(pools) {
  if (!pools.length) return `<p class="empty-note">${escapeHtml(t('pools.empty'))}</p>`;
  return `<div class="pool-grid">${pools.map((pool) => `
    <article class="pool-card">
      <div class="pool-title"><strong>${escapeHtml(pool.poolName)}</strong>${statusPill(pool.reachable ? t('status.online') : t('status.catalog'), pool.reachable ? 'safe' : 'neutral')}</div>
      <dl>
        <div><dt>${escapeHtml(t('pools.miners'))}</dt><dd>${escapeHtml(pool.miners ?? '—')}</dd></div>
        <div><dt>${escapeHtml(t('pools.poolHashrate'))}</dt><dd>${escapeHtml(pool.poolHashrate ?? '—')}</dd></div>
        <div><dt>${escapeHtml(t('pools.networkHashrate'))}</dt><dd>${escapeHtml(pool.networkHashrate ?? '—')}</dd></div>
        <div><dt>${escapeHtml(t('pools.height'))}</dt><dd>${escapeHtml(pool.blockHeight ?? '—')}</dd></div>
      </dl>
      <small>${escapeHtml(pool.message)}</small>
    </article>`).join('')}</div>`;
}

function renderSettings() {
  return `
    <section class="panel settings-panel">
      <h2>${escapeHtml(t('settings.title'))}</h2>
      <div class="settings-grid">
        <div><span>${escapeHtml(t('settings.language'))}</span><strong>${escapeHtml(localeNames[appState.locale])}</strong></div>
        <div><span>${escapeHtml(t('settings.platform'))}</span><strong>${escapeHtml(appState.bootstrap.platform)}</strong></div>
        <div><span>${escapeHtml(t('settings.version'))}</span><strong>${escapeHtml(appState.bootstrap.version)}</strong></div>
        <div><span>${escapeHtml(t('settings.repo'))}</span><strong>${escapeHtml(appState.bootstrap.repoUrl)}</strong></div>
      </div>
    </section>
    <section class="panel">
      <h3>${escapeHtml(t('settings.localFiles'))}</h3>
      <ul class="plain-list">
        <li><code>pools.local.json</code> — ${escapeHtml(t('settings.poolLocal'))}</li>
        <li><code>wallet.config.json</code> — ${escapeHtml(t('settings.walletLocal'))}</li>
        <li><code>*.sqlite</code>, <code>*.csv</code>, <code>*.log</code> — ${escapeHtml(t('settings.runtimeLocal'))}</li>
      </ul>
    </section>`;
}

function render() {
  const pageHtml = {
    dashboard: renderDashboard,
    monitor: renderMonitor,
    history: renderHistory,
    curves: renderCurves,
    pools: renderPoolsPage,
    settings: renderSettings
  }[appState.activePage]();
  renderShell(pageHtml);

  const dryButton = document.getElementById('dryRunCheck');
  if (dryButton) dryButton.addEventListener('click', runDryCheck);
  const historyInput = document.getElementById('historyFilter');
  if (historyInput) historyInput.addEventListener('input', (event) => {
    appState.historyFilter = event.target.value;
    render();
  });
  const poolButton = document.getElementById('syncPoolsPage');
  if (poolButton) poolButton.addEventListener('click', syncPoolsAndRender);
}

async function runDryCheck() {
  const wallet = getWallet();
  appState.dryRun = await window.pearlGuard.dryRunSweepCheck(wallet);
  appState.transferRequests = appState.dryRun.transferRequests || 0;
  render();
}

async function syncPoolsAndRender() {
  appState.poolSync = await window.pearlGuard.syncPools({ fixtureOnly: true });
  render();
}

async function boot() {
  appState.bootstrap = await window.pearlGuard.getBootstrap();
  appState.locale = chooseLocale(appState.bootstrap.locale, navigator.language);
  appState.messages = await loadMessages(appState.locale);
  appState.poolSync = await window.pearlGuard.syncPools({ fixtureOnly: true });
  appState.dryRun = await window.pearlGuard.dryRunSweepCheck(appState.bootstrap.demoState.wallet);
  appState.transferRequests = appState.dryRun.transferRequests || 0;
  render();
}

window.__pearlguardReady = boot();
window.__pearlguardSelfTest = async () => {
  await window.__pearlguardReady;
  const poolSync = await window.pearlGuard.syncPools({ fixtureOnly: true });
  const dryRun = await window.pearlGuard.dryRunSweepCheck(appState.bootstrap.demoState.wallet);
  const labels = Array.from(document.querySelectorAll('.nav-button')).map((node) => node.textContent.trim()).join('|');
  return {
    ok: Boolean(document.querySelector('.app-shell') && poolSync.observations.length >= 3 && dryRun.mode === 'dry-run' && labels.includes(t('nav.dashboard'))),
    locale: appState.locale,
    poolCount: poolSync.observations.length,
    transferRequests: dryRun.transferRequests,
    decision: dryRun.decision
  };
};
