#!/usr/bin/env node

import { importIMessages } from "./connectors/imessage.js";
import { importSlack } from "./connectors/slack.js";
import { importGmail } from "./connectors/gmail.js";
import { importTwitter } from "./connectors/twitter.js";
import { importDiscord } from "./connectors/discord.js";
import { importReddit } from "./connectors/reddit.js";
import { importInstagram } from "./connectors/instagram.js";
import { importGitHub } from "./connectors/github.js";
import { analyzeStyle, styleProfileToPrompt } from "./style/analyzer.js";
import { generateMessage } from "./generate/generate.js";
import {
  loadConfig,
  saveConfig,
  saveProfile,
  loadProfile,
  getConfigDir,
} from "./config.js";
import type { RawMessage, StyleProfile } from "./corpus/types.js";

const args = process.argv.slice(2);
const command = args[0];

const SOURCES = [
  "imessage",
  "slack",
  "gmail",
  "twitter",
  "discord",
  "reddit",
  "instagram",
  "github",
] as const;

async function main() {
  switch (command) {
    case "ingest":
      await cmdIngest();
      break;
    case "profile":
      cmdProfile();
      break;
    case "generate":
    case "gen":
      await cmdGenerate();
      break;
    case "config":
      cmdConfig();
      break;
    case "sources":
      cmdSources();
      break;
    case "help":
    default:
      cmdHelp();
      break;
  }
}

function cmdHelp() {
  console.log(`
replicateme - learn your writing style, generate messages that sound like you

All data stays on your machine. Bring your own API key.

Commands:
  ingest              Import messages from a data source
    --source SOURCE    Source: ${SOURCES.join(", ")}
    --file PATH        Path to archive ZIP or directory (required for most sources)
    --since DATE       Only import messages after this date
    --db-path PATH     Custom path to chat.db (imessage only)
    --user-id ID       User ID to filter (slack)
    --user-name NAME   Display name to filter (slack)
    --email EMAIL      Email to filter sent mail (gmail) or git author (github)
    --username USER    Username to filter (instagram)
    --repos PATH,...   Comma-separated repo paths (github)

  sources             List available data sources and how to get your data

  profile             Show your analyzed writing style profile

  generate (gen)      Generate a message in your style
    --platform P       Platform style: imessage, slack, email, github, twitter (default: from config)
    --quirk N          Quirk level 0-100 (default: from config)
    --variants N       Number of variants to generate (default: 3)
    <context>          The rest of the args are the context/prompt

  config              View or set configuration
    --provider P       LLM provider: anthropic or openai
    --model M          Model name
    --quirk-level N    Default quirk level 0-100
    --platform P       Default platform

Setup:
  1. export ANTHROPIC_API_KEY=sk-...
  2. replicateme ingest --source imessage
  3. replicateme gen "Friend asks: want to grab dinner tonight?"

Config stored at: ${getConfigDir()}
`);
}

function cmdSources() {
  console.log(`
Data sources and how to get your data:

imessage (macOS only)
  No file needed - reads ~/Library/Messages/chat.db directly.
  Requires Full Disk Access for your terminal app.
  replicateme ingest --source imessage

slack
  Export your workspace: Workspace Settings > Import/Export > Export.
  Or ask your admin for the export ZIP.
  replicateme ingest --source slack --file slack-export.zip --user-name "Your Name"

gmail
  Go to takeout.google.com, select Gmail, download as mbox.
  replicateme ingest --source gmail --file Takeout.zip --email you@gmail.com

twitter
  Settings > Your Account > Download an archive of your data.
  replicateme ingest --source twitter --file twitter-archive.zip

discord
  Settings > Privacy & Safety > Request all of my Data.
  (Takes up to 30 days for Discord to prepare.)
  replicateme ingest --source discord --file discord-package.zip

reddit
  Settings > Request Your Data (GDPR).
  Or: https://www.reddit.com/settings/data-request
  replicateme ingest --source reddit --file reddit-export.zip

instagram
  Settings > Your Activity > Download Your Information > Request Download.
  Select JSON format.
  replicateme ingest --source instagram --file instagram-data.zip --username yourusername

github
  No archive needed - reads from local git repos.
  replicateme ingest --source github --repos ~/projects/repo1,~/projects/repo2 --email you@email.com
`);
}

