import { execSync, spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { setTimeout as sleep } from "node:timers/promises";
import pg from "pg";

import { e2eEnv } from "./env";

function hasDocker() {
  return spawnSync("docker", ["compose", "version"], { stdio: "ignore" })
    .status === 0;
}

function run(command: string) {
  execSync(command, {
    cwd: e2eEnv.repoRoot,
    stdio: "inherit",
    env: { ...process.env, DATABASE_URL: e2eEnv.databaseUrl },
  });
}

function ensureBinary() {
  if (existsSync(e2eEnv.binaryPath)) {
    return;
  }

  execSync("go build -o bin/assetagent ./cmd/assetagent", {
    cwd: e2eEnv.repoRoot,
    stdio: "inherit",
  });
}

async function waitForPostgres() {
  const client = new pg.Client({ connectionString: e2eEnv.databaseUrl });

  for (let attempt = 1; attempt <= 50; attempt++) {
    try {
      await client.connect();
      await client.end();
      return;
    } catch {
      if (attempt === 50) {
        throw new Error("postgres not ready");
      }
      await sleep(200);
    }
  }
}

async function resetImportData() {
  const client = new pg.Client({ connectionString: e2eEnv.databaseUrl });
  await client.connect();
  await client.query(
    "TRUNCATE transactions, import_runs, accounts RESTART IDENTITY CASCADE",
  );
  await client.end();
}

export default async function globalSetup() {
  if (process.env.E2E_SKIP_SETUP === "1") {
    return;
  }

  ensureBinary();

  if (hasDocker()) {
    run("docker compose up -d postgres");
  }

  await waitForPostgres();
  run(`"${e2eEnv.binaryPath}" migrate up`);
  await resetImportData();
  run(`"${e2eEnv.binaryPath}" import --account-name "E2E Seed" "${e2eEnv.seedFile}"`);
}
