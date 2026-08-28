import { readFileSync, existsSync, readdirSync } from "fs";
import { join } from "path";
import { execSync } from "child_process";
import type { RawMessage } from "../corpus/types.js";

// Instagram data download format:
// ZIP containing:
// - messages/inbox/<conversation>/message_1.json (DMs)
// - content/posts_1.json (post captions)
// - comments/post_comments_1.json (comments on other posts)
//
// Message format:
// { participants: [...], messages: [{ sender_name, timestamp_ms, content, ... }] }
//
// Posts format:
// [{ media: [...], title: "caption text", creation_timestamp: 123456 }]

interface IGMessage {
  sender_name: string;
  timestamp_ms: number;
  content?: string;
  type?: string;
}

interface IGConversation {
  participants: Array<{ name: string }>;
  messages: IGMessage[];
}

interface IGPost {
  title?: string;
  creation_timestamp?: number;
  media?: Array<{ title?: string; creation_timestamp?: number }>;
}

interface IGComment {
  string_map_data?: {
    Comment?: { value?: string; timestamp?: number };
    "Media Owner"?: { value?: string };
  };
}

export interface InstagramImportOptions {
  file: string; // path to ZIP or extracted directory
  username?: string; // your Instagram username to identify your messages
  since?: Date;
}

export function importInstagram(opts: InstagramImportOptions): RawMessage[] {
  let dir = opts.file;

  if (opts.file.endsWith(".zip")) {
    dir = "/tmp/replicateme-instagram-export";
    execSync(`rm -rf ${dir} && mkdir -p ${dir} && unzip -q -o "${opts.file}" -d ${dir}`);
  }

  if (!existsSync(dir)) {
    throw new Error(`Instagram export not found at ${dir}`);
  }

  const messages: RawMessage[] = [];

  // import DMs
  const inboxDir = join(dir, "messages", "inbox");
  if (existsSync(inboxDir)) {
    const conversations = readdirSync(inboxDir);
    for (const conv of conversations) {
      const convDir = join(inboxDir, conv);
      // each conversation may have message_1.json, message_2.json, etc.
      const msgFiles = readdirSync(convDir).filter((f) =>
        f.startsWith("message_") && f.endsWith(".json")
      );

      for (const file of msgFiles) {
        let data: IGConversation;
        try {
          data = JSON.parse(readFileSync(join(convDir, file), "utf-8"));
        } catch {
          continue;
        }

        for (const msg of data.messages ?? []) {
          if (!msg.content) continue;
          // filter to user's messages if username provided
          if (opts.username && decodeIGText(msg.sender_name) !== opts.username)
            continue;

          const timestamp = new Date(msg.timestamp_ms);
          if (opts.since && timestamp < opts.since) continue;

          messages.push({
            id: `instagram-dm-${msg.timestamp_ms}`,
            text: decodeIGText(msg.content),
            platform: "instagram",
            timestamp,
            isFromUser: true,
            metadata: {
              type: "dm",
              conversation: conv,
            },
          });
        }
      }
    }
  }

  // import post captions
  const postsFile = findPostsFile(dir);
  if (postsFile) {
    let posts: IGPost[];
    try {
      posts = JSON.parse(readFileSync(postsFile, "utf-8"));
    } catch {
      posts = [];
    }

    for (let i = 0; i < posts.length; i++) {
      const post = posts[i]!;
      const caption =
        post.title ??
        post.media?.[0]?.title;
      if (!caption) continue;

      const ts =
        post.creation_timestamp ??
        post.media?.[0]?.creation_timestamp;
      if (!ts) continue;

      const timestamp = new Date(ts * 1000);
      if (opts.since && timestamp < opts.since) continue;

      messages.push({
        id: `instagram-post-${i}`,
        text: decodeIGText(caption),
        platform: "instagram",
        timestamp,
        isFromUser: true,
        metadata: { type: "post" },
      });
    }
  }

  // import comments
  const commentsFile = findCommentsFile(dir);
  if (commentsFile) {
    let comments: IGComment[];
    try {
      const raw = JSON.parse(readFileSync(commentsFile, "utf-8"));
      comments = Array.isArray(raw) ? raw : raw.comments_media_comments ?? [];
    } catch {
      comments = [];
    }

    for (let i = 0; i < comments.length; i++) {
      const c = comments[i]!;
      const text = c.string_map_data?.Comment?.value;
      const ts = c.string_map_data?.Comment?.timestamp;
      if (!text || !ts) continue;

      const timestamp = new Date(ts * 1000);
      if (opts.since && timestamp < opts.since) continue;

      messages.push({
        id: `instagram-comment-${i}`,
        text: decodeIGText(text),
        platform: "instagram",
        timestamp,
        isFromUser: true,
        metadata: {
          type: "comment",
          mediaOwner: c.string_map_data?.["Media Owner"]?.value,
        },
      });
    }
  }

  messages.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
  return messages;
}

function findPostsFile(dir: string): string | null {
  const candidates = [
    join(dir, "content", "posts_1.json"),
    join(dir, "content", "posts.json"),
    join(dir, "posts_1.json"),
  ];
  return candidates.find((f) => existsSync(f)) ?? null;
}

function findCommentsFile(dir: string): string | null {
  const candidates = [
    join(dir, "comments", "post_comments_1.json"),
    join(dir, "comments", "post_comments.json"),
    join(dir, "comments_1.json"),
  ];
  return candidates.find((f) => existsSync(f)) ?? null;
}

// Instagram encodes non-ASCII as escaped UTF-8 bytes in latin1
function decodeIGText(text: string): string {
  try {
    return Buffer.from(
      text.replace(/\\u00([0-9a-f]{2})/gi, (_, hex) =>
        String.fromCharCode(parseInt(hex, 16))
      ),
      "latin1"
    ).toString("utf-8");
  } catch {
    return text;
  }
}
