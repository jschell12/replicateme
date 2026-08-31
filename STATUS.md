# replicateme status

## 2026-08-31

### Completed

- **3 new LLM providers** (PR #28) - claude-cli (Max subscription, no API cost), ollama (local models, zero cost), openai --base-url (any OpenAI-compatible server). Tested ollama with mistral-small:24b.
- **Privacy card fix** (PR #29) - features card now discloses site analytics
- **RAG embedding validation** (PR #30) - validate embedding dimension before Qdrant upsert. Prevents batch failures from malformed embeddings.
- **RAG corpus** - 34k of 45k iMessages indexed in Qdrant before the embedding bug killed the first run. Fix allows re-running to completion.

### Next steps

- Re-run full iMessage ingest to index remaining ~11k messages
- Connector tests (all 9 sources)
- Update marketing site with new provider options
- Tag v0.3.1 or v0.4.0

## 2026-08-30

- v0.2.0 (persona spec, git hook, clipboard, TikTok, quirk toggles), v0.3.0 (RAG), marketing site updates

## 2026-08-29

- goreleaser + brew tap, Gatekeeper fix, CI/CD, v0.1.3

## 2026-08-28

- Project created, Go rewrite, corpus store, marketing site, 9 connectors, domain
