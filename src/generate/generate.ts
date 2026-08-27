import Anthropic from "@anthropic-ai/sdk";
import OpenAI from "openai";
import type { RawMessage, StyleProfile } from "../corpus/types.js";
import { buildSystemPrompt, buildUserPrompt } from "./prompt.js";
import type { Config } from "../config.js";

export interface GenerateRequest {
  platform: string;
  context: string;
  instruction?: string;
  quirkLevel?: number;
  similarMessages: RawMessage[];
  styleProfile: StyleProfile;
  variants?: number;
  config: Config;
}

export async function generateMessage(
  req: GenerateRequest
): Promise<string[]> {
  const quirkLevel = req.quirkLevel ?? req.config.quirkLevel ?? 50;
  const variants = req.variants ?? 3;

  const systemPrompt = buildSystemPrompt({
    platform: req.platform,
    context: req.context,
    similarMessages: req.similarMessages,
    styleProfile: req.styleProfile,
    quirkLevel,
    instruction: req.instruction,
  });

  const userPrompt = buildUserPrompt({
    platform: req.platform,
    context: req.context,
    similarMessages: req.similarMessages,
    styleProfile: req.styleProfile,
    quirkLevel,
    instruction: req.instruction,
  });

  if (req.config.provider === "openai") {
    return generateOpenAI(systemPrompt, userPrompt, variants, req.config);
  }
  return generateAnthropic(systemPrompt, userPrompt, variants, req.config);
}

async function generateAnthropic(
  system: string,
  user: string,
  variants: number,
  config: Config
): Promise<string[]> {
  const client = new Anthropic();
  const model = config.model ?? "claude-sonnet-4-6-20250725";
  const results: string[] = [];

  for (let i = 0; i < variants; i++) {
    const response = await client.messages.create({
      model,
      max_tokens: 256,
      temperature: 0.8 + i * 0.05,
      system,
      messages: [{ role: "user", content: user }],
    });

    const text =
      response.content[0].type === "text" ? response.content[0].text : "";
    results.push(text.trim());
  }

  return results;
}

async function generateOpenAI(
  system: string,
  user: string,
  variants: number,
  config: Config
): Promise<string[]> {
  const client = new OpenAI();
  const model = config.model ?? "gpt-4o";
  const results: string[] = [];

  for (let i = 0; i < variants; i++) {
    const response = await client.chat.completions.create({
      model,
      max_tokens: 256,
      temperature: 0.8 + i * 0.05,
      messages: [
        { role: "system", content: system },
        { role: "user", content: user },
      ],
    });

    results.push(response.choices[0]?.message?.content?.trim() ?? "");
  }

  return results;
}
