# Security Policy

MatchMind is designed as a **trusted local application**: the API binds to `127.0.0.1` by default and is not a hardened multi-user public service (see "Security and operational scope" in the README).

## Reporting a vulnerability

Please **do not open a public issue**. Use GitHub's private vulnerability reporting: **Security → Report a vulnerability** on this repository. Include steps to reproduce, the affected version or commit, and the impact you observed.

You can expect an initial response within a few days. Fixes are released on the `main` branch.

## Secrets

API keys (`API_FUTEBOL_KEY`, `API_FOOTBALL_KEY`) and database credentials live only in `.env`, which is git-ignored, and are never sent to the browser. If you accidentally commit a secret, revoke it at the provider immediately; rewriting history alone is not enough.
