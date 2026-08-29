# replicateme status

## 2026-08-28 (session 2)

### Completed

- **Go rewrite** (PR #8) - full rewrite from TypeScript/Node.js to Go. 15MB single binary, pure Go SQLite (modernc.org/sqlite), no CGO, no runtime deps. All 8 connectors, style analyzer, generation layer, corpus store ported.
- **Corpus store** (PR #7) - local SQLite at ~/.replicateme/corpus.db accumulates messages across sources. Duplicates skipped. Per-platform + combined profiles.
- **Multi-source merging** (PR #7) - ingest from multiple sources without overwriting. `stats` and `profile --platform` commands.
- **Marketing site updated** (PRs #4, #9) - all 8 sources live, Go install instructions
- **Example persona spec** (PR #5) - sanitized production persona spec

### Next steps

- Persona spec integration (--persona flag)
- goreleaser + brew tap for binary releases
- Git hook integration (prepare-commit-msg)
- Clipboard copy
- RAG with vector search

### Decisions

- Go over TypeScript: better-sqlite3 native addon caused install friction, single binary easier to distribute
- modernc.org/sqlite over mattn/go-sqlite3: pure Go, no CGO, cross-compiles cleanly
- Raw HTTP for LLM APIs over SDKs: keeps binary small, avoids version churn

## 2026-08-28

### Completed

- **Project created from scratch** - repo initialized, TypeScript/Node.js stack, MIT license
- **iMessage connector** (PR #1) - reads local chat.db, imports 45k+ messages, style profile
- **Style analyzer** - writing fingerprint with quirk detection
- **Generation engine** (PR #1) - multi-provider, per-platform prompting, quirk slider
- **Marketing site** (PR #2) - deployed to Cloudflare Pages at replicateme.cc
- **7 additional data source connectors** (PR #3) - Slack, Gmail, Twitter, Discord, Reddit, Instagram, GitHub
- **Domain** - replicateme.cc on Cloudflare ($8/yr .cc)
