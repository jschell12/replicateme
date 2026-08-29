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

### Remaining CI/CD gaps

The pipeline builds and deploys the site, but **merging still does not cut a release** — `release.yml` fires on `v*` tags and nothing pushes those tags, so shipping from mobile still ends with a manual tag. Closing that is the next piece of work:

1. **Auto-tag on merge to `main`** — the actual blocker. Add a job that reads a `VERSION` file and pushes the tag when it changes; the existing tag-triggered goreleaser job then runs unchanged. Bumping one line in `VERSION` is an edit Claude can make from mobile in the same PR as the change.
   - Gotcha: a tag pushed with the default `GITHUB_TOKEN` does **not** trigger other workflows. Push it with a PAT (reuse `HOMEBREW_TAP_GITHUB_TOKEN` or add `RELEASE_TOKEN`), or have the tag job invoke goreleaser directly.
2. **Tests** — still zero test files, so `go test ./...` in CI passes vacuously and the PR gate only proves the code compiles. Best targets: `pkg/style/analyzer.go` (pure functions over text, easy to table-test) and `pkg/corpus/store.go` (dedup, per-platform profiles, against a temp-dir SQLite file).
3. **Harden `ci.yml`** — currently `go build` + `go test` on `ubuntu-latest` only:
   - add `go vet ./...` and a `gofmt -l .` check that fails on non-empty output
   - pin the toolchain with `go-version-file: go.mod` instead of `go-version: stable` (repo is on Go 1.26.1)
   - add `cache: true` to `setup-go`
   - add a `macos-latest` matrix leg — the iMessage connector is the most macOS-coupled code in the repo
4. **Branch protection** — make CI a required status check on `main` so a merge from mobile can't land red.

### Open question

Release on every merge to `main`, or only when `VERSION` changes? Every-merge is simpler to reason about from a phone but burns version numbers on doc-only commits.

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
