import { readFileSync, existsSync } from "fs";
import { join } from "path";
import { execSync } from "child_process";
import type { RawMessage } from "../corpus/types.js";

// Twitter/X data archive format:
// ZIP containing data/ directory with JS files.
// tweets.js: window.YTD.tweet.part0 = [ { tweet: { full_text, created_at, id_str, ... } } ]
// direct-messages.js: similar structure

interface TweetEntry {
  tweet: {
    id_str: string;
    full_text: string;
    created_at: string;
    in_reply_to_status_id_str?: string;
    in_reply_to_screen_name?: string;
  };
}

export interface TwitterImportOptions {
  file: string; // path to ZIP or extracted archive directory
  since?: Date;
  includeDMs?: boolean;
}

export function importTwitter(opts: TwitterImportOptions): RawMessage[] {
  let dir = opts.file;

  if (opts.file.endsWith(".zip")) {
    dir = "/tmp/replicateme-twitter-export";
    execSync(`rm -rf ${dir} && mkdir -p ${dir} && unzip -q -o "${opts.file}" -d ${dir}`);
  }

  if (!existsSync(dir)) {
    throw new Error(`Twitter archive not found at ${dir}`);
  }

  // twitter archive may have a nested directory
  const dataDir = existsSync(join(dir, "data")) ? join(dir, "data") : dir;

  const messages: RawMessage[] = [];

  // import tweets
  const tweetsFile = findFile(dataDir, ["tweets.js", "tweet.js"]);
  if (tweetsFile) {
    const tweets = parseTwitterJS<TweetEntry[]>(tweetsFile);
    for (const entry of tweets) {
      const t = entry.tweet;
      if (!t.full_text) continue;

      const timestamp = new Date(t.created_at);
      if (opts.since && timestamp < opts.since) continue;

      messages.push({
        id: `twitter-${t.id_str}`,
        text: cleanTweetText(t.full_text),
        platform: "twitter",
        timestamp,
        isFromUser: true,
        metadata: {
          isReply: !!t.in_reply_to_status_id_str,
          replyTo: t.in_reply_to_screen_name,
        },
      });
    }
  }

  // import DMs if requested
  if (opts.includeDMs !== false) {
    const dmsFile = findFile(dataDir, [
      "direct-messages.js",
      "direct_messages.js",
    ]);
    if (dmsFile) {
      const dms = parseTwitterJS<Array<{ dmConversation: DMConversation }>>(dmsFile);
      for (const conv of dms) {
        for (const msg of conv.dmConversation?.messages ?? []) {
          const dm = msg.messageCreate;
          if (!dm?.text) continue;

          const timestamp = new Date(parseInt(dm.createdAt));
          if (opts.since && timestamp < opts.since) continue;

          // we can't reliably tell which is "from me" without the user ID,
          // but twitter archive DMs include senderId - mark all for now,
          // user can filter later
          messages.push({
            id: `twitter-dm-${dm.id}`,
            text: dm.text,
            platform: "twitter",
            timestamp,
            isFromUser: true,
            metadata: {
              isDM: true,
              senderId: dm.senderId,
              recipientId: dm.recipientId,
            },
          });
        }
      }
    }
  }

  messages.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
  return messages;
}

interface DMConversation {
  messages: Array<{
    messageCreate?: {
      id: string;
      text: string;
      createdAt: string;
      senderId: string;
      recipientId: string;
    };
  }>;
}

function parseTwitterJS<T>(filePath: string): T {
  let content = readFileSync(filePath, "utf-8");
  // strip the "window.YTD.xxx.part0 = " prefix
  const eqIdx = content.indexOf("=");
  if (eqIdx !== -1) {
    content = content.substring(eqIdx + 1).trim();
  }
  // strip trailing semicolon
  if (content.endsWith(";")) {
    content = content.slice(0, -1);
  }
  return JSON.parse(content);
}

function findFile(dir: string, names: string[]): string | null {
  for (const name of names) {
    const path = join(dir, name);
    if (existsSync(path)) return path;
  }
  return null;
}

function cleanTweetText(text: string): string {
  // remove t.co URLs
  let cleaned = text.replace(/https?:\/\/t\.co\/\w+/g, "").trim();
  // decode HTML entities
  cleaned = cleaned
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"');
  return cleaned;
}
