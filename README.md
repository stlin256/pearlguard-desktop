![PearlGuard Desktop banner](assets/banner.svg)

# PearlGuard Desktop

PearlGuard Desktop is a lightweight desktop workspace for Pearl Wallet operators and miners. It focuses on local wallet status, guarded threshold decisions, address-history review, balance curves, and mining-pool observations.

## Features

- Lightweight Wails/WebView desktop app for Windows, macOS, and Linux.
- Windows build outputs a portable single-file `.exe` that uses the system WebView runtime.
- Guided startup settings for wallet unlock password, optional transfer policy, mining-pool sync, proxy, network, and language.
- Internationalized interface for Arabic, Chinese, English, French, Russian, and Spanish with system-language startup detection.
- Continuous monitor controls for scheduled wallet refresh and threshold evaluation.
- Pearl Wallet v100/SPV-compatible read-only status fallback using wallet balance and block-count RPC when chain-info RPC is inactive.
- SafeTrade PRL/USDT live quote card with manual refresh and proxy-aware HTTP access.
- Address-history CSV import with local balance-curve verification.
- Miner-focused pool sync for the Hashrate.no PRL pool index and supported public aggregate APIs from listed PRL pools.
- Encrypted local-first storage model for settings, audit state, and private endpoint data.

## Configuration

Use **Settings** inside the app. The startup guide keeps the normal path short: wallet unlock password, optional destination address, threshold, mining-pool sync, proxy address, network, and language. Advanced local RPC fields remain available in a folded section for custom nodes.

PearlGuard writes runtime settings to the local application data folder as encrypted JSON envelopes. Existing plaintext local config can still be read and is migrated the next time the app saves settings. The public repository ships only examples and test fixtures.

Recommended normal settings:

```json
{
  "network": "mainnet",
  "thresholdPRL": 1.1,
  "autoTransferEnabled": false,
  "miningPoolEnabled": false,
  "proxyUrl": "",
  "uiLanguage": "system",
  "readOnly": true
}
```

The desktop monitor records local readiness decisions and keeps the current preview from requesting transaction broadcasts.

## Mining Pool Intelligence

PearlGuard includes a Hashrate.no PRL pool-index adapter plus public aggregate API adapters for Kryptex, Pearlhash, LuckyPool, AlphaPool, Akoya, BaikalMine, JETSKI, MinePRL, Himpool, GrandPool, HeroMiners, and NushyPool. Each adapter normalizes the fields that are publicly available, such as miners, pool hashrate, network hashrate, chain height, fee, and payout mode.

Generic adapters remain available for custom local endpoints:

- `zpool-status`
- `yiimp-status`
- `miningcore-pool`
- `nomp-pool`
- `generic-json`

Pool endpoints are configured from the GUI and stored locally in encrypted runtime config. Do not commit private endpoints, API keys, wallet addresses, exported records, or raw pool responses.

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

## Proxy And Market Data

A configured proxy is applied to external backend HTTP requests such as pool sync and SafeTrade quotes. Local loopback wallet RPC stays direct so wallet RPC credentials are not sent through a proxy.

SafeTrade quote refresh is best-effort. If an exchange endpoint blocks the current network, configure a trusted proxy in Settings.

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
