import Database from "better-sqlite3";
import { existsSync, mkdirSync } from "fs";
import { join } from "path";
import { homedir } from "os";
import type { RawMessage, Platform, StyleProfile } from "./types.js";

const CONFIG_DIR = join(homedir(), ".replicateme");
const DB_PATH = join(CONFIG_DIR, "corpus.db");

let _db: InstanceType<typeof Database> | null = null;

function getDb(): InstanceType<typeof Database> {
  if (_db) return _db;

  if (!existsSync(CONFIG_DIR)) {
    mkdirSync(CONFIG_DIR, { recursive: true });
  }

  _db = new Database(DB_PATH);
  _db.pragma("journal_mode = WAL");
  _db.pragma("foreign_keys = ON");

  _db.exec(`
    CREATE TABLE IF NOT EXISTS messages (
      id TEXT PRIMARY KEY,
      text TEXT NOT NULL,
      platform TEXT NOT NULL,
      timestamp TEXT NOT NULL,
      is_from_user INTEGER NOT NULL DEFAULT 1,
      metadata TEXT,
      ingested_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE INDEX IF NOT EXISTS idx_messages_platform ON messages(platform);
    CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);

    CREATE TABLE IF NOT EXISTS profiles (
      platform TEXT PRIMARY KEY,
      profile TEXT NOT NULL,
      message_count INTEGER NOT NULL,
      updated_at TEXT NOT NULL DEFAULT (datetime('now'))
    );

    CREATE TABLE IF NOT EXISTS ingest_log (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      source TEXT NOT NULL,
      message_count INTEGER NOT NULL,
      ingested_at TEXT NOT NULL DEFAULT (datetime('now'))
    );
  `);

  return _db;
}

export function storeMessages(messages: RawMessage[]): {
  inserted: number;
  skipped: number;
} {
  const db = getDb();

  const insert = db.prepare(`
    INSERT OR IGNORE INTO messages (id, text, platform, timestamp, is_from_user, metadata)
    VALUES (@id, @text, @platform, @timestamp, @isFromUser, @metadata)
  `);

  let inserted = 0;
  let skipped = 0;

  const tx = db.transaction(() => {
    for (const msg of messages) {
      const result = insert.run({
        id: msg.id,
        text: msg.text,
        platform: msg.platform,
        timestamp: msg.timestamp.toISOString(),
        isFromUser: msg.isFromUser ? 1 : 0,
        metadata: msg.metadata ? JSON.stringify(msg.metadata) : null,
      });
      if (result.changes > 0) {
        inserted++;
      } else {
        skipped++;
      }
    }
  });

  tx();
  return { inserted, skipped };
}

export function getMessages(opts?: {
  platform?: Platform;
  limit?: number;
  since?: Date;
  random?: boolean;
}): RawMessage[] {
  const db = getDb();

  const conditions: string[] = ["is_from_user = 1"];
  const params: Record<string, unknown> = {};

  if (opts?.platform) {
    conditions.push("platform = @platform");
    params.platform = opts.platform;
  }

  if (opts?.since) {
    conditions.push("timestamp >= @since");
    params.since = opts.since.toISOString();
  }

  const where = conditions.length > 0 ? `WHERE ${conditions.join(" AND ")}` : "";
  const order = opts?.random ? "ORDER BY RANDOM()" : "ORDER BY timestamp DESC";
  const limit = opts?.limit ? `LIMIT ${opts.limit}` : "";

  const rows = db
    .prepare(`SELECT * FROM messages ${where} ${order} ${limit}`)
    .all(params) as Array<{
    id: string;
    text: string;
    platform: string;
    timestamp: string;
    is_from_user: number;
    metadata: string | null;
  }>;

  return rows.map((row) => ({
    id: row.id,
    text: row.text,
    platform: row.platform as Platform,
    timestamp: new Date(row.timestamp),
    isFromUser: row.is_from_user === 1,
    metadata: row.metadata ? JSON.parse(row.metadata) : undefined,
  }));
}

export function getExamples(
  platform?: Platform,
  count: number = 20
): RawMessage[] {
  // prefer examples from the same platform, fall back to any
  if (platform) {
    const platformMsgs = getMessages({
      platform,
      limit: count,
      random: true,
    });
    if (platformMsgs.length >= count) return platformMsgs;

    // supplement with other platforms
    const remaining = count - platformMsgs.length;
    const otherMsgs = getMessages({ limit: remaining, random: true });
    return [...platformMsgs, ...otherMsgs];
  }

  return getMessages({ limit: count, random: true });
}

export function savePerPlatformProfile(
  platform: Platform | "combined",
  profile: StyleProfile,
  messageCount: number
): void {
  const db = getDb();
  db.prepare(`
    INSERT OR REPLACE INTO profiles (platform, profile, message_count, updated_at)
    VALUES (@platform, @profile, @messageCount, datetime('now'))
  `).run({
    platform,
    profile: JSON.stringify(profile),
    messageCount,
  });
}

export function getPerPlatformProfile(
  platform: Platform | "combined"
): { profile: StyleProfile; messageCount: number } | null {
  const db = getDb();
  const row = db
    .prepare("SELECT profile, message_count FROM profiles WHERE platform = ?")
    .get(platform) as { profile: string; message_count: number } | undefined;

  if (!row) return null;
  return {
    profile: JSON.parse(row.profile),
    messageCount: row.message_count,
  };
}

export function getAllPlatformProfiles(): Array<{
  platform: string;
  profile: StyleProfile;
  messageCount: number;
}> {
  const db = getDb();
  const rows = db
    .prepare("SELECT platform, profile, message_count FROM profiles ORDER BY platform")
    .all() as Array<{ platform: string; profile: string; message_count: number }>;

  return rows.map((row) => ({
    platform: row.platform,
    profile: JSON.parse(row.profile),
    messageCount: row.message_count,
  }));
}

export function logIngest(source: string, messageCount: number): void {
  const db = getDb();
  db.prepare(
    "INSERT INTO ingest_log (source, message_count) VALUES (?, ?)"
  ).run(source, messageCount);
}

export function getCorpusStats(): {
  totalMessages: number;
  byPlatform: Array<{ platform: string; count: number }>;
  profiles: Array<{ platform: string; messageCount: number; updatedAt: string }>;
} {
  const db = getDb();

  const total = db
    .prepare("SELECT COUNT(*) as count FROM messages")
    .get() as { count: number };

  const byPlatform = db
    .prepare(
      "SELECT platform, COUNT(*) as count FROM messages GROUP BY platform ORDER BY count DESC"
    )
    .all() as Array<{ platform: string; count: number }>;

  const profiles = db
    .prepare(
      "SELECT platform, message_count, updated_at as updatedAt FROM profiles ORDER BY platform"
    )
    .all() as Array<{ platform: string; message_count: number; updatedAt: string }>;

  return {
    totalMessages: total.count,
    byPlatform,
    profiles: profiles.map((p) => ({
      platform: p.platform,
      messageCount: p.message_count,
      updatedAt: p.updatedAt,
    })),
  };
}

export function closeDb(): void {
  if (_db) {
    _db.close();
    _db = null;
  }
}
