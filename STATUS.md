# replicateme status

## 2026-08-30 (session 2)

### Completed

- **v0.2.0 released** (PR #22) - 7 features shipped:
  - Persona spec integration (`--persona PATH` flag + config default)
  - Git hook (`examples/prepare-commit-msg` for auto commit messages)
  - Clipboard copy (`--copy [N]` flag, detects pbcopy/xclip/clip)
  - TikTok connector (comments + DMs from data download ZIP)
  - Quirk toggles (`--enable-quirk`/`--disable-quirk` for 7 granular controls)
  - Per-platform example selection verified working
  - Removed deprecated goreleaser brews, added manual formula template
- **Homebrew formula updated** - manually pushed v0.2.0 to jschell12/homebrew-tap, `brew upgrade` tested
- **WireGuard NAT fix** - pfctl NAT anchor was empty on mini, reloaded to restore VPN forwarding to Framework

### Next steps

- Script the homebrew formula update into the release workflow (manual update works but could be automated)
- RAG with vector search (Qdrant + bge-m3 on Framework)
- Update marketing site with new features (persona spec, quirk toggles, git hook)

### Decisions

- Manual homebrew formula over goreleaser brews: brews deprecated, homebrew_casks has Gatekeeper quarantine problem. Manual formula update per release is about 1 minute of work.
- WireGuard NAT on mini: pfctl anchor gets cleared on reboot if the wireguard-up.sh script fails or runs late. Need to investigate why the anchor wasn't loaded.

## 2026-08-30

### Completed

- workflow_dispatch (PR #18), WireGuard forwarding fix (schellout PR #268)

## 2026-08-29

### Completed

- goreleaser + brew tap (PRs #11-#15), Gatekeeper quarantine fix, CI/CD pipeline (PR #16), v0.1.3 released

## 2026-08-28

### Completed

- Project created, Go rewrite (PR #8), corpus store (PR #7), marketing site, 8 data source connectors, domain replicateme.cc