async function cmdIngest() {
  const source = getFlag("--source") ?? "imessage";
  const sinceStr = getFlag("--since");
  const file = getFlag("--file");

  if (!SOURCES.includes(source as (typeof SOURCES)[number])) {
    console.log(
      `Unknown source "${source}". Available: ${SOURCES.join(", ")}`
    );
    console.log("Run `replicateme sources` for setup instructions.");
    process.exit(1);
  }

  const since = sinceStr ? new Date(sinceStr) : undefined;

  let messages: RawMessage[];

  switch (source) {
    case "imessage": {
      console.log("Importing iMessages...");
      const dbPath = getFlag("--db-path");
      const opts: Record<string, unknown> = {};
      if (since) opts.since = since;
      if (dbPath) {
        opts.dbPath = dbPath;
        opts.copyToTmp = false;
      }
      messages = importIMessages(opts);
      break;
    }

    case "slack": {
      if (!file) {
        console.log("--file is required for Slack import.");
        console.log(
          "Example: replicateme ingest --source slack --file slack-export.zip"
        );
        process.exit(1);
      }
      console.log("Importing Slack messages...");
      messages = importSlack({
        file,
        userId: getFlag("--user-id"),
        userName: getFlag("--user-name"),
        since,
      });
      break;
    }

    case "gmail": {
      if (!file) {
        console.log("--file is required for Gmail import.");
        console.log(
          "Example: replicateme ingest --source gmail --file Takeout.zip --email you@gmail.com"
        );
        process.exit(1);
      }
      console.log("Importing Gmail...");
      messages = importGmail({
        file,
        email: getFlag("--email"),
        since,
      });
      break;
    }

    case "twitter": {
      if (!file) {
        console.log("--file is required for Twitter import.");
        console.log(
          "Example: replicateme ingest --source twitter --file twitter-archive.zip"
        );
        process.exit(1);
      }
      console.log("Importing Twitter/X...");
      messages = importTwitter({ file, since });
      break;
    }

    case "discord": {
      if (!file) {
        console.log("--file is required for Discord import.");
        console.log(
          "Example: replicateme ingest --source discord --file discord-package.zip"
        );
        process.exit(1);
      }
      console.log("Importing Discord...");
      messages = importDiscord({ file, since });
      break;
    }

    case "reddit": {
      if (!file) {
        console.log("--file is required for Reddit import.");
        console.log(
          "Example: replicateme ingest --source reddit --file reddit-export.zip"
        );
        process.exit(1);
      }
      console.log("Importing Reddit...");
      messages = importReddit({ file, since });
      break;
    }

    case "instagram": {
      if (!file) {
        console.log("--file is required for Instagram import.");
        console.log(
          "Example: replicateme ingest --source instagram --file instagram-data.zip --username yourusername"
        );
        process.exit(1);
      }
      console.log("Importing Instagram...");
      messages = importInstagram({
        file,
        username: getFlag("--username"),
        since,
      });
      break;
    }

    case "github": {
      const reposStr = getFlag("--repos");
      if (!reposStr) {
        console.log("--repos is required for GitHub import.");
        console.log(
          "Example: replicateme ingest --source github --repos ~/project1,~/project2 --email you@email.com"
        );
        process.exit(1);
      }
      console.log("Importing GitHub commits...");
      messages = importGitHub({
        repos: reposStr.split(",").map((r) => r.trim()),
        email: getFlag("--email"),
        since,
      });
      break;
    }

    default:
      console.log(`Source "${source}" not yet implemented.`);
      process.exit(1);
  }

  // filter out messages with invalid dates
  messages = messages.filter(
    (m) => m.timestamp instanceof Date && !isNaN(m.timestamp.getTime())
  );

  console.log(`Imported ${messages.length} messages`);

  if (messages.length === 0) {
    console.log("No messages found.");
    if (source === "imessage") {
      console.log("Make sure Full Disk Access is granted.");
    }
    process.exit(1);
  }

  const earliest = messages[0]!.timestamp.toISOString().split("T")[0];
  const latest =
    messages[messages.length - 1]!.timestamp.toISOString().split("T")[0];
  console.log(`Date range: ${earliest} to ${latest}`);

  console.log("\nAnalyzing writing style...");
  const profile = analyzeStyle(messages);

  saveProfile(profile as unknown as Record<string, unknown>);
  console.log(`Profile saved to ${getConfigDir()}/profile.json`);

  console.log("\n" + styleProfileToPrompt(profile));
}

