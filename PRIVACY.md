# Privacy

PearlGuard Desktop is designed as a local-first desktop application.

## Data Kept Local

- Wallet snapshots.
- Address observations.
- Pool observations.
- Audit events.
- Local configuration.
- Wallet unlock password, only when the user saves it in Settings.

## Data Not Stored

- Plaintext private keys.
- Seed phrases.
- Repository copies of private wallet addresses, pool API keys, logs, or runtime exports.

## Local Encryption

Runtime JSON files are written as AES-GCM encrypted envelopes using a key derived from the local machine and user context. This protects local config from casual inspection, but it is not a replacement for full-disk encryption or an operating-system keychain.

## Network Access

Pool sync may contact endpoints configured by the user. SafeTrade quote refresh contacts the public exchange endpoint for PRL/USDT. When a proxy is configured, external backend HTTP requests use it; loopback wallet RPC remains direct to avoid sending local wallet RPC credentials through a proxy. The application does not require a project-operated server.

## Repository Hygiene

Runtime files are ignored by Git. Before publishing releases, run:

```powershell
npm run privacy:scan
```

