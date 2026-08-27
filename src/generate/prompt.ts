import type { RawMessage, StyleProfile } from "../corpus/types.js";
import { styleProfileToPrompt } from "../style/analyzer.js";

export interface GenerateOptions {
  platform: string;
  context: string;
  similarMessages: RawMessage[];
  styleProfile: StyleProfile;
  quirkLevel: number; // 0-100
  instruction?: string;
}

export function buildSystemPrompt(opts: GenerateOptions): string {
  const { styleProfile, similarMessages, quirkLevel, platform } = opts;

  const lines: string[] = [];

  lines.push(
    "You are a writing style replicator. Your job is to write messages that sound exactly like a specific person based on their writing patterns and examples."
  );
  lines.push("");
  lines.push(
    "You will be given their style profile (statistical patterns), example messages they have written, and context for what needs to be written. Your output should be indistinguishable from something they would actually write."
  );
  lines.push("");

  lines.push(`## Platform: ${platform}`);
  lines.push("");
  lines.push(platformGuidance(platform));
  lines.push("");

  lines.push(styleProfileToPrompt(styleProfile));
  lines.push("");

  lines.push(`## Quirk level: ${quirkLevel}%`);
  lines.push("");
  if (quirkLevel === 0) {
    lines.push(
      "Use their vocabulary and voice but with clean grammar and spelling. No intentional errors."
    );
  } else if (quirkLevel <= 30) {
    lines.push(
      "Mostly clean writing but include their most common habits like missing apostrophes in contractions (im, dont, cant) occasionally."
    );
  } else if (quirkLevel <= 70) {
    lines.push(
      "Write naturally in their style including their typical quirks: missing apostrophes, lowercase starts, fragments, and their common phrases. This should read like their normal messages."
    );
  } else {
    lines.push(
      "Full authenticity. Include all their writing quirks: missing apostrophes, lowercase i, sentence fragments, double spaces, repeated words, minimal punctuation. This should be indistinguishable from their actual messages."
    );
  }
  lines.push("");

  if (similarMessages.length > 0) {
    lines.push("## Example messages from this person");
    lines.push("");
    for (const msg of similarMessages.slice(0, 20)) {
      lines.push(`- "${msg.text}"`);
    }
    lines.push("");
  }

  lines.push("## Rules");
  lines.push("");
  lines.push("- Output ONLY the message text, nothing else");
  lines.push("- Do not add quotes around the message");
  lines.push("- Do not explain or caveat");
  lines.push("- Match their typical message length for this type of content");
  lines.push(
    "- If they rarely use periods, dont add periods. If they rarely capitalize, dont capitalize."
  );
  lines.push(
    "- Never use words or phrases this person wouldnt use based on their examples"
  );

  return lines.join("\n");
}

export function buildUserPrompt(opts: GenerateOptions): string {
  const lines: string[] = [];

  if (opts.instruction) {
    lines.push(`Task: ${opts.instruction}`);
    lines.push("");
  }

  lines.push("Context:");
  lines.push(opts.context);
  lines.push("");
  lines.push(
    "Write a response in this person's voice and style. Output only the message."
  );

  return lines.join("\n");
}

function platformGuidance(platform: string): string {
  switch (platform) {
    case "imessage":
      return "This is a text message / iMessage. Messages are typically very short, casual, and conversational. Multiple short messages are common instead of one long one.";
    case "slack":
      return "This is a Slack message at work. Slightly more structured than texts but still casual. May reference channels, threads, or colleagues.";
    case "email":
      return "This is an email. More structured than chat but still in the person's voice. Includes greeting and sign-off only if the person typically uses them.";
    case "github":
      return "This is a GitHub comment, PR description, or commit message. Technical and concise. Follows the person's typical commit/PR style.";
    case "twitter":
      return "This is a tweet or reply. Very concise (280 char limit). Matches the person's typical Twitter tone.";
    case "discord":
      return "This is a Discord message. Similar to texting but may be in a server with specific context.";
    case "reddit":
      return "This is a Reddit post or comment. May be longer and more detailed than texts but still in the person's voice.";
    default:
      return "Write in the person's natural style for this platform.";
  }
}