function cmdProfile() {
  const profile = loadProfile();
  if (!profile) {
    console.log("No profile found. Run `replicateme ingest` first.");
    process.exit(1);
  }

  console.log(styleProfileToPrompt(profile as unknown as StyleProfile));
}

async function cmdGenerate() {
  const config = loadConfig();
  const profile = loadProfile() as unknown as StyleProfile | null;

  if (!profile) {
    console.log("No profile found. Run `replicateme ingest` first.");
    process.exit(1);
  }

  const platform = getFlag("--platform") ?? config.defaultPlatform;
  const quirkLevel = parseInt(
    getFlag("--quirk") ?? String(config.quirkLevel),
    10
  );
  const variants = parseInt(getFlag("--variants") ?? "3", 10);

  const context = args
    .slice(1)
    .filter((a) => !a.startsWith("--") && !isValueOfFlag(a))
    .join(" ");

  if (!context) {
    console.log("Usage: replicateme gen [--platform P] [--quirk N] <context>");
    console.log('Example: replicateme gen "Friend asks: want to grab dinner?"');
    process.exit(1);
  }

  console.log(
    `Platform: ${platform} | Quirk: ${quirkLevel}% | Variants: ${variants}`
  );
  console.log(`Context: ${context}\n`);

  let similarMessages: RawMessage[] = [];
  try {
    const messages = importIMessages({ copyToTmp: true });
    const shuffled = messages.sort(() => Math.random() - 0.5);
    similarMessages = shuffled.slice(0, 20);
  } catch {
    similarMessages = [];
  }

  const results = await generateMessage({
    platform,
    context,
    quirkLevel,
    similarMessages,
    styleProfile: profile,
    variants,
    config,
  });

  for (let i = 0; i < results.length; i++) {
    console.log(`[${i + 1}] ${results[i]}`);
  }
}

function cmdConfig() {
  const config = loadConfig();

  const provider = getFlag("--provider");
  const model = getFlag("--model");
  const quirkLevel = getFlag("--quirk-level");
  const platform = getFlag("--platform");

  const hasChanges = provider || model || quirkLevel || platform;

  if (provider) config.provider = provider as "anthropic" | "openai";
  if (model) config.model = model;
  if (quirkLevel) config.quirkLevel = parseInt(quirkLevel, 10);
  if (platform) config.defaultPlatform = platform;

  if (hasChanges) {
    saveConfig(config);
    console.log("Config updated:");
  } else {
    console.log("Current config:");
  }

  console.log(JSON.stringify(config, null, 2));
  console.log(`\nConfig file: ${getConfigDir()}/config.json`);
}

function getFlag(name: string): string | undefined {
  const idx = args.indexOf(name);
  if (idx === -1 || idx + 1 >= args.length) return undefined;
  return args[idx + 1];
}

function isValueOfFlag(arg: string): boolean {
  const idx = args.indexOf(arg);
  if (idx <= 0) return false;
  return args[idx - 1]!.startsWith("--");
}

main().catch((err) => {
  console.error(err.message ?? err);
  process.exit(1);
});
