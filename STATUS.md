# replicateme status

## 2026-08-30

### Completed

- **workflow_dispatch** (PR #18) - CI workflow can be triggered manually via `gh workflow run` or GitHub Actions UI. Deploy gate updated to allow dispatch. Tested and working.
- **WireGuard forwarding fix** (schellout PR #268) - persisted IP forwarding on mini via /etc/sysctl.conf so VPN traffic reaches Framework even if WG starts late after reboot

### In progress

- None

### Blockers

- MacBook not on home network during session; WireGuard tunnel wasn't flowing. mem0 saves timed out. Not a code issue.

### Next steps

- Persona spec integration (--persona flag to load .md into generation prompt)
- Git hook integration (prepare-commit-msg example)
- Clipboard copy (pick a variant, copy to clipboard)
- RAG with vector search instead of random example selection

## 2026-08-29

### Completed

- **goreleaser + brew tap** (PRs #11-#15) - cross-platform binary releases (macOS/Linux/Windows x amd64/arm64) via GitHub Actions on tag push. Homebrew formula at `jschell12/homebrew-tap`. `brew install jschell12/tap/replicateme` tested and working.
- **Gatekeeper quarantine fix** (PRs #13-#15) - switched from Homebrew cask to formula. Casks quarantine downloaded binaries; formulas handle xattr stripping internally.
- **CI/CD pipeline** (PR #16) - `go build` + `go test` on every PR. Cloudflare Pages deploy of `site/` on merge to main. Enables full workflow from Claude Code mobile.
- **v0.1.3 released** - latest stable release with all fixes

### Decisions

- Homebrew formula over cask: casks quarantine unsigned binaries, formulas don't
- CLOUDFLARE_API_TOKEN + CLOUDFLARE_ACCOUNT_ID + HOMEBREW_TAP_GITHUB_TOKEN stored in GitHub Actions secrets + scredmgr

## 2026-08-28 (session 2)

### Completed

- **Go rewrite** (PR #8) - full rewrite from TypeScript/Node.js to Go. 15MB single binary, pure Go SQLite (modernc.org/sqlite), no CGO, no runtime deps.
- **Corpus store** (PR #7) - local SQLite at ~/.replicateme/corpus.db accumulates messages across sources. Per-platform + combined profiles.
- **Marketing site updated** (PRs #4, #9) - all 8 sources live, Go install instructions
- **Example persona spec** (PR #5) - sanitized production persona spec

## 2026-08-28

### Completed

- **Project created from scratch** - repo, domain (replicateme.cc), marketing site, 8 data source connectors, style analyzer, generation engine
