# Security Policy

## Supported Versions

PearlGuard Desktop is currently in preview. Security fixes are applied to the latest `main` branch and tagged preview releases.

## Reporting a Vulnerability

Open a GitHub security advisory or a private issue with enough detail to reproduce the problem. Do not publish wallet secrets, passphrases, private transaction ids, API keys, or local runtime files in public issues.

## Sensitive Data Rules

The public repository must not contain:

- Real Pearl wallet addresses.
- Wallet passphrases or password fragments.
- Local usernames or private filesystem paths.
- Private transaction ids.
- Runtime audit CSV files.
- Runtime JSON logs.
- Local SQLite databases.
- Pool API keys or private pool endpoints.

## Local Storage

Runtime JSON files are encrypted at rest with AES-GCM envelopes. Treat the host account and machine as the trust boundary, and use operating-system disk encryption for stronger protection.

## Transfer Safety

The current preview only exposes dry-run transfer checks. Future transfer-capable code must require explicit local configuration, runtime confirmation, and audit events before any broadcast path can run.

