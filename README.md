![PearlGuard Desktop banner](assets/banner.svg)

# PearlGuard Desktop

PearlGuard Desktop is a lightweight desktop workspace for Pearl Wallet operators and miners. It focuses on local wallet status, guarded threshold decisions, address-history review, balance curves, and mining-pool observations.

## Features

- Lightweight Wails/WebView desktop app for Windows, macOS, and Linux.
- Windows build outputs a portable single-file `.exe` that uses the system WebView runtime.
- GUI settings for wallet RPC, guard thresholds, reserve amount, language, refresh timing, and mining-pool endpoints.
- Internationalized interface for Arabic, Chinese, English, French, Russian, and Spanish.
- Continuous monitor controls for scheduled wallet refresh and threshold evaluation.
- Address-history CSV import with local balance-curve verification.
- Miner-focused pool sync for Miningcore-style, Yiimp-style, NOMP-style, Zpool-style, and generic JSON endpoints.
- Local-first storage model for settings, audit state, and private endpoint data.

## Configuration

Use **Settings** inside the app to configure wallet RPC access, guard policy, language, and mining-pool endpoints. PearlGuard writes runtime settings to the local application data folder. The public repository ships only examples and test fixtures.

Recommended wallet settings:

```json
{
  "walletLabel": "Local Pearl Wallet",
  "network": "mainnet",
  "rpcHost": "127.0.0.1",
  "rpcPort": 8335,
  "reservePRL": 0.02,
  "thresholdPRL": 1.1,
  "refreshSeconds": 30,
  "poolSyncSeconds": 120,
  "readOnly": true
}
```

The desktop monitor records local readiness decisions and keeps automated test builds from requesting transaction broadcasts.

## Mining Pool Intelligence

PearlGuard supports these adapter families:

- `zpool-status`
- `yiimp-status`
- `miningcore-pool`
- `nomp-pool`
- `generic-json`

Pool endpoints are configured from the GUI and stored locally. Do not commit private endpoints, API keys, wallet addresses, or exported records.

## Development

Install Go, Node.js, npm, and Wails v2. On Windows, WebView2 must be available.

```powershell
npm install
npm run lint
npm test
npm run test:e2e
npm run dist
```

Useful commands:

```powershell
npm start                 # Wails dev mode
npm run build:wails-assets # Prepare embedded frontend assets
npm run privacy:scan       # Scan the repository for sensitive patterns
```

## Release

Tagged releases are built by `.github/workflows/release.yml`.

```text
v0.3.0
```

Release builds run lint, unit tests, Wails smoke checks, privacy scanning, and platform packaging. Draft GitHub Releases are published for `stlin256/pearlguard-desktop`.

## Local Runtime Files

The following file types are intentionally ignored:

```text
pools.local.json
wallet.config.json
addresses.local.json
*.csv
*.log
*.sqlite
*.local.json
```

Do not commit real wallet addresses, transaction ids, passphrases, pool API keys, local usernames, private paths, or runtime logs.

## Disclaimer

PearlGuard Desktop is not an official Pearl Wallet product. It can display cryptocurrency wallet and mining information. Use it at your own risk.
