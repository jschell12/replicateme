# replicateme status

## 2026-08-31 (session 2)

### Completed

- **88 connector tests** (PR #33) - all 9 connectors tested with synthetic fixtures, real SQLite DBs, real git repos, actual ZIP archives
- **4 LLM providers** (PR #28) - claude-cli (Max sub, no API cost), ollama (local), openai --base-url (any compatible endpoint). Tested ollama with mistral-small:24b.
- **Privacy card fix** (PR #29) - discloses site analytics
- **RAG embedding validation** (PR #30) - prevents batch failures from malformed embeddings
- **Desktop app** (PR #34) - Wails v2 + React + Tailwind. 4 screens: Generate, Profile, Ingest, Settings. Live model fetching from provider APIs. Casual/Formal tone slider.
- **Portable rule export** (PR #35) - `replicateme export` generates a rule file (.md) that works as Claude Code rule, Cursor rule, ChatGPT custom instructions, or --persona. `replicateme enrich` adds local corpus data to an existing rule. No raw messages exported.
- **Bug report links** (PR #36) - GitHub issues link in hero and install sections
- **Quirk renamed to Tone** (PR #37) - all user-facing "quirk" references replaced with Casual/Formal language on marketing site

### Decisions

- Renamed "quirk level" to "tone" (Casual/Formal) everywhere user-facing. The underlying config field is still `quirkLevel` for backwards compatibility.
- Simplified export to a single rule file instead of JSON bundle + import. One file that drops into any AI tool. `enrich` adds local data on the destination machine.
- Desktop app uses Wails v2 (Go + webview) with React + Tailwind frontend. Model selector fetches from live provider APIs with hardcoded fallbacks.

### Next steps

- Tag v0.4.0 with desktop app + all new features
- Re-run full iMessage RAG indexing with the embedding validation fix
- Desktop app: add to goreleaser for cross-platform builds
- Interactive REPL mode for the CLI

## 2026-08-31

- 3 new providers (PR #28), privacy fix (PR #29), RAG fix (PR #30)

## 2026-08-30

- v0.2.0, v0.3.0 (RAG), marketing site updates

## 2026-08-29

- goreleaser + brew tap, Gatekeeper fix, CI/CD, v0.1.3

## 2026-08-28

- Project created, Go rewrite, corpus store, marketing site, 9 connectors, domain
