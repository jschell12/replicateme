# replicateme status

## 2026-08-30 (session 4)

### Completed

- **RAG vector search** (PR #26, v0.3.0) - messages embedded via Ollama bge-m3, stored in Qdrant. Generation pulls 15 most similar examples instead of random sampling. Falls back to random if infra unreachable. Progress indicator during indexing.
- **v0.3.0 released** - goreleaser + homebrew formula updated
- **Marketing site updated** (PR #24) - all v0.2.0 features added

### Next steps

- Automate homebrew formula update in release workflow
- Full corpus indexing (45k messages, running in background)
- More real-world testing with RAG-powered generation

## 2026-08-30 (session 3)

- v0.2.0: persona spec, git hook, clipboard, TikTok, quirk toggles. Marketing site updated.

## 2026-08-30 (sessions 1-2)

- workflow_dispatch, WireGuard NAT fix, v0.2.0 features implemented

## 2026-08-29

- goreleaser + brew tap, Gatekeeper fix, CI/CD, v0.1.3

## 2026-08-28

- Project created, Go rewrite, corpus store, marketing site, 9 connectors, domain
