export type Platform =
  | "imessage"
  | "slack"
  | "email"
  | "github"
  | "twitter"
  | "discord"
  | "reddit"
  | "instagram"
  | "tiktok"
  | "teams"
  | "googledocs";

export interface RawMessage {
  id: string;
  text: string;
  platform: Platform;
  timestamp: Date;
  isFromUser: boolean;
  metadata?: Record<string, unknown>;
}

export interface StyleProfile {
  averageLength: number;
  capitalizesFirstWord: number; // 0-1 ratio
  usesContractions: number;
  usesPeriods: number;
  usesCommas: number;
  usesExclamation: number;
  usesQuestionMark: number;
  usesEmoji: number;
  commonPhrases: string[];
  typicalErrors: TypicalError[];
  sentenceFragmentRatio: number;
  lowercaseIRatio: number;
}

export interface TypicalError {
  pattern: string;
  frequency: number;
  examples: string[];
}

export interface CorpusEntry {
  id: string;
  text: string;
  platform: Platform;
  timestamp: string;
  embedding?: number[];
  conversationContext?: string;
}
