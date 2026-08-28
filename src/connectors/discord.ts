import { readFileSync, existsSync, readdirSync } from "fs";
import { join } from "path";
import { execSync } from "child_process";
import type { RawMessage } from "../corpus/types.js";

// Discord data export format:
// ZIP containing messages/ directory with subdirectories per channel.
// Each channel dir has messages.csv with columns:
// ID,Timestamp,Contents,Attachments
// Also has index.json mapping channel IDs to names.

export interface DiscordImportOptions {
  file: string; // path to ZIP or extracted directory
  since?: Date;
}

export function importDiscord(opts: DiscordImportOptions): RawMessage[] {
  let dir = opts.file;

  if (opts.file.endsWith(".zip")) {
    dir = "/tmp/replicateme-discord-export";
    execSync(`rm -rf ${dir} && mkdir -p ${dir} && unzip -q -o "${opts.file}" -d ${dir}`);
  }

  if (!existsSync(dir)) {
    throw new Error(`Discord export not found at ${dir}`);
  }

  const messagesDir = existsSync(join(dir, "messages"))
    ? join(dir, "messages")
    : dir;

  // load channel index if available
  const channelNames: Record<string, string> = {};
  const indexFile = join(messagesDir, "index.json");
  if (existsSync(indexFile)) {
    const index = JSON.parse(readFileSync(indexFile, "utf-8")) as Record<
      string,
      string | null
    >;
    for (const [id, name] of Object.entries(index)) {
      if (name) channelNames[id] = name;
    }
  }

  const messages: RawMessage[] = [];

  const channelDirs = readdirSync(messagesDir, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);

  for (const channelId of channelDirs) {
    const csvPath = join(messagesDir, channelId, "messages.csv");
    if (!existsSync(csvPath)) continue;

    const channelName = channelNames[channelId] ?? channelId;
    const csv = readFileSync(csvPath, "utf-8");
    const rows = parseCSV(csv);

    for (const row of rows) {
      const [id, timestampStr, contents] = row;
      if (!id || !contents || !timestampStr) continue;

      const timestamp = new Date(timestampStr);
      if (isNaN(timestamp.getTime())) continue;
      if (opts.since && timestamp < opts.since) continue;

      messages.push({
        id: `discord-${id}`,
        text: contents,
        platform: "discord",
        timestamp,
        isFromUser: true, // discord export only contains your own messages
        metadata: {
          channel: channelName,
          channelId,
        },
      });
    }
  }

  messages.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
  return messages;
}

function parseCSV(content: string): string[][] {
  const rows: string[][] = [];
  const lines = content.split("\n");

  // skip header row
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
          i++; // skip escaped quote
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
