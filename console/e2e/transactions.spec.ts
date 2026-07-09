import { expect, test } from "@playwright/test";

test("transactions page shows seeded rows", async ({ page }) => {
  await page.goto("/transactions");

  await expect(
    page.getByRole("heading", { name: "Transactions" }),
  ).toBeVisible();

  const rows = page.locator("tbody tr");
  await expect(rows.first()).toBeVisible();
  await expect(rows).not.toHaveCount(0);

  await expect(page.getByText(/PayPal Europe/i)).toBeVisible();
});
