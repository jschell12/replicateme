import { existsSync, mkdirSync, readFileSync, writeFileSync } from "fs";
import { homedir } from "os";
import { join } from "path";

const CONFIG_DIR = join(homedir(), ".replicateme");
const CONFIG_FILE = join(CONFIG_DIR, "config.json");
const PROFILE_FILE = join(CONFIG_DIR, "profile.json");

export interface Config {
  provider: "anthropic" | "openai";
  model?: string;
  quirkLevel: number;
  defaultPlatform: string;
}

const DEFAULTS: Config = {
  provider: "anthropic",
  quirkLevel: 50,
  defaultPlatform: "imessage",
};

export function ensureConfigDir(): void {
  if (!existsSync(CONFIG_DIR)) {
    mkdirSync(CONFIG_DIR, { recursive: true });
  }
}

export function loadConfig(): Config {
  ensureConfigDir();
  if (!existsSync(CONFIG_FILE)) {
    return { ...DEFAULTS };
  }
  const raw = readFileSync(CONFIG_FILE, "utf-8");
  return { ...DEFAULTS, ...JSON.parse(raw) };
}

export function saveConfig(config: Config): void {
  ensureConfigDir();
  writeFileSync(CONFIG_FILE, JSON.stringify(config, null, 2) + "\n");
}

export function loadProfile(): Record<string, unknown> | null {
  if (!existsSync(PROFILE_FILE)) return null;
  return JSON.parse(readFileSync(PROFILE_FILE, "utf-8"));
}

export function saveProfile(profile: Record<string, unknown>): void {
  ensureConfigDir();
  writeFileSync(PROFILE_FILE, JSON.stringify(profile, null, 2) + "\n");
}

export function getConfigDir(): string {
  return CONFIG_DIR;
}
