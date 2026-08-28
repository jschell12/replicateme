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
import { loadConfig, saveConfig, getConfigDir } from "./config.js";
import {
  storeMessages,
  getMessages,
  getExamples,
  savePerPlatformProfile,
  getPerPlatformProfile,
  getAllPlatformProfiles,
  logIngest,
  getCorpusStats,
  closeDb,
} from "./corpus/store.js";
import type { RawMessage, StyleProfile, Platform } from "./corpus/types.js";

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
  try {
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
      case "stats":
        cmdStats();
        break;
      case "help":
      default:
        cmdHelp();
        break;
    }
  } finally {
    closeDb();
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

  stats               Show corpus statistics (messages per platform, profiles)

  profile             Show your analyzed writing style profile
    --platform P       Show profile for a specific platform (or "combined")

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
  3. replicateme ingest --source slack --file export.zip --user-name "You"
  4. replicateme gen "Friend asks: want to grab dinner tonight?"

Messages are stored locally at ${getConfigDir()}/corpus.db
Ingest multiple sources - they accumulate, never overwrite.
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

Ingest as many sources as you want. Messages accumulate in a local database.
Duplicates are automatically skipped.
`);
}

function cmdStats() {
  const stats = getCorpusStats();

  if (stats.totalMessages === 0) {
    console.log("No messages in corpus. Run `replicateme ingest` first.");
    return;
  }

  console.log(`Corpus: ${stats.totalMessages.toLocaleString()} messages\n`);

  console.log("Messages by platform:");
  for (const { platform, count } of stats.byPlatform) {
    console.log(`  ${platform.padEnd(12)} ${count.toLocaleString()}`);
  }

  if (stats.profiles.length > 0) {
    console.log("\nStyle profiles:");
    for (const { platform, messageCount, updatedAt } of stats.profiles) {
      console.log(
        `  ${platform.padEnd(12)} ${messageCount.toLocaleString()} msgs  (${updatedAt})`
      );
    }
  }

  console.log(`\nCorpus stored at: ${getConfigDir()}/corpus.db`);
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

  console.log(`Parsed ${messages.length} messages from ${source}`);

  if (messages.length === 0) {
    console.log("No messages found.");
    if (source === "imessage") {
      console.log("Make sure Full Disk Access is granted.");
    }
    process.exit(1);
  }

  // store in corpus
  const { inserted, skipped } = storeMessages(messages);
  console.log(`Stored ${inserted} new messages (${skipped} duplicates skipped)`);
  logIngest(source, inserted);

  // build per-platform profile
  const platform = platformForSource(source);
  const platformMessages = getMessages({ platform });
  console.log(
    `\nAnalyzing ${platform} style (${platformMessages.length} total messages)...`
  );
  const platformProfile = analyzeStyle(platformMessages);
  savePerPlatformProfile(platform, platformProfile, platformMessages.length);

  // rebuild combined profile from all messages
  const allMessages = getMessages();
  console.log(
    `Rebuilding combined profile (${allMessages.length} total messages across all sources)...`
  );
  const combinedProfile = analyzeStyle(allMessages);
  savePerPlatformProfile("combined", combinedProfile, allMessages.length);

  console.log(
    `\nProfiles saved. Use 'replicateme profile' to view combined, or 'replicateme profile --platform ${platform}' for ${platform} only.`
  );

  // show the per-platform profile
  console.log(`\n--- ${platform} style ---\n`);
  console.log(styleProfileToPrompt(platformProfile));
}

function cmdProfile() {
  const platform = getFlag("--platform") ?? "combined";

  if (platform === "all") {
    const profiles = getAllPlatformProfiles();
    if (profiles.length === 0) {
      console.log("No profiles found. Run `replicateme ingest` first.");
      process.exit(1);
    }
    for (const p of profiles) {
      console.log(`\n=== ${p.platform} (${p.messageCount} messages) ===\n`);
      console.log(styleProfileToPrompt(p.profile));
    }
    return;
  }

  const result = getPerPlatformProfile(platform as Platform | "combined");
  if (!result) {
    console.log(
      `No profile found for "${platform}". Run \`replicateme ingest\` first.`
    );
    const available = getAllPlatformProfiles().map((p) => p.platform);
    if (available.length > 0) {
      console.log(`Available profiles: ${available.join(", ")}`);
    }
    process.exit(1);
  }

  console.log(
    `${platform} profile (${result.messageCount} messages):\n`
  );
  console.log(styleProfileToPrompt(result.profile));
}

async function cmdGenerate() {
  const config = loadConfig();

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

  // pick the best profile: per-platform if available, else combined
  const platformProfile = getPerPlatformProfile(platform as Platform);
  const combinedProfile = getPerPlatformProfile("combined");
  const profileResult = platformProfile ?? combinedProfile;

  if (!profileResult) {
    console.log("No profile found. Run `replicateme ingest` first.");
    process.exit(1);
  }

  console.log(
    `Platform: ${platform} | Quirk: ${quirkLevel}% | Variants: ${variants}`
  );
  if (platformProfile) {
    console.log(`Using ${platform} profile (${platformProfile.messageCount} messages)`);
  } else {
    console.log(`No ${platform} profile found, using combined profile`);
  }
  console.log(`Context: ${context}\n`);

  // pull examples from corpus, preferring same platform
  const similarMessages = getExamples(platform as Platform, 20);

  const results = await generateMessage({
    platform,
    context,
    quirkLevel,
    similarMessages,
    styleProfile: profileResult.profile,
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

function platformForSource(source: string): Platform {
  const map: Record<string, Platform> = {
    imessage: "imessage",
    slack: "slack",
    gmail: "email",
    twitter: "twitter",
    discord: "discord",
    reddit: "reddit",
    instagram: "instagram",
    github: "github",
  };
  return map[source] ?? (source as Platform);
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
