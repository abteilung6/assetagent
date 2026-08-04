import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

describe("Insights categories page", () => {
  beforeEach(() => {
    vi.spyOn(sdk, "getHealth").mockResolvedValue(
      mockApiResponse({ status: "ok" }),
    );
    vi.spyOn(sdk, "getTransferCandidates").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    vi.spyOn(sdk, "getClassificationQueue").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    vi.spyOn(sdk, "getUncertainRecurring").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    vi.spyOn(sdk, "getBaselineCategorySpend").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            category_slug: "housing",
            category_name: "Housing",
            total: "2400.00",
            transaction_count: 2,
          },
          {
            category_slug: "groceries",
            category_name: "Groceries",
            total: "600.00",
            transaction_count: 12,
          },
          {
            category_slug: "travel",
            category_name: "Travel",
            total: "200.00",
            transaction_count: 1,
          },
        ],
      }),
    );
    vi.spyOn(sdk, "getBaselineCategoryMerchants").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            merchant: "Vermieter GmbH",
            total: "2200.00",
            transaction_count: 2,
          },
        ],
      }),
    );
    vi.spyOn(sdk, "getBaselineCategorySpendMonthly").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            month_start: "2026-01-01",
            category_slug: "housing",
            category_name: "Housing",
            total: "1200.00",
          },
          {
            month_start: "2026-02-01",
            category_slug: "housing",
            category_name: "Housing",
            total: "1200.00",
          },
          {
            month_start: "2026-01-01",
            category_slug: "groceries",
            category_name: "Groceries",
            total: "300.00",
          },
          {
            month_start: "2026-02-01",
            category_slug: "groceries",
            category_name: "Groceries",
            total: "300.00",
          },
        ],
      }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows spend mix and expands merchants", async () => {
    testRender({ route: "/insights/categories" });

    expect(await screen.findByText("Development")).toBeInTheDocument();
    expect(
      await screen.findByRole("img", { name: /Monthly spend by top categories/i }),
    ).toBeInTheDocument();
    expect(screen.getByText("Spend mix")).toBeInTheDocument();
    expect(screen.getAllByText("Housing").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Groceries").length).toBeGreaterThan(0);
    expect(screen.getByText(/Classified spend/i)).toBeInTheDocument();

    const housingButtons = await screen.findAllByRole("button", {
      name: /Housing/i,
    });
    await userEvent.click(housingButtons[0]!);

    expect(await screen.findByText("Vermieter GmbH")).toBeInTheDocument();
    expect(sdk.getBaselineCategoryMerchants).toHaveBeenCalled();
  });

  it("toggles the 6 mo window", async () => {
    testRender({ route: "/insights/categories" });

    expect(await screen.findByText("Spend mix")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "6 mo" }));
    expect(sdk.getBaselineCategorySpend).toHaveBeenCalled();
  });
});
