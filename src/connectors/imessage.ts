import Database from "better-sqlite3";
import { homedir } from "os";
import { existsSync, copyFileSync } from "fs";
import { join } from "path";
import type { RawMessage } from "../corpus/types.js";

const APPLE_EPOCH_OFFSET = 978307200;

interface IMRow {
  rowid: number;
  text: string;
  date: number;
  is_from_me: number;
  handle_id: string | null;
  cache_roomnames: string | null;
  service: string;
  thread_originator_guid: string | null;
}

export interface ImportOptions {
  since?: Date;
  dbPath?: string;
  copyToTmp?: boolean;
}

export function importIMessages(opts: ImportOptions = {}): RawMessage[] {
  const source = opts.dbPath ?? join(homedir(), "Library/Messages/chat.db");

  if (!existsSync(source)) {
    throw new Error(`iMessage database not found at ${source}`);
  }

  let dbPath = source;
  if (opts.copyToTmp !== false) {
    dbPath = "/tmp/replicateme-chat.db";
    copyFileSync(source, dbPath);
  }

  const db = new Database(dbPath, { readonly: true });

  let query = `
    SELECT
      m.ROWID as rowid,
      m.text,
      m.date,
      m.is_from_me,
      h.id as handle_id,
      m.cache_roomnames,
      m.service,
      m.thread_originator_guid
    FROM message m
    LEFT JOIN handle h ON m.handle_id = h.ROWID
    WHERE m.is_from_me = 1
      AND m.text IS NOT NULL
      AND length(m.text) > 0
      AND m.associated_message_type = 0
  `;

  const params: Record<string, unknown> = {};

  if (opts.since) {
    const appleTimestamp =
      (opts.since.getTime() / 1000 - APPLE_EPOCH_OFFSET) * 1000000000;
    query += ` AND m.date >= @since`;
    params.since = appleTimestamp;
  }

  query += ` ORDER BY m.date ASC`;

  const rows = db.prepare(query).all(params) as IMRow[];

  const messages: RawMessage[] = rows.map((row) => ({
    id: `imessage-${row.rowid}`,
    text: row.text,
    platform: "imessage" as const,
    timestamp: new Date(
      (row.date / 1000000000 + APPLE_EPOCH_OFFSET) * 1000
    ),
    isFromUser: true,
    metadata: {
      recipient: row.handle_id ?? undefined,
      isGroupChat: row.cache_roomnames !== null,
      service: row.service,
      threadId: row.thread_originator_guid ?? undefined,
    },
  }));

  db.close();
  return messages;
}

export function getConversationContext(
  messageId: number,
  windowSize: number = 5,
  dbPath: string = "/tmp/replicateme-chat.db"
): RawMessage[] {
  const db = new Database(dbPath, { readonly: true });

  const target = db
    .prepare(
      "SELECT handle_id, cache_roomnames, date FROM message WHERE ROWID = ?"
    )
    .get(messageId) as
    | { handle_id: number; cache_roomnames: string | null; date: number }
    | undefined;

  if (!target) {
    db.close();
    return [];
  }

  let contextQuery: string;
  const params: Record<string, unknown> = {
    date: target.date,
    limit: windowSize,
  };

  if (target.cache_roomnames) {
    contextQuery = `
      SELECT m.ROWID as rowid, m.text, m.date, m.is_from_me,
             h.id as handle_id, m.cache_roomnames, m.service,
             m.thread_originator_guid
      FROM message m
      LEFT JOIN handle h ON m.handle_id = h.ROWID
      WHERE m.cache_roomnames = @roomname
        AND m.text IS NOT NULL AND length(m.text) > 0
        AND m.date < @date
        AND m.associated_message_type = 0
      ORDER BY m.date DESC
      LIMIT @limit
    `;
    params.roomname = target.cache_roomnames;
  } else {
    contextQuery = `
      SELECT m.ROWID as rowid, m.text, m.date, m.is_from_me,
             h.id as handle_id, m.cache_roomnames, m.service,
             m.thread_originator_guid
      FROM message m
      LEFT JOIN handle h ON m.handle_id = h.ROWID
      WHERE m.handle_id = @handleId
        AND m.text IS NOT NULL AND length(m.text) > 0
        AND m.date < @date
        AND m.associated_message_type = 0
      ORDER BY m.date DESC
      LIMIT @limit
    `;
    params.handleId = target.handle_id;
  }

  const rows = db.prepare(contextQuery).all(params) as IMRow[];
  db.close();

  return rows.reverse().map((row) => ({
    id: `imessage-${row.rowid}`,
    text: row.text,
    platform: "imessage" as const,
    timestamp: new Date(
      (row.date / 1000000000 + APPLE_EPOCH_OFFSET) * 1000
    ),
    isFromUser: row.is_from_me === 1,
    metadata: {
      recipient: row.handle_id ?? undefined,
      isGroupChat: row.cache_roomnames !== null,
      service: row.service,
      threadId: row.thread_originator_guid ?? undefined,
    },
  }));
}
