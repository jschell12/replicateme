# replicateme status

## 2026-08-29

### Completed

- **goreleaser + brew tap** (PRs #11-#15) - cross-platform binary releases (macOS/Linux/Windows x amd64/arm64) via GitHub Actions on tag push. Homebrew formula at `jschell12/homebrew-tap`. `brew install jschell12/tap/replicateme` tested and working.
- **Gatekeeper quarantine fix** (PRs #13-#15) - switched from Homebrew cask to formula. Casks quarantine downloaded binaries; formulas handle xattr stripping internally. No more "not verified" dialog.
- **CI/CD pipeline** (PR #16) - `go build` + `go test` on every PR. Cloudflare Pages deploy of `site/` on merge to main. Enables full workflow from Claude Code mobile.
- **v0.1.3 released** - latest stable release with all fixes

### Next steps

- Persona spec integration (--persona flag to load .md into generation prompt)
- Git hook integration (prepare-commit-msg example)
- Clipboard copy (pick a variant, copy to clipboard)
- RAG with vector search instead of random example selection

### Decisions

- Homebrew formula over cask: casks quarantine unsigned binaries, formulas don't. `brews` field in goreleaser is deprecated but functional and the only way to generate a formula.
- CLOUDFLARE_API_TOKEN stored in both scredmgr and GitHub Actions secrets for CI deploys
- HOMEBREW_TAP_GITHUB_TOKEN: fine-grained PAT scoped to homebrew-tap repo only (Contents RW)

## 2026-08-28 (session 2)

### Completed

- **Go rewrite** (PR #8) - full rewrite from TypeScript/Node.js to Go. 15MB single binary, pure Go SQLite (modernc.org/sqlite), no CGO, no runtime deps.
- **Corpus store** (PR #7) - local SQLite at ~/.replicateme/corpus.db accumulates messages across sources. Duplicates skipped. Per-platform + combined profiles.
- **Multi-source merging** (PR #7) - ingest from multiple sources without overwriting. `stats` and `profile --platform` commands.
- **Marketing site updated** (PRs #4, #9) - all 8 sources live, Go install instructions
- **Example persona spec** (PR #5) - sanitized production persona spec

## 2026-08-28

### Completed

- **Project created from scratch** - repo initialized, TypeScript/Node.js stack, MIT license
- **iMessage connector** (PR #1) - reads local chat.db, imports 45k+ messages, style profile
- **Style analyzer** - writing fingerprint with quirk detection
- **Generation engine** (PR #1) - multi-provider, per-platform prompting, quirk slider
- **Marketing site** (PR #2) - deployed to Cloudflare Pages at replicateme.cc
- **7 additional data source connectors** (PR #3) - Slack, Gmail, Twitter, Discord, Reddit, Instagram, GitHub
- **Domain** - replicateme.cc on Cloudflare ($8/yr .cc)
