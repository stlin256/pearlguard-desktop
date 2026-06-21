const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('pearlGuard', {
  getBootstrap: () => ipcRenderer.invoke('app:get-bootstrap'),
  getMessages: (locale) => ipcRenderer.invoke('i18n:get-messages', locale),
  syncPools: (options) => ipcRenderer.invoke('pools:sync', options || {}),
  dryRunSweepCheck: (input) => ipcRenderer.invoke('wallet:dry-run-sweep-check', input || {}),
  appendDemoAudit: (entry) => ipcRenderer.invoke('audit:append-demo', entry || {})
});
