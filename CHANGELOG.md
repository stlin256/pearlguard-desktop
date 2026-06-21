# Changelog

## 0.3.0

- Migrated the primary desktop build from Electron to Wails for a smaller, faster single-file Windows executable.
- Added GUI-based wallet, guard policy, language, refresh, and mining-pool settings.
- Reworked the dashboard around wallet health, sweep readiness, pool status, and recent audit activity.
- Added continuous monitor controls for scheduled local wallet refresh and threshold evaluation.
- Added a guided startup settings flow with wallet password, optional transfer policy, mining-pool sync, proxy, network, and language controls.
- Added encrypted local JSON storage for runtime settings and audit state.
- Added Pearl Wallet v100/SPV read-only RPC fallback when chain-info RPC is inactive.
- Added SafeTrade PRL/USDT quote refresh with proxy-aware external HTTP access.
- Removed demo-first and file-editing wording from the main user flow.
- Added Hashrate.no PRL pool-index parsing and public aggregate API adapters for listed PRL pools.

## 0.2.0

- Defaulted the app to local runtime data instead of demo data.
- Added cross-platform release workflow and privacy scanning.
