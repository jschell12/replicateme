# replicateme status

## 2026-08-30 (session 3)

### Completed

- **v0.2.0 released** (PR #22) with 7 features: persona spec (`--persona`), git hook (`examples/prepare-commit-msg`), clipboard copy (`--copy`), TikTok connector, quirk toggles (7 granular controls), per-platform examples verified, goreleaser formula template
- **Homebrew formula updated** to v0.2.0, `brew upgrade` tested
- **Marketing site updated** (PR #24) - added persona specs, git hook, clipboard, quirk toggles, TikTok to features grid and data sources. Hero install switched to brew. Install section shows new flags.

### Next steps

- RAG with vector search (Qdrant + bge-m3 on Framework) to replace random example selection
- Automate homebrew formula update in release workflow
- More real-world testing with multi-source corpus

## 2026-08-30 (session 2)

- v0.2.0 features implemented, WireGuard NAT fix

## 2026-08-30

- workflow_dispatch (PR #18), WireGuard forwarding fix (schellout PR #268)

## 2026-08-29

- goreleaser + brew tap (PRs #11-#15), Gatekeeper fix, CI/CD (PR #16), v0.1.3

## 2026-08-28

- Project created, Go rewrite (PR #8), corpus store (PR #7), marketing site, 8 connectors, domain
