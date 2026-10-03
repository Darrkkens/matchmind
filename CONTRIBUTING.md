# Contributing to MatchMind

Thanks for your interest! MatchMind is a local-first Brazilian football analyst: a Go API, a Vue 3 frontend and an open-weight model (Gemma) running through Ollama. Contributions of any size are welcome: bug reports, data-quality fixes, translations, tests, docs and features.

> 🇧🇷 Contribuições em português são bem-vindas. Issues e pull requests podem ser escritos em português ou inglês.

## Before you start

- Look for an existing issue, or open one to discuss larger changes before writing code.
- Issues labeled `good first issue` are scoped for newcomers.
- Please read the [Code of Conduct](CODE_OF_CONDUCT.md).

> **Hacktoberfest 2026:** pull requests no longer count toward Hacktoberfest rewards, so there is no PR quota to meet here. We value one thoughtful, tested contribution over many small ones. Low-effort or AI-generated PRs that were not reviewed and tested by their author will be closed.

## Development setup

Requirements: Go 1.25+, Node.js 22.12+, and (only for the chat) [Ollama](https://ollama.com) with `gemma3:4b`. Docker is optional (PostgreSQL response cache).

```sh
cp .env.example .env          # optional; defaults work without it
ollama serve & ollama pull gemma3:4b
cd backend && go run ./cmd/server     # API on http://127.0.0.1:8080
cd frontend && npm install && npm run dev   # UI on http://localhost:5173
```

The dashboard works without Ollama; only the chat needs it. No API key is required.

## Checks to run before opening a PR

```sh
cd backend
gofmt -l .            # must print nothing
go vet ./...
go test -race ./...

cd ../frontend
npm ci
npm run build         # runs vue-tsc type checking first
```

CI runs the same checks on every pull request.

## Guidelines

- **Never invent football data.** Missing data must stay missing (`null`, empty list, "Indisponível"). Never fill gaps with fabricated zeros, guesses or model output.
- **Respect data licenses and terms.** Only add sources with a documented API and terms that allow the use. Do not scrape sites or call private endpoints. See "Adding another authorized provider" in the README.
- **Never commit secrets.** Keys go in `.env` (git-ignored). Use obviously fake values in tests (`test-key`).
- **Tests use fixtures**, never real network calls or a real model. Use `httptest` servers like the existing tests.
- **Keep the UI in Brazilian Portuguese** and the code, comments and docs in English.
- Match the surrounding code style; keep changes focused; update the README when behavior or configuration changes.
- Crest images need verified provenance and license; record them in `frontend/public/crests/manifest.json` and `credits.html` (see `docs/club-crests.md`).

## Commit and PR

- Write clear commit messages describing *why*.
- Fill in the pull request template, including how you tested.
- If you used AI assistance, say so in the PR and confirm you reviewed and tested the result.

## Reporting security issues

Please do not open public issues for vulnerabilities. See [SECURITY.md](SECURITY.md).
