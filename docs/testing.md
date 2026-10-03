# Testing MatchMind

## Original MVP verification — October 2, 2026

| Check | Result |
| --- | --- |
| `gofmt -w cmd internal` | Completed |
| `go test ./...` | Passed |
| `go test -race -cover -timeout 60s ./...` | Passed, no races detected |
| `go vet ./...` | Passed |
| `go build -o /tmp/matchmind-server ./cmd/server` | Passed |
| `npm install` | Completed; audit reported zero vulnerabilities at that time |
| `npm run build` | Type checking and Vite production build passed |
| Chromium desktop and 390 px mobile smoke test | Passed; screenshots captured |
| Real Gemma inference | Not run during the original MVP pass; see integration follow-up below |

Tested with Go 1.25.0, Node 22.23.2 and npm 10.9.8. Package coverage: AI 86.4%, API 90.2%, football 84.7%, services 74.0%. The executable entry point has no unit coverage; it was built and started for the browser smoke test.

The browser test used the real Go API for URL resolution, displayed form metrics, three scorecards, six squad members, expandable statistics, offline AI failure and recovery from an unknown club. An intercepted, explicitly marked test response checked chat rendering and retry without pretending to be Gemma. Switching clubs cleared the prior conversation. No JavaScript exceptions or horizontal overflow at 390 px were observed. Screenshot captures contain no mocked answer.

The build emitted about 83.81 kB JavaScript (32.59 kB gzip) and 16.62 kB CSS (4.59 kB gzip). These sizes describe this build, not a permanent performance guarantee.

## Automated checks

Backend tests use Go's standard `testing` package, fakes and `httptest`. They never need Ollama, a downloaded model or provider credentials. Parser tests include loopback URLs: successful parsing is not a network request.

```sh
cd backend
go test ./...
go test -race -cover -timeout 60s ./...
go vet ./...
```

```sh
cd frontend
npm ci
npm run build
```

The build runs strict TypeScript/Vue type checking before bundling.

## Manual acceptance

1. Start the backend/frontend without Ollama. Health must report `degraded`; the landing page must show the Brasileirão table and club resolve must still work.
2. Check that the table has 20 clubs ordered by points, wins, goal difference and goals scored, with the Mirassol crest visible. Click a row; the club dashboard must open with that row highlighted.
3. Resolve `Mirassol` and `Palmeiras`. Check the crest, the "Nº na tabela" badge, form (V/E/D) and Portuguese dates.
4. Match cards must show "Estatísticas indisponíveis" for score-only data.
5. Resolve an unknown club and an unsupported URL. Expect readable errors and a recoverable search form.
6. Ask a question with Ollama offline. Expect an actionable error and retry; club data stays usable.
7. Start Ollama and pull `gemma3:4b`. Recheck status, then ask `Quem é o treinador?`, `Como está o ataque?` and `Em que posição o clube está na tabela?`.
8. Answers must be in Portuguese, labeled FATO/INTERPRETAÇÃO, cite `standings` for table questions and say that coach and shots are unavailable.
9. Ask an unavailable fact, such as the club's head coach or possession. Expect insufficient data, not a guessed answer.
10. Try an instruction override. Check that no unrelated facts or privileged actions occur. Model refusal quality requires live evaluation; unit tests only verify structural defenses.
11. Send a slow question and switch clubs. The old chat request should be canceled, and its response must not appear under the new club.
12. Resize to 390 px; check horizontal overflow, touch targets, keyboard focus, loading and empty/error states.

## Interpretation of test results

A successful mocked Ollama response verifies the HTTP contract, not Gemma's real output quality or hardware performance. Schema validation checks structure and source names, not factual entailment. The README must not claim live inference was verified unless a real installed model was actually used.

No fake AI fallback is served by the application: when Ollama is offline, it returns an explicit error instead of a canned answer.


## OpenFootball integration verification

The new HTTP adapter tests cover both published score formats, explicit 0–0 draws, absent results, negative/null scores, future fixtures, ordering, accented names, stable provider IDs, missing sections, metadata propagation, cache isolation, concurrent downloads, cancellation, timeouts, redirects, oversized/invalid JSON, 404/429/503 responses, failure cooldown and expired-cache refusal. API tests verify the removed `provider` field is rejected and that a source outage returns an explicit 503.

