import type { RawMessage, StyleProfile, TypicalError } from "../corpus/types.js";

export function analyzeStyle(messages: RawMessage[]): StyleProfile {
  if (messages.length === 0) {
    throw new Error("No messages to analyze");
  }

  const texts = messages.map((m) => m.text);

  const avgLength =
    texts.reduce((sum, t) => sum + t.length, 0) / texts.length;

  const capsFirst =
    texts.filter((t) => /^[A-Z]/.test(t)).length / texts.length;

  const withContractions = texts.filter((t) =>
    /\b(i'm|i'll|i've|it's|that's|there's|don't|can't|won't|didn't|isn't|aren't|wasn't|weren't|hasn't|haven't|couldn't|wouldn't|shouldn't|y'all|let's|he's|she's|we're|they're|you're|who's|what's|where's|how's)\b/i.test(
      t
    )
  ).length;
  const contractionsRatio = withContractions / texts.length;

  const withPeriods =
    texts.filter((t) => /\.\s*$/.test(t)).length / texts.length;
  const withCommas = texts.filter((t) => t.includes(",")).length / texts.length;
  const withExclamation =
    texts.filter((t) => t.includes("!")).length / texts.length;
  const withQuestion =
    texts.filter((t) => t.includes("?")).length / texts.length;

  const emojiRegex =
    /[\u{1F600}-\u{1F64F}\u{1F300}-\u{1F5FF}\u{1F680}-\u{1F6FF}\u{1F1E0}-\u{1F1FF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}]/u;
  const withEmoji =
    texts.filter((t) => emojiRegex.test(t)).length / texts.length;

  // sentence fragments: messages without a verb-like word and under 30 chars
  const fragmentish = texts.filter(
    (t) => t.length < 30 && t.split(/\s+/).length <= 5
  ).length;
  const fragmentRatio = fragmentish / texts.length;

  // lowercase "i" as a standalone word (not I'm, I'll, etc.)
  const lowercaseI = texts.filter((t) =>
    /\bi\b(?!')/.test(t)
  ).length;
  const iTotal = texts.filter((t) =>
    /\b[iI]\b/i.test(t)
  ).length;
  const lowercaseIRatio = iTotal > 0 ? lowercaseI / iTotal : 0;

  const commonPhrases = findCommonPhrases(texts);
  const typicalErrors = findTypicalErrors(texts);

  return {
    averageLength: Math.round(avgLength),
    capitalizesFirstWord: round(capsFirst),
    usesContractions: round(contractionsRatio),
    usesPeriods: round(withPeriods),
    usesCommas: round(withCommas),
    usesExclamation: round(withExclamation),
    usesQuestionMark: round(withQuestion),
    usesEmoji: round(withEmoji),
    commonPhrases,
    typicalErrors,
    sentenceFragmentRatio: round(fragmentRatio),
    lowercaseIRatio: round(lowercaseIRatio),
  };
}

function round(n: number): number {
  return Math.round(n * 1000) / 1000;
}

function findCommonPhrases(texts: string[], minCount = 5): string[] {
  const ngrams = new Map<string, number>();

  for (const text of texts) {
    const words = text.toLowerCase().split(/\s+/);
    // bigrams and trigrams
    for (let n = 2; n <= 3; n++) {
      for (let i = 0; i <= words.length - n; i++) {
        const gram = words.slice(i, i + n).join(" ");
        ngrams.set(gram, (ngrams.get(gram) ?? 0) + 1);
      }
    }
  }

  return Array.from(ngrams.entries())
    .filter(([, count]) => count >= minCount)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 30)
    .map(([phrase]) => phrase);
}

function findTypicalErrors(texts: string[]): TypicalError[] {
  const errors: TypicalError[] = [];

  // missing apostrophes in common contractions
  const missingApostrophe = [
    { wrong: /\bdont\b/gi, label: "dont (missing apostrophe)" },
    { wrong: /\bcant\b/gi, label: "cant (missing apostrophe)" },
    { wrong: /\bwont\b/gi, label: "wont (missing apostrophe)" },
    { wrong: /\bdidnt\b/gi, label: "didnt (missing apostrophe)" },
    { wrong: /\bisnt\b/gi, label: "isnt (missing apostrophe)" },
    { wrong: /\bthats\b/gi, label: "thats (missing apostrophe)" },
    { wrong: /\btheres\b/gi, label: "theres (missing apostrophe)" },
    { wrong: /\bits\b/gi, label: "its (possessive vs contraction)" },
    { wrong: /\bim\b/gi, label: "im (missing apostrophe)" },
    { wrong: /\byoure\b/gi, label: "youre (missing apostrophe)" },
  ];

  for (const { wrong, label } of missingApostrophe) {
    const matches = texts.filter((t) => wrong.test(t));
    if (matches.length >= 3) {
      errors.push({
        pattern: label,
        frequency: matches.length,
        examples: matches.slice(0, 3).map((t) => t.substring(0, 80)),
      });
    }
  }

  // double spaces
  const doubleSpaces = texts.filter((t) => /  /.test(t));
  if (doubleSpaces.length >= 3) {
    errors.push({
      pattern: "double spaces",
      frequency: doubleSpaces.length,
      examples: doubleSpaces.slice(0, 3).map((t) => t.substring(0, 80)),
    });
  }

  // repeated words
  const repeatedWords = texts.filter((t) =>
    /\b(\w+)\s+\1\b/i.test(t)
  );
  if (repeatedWords.length >= 2) {
    errors.push({
      pattern: "repeated words",
      frequency: repeatedWords.length,
      examples: repeatedWords.slice(0, 3).map((t) => t.substring(0, 80)),
    });
  }

  return errors.sort((a, b) => b.frequency - a.frequency);
}

export function styleProfileToPrompt(profile: StyleProfile): string {
  const lines: string[] = [];

  lines.push("## Writing style characteristics");
  lines.push("");
  lines.push(`- Average message length: ${profile.averageLength} characters`);
  lines.push(
    `- Capitalizes first word: ${pct(profile.capitalizesFirstWord)} of the time`
  );
  lines.push(`- Uses contractions: ${pct(profile.usesContractions)} of the time`);
  lines.push(
    `- Ends with period: ${pct(profile.usesPeriods)} of the time`
  );
  lines.push(`- Uses commas: ${pct(profile.usesCommas)} of the time`);
  lines.push(`- Uses exclamation marks: ${pct(profile.usesExclamation)} of the time`);
  lines.push(`- Uses question marks: ${pct(profile.usesQuestionMark)} of the time`);
  lines.push(`- Uses emoji: ${pct(profile.usesEmoji)} of the time`);
  lines.push(
    `- Short fragment messages: ${pct(profile.sentenceFragmentRatio)} of the time`
  );

  if (profile.lowercaseIRatio > 0.1) {
    lines.push(
      `- Uses lowercase "i" instead of "I": ${pct(profile.lowercaseIRatio)} of the time`
    );
  }

  if (profile.commonPhrases.length > 0) {
    lines.push("");
    lines.push("## Common phrases");
    lines.push(
      profile.commonPhrases.slice(0, 15).map((p) => `- "${p}"`).join("\n")
    );
  }

  if (profile.typicalErrors.length > 0) {
    lines.push("");
    lines.push("## Typical writing quirks/errors");
    for (const err of profile.typicalErrors) {
      lines.push(
        `- ${err.pattern} (${err.frequency} occurrences). Examples: ${err.examples.map((e) => `"${e}"`).join(", ")}`
      );
    }
  }

  return lines.join("\n");
}

function pct(ratio: number): string {
  return `${Math.round(ratio * 100)}%`;
}
