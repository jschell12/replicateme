# replicateme

CLI that learns your writing style and generates messages that sound like you. Typos, grammar quirks, and all.

Your data never leaves your machine. Bring your own API key. Single binary, no runtime dependencies.

## Install

```bash
# with Go
go install github.com/jschell12/replicateme/cmd/replicateme@latest

# or build from source
git clone https://github.com/jschell12/replicateme.git
cd replicateme
make install
```

## Setup

Set your LLM API key:

```bash
# Anthropic (default)
export ANTHROPIC_API_KEY=sk-ant-...

# or OpenAI
export OPENAI_API_KEY=sk-...
replicateme config --provider openai
```

Grant Full Disk Access to your terminal app (System Settings > Privacy & Security > Full Disk Access) so replicateme can read your iMessage history.

## Usage

### 1. Ingest your messages

```bash
replicateme ingest --source imessage
```

Reads your local iMessage database, stores messages in a local corpus, and analyzes your writing patterns. Nothing is sent anywhere.

Ingest from multiple sources - they accumulate, never overwrite:

```bash
replicateme ingest --source imessage
replicateme ingest --source slack --file slack-export.zip --user-name "Your Name"
replicateme ingest --source twitter --file twitter-archive.zip
```

### 2. Check your style profile

```bash
replicateme profile                    # combined profile
replicateme profile --platform slack   # platform-specific
replicateme profile --platform all     # compare all platforms
replicateme stats                      # corpus overview
```

### 3. Generate messages

```bash
# reply to a text
replicateme gen "Friend asks: want to grab dinner tonight?"

# write a Slack message
replicateme gen --platform slack "Tell the team the deploy is done"

# commit message
replicateme gen --platform github "Fixed the auth middleware to handle expired tokens"

# adjust quirk level (0 = clean, 100 = full authenticity)
replicateme gen --quirk 0 "Reply to boss asking about project status"
replicateme gen --quirk 100 "Reply to friend asking what's for dinner"
```

## Quirk levels

| Level | Description |
|-------|-------------|
| 0 | Your voice and vocabulary, clean grammar |
| 25 | Mostly clean, occasional habits slip through |
| 50 | Natural mix of your style and quirks (default) |
| 75 | Authentic - missing apostrophes, lowercase, fragments |
| 100 | Full you - typos, double spaces, all of it |

## Data sources

| Source | How |
|--------|-----|
| iMessage | Reads local `~/Library/Messages/chat.db` |
| Slack | Workspace export ZIP |
| Gmail | Google Takeout mbox |
| Twitter/X | Data archive ZIP (tweets + DMs) |
| GitHub | Local git repos (commit messages) |
| Discord | Data export ZIP |
| Reddit | Data archive ZIP (posts + comments) |
| Instagram | Data download ZIP (DMs, captions, comments) |

Run `replicateme sources` for step-by-step instructions on getting your data from each platform.

## Configuration

```bash
replicateme config --provider anthropic    # or openai
replicateme config --model claude-sonnet-4-6-20250725
replicateme config --quirk-level 70
replicateme config --platform imessage
```

Config stored at `~/.replicateme/config.json`. Corpus at `~/.replicateme/corpus.db`.

## Privacy

- All message data stays on your machine in a local SQLite database
- Style profiles are stored locally at `~/.replicateme/`
- The only network call is to your chosen LLM provider with the style profile and example messages for generation
- No telemetry, no analytics, no accounts
- MIT licensed, single binary, no dependencies to audit

## Persona specs

The style profile that `replicateme ingest` generates captures statistical patterns (punctuation frequency, capitalization ratios, common errors). For even better results, layer a hand-written persona spec on top.

See [`examples/persona-spec.md`](examples/persona-spec.md) for a production example that's been battle-tested with Claude Code.

## Contributing

PRs welcome, especially for new data source connectors. See `pkg/connectors/` for the pattern.

## License

MIT
