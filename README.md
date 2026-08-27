# replicateme

CLI that learns your writing style and generates messages that sound like you. Typos, grammar quirks, and all.

Your data never leaves your machine. Bring your own API key.

## Install

```bash
npm install -g replicateme
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

This reads your local iMessage database, analyzes your writing patterns, and saves a style profile to `~/.replicateme/profile.json`. Nothing is sent anywhere.

### 2. Check your style profile

```bash
replicateme profile
```

Shows your writing fingerprint: how often you capitalize, use punctuation, skip apostrophes, send fragments, etc.

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

| Source | Status | How |
|--------|--------|-----|
| iMessage | supported | Reads local `~/Library/Messages/chat.db` |
| Slack | planned | Workspace export (JSON) |
| Gmail | planned | Google Takeout (mbox) |
| Twitter/X | planned | Data archive (JSON) |
| GitHub | planned | Git log + API |
| Discord | planned | Data export (JSON) |
| Reddit | planned | Data archive |

## Configuration

```bash
replicateme config --provider anthropic    # or openai
replicateme config --model claude-sonnet-4-20250514
replicateme config --quirk-level 70
replicateme config --platform imessage
```

Config stored at `~/.replicateme/config.json`.

## Privacy

- All message data stays on your machine
- Style profiles are stored locally at `~/.replicateme/`
- The only network call is to your chosen LLM provider (Anthropic or OpenAI) with your style profile and example messages for generation
- No telemetry, no analytics, no accounts
- MIT licensed - read every line yourself

## Contributing

PRs welcome, especially for new data source connectors. See `src/connectors/` for the pattern.

## License

MIT