Actual network validation resolved the Sofascore example URL locally and fetched the public OpenFootball Brazil 2026 dataset. The backend returned `SE Palmeiras`, source `openfootball`, three dated results, computed D–W–D and empty/unavailable profile sections. The API health check confirmed the real provider and installed `gemma3:4b` were available. The first live answer exposed a language/formatting issue; the input was changed to one user JSON envelope with separate CONTEXT/QUESTION fields and the factual-versus-interpretive instructions were strengthened.

After code changes, Go tests (including `-race`) and the Vue/TypeScript production build passed. The external dataset is mutable: numerical results in a later run may differ. Unit tests use invented fixtures and do not call the live source or model.


### Crest verification

All 17 original crest files loaded successfully in Chromium from the local credits page. The Palmeiras dashboard displayed seven images (the selected club plus six home/away appearances), including Grêmio, São Paulo and Botafogo. Fictional demo teams rendered initials, and a simulated image error changed the selected club badge to initials. No JavaScript errors or horizontal overflow appeared at 390 px. Original SVGs were inspected for active elements, event handlers and external href references. Attribution, licenses and SHA-256 hashes are bundled with the assets.

Final frontend build: 88.64 kB JavaScript (34.29 kB gzip), 18.11 kB CSS (4.93 kB gzip), plus approximately 509 kB of original crest assets loaded on demand. `npm run build`, `go vet ./...`, `go build` and `go test -race -cover -timeout 60s ./...` passed. AI schema tests also verify missing fields, unsupported source names and omission of an unjustified interpretation.

### Live Gemma follow-up

The structured model response now has separate `facts` and `interpretation` strings. The Go client validates those fields and adds FACT/INTERPRETATION labels to the unchanged public `answer` contract. An actual local `gemma3:4b` call with the OpenFootball Palmeiras snapshot returned HTTP 200 in approximately 34 seconds: one win, two draws, two goals scored and none conceded, with cited `recent_matches` and `recent_form`. These facts matched the snapshot. The model's interpretation remains subjective; structural validation does not prove entailment. One earlier, longer browser request did not produce an answer within the test deadline, so latency is hardware-dependent and timeout/retry remains relevant. The daemon reported CPU inference (`size_vram: 0`).

A browser run then exposed an unsupported defensive-vulnerability interpretation despite zero conceded goals. The prompt was tightened to require concrete evidence for each interpretation, limit score-only analysis to observed scoring/conceding, and leave interpretation empty for factual lookups. This is an observed model failure, not something schema validation can catch; generated analysis still requires review.

## Data sources, history and lineups — October 2, 2026

Automated: `gofmt -l .` (empty), `go vet ./...`, `go test -race -timeout 120s ./...` and a clean `npm ci && npm run build` passed. New tests cover the Brasileirão table and tie-breakers, Brazilian nickname search, AlmanacStats (match linking by round/opponent/side/score, rate limiting, caching across instances, goals, cards, venue, referee, squads, lineups, abbreviated-name matching, HTML-entity and Latin-1 repairs), API-Futebol (test-key labeling, plan errors, negative caching), API-Football, the PostgreSQL-compatible response cache contract, Série A history (season inference, champions, positions, coaches, averages with filled-data checks, scorers, bookings, meetings), `.env` loading, AI context selection (table/history/squad only when asked) and the enforced sample-data warning.

Live checks against the real sources:

- The computed 2026 table matched the source standings (e.g. Flamengo 1st with 60 points after 28 rounds).
- Champions computed from the history dataset matched the official list for every season 2003–2024; scorers matched the final score in 100% of 2015+ matches.
- With AlmanacStats enabled, all 15 checked recent matches of five clubs received statistics, and scores agreed with OpenFootball. A second pass was served entirely from PostgreSQL (≈20 ms per club, no new rows).
- Lineups: 463 of 484 starters (95%) across 22 cached matches were matched to minutes and ratings; the rest are nickname-versus-full-name cases and are shown without numbers rather than guessed.
- The API-Futebol production key returned "championship not in plan"; the dashboard kept working and showed the provider's message.
- Local Gemma 3 4B on CPU answered table, titles, scorers and statistics questions in about 1.5–3 minutes after the context was made question-dependent; before that change, answers exceeded the 120 s timeout. The 4B model still occasionally omits data that is in its context (e.g. a top scorer), so answers must be reviewed.
- Playwright checks at 1280 px and 390 px: no JavaScript errors and no horizontal overflow (after fixing a wide-table overflow in the history panel); the lineups dialog opens, loads and closes with Escape.
