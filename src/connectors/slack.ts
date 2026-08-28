import { readFileSync, existsSync, readdirSync } from "fs";
import { join, extname } from "path";
import { execSync } from "child_process";
import type { RawMessage } from "../corpus/types.js";

// Slack workspace export format:
// ZIP containing directories per channel, each with JSON files.
// Each JSON file is an array of message objects:
// { type: "message", user: "U...", text: "...", ts: "1234567890.123456" }

interface SlackMessage {
  type?: string;
  subtype?: string;
  user?: string;
  text?: string;
  ts?: string;
  thread_ts?: string;
}

export interface SlackImportOptions {
  file: string; // path to ZIP or extracted directory
  userId?: string; // Slack user ID to filter (if known)
  userName?: string; // display name to filter from users.json
  since?: Date;
}

export function importSlack(opts: SlackImportOptions): RawMessage[] {
  let dir = opts.file;

  // if it's a zip, extract to tmp
  if (opts.file.endsWith(".zip")) {
    dir = "/tmp/replicateme-slack-export";
    execSync(`rm -rf ${dir} && mkdir -p ${dir} && unzip -q -o "${opts.file}" -d ${dir}`);
  }

  if (!existsSync(dir)) {
    throw new Error(`Slack export not found at ${dir}`);
  }

  // try to resolve user ID from users.json
  let userId = opts.userId;
  if (!userId && opts.userName) {
    const usersFile = join(dir, "users.json");
    if (existsSync(usersFile)) {
      const users = JSON.parse(readFileSync(usersFile, "utf-8")) as Array<{
        id: string;
        name: string;
        real_name?: string;
        profile?: { display_name?: string };
      }>;
      const match = users.find(
        (u) =>
          u.name === opts.userName ||
          u.real_name === opts.userName ||
          u.profile?.display_name === opts.userName
      );
      if (match) userId = match.id;
    }
  }

  const messages: RawMessage[] = [];
  const channels = readdirSync(dir, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);

  for (const channel of channels) {
    const channelDir = join(dir, channel);
    const jsonFiles = readdirSync(channelDir).filter(
      (f) => extname(f) === ".json"
    );

    for (const file of jsonFiles) {
      const raw = readFileSync(join(channelDir, file), "utf-8");
      let msgs: SlackMessage[];
      try {
        msgs = JSON.parse(raw);
      } catch {
        continue;
      }

      for (const msg of msgs) {
        if (msg.type !== "message" || msg.subtype) continue;
        if (!msg.text || !msg.ts) continue;
        if (userId && msg.user !== userId) continue;

        const timestamp = new Date(parseFloat(msg.ts) * 1000);
        if (opts.since && timestamp < opts.since) continue;

        messages.push({
          id: `slack-${msg.ts}`,
          text: cleanSlackText(msg.text),
          platform: "slack",
          timestamp,
          isFromUser: true,
          metadata: {
            channel,
            threadTs: msg.thread_ts,
            userId: msg.user,
          },
        });
      }
    }
  }

  messages.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
  return messages;
}

function cleanSlackText(text: string): string {
  // strip user mentions <@U123> -> @user
  let cleaned = text.replace(/<@[A-Z0-9]+>/g, "@user");
  // strip channel refs <#C123|channel> -> #channel
  cleaned = cleaned.replace(/<#[A-Z0-9]+\|([^>]+)>/g, "#$1");
  // strip URLs <http://...|label> -> label or url
  cleaned = cleaned.replace(/<(https?:\/\/[^|>]+)\|([^>]+)>/g, "$2");
  cleaned = cleaned.replace(/<(https?:\/\/[^>]+)>/g, "$1");
  return cleaned;
}
