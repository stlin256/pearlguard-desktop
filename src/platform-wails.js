(function () {
  if (window.pearlGuard) return;

  async function waitForAPI() {
    for (let index = 0; index < 200; index += 1) {
      const api = window.go && window.go.main && window.go.main.App;
      if (api) return api;
      await new Promise((resolve) => setTimeout(resolve, 50));
    }
    throw new Error('PearlGuard desktop runtime is not available.');
  }

  async function call(method, ...args) {
    const api = await waitForAPI();
    if (typeof api[method] !== 'function') throw new Error(`Missing runtime method: ${method}`);
    return api[method](...args);
  }

  window.pearlGuard = {
    getBootstrap: () => call('GetBootstrap'),
    getMessages: (locale) => call('GetMessages', locale),
    syncPools: (options) => call('SyncPools', options || {}),
    getMarketQuote: () => call('GetMarketQuote'),
    dryRunSweepCheck: (input) => call('DryRunSweepCheck', input || {}),
    readWalletStatus: () => call('ReadWalletStatus'),
    importAuditCsv: () => call('ImportAuditCsv'),
    saveSettings: (settings) => call('SaveSettings', settings || {}),
    testRpcConnection: (settings) => call('TestRPCConnection', settings || {}),
    savePoolConfig: (config) => call('SavePoolConfig', config || {})
  };
}());
