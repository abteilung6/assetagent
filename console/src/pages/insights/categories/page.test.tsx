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
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows spend mix and expands merchants", async () => {
    testRender({ route: "/insights/categories" });

    expect(await screen.findByText("Housing")).toBeInTheDocument();
    expect(screen.getByText("Spend mix")).toBeInTheDocument();
    expect(screen.getByText("Groceries")).toBeInTheDocument();
    expect(screen.getByText(/Classified spend/i)).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: /Housing/i }));

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
