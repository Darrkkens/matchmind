# MatchMind

**Your local AI analyst for Brazilian football.** · *Seu analista de futebol brasileiro com IA local.*

[![CI](https://github.com/Darrkkens/matchmind/actions/workflows/ci.yml/badge.svg)](https://github.com/Darrkkens/matchmind/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-c4f66b.svg)](LICENSE)

MatchMind is a Brasileirão dashboard with a Vue 3 interface (in Brazilian Portuguese), a Go REST API and an **open-weight model (Gemma) running locally through Ollama**. Pick a club and see the league table, recent form, the last matches with statistics, scorers, cards and lineups, 20+ seasons of Série A history, and ask an AI analyst questions that are answered only from that data.

![MatchMind dashboard](docs/screenshots/dashboard.png)

- **No account, no cloud AI, no API key required.** Optional keys only add more statistics.
- **Facts first.** Every number comes from a cited source or a deterministic Go calculation; the AI separates `FATO` from `INTERPRETAÇÃO` and must say when data is missing.
- **Honest about coverage.** Missing data stays missing; the UI shows sources, retrieval times and notices.

## Contents

[Features](#features) · [Quick start](#quick-start) · [Data sources](#data-sources) · [Next-match simulation](#next-match-simulation) · [Open-source AI](#open-source-ai) · [Architecture](#architecture) · [Configuration](#configuration) · [API](#api) · [Security](#security-and-operational-scope) · [Tests](#tests-and-builds) · [Contributing](#contributing) · [Hacktoberfest 2026](#hacktoberfest-2026) · [License](#license)

## Features

- **Brasileirão table** computed from the season's results, with all 20 club crests, indicative Libertadores / Sul-Americana / relegation zones, and click-to-analyze.
- **Club search** by name, nickname (`Galo`, `Timão`, `Verdão`…) or a public team URL (parsed locally as text, never fetched).
- **Recent form**: W/D/L sequence, goals, average and points percentage.
- **Next scheduled match** (opponent, date, kick-off time, home or away) and the **last five matches** with possession, shots, shots on target, corners, fouls and cards; **goal scorers** (with assists, penalties and own goals), **bookings**, **stadium** and **referee**.
- **"Ver escalações"**: both team sheets on demand — formation, coach, starters, substitutes, minutes, ratings, goals and cards.
- **Squad** with appearances, goals, assists and ratings.
- **Série A history 2003–2024**: titles, season-by-season position/points/coaches/averages, the club's top scorers, discipline, and head-to-head records with scorers of the last meetings.
- **Season panel** (optional, local FBref CSV export): attack, defense and discipline per match against the league average with ranks, home/away records, top scorer, assists leader, goalkeeper, average attendance and home stadium.
- **Next-match preview and simulation**: both clubs' last five results, home/away records and season numbers side by side, and a Monte Carlo simulation (50, 1,000 or 10,000 matches) with win/draw/loss percentages, expected goals and the most simulated scores. See [Next-match simulation](#next-match-simulation).
- **AI chat** in Portuguese with quick questions, cited context sections and clear limits.
- Responsive dark UI, keyboard navigation, loading/error/offline states; dashboard works without the AI.

<details>
<summary>More screenshots</summary>

| Landing page and table | Lineups | Série A history |
| --- | --- | --- |
| ![Landing](docs/screenshots/landing.png) | ![Lineups](docs/screenshots/lineups.png) | ![History](docs/screenshots/history.png) |

| Real local Gemma answer | Mobile |
| --- | --- |
| ![Chat](docs/screenshots/chat.png) | ![Mobile](docs/screenshots/mobile.png) |

Captured on October 2, 2026 from the running application with real data. The chat image shows an actual Gemma 3 4B response, not a mock. See the [screenshot guide](docs/screenshots/README.md).

</details>

## Quick start

Requirements: **Go 1.25+**, **Node.js 22.12+** and npm. For the chat: **[Ollama](https://ollama.com)** with enough RAM for the model (CPU works, but answers take 1–3 minutes). Docker is optional.

```sh
git clone https://github.com/Darrkkens/matchmind.git
cd matchmind
cp .env.example .env            # optional: defaults work without it

# 1. Local AI (only needed for the chat)
ollama serve                    # skip if the Ollama service is already running
ollama pull gemma3:4b

# 2. Backend (another terminal)
cd backend && go run ./cmd/server

# 3. Frontend (another terminal)
cd frontend && npm install && npm run dev
```

Open **http://localhost:5173**, click a club in the table or search for one.

<details>
<summary>Installing Ollama</summary>

Ollama is a separate application; `npm install` and `go run` do not install it. On Arch/CachyOS: `sudo pacman -S --needed ollama`. For other systems follow the [official instructions](https://docs.ollama.com/linux). GPU acceleration is optional.

</details>

**Optional extras**

| Want | Do |
| --- | --- |
| Match statistics, squads and lineups (free, no key) | Set `ALMANACSTATS=on` in `.env` — read its [terms](#match-statistics-and-lineups-optional) first |
| Statistics from a paid/free-plan API | Set `API_FUTEBOL_KEY` or `API_FOOTBALL_KEY` |
| Keep API responses across restarts (saves quota) | `docker compose up -d` and set `DATABASE_URL` (see `.env.example`) |

The backend loads `.env` automatically from the working directory or its parent; variables already set in the shell win. The log line `loaded environment file` confirms it.

## Data sources

MatchMind never scrapes websites or calls private endpoints. Every outgoing URL is built from a fixed host and an allowlist; browser input never chooses an endpoint.

| Data | Source | License / terms | Required |
| --- | --- | --- | --- |
| Results, table, form | [OpenFootball](https://github.com/openfootball/football.json) | CC0-1.0 | Yes (default) |
| Série A history 2003–2024 | [Brasileirao_Dataset](https://github.com/adaoduque/Brasileirao_Dataset) by Adão Duque ([Kaggle](https://www.kaggle.com/datasets/adaoduque/campeonato-brasileiro-de-futebol)) | GPL-2.0, fetched at runtime, not bundled | On by default |
| Match stats, events, squads, lineups | [AlmanacStats API](https://almanacstats.com/pt-br/api-docs) | Personal, non-commercial use (see below) | Off by default |
| Match stats (alternative) | [API-Futebol](https://www.api-futebol.com.br), [API-Football](https://www.api-football.com) | Their plans and terms; key required | Off |
| Club crests | Wikimedia Commons and official sources | Per file, see [credits](frontend/public/crests/credits.html) | Bundled |

### Results and table: OpenFootball

The default dataset is [Brasileiro Série A 2026](https://raw.githubusercontent.com/openfootball/football.json/master/2026/br.1.json). It supplies team names, dates, rounds and final scores — nothing else. The app uses the last five scored matches dated up to today in that dataset (not other competitions), shows the club's next dated, non-postponed fixture from it, accepts both published score formats, and distinguishes a 0–0 draw from a missing score. Names are matched case- and accent-insensitively, with Brazilian nicknames as aliases.

The table uses the Brasileirão criteria available from scores: points, wins, goal difference, goals scored (head-to-head and disciplinary criteria are not modeled). The dataset is cached in memory for five minutes, with coalesced downloads and a 15-second cooldown after failures; an unavailable source produces an explicit error, never substitute data. Supported leagues: `br.1` (all features), `en.1`, `de.1`, `es.1`, `it.1`, `fr.1` (results and table only).

### Match statistics and lineups (optional)

Sources are used in this order when enabled: **AlmanacStats** (`ALMANACSTATS=on`) → **API-Futebol** (`API_FUTEBOL_KEY`) → **API-Football** (`API_FOOTBALL_KEY`). A match receives data only when round/date, both clubs and the final score agree with OpenFootball; otherwise its statistics stay `null`. Failures never block the dashboard: `data_metadata.statistics_notice` explains them.

- **AlmanacStats** (free, keyless) provides team statistics, goal and card events, stadium, referee, lineups and squads. MatchMind requests only the opened club (its profile plus up to five match pages), at most one request per second as the provider asks, and credits it in the UI.
  > **Terms of use.** The API page invites free use with caching and attribution, but the site's [Terms and Conditions](https://almanacstats.com/terms-and-conditions) allow only personal, non-commercial use and forbid extracting substantial parts of the data, redistribution, derivative datasets and using the platform to "train, fine-tune or evaluate" AI models without a licence. MatchMind only passes the data to a local model as answer context and trains nothing, but a strict reading could still apply. It is **off by default**; ask AlmanacStats for written permission before using it beyond personal use.
- **API-Futebol** (championship 10): rounds are linked by number. Test keys (`test_…`) return the same sample match for every request, so the UI labels them "dados de exemplo" and the backend appends a warning to AI answers that use them.
- **API-Football** (league 71): the season fixture list is cached for six hours and finished matches forever, to stay within the free plan's 100 requests/day.

**Response cache.** Every statistics response is cached: finished matches never expire, open rounds and profiles after hours, 404s after 6 hours. With `DATABASE_URL`, the cache lives in PostgreSQL (`api_responses` table, `jsonb`) and survives restarts; `docker compose up -d` starts PostgreSQL 18 on `127.0.0.1:5435` with credentials from `.env`. Without it, the cache is in memory.

### Série A history (2003–2024)

All four CSV files of the dataset are used: results, stadiums, coaches and formations (required), plus scorers (2014–2024), bookings (2014–2024) and per-match statistics (optional). For each club the snapshot's `history` contains titles, the all-time record, `seasons` (final position, points, record, coaches by matches in charge, formation, home stadium and per-match averages), `top_scorers` (own goals excluded), `discipline`, and head-to-head records with the last five `meetings` (stadium and scorers).

Titles and positions come from each complete season's final table and match the official champions for 2003–2024. Seasons are inferred from dates (2020 ended in February 2021). Averages are computed only where the source actually filled them (2017–2023; 2016 and 2024 rows are mostly zeros). Scorers are shown only when they add up to the final score. Player and coach names appear as recorded by the source (often full legal names). A missing optional file is listed in `history.unavailable`. The dataset is cached for 12 hours; set `BRASILEIRAO_HISTORY=off` to disable it.

### Season statistics: local FBref export (optional)

Set `FBREF_DIR` to a folder of CSV exports of FBref's Série A tables (`times_padrao`, `times_finalizacao`, `times_finalizacao_adversarios`, `times_goleiros`, `times_diversos`, `classificacao`, `classificacao_casa_fora`, `jogos`, `jogadores_padrao`, `jogadores_tempo_jogo`, `jogadores_finalizacao`, `goleiros`). The folder is read once at start-up; nothing is downloaded. Clubs are matched to OpenFootball names through the shared aliases. The snapshot then carries `season_stats` (with `key_players` by on/off impact among regulars and `finishers` with shots, accuracy and goals per shot), `next_opponent_season` and `next_match_availability`: likely card suspensions for both sides (a red card, or the 3rd/6th/9th yellow, in the last league match, using the statistics source's cards for that match) and players one yellow away. Tribunal decisions and injuries are not known. The stadium becomes the venue of most home matches, and the UI shows the data date. FBref/Sports Reference terms restrict scraping and reuse: keep these files out of the repository and use them only locally. A missing file or column disables the source with a warning in the log.

### Club crests

All 20 Série A 2026 crests are bundled locally; nothing is fetched from third parties while using the dashboard. Unknown clubs and failed images fall back to initials. Provenance, authors, licenses and SHA-256 hashes are in the [crest manifest](frontend/public/crests/manifest.json) and the [credits page](frontend/public/crests/credits.html).

Seventeen files are public domain on Wikimedia Commons and the Corinthians image is CC BY-SA 4.0 (by Fratino.koko). **The current RB Bragantino (Red Bull trademark, from the club's official site) and Vasco (marked as restricted content on Portuguese Wikipedia) crests are not freely licensed**; they are included only to identify the clubs and will be removed if the rights holders object. Crest files are **not covered by the MIT license**. See [crest maintenance](docs/club-crests.md).

### Adding another authorized provider

1. Verify the provider's documented API, license, terms, quotas and redistribution rights. Do not scrape or reverse engineer private endpoints.
2. Implement `FootballProvider` (`backend/internal/football/provider.go`), or `StatisticsSource` / `SquadSource` / `LineupSource` to enrich OpenFootball results.
3. Return completed matches newest first, propagate `context.Context`, set timeouts, and use `nil`/empty values for missing data — never fabricated zeros.
4. Wire it in `cmd/server/main.go` behind explicit configuration; keep credentials in backend environment variables only.
5. Add `httptest` fixture tests and document provenance, freshness and limits.

## Next-match simulation

For the selected club's next league fixture, MatchMind compares both sides and estimates the result by **simulating the match many times**. A deterministic Go model does the math, the local AI reviews it, and the draws are counted in Go, so the percentages are reproducible and never invented by the language model.

![Next-match card with the simulation](docs/screenshots/simulation.png)

### What the page shows

The **Próximo jogo** card on the dashboard has three parts:

1. **The fixture**: date, local kick-off time, round, opponent and whether the club plays at home or away (from the OpenFootball feed).
2. **Como chegam** (how both sides arrive): last five results of each club, each one's record in its role for this match (home vs. away, points per match), likely **card suspensions** and players **one yellow away** (*pendurados*), and season numbers side by side (goals for and against per match, shots on target, goalkeeper save %), with the better value highlighted. Season numbers need the optional [FBref export](#season-statistics-local-fbref-export-optional).
3. **Simulação do jogo**: pick **50, 1,000 or 10,000** simulated matches and press *Simular*. While the local AI works, short status phrases rotate with a seconds counter. The result shows:
   - win / draw / loss percentages from the selected club's side, with a proportional bar;
   - the **AI analyst's reason** for its adjustment (marked as generated text);
   - expected goals for both sides, the last five results and the five most simulated scores;
   - the **factor table**: how each input moved each side's expected goals (see below);
   - notes about the data (e.g. rest only counts league matches; 50 runs are noisy);
   - **Explicar com a IA**, which sends the result to the chat for a written explanation.

### How it works

`POST /api/simulate` (and the button) runs three steps:

1. **Base model (Go).** Expected goals start from each club's season attack and defense in the OpenFootball table, shrunk toward the league average so early seasons are not extreme. Bounded factors then multiply each side's expected goals.
2. **AI analyst (Ollama, optional).** The local model reviews a short summary of all those facts and may give one club a small extra edge (details below).
3. **Draws (Go).** Each simulated match varies both sides' strength (log-normal, about ±20%) and draws the goals from a Poisson distribution. The random seed comes from the fixture and the run count, so the same request always returns the same numbers.

| Factor (UI label) | Source | Effect |
| --- | --- | --- |
| Força na temporada (base) | OpenFootball table | Starting expected goals of each side, shown in goals |
| Mando de campo (média da liga) | This season's matches | League-wide home edge, e.g. home +14% / away −14% |
| Estatísticas da temporada | FBref export | Goals and shots on target, for and against, vs. the league average; blended 50/50 with the base, ±15% max |
| Campanha em casa e fora | FBref export | Each club's scoring in this match's role compared with its own average, *beyond* the league home edge, ±15% max |
| Últimos 5 jogos | OpenFootball | Recent scoring/conceding blended (30%) into the season rates, ±20% max |
| Sequência e descanso | OpenFootball dates | 2 days or fewer −8%, 3 days −4%, 3+ matches in 10 days −3%; a tired side also concedes a bit more |
| Suspensões por cartão | AlmanacStats cards + FBref totals | A red card or the 3rd/6th/9th yellow in the last league match; weighted by the player's minutes and on/off impact, ±12% per side max |
| Lesionados | — | Shown but **not used**: no configured source provides injuries |
| Confrontos históricos | Série A dataset (since 2003) + this season | Points share against this opponent, weighted by sample size (15 meetings carry half the weight), ±15% max |
| Análise da IA (Gemma local) | Ollama | `leve` 3% or `moderado` 6% for the club it picks, or nothing |
| Aleatoriedade | — | Day-to-day ±20% noise in every simulated match |

Reading the table: **+x% / −x%** is the change to that side's expected goals; **0%** means the factor ran but changed nothing, and its text says why (e.g. nobody played within 3 days, a balanced or tiny head-to-head sample); **—** means not used. The base row shows goals instead of a percentage.

### The AI analyst and its safeguards

With `use_ai: true` (always on in the UI) the backend sends the model the base expected goals, every applied factor with its explanation, both clubs' season metrics and each club's record **in its role for this match**, written as sentences with club names ("Grêmio FBPA fora: 14 jogos, 0 vitórias, 4 empates, 10 derrotas, 8 gols marcados e 22 gols sofridos"). The model answers in a constrained JSON format:

- `reason` first (at most two sentences in Portuguese), then
- `favored`: the exact name of one of the two clubs, or `"nenhum"`;
- `strength`: `leve` (3%) or `moderado` (6%).

Go turns that choice into the multipliers; the model never writes numbers for the simulation. The adjustment is **discarded** (and the page says so) when the reason does not name the chosen club, cites a number that is not in the facts, or argues for the other side. If Ollama is offline or too slow, the simulation runs without the AI step. Analyst calls share the API's single inference slot with the chat (HTTP 429 while busy).

These checks catch invented numbers and contradictions, not every weak argument: on CPU, `gemma3:4b` is a modest analyst, which is why its weight is kept at 3–6%.

### Explaining a result in the chat

*Explicar com a IA* asks the chat about the simulation. The backend reuses the AI-reviewed result the user just ran (kept in memory for 15 minutes for the same fixture), sends the model a compact context with ready-made sentences ("SE Palmeiras vence em 59,6%…", "SE Palmeiras 1 x 0 EC Bahia: 12,3%") and factors ordered by weight, and prints the **exact** result above the model's answer. The model only explains which factors weighed most. Questions such as "qual a chance do Palmeiras?" typed in the chat use the same path (a quick simulation without the AI step when none was run).

### Example

Palmeiras x Bahia (round 29), 10,000 simulations, real data and `gemma3:4b`, October 3, 2026:

| | Palmeiras | Draw | Bahia |
| --- | --- | --- | --- |
| Result | **59.6%** | 22.6% | 17.8% |
| Expected goals | 1.83 | | 0.86 |

Most simulated scores: 1–0 (12.3%), 2–0 (10.8%), 1–1 (10.7%), 2–1 (9.1%), 0–0 (7.2%). Biggest factors: home advantage (+14% / −14%), Bahia's away scoring (+11%) and head-to-head (21 meetings, 11W 8D 2L: +3% / −3%); the AI analyst favored Palmeiras (`leve`) citing goals conceded per match (0.75 vs. 1.29).

### Performance and hardware

On an 8-core CPU without GPU, with `gemma3:4b`: the simulation without AI takes a second or two (data is cached); with the AI step about 25–90 s; a chat explanation about 100–160 s (`OLLAMA_TIMEOUT=240s` covers both). `gemma3:12b` reasons better but needs about 9 GB of **free** RAM; with less, the operating system kills the model process and the simulation continues without the AI step. A 7–8B model is a middle ground on 16 GB machines.

### Limitations

- A statistical estimate from league data, not a forecast: no injuries, tribunal (STJD) decisions, lineups, weather or news.
- Rest and suspensions count only league matches in the dataset; cups and other competitions are not included.
- FBref numbers are a dated local copy (the date is shown); suspensions assume the season totals and the last match's cards are complete.
- 50 simulations vary by several points between fixtures; use 1,000 or 10,000 for stable percentages.

## Open-source AI

[Ollama](https://github.com/ollama/ollama) runs [Gemma 3](https://ai.google.dev/gemma/docs/core/model_card_3) (default [`gemma3:4b`](https://ollama.com/library/gemma3:4b)) through its documented [`POST /api/chat`](https://docs.ollama.com/api/chat), with structured JSON output and a low temperature. With the default loopback `OLLAMA_URL`, questions and context never leave your machine. Gemma weights follow the [Gemma terms](https://ai.google.dev/gemma/terms) and are not included in this repository.

How answers stay grounded:

- The browser cannot send context. For each question the backend rebuilds the club snapshot and sends the model a compact JSON context (no URLs or artwork), including the full table, history, or squad **only when the question is about them**, so CPU inference stays within the timeout.
- The system prompt treats all data as untrusted evidence, forbids prior football knowledge and invented facts, and explains coverage rules (e.g. player totals are not recent-match goals; own goals belong to the other side).
- Ollama returns `facts`, `interpretation` and `sources_used`; Go validates the structure and the allowed source names and adds the `FATO` / `INTERPRETAÇÃO` labels itself. When statistics come from a test key, Go appends the sample-data warning instead of trusting the model to mention it.

These are behavioral controls, not a guarantee: a 4B model can still omit or misread data. Check answers against the visible numbers. Larger models (`OLLAMA_MODEL=gemma3:12b`) follow the instructions better if your hardware allows.

## Architecture

```mermaid
flowchart LR
    User --> Vue[Vue 3 + TypeScript]
    Vue --> API[Go REST API]
    API --> Team[TeamService]
    API --> Chat[ChatService]
    Team --> OF[OpenFootball results]
    OF --> Stats[Statistics / squad / lineups source]
    OF --> Hist[Série A history CSVs]
    Stats --> Cache[(Response cache: memory or PostgreSQL)]
    Chat --> Ctx[Context builder]
    Team --> Ctx
    Ctx --> Ollama --> Gemma
```

```text
matchmind/
├── backend/
│   ├── cmd/server/            # Configuration, .env loading, HTTP server
│   └── internal/
│       ├── api/               # Routes, validation, CORS, rate and concurrency limits
│       ├── football/          # Providers, models, table, form, history, sources
│       ├── footballtest/      # Fictional in-memory provider used only by tests
│       ├── service/           # Team snapshots and chat orchestration
│       ├── store/             # PostgreSQL response cache (pgx)
│       └── ai/                # Ollama client, system prompt, context builder
├── frontend/
│   ├── public/crests/         # Club crests, manifest and credits
│   └── src/                   # App, components, API client, types, styles
├── docs/                      # Testing notes, crest maintenance, research, screenshots
├── .github/                   # CI, issue and pull request templates
├── docker-compose.yml         # Optional PostgreSQL
└── .env.example
```

## Configuration

| Variable | Default | Meaning |
| --- | --- | --- |
| `API_ADDR` | `127.0.0.1:8080` | API listening address |
| `OPENFOOTBALL_LEAGUE` | `br.1` | League file code from the allowlist |
| `OPENFOOTBALL_SEASON` | `2026` | Season directory, e.g. `2026` or `2026-27` |
| `BRASILEIRAO_HISTORY` | `on` | `off` disables the Série A history download (`br.1` only) |
| `ALMANACSTATS` | `off` | `on` enables AlmanacStats statistics, squads and lineups (`br.1`); read its terms |
| `FBREF_DIR` | — | Folder of a local FBref CSV export: season panel, key players, finishers, suspensions and the simulation's season factors ([details](#season-statistics-local-fbref-export-optional)) |
| `API_FUTEBOL_KEY` | empty | API-Futebol key for match statistics (`br.1`) |
| `API_FOOTBALL_KEY` | empty | API-Football key for match statistics (`br.1`) |
| `DATABASE_URL` | empty | PostgreSQL URL for the response cache; empty keeps it in memory |
| `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_PORT` | see `.env.example` | Used by `docker compose` |
| `CORS_ORIGINS` | `http://localhost:5173,http://127.0.0.1:5173` | Exact allowed origins, comma-separated |
| `OLLAMA_URL` | `http://localhost:11434` | Inference host; never supplied by a request |
| `OLLAMA_MODEL` | `gemma3:4b` | Installed Ollama model |
| `OLLAMA_TIMEOUT` | `240s` | Inference timeout (max `10m`); the browser waits up to 260 s |
| `VITE_API_URL` | empty | Frontend API origin without `/api` (`frontend/.env.local` or build env) |

Never commit `.env`; it is git-ignored. If you change the backend port, update the Vite proxy or `VITE_API_URL`, and keep `CORS_ORIGINS` in sync with the frontend origin.

## API

POST requests require `Content-Type: application/json`; unknown fields and trailing JSON are rejected; bodies are limited to 16 KiB. Errors are `{"error": "..."}` in Portuguese.

| Endpoint | Purpose |
| --- | --- |
| `GET /api/health` | `status` (`ok`/`degraded`), `ollama` (daemon up and model installed), `football_provider`, `data_source` |
| `POST /api/team/resolve` `{"input": "Palmeiras"}` | Club snapshot: `team`, `recent_matches` (with `statistics`, `goals`, `cards`, `venue`, `referee`, `lineup_ref`), `recent_form`, `standings`, `squad`, `trophies`, `history`, `data_notice`, `data_metadata` |
| `GET /api/standings` | League table with competition, season, source URL and retrieval time |
| `GET /api/lineups/{ref}` | Both team sheets for a match's numeric `lineup_ref` (400 if invalid, 404 if unavailable) |
| `POST /api/simulate` `{"team_id": "...", "runs": 10000, "use_ai": true}` | Next-fixture simulation: `win_pct`, `draw_pct`, `loss_pct`, expected goals, `top_scorelines`, `factors` (multipliers and reasons), last five results of both sides and `notes`; `runs` 1–10000 (default 50). With `use_ai` it shares the single inference slot (429 when busy). 404 when no fixture is scheduled |
| `POST /api/chat` `{"team_id": "...", "question": "..."}` | `{"answer": "FATO: …\n\nINTERPRETAÇÃO: …", "sources_used": [...]}` |

```sh
curl -s http://localhost:8080/api/team/resolve -H 'Content-Type: application/json' -d '{"input":"Palmeiras"}'
curl -s http://localhost:8080/api/chat -H 'Content-Type: application/json' \
  -d '{"team_id":"TEAM_ID_FROM_RESOLVE","question":"Como foi o desempenho nos últimos 3 jogos?"}'
```

Use the returned `team.id` for chat. The backend is stateless: each question is answered on its own, so follow-ups that depend on a previous answer are not reliable. Common statuses: 400 invalid input, 404 not found, 413 body too large, 415 content type, 429 rate or one-inference-at-a-time limit, 502 invalid upstream data, 503 AI/provider unavailable, 504 timeout.

## Security and operational scope

- User URLs are parsed locally; there is no arbitrary URL retrieval. Lineup references must be numeric.
- Provider data is encoded as JSON separate from the system prompt and treated as untrusted; the model has no tools, shell or HTML rendering, and Vue renders text escaped (no `v-html`).
- Exact-origin CORS, body limits, one concurrent inference and a global 120 requests/minute window protect the local app. Upstream calls have timeouts, size limits, no redirects, failure cooldowns and (AlmanacStats) a 1 request/second limit.
- API keys and database credentials stay in the backend environment and are never sent to the browser.
- The API binds to loopback by default. CORS is not authentication: this is a trusted local application, not a hardened multi-user service. See [SECURITY.md](SECURITY.md).

## Tests and builds

```sh
cd backend
gofmt -l .                 # must print nothing
go vet ./...
go test -race ./...

cd ../frontend
npm ci
npm run build              # vue-tsc type checking + Vite build
```

Tests use local `httptest` servers and fakes — never real APIs or a real model. They cover URL/name resolution and nicknames, form and table calculations, every data source (matching rules, caching, rate limiting, plan/quota errors, test keys), history parsing (seasons, champions, scorers, cards, averages), lineup and name matching, `.env` loading, AI context selection and response validation, API validation, CORS and rate limits. CI runs these checks on every push and pull request. See [testing notes](docs/testing.md) for manual checks.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| API offline | Start `go run ./cmd/server` in `backend/` and check port 8080. |
| "Estatísticas indisponíveis" everywhere | No statistics source is enabled: set `ALMANACSTATS=on` or a key in `.env`, restart the backend and look for `match statistics enabled` in the log. |
| Local AI offline | Start Ollama, run `ollama list`, pull the configured model, then click the connection status. |
| AI answer times out | CPU inference is slow; close heavy apps, keep `OLLAMA_TIMEOUT=240s` or try a smaller model. |
| Club not found | Only clubs in the configured league/season exist; try the club's common name. |
| OpenFootball unavailable | Check access to `raw.githubusercontent.com`, then retry after 15 seconds. |
| Origin is not allowed | Put the exact frontend origin (scheme and port) in `CORS_ORIGINS`. |
| Port already in use | Stop the earlier server; Vite dev and preview both use port 5173. |

## Contributing

Contributions are welcome — bug reports, data-quality fixes, tests, docs and features. Read [CONTRIBUTING.md](CONTRIBUTING.md) and the [Code of Conduct](CODE_OF_CONDUCT.md). Ideas that would help:

- More leagues with history and statistics (Série B, Copa do Brasil) from openly licensed sources.
- Club and match comparison; player pages.
- Reproducible evaluations of answer grounding with local models.
- Carefully bounded conversation context for follow-up questions.

## Hacktoberfest 2026

MatchMind was created for **[Hacktoberfest 2026](https://hacktoberfest.com)**, whose theme is *"AI belongs to everyone"* — building with open-weight models and open-source AI. It was built for the [Weekend Challenge — Build for a Friend](https://dev.to/challenges/hacktoberfest-weekend-2026-10-01) (tag `hf26challenge`), with Gemma running locally as the core of the experience. See the [preparation checklist](docs/hacktoberfest-checklist.md).

In 2026, pull requests no longer count toward Hacktoberfest rewards, so there is no PR quota here — thoughtful, tested contributions are what we are looking for.

### AI assistance

Development was assisted by an AI coding assistant (Claude Code), directed and reviewed by the maintainer. Data, screenshots and the sample model answer come from running the real application; nothing in this repository is presented as real data or a real model answer unless it is one.

## License

Code: MIT — see [LICENSE](LICENSE). Not covered by the MIT license: Gemma weights ([Gemma terms](https://ai.google.dev/gemma/terms)), football data from the sources above (their own licenses and terms; the GPL-2.0 history dataset is downloaded at runtime, not bundled), and club crests (individual licenses or trademarks; see the [credits](frontend/public/crests/credits.html)).
