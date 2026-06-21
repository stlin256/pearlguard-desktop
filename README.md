![PearlGuard Desktop banner](assets/banner.svg)

# PearlGuard Desktop

PearlGuard Desktop is a lightweight cross-platform desktop companion for local Pearl Wallet monitoring, address history review, balance curve verification, and mining pool intelligence.

The application starts in **normal local mode**. It does not load demo balances by default. Demo data is only used when the app is launched with `--demo` or when a user intentionally loads demo data from the interface.

## Features

- Cross-platform desktop app for Windows, macOS, and Linux.
- Windows release target is a portable single-file `.exe`.
- Internationalization for all UN official languages: Arabic, Chinese, English, French, Russian, and Spanish.
- Startup language selection from the operating system locale, with an in-app override.
- Local wallet status refresh through read-only JSON-RPC configuration.
- CSV audit import for address history browsing and balance curve verification.
- Dashboard for wallet health, sweep-readiness checks, recent audit events, and mining pool intelligence.
- Miner-focused pool sync layer with Miningcore-style, Yiimp-style, NOMP-style, Zpool-style, and generic JSON adapters.
- Local-first privacy model with ignored runtime config, logs, audit databases, and CSV exports.
- Release workflow configured for GitHub Releases from this repository.

## Normal Local Mode

On first launch the dashboard may show that local setup is required. That is expected when no local config or imported audit data exists yet.

Create a local wallet config in the application config folder:

```json
{
  "version": 1,
  "mode": "read-only",
  "walletLabel": "Local Pearl Wallet",
  "network": "mainnet",
  "rpcHost": "127.0.0.1",
  "rpcPort": 8335,
  "rpcUsername": "",
  "rpcPassword": "",
  "reservePRL": 0.02,
  "thresholdPRL": 1.1,
  "pollSeconds": 10
}
```

Then use **Refresh wallet** for read-only status. The desktop app does not broadcast transactions.

You can also import an existing Pearl Auto Sweep CSV audit file from the Address History page. Imported records are stored in local runtime state and are ignored by Git.

## Mining Pool Intelligence

PearlGuard includes a pool sync layer for mainstream mining-pool API styles. The public repository ships only safe example configuration. Private pool endpoints, API keys, and wallet addresses belong in ignored local files.

Supported adapter families:

- `zpool-status`
- `yiimp-status`
- `miningcore-pool`
- `nomp-pool`
- `generic-json`

Copy `data/pools.example.json` to `pools.local.json` in the application config folder before enabling real endpoints.

## Development

```powershell
npm install
npm run lint
npm test
npm run test:e2e
npm start
```

The end-to-end smoke test launches the desktop app in normal local mode and asserts that no transfer operation is requested.

## Release

The repository includes `.github/workflows/release.yml` for tagged releases:

```text
v0.2.0
```

The release workflow builds Windows, macOS, and Linux artifacts and publishes a draft GitHub Release for `stlin256/pearlguard-desktop`. The Windows target is a portable single-file executable.

## Local Runtime Files

The following files are intentionally ignored:

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
