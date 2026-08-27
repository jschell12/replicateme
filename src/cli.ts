#!/usr/bin/env node

import { importIMessages } from "./connectors/imessage.js";
import { analyzeStyle, styleProfileToPrompt } from "./style/analyzer.js";
import { generateMessage } from "./generate/generate.js";
import { loadConfig, saveConfig, saveProfile, loadProfile, getConfigDir } from "./config.js";
import type { StyleProfile } from "./corpus/types.js";

const args = process.argv.slice(2);
const command = args[0];

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
    --source imessage  Source to import from (default: imessage)
    --since DATE       Only import messages after this date
    --db-path PATH     Custom path to chat.db

  profile             Show your analyzed writing style profile

  generate (gen)      Generate a message in your style
    --platform P       Platform style: imessage, slack, email, github, twitter (default: from config)
    --quirk N          Quirk level 0-100 (default: from config)
    --variants N       Number of variants to generate (default: 3)
    <context>          The rest of the args are the context/prompt

  config              View or set configuration
    --provider P       LLM provider: anthropic or openai
    --model M          Model name (e.g. claude-sonnet-4-20250514, gpt-4o)
    --quirk-level N    Default quirk level 0-100
    --platform P       Default platform

Setup:
  1. Set your API key:
     export ANTHROPIC_API_KEY=sk-...   # for Anthropic
     export OPENAI_API_KEY=sk-...      # for OpenAI

  2. Ingest your messages:
     replicateme ingest --source imessage

  3. Generate:
     replicateme gen "Friend asks: want to grab dinner tonight?"

Config stored at: ${getConfigDir()}
`);
}

async function cmdIngest() {
  const source = getFlag("--source") ?? "imessage";
  const sinceStr = getFlag("--since");
  const dbPath = getFlag("--db-path");

  if (source !== "imessage") {
    console.log(`Source "${source}" not yet supported. Available: imessage`);
    process.exit(1);
  }

  console.log("Importing iMessages...");

  const opts: Record<string, unknown> = {};
  if (sinceStr) opts.since = new Date(sinceStr);
  if (dbPath) {
    opts.dbPath = dbPath;
    opts.copyToTmp = false;
  }

  const messages = importIMessages(opts);
  console.log(`Imported ${messages.length} messages`);

  if (messages.length === 0) {
    console.log("No messages found. Make sure Full Disk Access is granted.");
    process.exit(1);
  }

  const earliest = messages[0].timestamp.toISOString().split("T")[0];
  const latest = messages[messages.length - 1].timestamp.toISOString().split("T")[0];
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
  const quirkLevel = parseInt(getFlag("--quirk") ?? String(config.quirkLevel), 10);
  const variants = parseInt(getFlag("--variants") ?? "3", 10);

  // everything after flags is the context
  const context = args
    .slice(1)
    .filter((a) => !a.startsWith("--") && !isValueOfFlag(a))
    .join(" ");

  if (!context) {
    console.log("Usage: replicateme gen [--platform P] [--quirk N] <context>");
    console.log('Example: replicateme gen "Friend asks: want to grab dinner?"');
    process.exit(1);
  }

  console.log(`Platform: ${platform} | Quirk: ${quirkLevel}% | Variants: ${variants}`);
  console.log(`Context: ${context}\n`);

  // load raw messages for examples (re-import from cached copy)
  let similarMessages: import("./corpus/types.js").RawMessage[] = [];
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

// flag parsing helpers
function getFlag(name: string): string | undefined {
  const idx = args.indexOf(name);
  if (idx === -1 || idx + 1 >= args.length) return undefined;
  return args[idx + 1];
}

function isValueOfFlag(arg: string): boolean {
  const idx = args.indexOf(arg);
  if (idx <= 0) return false;
  return args[idx - 1].startsWith("--");
}

main().catch((err) => {
  console.error(err.message ?? err);
  process.exit(1);
});
