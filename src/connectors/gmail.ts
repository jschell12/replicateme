import { readFileSync, existsSync } from "fs";
import { execSync } from "child_process";
import type { RawMessage } from "../corpus/types.js";

// Google Takeout Gmail format:
// ZIP or directory containing .mbox files
// mbox is a concatenation of RFC 2822 messages separated by "From " lines

export interface GmailImportOptions {
  file: string; // path to .mbox file, ZIP, or extracted Takeout directory
  email?: string; // user's email to identify sent messages
  since?: Date;
}

export function importGmail(opts: GmailImportOptions): RawMessage[] {
  let mboxPath = opts.file;

  // if ZIP, extract and find the mbox
  if (opts.file.endsWith(".zip")) {
    const dir = "/tmp/replicateme-gmail-export";
    execSync(`rm -rf ${dir} && mkdir -p ${dir} && unzip -q -o "${opts.file}" -d ${dir}`);
    // Takeout structure: Takeout/Mail/*.mbox
    const found = execSync(`find ${dir} -name "*.mbox" -type f 2>/dev/null`)
      .toString()
      .trim()
      .split("\n")
      .filter(Boolean);
    if (found.length === 0) {
      throw new Error("No .mbox files found in archive");
    }
    // prefer "Sent Mail.mbox" or "All mail.mbox"
    mboxPath =
      found.find((f) => /sent/i.test(f)) ??
      found.find((f) => /all.mail/i.test(f)) ??
      found[0]!;
  }

  if (!existsSync(mboxPath)) {
    throw new Error(`mbox file not found at ${mboxPath}`);
  }

  const content = readFileSync(mboxPath, "utf-8");
  return parseMbox(content, opts);
}

function parseMbox(content: string, opts: GmailImportOptions): RawMessage[] {
  const messages: RawMessage[] = [];

  // split on mbox "From " delimiter lines
  const rawMessages = content.split(/^From /m).filter((s) => s.trim());

  for (let i = 0; i < rawMessages.length; i++) {
    const raw = rawMessages[i]!;

    // split headers from body at first blank line
    const headerEnd = raw.search(/\n\n/);
    if (headerEnd === -1) continue;

    const headerBlock = raw.substring(0, headerEnd);
    const body = raw.substring(headerEnd + 2);

    const headers = parseHeaders(headerBlock);

    const from = headers.from ?? "";
    const date = headers.date;
    const subject = headers.subject ?? "";

    // only keep sent messages
    const isSent =
      opts.email
        ? from.toLowerCase().includes(opts.email.toLowerCase())
        : /sent/i.test(headers["x-gmail-labels"] ?? "");

    if (!isSent) continue;

    let timestamp: Date;
    try {
      timestamp = date ? new Date(date) : new Date(0);
      if (isNaN(timestamp.getTime())) continue;
    } catch {
      continue;
    }

    if (opts.since && timestamp < opts.since) continue;

    // extract plain text body, strip quoted replies
    const plainText = extractPlainText(body);
    if (!plainText || plainText.length < 2) continue;

    messages.push({
      id: `gmail-${i}`,
      text: plainText,
      platform: "email",
      timestamp,
      isFromUser: true,
      metadata: {
        subject,
        to: headers.to,
      },
    });
  }

  messages.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
  return messages;
}

function parseHeaders(block: string): Record<string, string> {
  const headers: Record<string, string> = {};
  // unfold continuation lines
  const unfolded = block.replace(/\n[ \t]+/g, " ");
  for (const line of unfolded.split("\n")) {
    const idx = line.indexOf(":");
    if (idx === -1) continue;
    const key = line.substring(0, idx).trim().toLowerCase();
    const value = line.substring(idx + 1).trim();
    headers[key] = value;
  }
  return headers;
}

function extractPlainText(body: string): string {
  let text = body;

  // if multipart, try to extract text/plain part
  if (text.includes("Content-Type: text/plain")) {
    const parts = text.split(/--[^\n]+/);
    const plainPart = parts.find((p) => p.includes("text/plain"));
    if (plainPart) {
      const bodyStart = plainPart.search(/\n\n/);
      if (bodyStart !== -1) {
        text = plainPart.substring(bodyStart + 2);
      }
    }
  }

  // strip quoted replies (lines starting with >)
  text = text
    .split("\n")
    .filter((line) => !line.startsWith(">"))
    .join("\n");

  // strip "On ... wrote:" blocks
  text = text.replace(/On .+wrote:\s*$/gm, "");

  // strip signatures (-- delimiter)
  const sigIdx = text.indexOf("\n-- \n");
  if (sigIdx !== -1) {
    text = text.substring(0, sigIdx);
  }

  // decode quoted-printable soft line breaks
  text = text.replace(/=\r?\n/g, "");
  // decode common QP entities
  text = text.replace(/=([0-9A-F]{2})/gi, (_, hex) =>
    String.fromCharCode(parseInt(hex, 16))
  );

  return text.trim();
}
