import path from "node:path";
import { fileURLToPath } from "node:url";

const consoleDir = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
);
const repoRoot = path.resolve(consoleDir, "..");

const consolePort = process.env.E2E_CONSOLE_PORT ?? "5174";

export const e2eEnv = {
  repoRoot,
  consoleDir,
  binaryPath: path.join(repoRoot, "bin/assetagent"),
  apiUrl: process.env.E2E_API_URL ?? "http://127.0.0.1:8080",
  consolePort,
  consoleUrl:
    process.env.E2E_CONSOLE_URL ?? `http://127.0.0.1:${consolePort}`,
  seedFile:
    process.env.E2E_SEED_FILE ??
    path.join(repoRoot, "testdata/sparkasse/sample.csv"),
  databaseUrl:
    process.env.DATABASE_URL ??
    "postgres://assetagent:assetagent@127.0.0.1:5432/assetagent?sslmode=disable",
};
