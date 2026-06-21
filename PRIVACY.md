# Privacy

PearlGuard Desktop is designed as a local-first desktop application.

## Data Kept Local

- Wallet snapshots.
- Address observations.
- Pool observations.
- Audit events.
- Local configuration.

## Data Not Stored

- Wallet passphrases.
- Plaintext private keys.
- Seed phrases.

## Network Access

Pool sync may contact endpoints configured by the user. The bundled demo mode and automated tests use local fixtures. The application does not require a project-operated server.

## Repository Hygiene

Runtime files are ignored by Git. Before publishing releases, run:

```powershell
npm run privacy:scan
```

