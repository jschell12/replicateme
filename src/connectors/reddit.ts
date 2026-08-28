import { readFileSync, existsSync } from "fs";
import { join } from "path";
import { execSync } from "child_process";
import type { RawMessage } from "../corpus/types.js";

// Reddit data archive format:
// ZIP containing CSV files:
// comments.csv: id,permalink,date,ip,subreddit,gildings,link,parent,body,media
// posts.csv: id,permalink,date,ip,subreddit,gildings,title,url,body,media

export interface RedditImportOptions {
  file: string; // path to ZIP or extracted directory
  since?: Date;
}

export function importReddit(opts: RedditImportOptions): RawMessage[] {
  let dir = opts.file;

  if (opts.file.endsWith(".zip")) {
    dir = "/tmp/replicateme-reddit-export";
    execSync(`rm -rf ${dir} && mkdir -p ${dir} && unzip -q -o "${opts.file}" -d ${dir}`);
  }

  if (!existsSync(dir)) {
    throw new Error(`Reddit export not found at ${dir}`);
  }

  const messages: RawMessage[] = [];

  // import comments
  const commentsFile = join(dir, "comments.csv");
  if (existsSync(commentsFile)) {
    const csv = readFileSync(commentsFile, "utf-8");
    const rows = parseCSV(csv);

    for (const row of rows) {
      // columns: id,permalink,date,ip,subreddit,gildings,link,parent,body,media
      const id = row[0];
      const dateStr = row[2];
      const subreddit = row[4];
      const body = row[8];

      if (!id || !body || !dateStr) continue;

      const timestamp = new Date(dateStr);
      if (isNaN(timestamp.getTime())) continue;
      if (opts.since && timestamp < opts.since) continue;

      messages.push({
        id: `reddit-comment-${id}`,
        text: cleanRedditText(body),
        platform: "reddit",
        timestamp,
        isFromUser: true,
        metadata: {
          type: "comment",
          subreddit,
        },
      });
    }
  }

  // import posts
  const postsFile = join(dir, "posts.csv");
  if (existsSync(postsFile)) {
    const csv = readFileSync(postsFile, "utf-8");
    const rows = parseCSV(csv);

    for (const row of rows) {
      // columns: id,permalink,date,ip,subreddit,gildings,title,url,body,media
      const id = row[0];
      const dateStr = row[2];
      const subreddit = row[4];
      const title = row[6];
      const body = row[8];

      if (!id || !dateStr) continue;
      // posts might have just a title, or title + body
      const text = [title, body].filter(Boolean).join("\n\n");
      if (!text) continue;

      const timestamp = new Date(dateStr);
      if (isNaN(timestamp.getTime())) continue;
      if (opts.since && timestamp < opts.since) continue;

      messages.push({
        id: `reddit-post-${id}`,
        text: cleanRedditText(text),
        platform: "reddit",
        timestamp,
        isFromUser: true,
        metadata: {
          type: "post",
          subreddit,
          title,
        },
      });
    }
  }

  messages.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
  return messages;
}

function cleanRedditText(text: string): string {
  // unescape HTML entities
  let cleaned = text
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'");

  // strip markdown links [text](url) -> text
  cleaned = cleaned.replace(/\[([^\]]+)\]\([^)]+\)/g, "$1");

  return cleaned.trim();
}

function parseCSV(content: string): string[][] {
  const rows: string[][] = [];
  const lines = content.split("\n");

  // skip header
  for (let i = 1; i < lines.length; i++) {
    const line = lines[i]!.trim();
    if (!line) continue;
    rows.push(parseCSVLine(line));
  }

  return rows;
}

function parseCSVLine(line: string): string[] {
  const fields: string[] = [];
  let current = "";
  let inQuotes = false;

  for (let i = 0; i < line.length; i++) {
    const ch = line[i]!;
    if (inQuotes) {
      if (ch === '"') {
        if (i + 1 < line.length && line[i + 1] === '"') {
          current += '"';
          i++;
        } else {
          inQuotes = false;
        }
      } else {
        current += ch;
      }
    } else {
      if (ch === '"') {
        inQuotes = true;
      } else if (ch === ",") {
        fields.push(current);
        current = "";
      } else {
        current += ch;
      }
    }
  }
  fields.push(current);
  return fields;
}
