import path from "node:path";

import { expect, test } from "@playwright/test";

import { e2eEnv } from "./env";

const importFlowCsv = path.join(
  e2eEnv.repoRoot,
  "testdata/sparkasse/import_flow.csv",
);

test("import preview, commit, reimport, and undo", async ({ page }) => {
  await page.goto("/imports");

  await expect(
    page.getByRole("heading", { name: "Import", exact: true }),
  ).toBeVisible();
  await expect(page.getByText(/Drop a Sparkasse CSV here/i)).toBeVisible();

  await page.getByTestId("import-file-input").setInputFiles(importFlowCsv);

  await expect(page.getByText("Sample rows")).toBeVisible();
  await expect(page.getByText("Flow Cafe")).toBeVisible();
  await expect(page.getByLabel("Account name")).toBeVisible();

  await page.getByLabel("Account name").fill("E2E Flow Account");
  await page.getByRole("button", { name: "Confirm import" }).click();

  await expect(
    page.getByRole("heading", { name: "Import complete" }),
  ).toBeVisible();
  await expect(page.getByText("New transactions")).toBeVisible();
  await expect(page.getByText("Recent imports")).toBeVisible();
  await expect(page.getByText("import_flow.csv").first()).toBeVisible();

  await page.getByRole("button", { name: "Import another file" }).click();
  await expect(page.getByText(/Drop a Sparkasse CSV here/i)).toBeVisible();

  await page.getByTestId("import-file-input").setInputFiles(importFlowCsv);
  await expect(page.getByText("Sample rows")).toBeVisible();
  await page.getByLabel("Account name").fill("E2E Flow Account");
  await page.getByRole("button", { name: "Confirm import" }).click();

  await expect(
    page.getByRole("heading", { name: "Nothing new to import" }),
  ).toBeVisible();
  await expect(page.getByText(/already in your ledger/i)).toBeVisible();

  const originalRun = page.locator("li").filter({ hasText: "2 inserted" });
  await originalRun.getByRole("button", { name: "Undo" }).click();
  await expect(page.getByText(/Undo this import\?/i)).toBeVisible();
  await page.getByRole("button", { name: "Undo import" }).click();

  await expect(originalRun.getByText("Undone")).toBeVisible();
});
