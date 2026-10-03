# Implementation references

Primary API references consulted during implementation:

- [Ollama chat API](https://docs.ollama.com/api/chat): non-streaming requests, message roles, structured JSON format and completion envelope.
- [Gemma 3:4b on Ollama](https://ollama.com/library/gemma3:4b): the exact default model selected by the project brief.
- [Official challenge](https://dev.to/challenges/hacktoberfest-weekend-2026-10-01): theme, dates and submission requirements, also checked through DevRelay event 78 `full_details`.

## Community Wisdom

Two relevant DEV posts were read through DevRelay, along with their comment threads. These are individual implementation reports, not evidence of a community-wide consensus.

### [Indirect prompt injection in a RAG pipeline: one attack, step by step, and what actually stopped it](https://dev.to/sergiobm99/indirect-prompt-injection-in-a-rag-pipeline-one-attack-step-by-step-and-what-actually-stopped-it-4l00)

Author: [Sergio Belmonte Morales](https://dev.to/sergiobm99). Tags: `security`, `ai`, `llm`, `nextjs`.

The author's attack replay distinguishes prompt-level isolation from deterministic controls at the point where an agent performs an action. Its demonstrated barrier is a tool gate, not proof that a model always ignores injected text. Comments by Jo Do and Max Quimby reinforce separating retrieved data from privileged actions. MatchMind applies that narrow lesson by exposing no model tools, encoding provider context as data and accepting no frontend-supplied context. This does not prove that every generated answer is grounded.

[Read the full discussion](https://dev.to/sergiobm99/indirect-prompt-injection-in-a-rag-pipeline-one-attack-step-by-step-and-what-actually-stopped-it-4l00).

### [Refactoring Go API Unit Tests: Breaking Down the Testing Monolith](https://dev.to/xaiphyr/refactoring-go-api-unit-tests-breaking-down-the-testing-monolith-1p4p)

Author profile: [xaiphyr](https://dev.to/xaiphyr). Tags: `go`, `api`, `testing`.

The author separates service behavior from transport tests and discusses the extra maintenance cost of mocks. MatchMind keeps small fakes in test files and uses `httptest` for HTTP boundaries, with direct domain tests for calculations. No comments were returned when consulted.

[Read the full discussion](https://dev.to/xaiphyr/refactoring-go-api-unit-tests-breaking-down-the-testing-monolith-1p4p).


## Real-data integration follow-up

The OpenFootball [README](https://github.com/openfootball/football.json) documents its public raw JSON service. The repository's [CC0-1.0 license](https://github.com/openfootball/football.json/blob/master/LICENSE.md) was inspected before integration. The actual `2026/br.1.json` download contained both score objects and compact score arrays, so the adapter handles both and tests scoreless draws explicitly. Coverage/freshness are carried into the UI and model context. No private endpoints or user-supplied URLs are fetched.

Two additional first-hand DEV implementation reports and their comment threads were read; both threads returned no comments when consulted. These are individual reports, not a consensus survey:

- [An Error Inside HTTP 200 Poisoned My Cache](https://dev.to/hexisteme/an-error-inside-http-200-poisoned-my-cache-why-responseok-is-not-a-success-check-1nd1), by [hexisteme](https://dev.to/hexisteme), tags `api`, `backend`, `programming`, `webdev`: validate the body before writing a success cache entry. MatchMind checks the dataset and scores before caching, distinguishes missing results from 0–0, and never converts an upstream failure into fictional data. [Full discussion](https://dev.to/hexisteme/an-error-inside-http-200-poisoned-my-cache-why-responseok-is-not-a-success-check-1nd1).
- [Retrying HTTP Requests in Go Without Making It Worse](https://dev.to/krishankumar01/retrying-http-requests-in-go-without-making-it-worse-48mj), by [Krishan Kumar](https://dev.to/krishankumar01), tags `go`, `http`, `distributedsystems`, `webdev`: deadlines and retry policy address different failure modes, and unbounded retries can amplify outages. This MVP uses an eight-second client deadline, context cancellation and a short failure cooldown instead of an automatic retry loop. [Full discussion](https://dev.to/krishankumar01/retrying-http-requests-in-go-without-making-it-worse-48mj).

## Club crest sources

Original artwork was retrieved through Wikimedia Commons' public `imageinfo` API, checking each file's source, author and license metadata. The 18 selected originals are preserved locally, with per-file provenance and checksums in [the manifest](../frontend/public/crests/manifest.json) and user-facing attribution in [credits.html](../frontend/public/crests/credits.html). Sports results still come exclusively from the selected football provider; the crest catalog adds identity artwork only. Missing or unsuitable artwork uses initials.
