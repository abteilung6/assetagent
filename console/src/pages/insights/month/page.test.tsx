import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { sampleTransaction } from "@/test/fixtures";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

describe("Baseline month page", () => {
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
    vi.spyOn(sdk, "getBaselineOneOffImpact").mockResolvedValue(
      mockApiResponse({ count: 0, expense_total: "0.00" }),
    );
    vi.spyOn(sdk, "getBaselineCategorySpend").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            category_slug: "housing",
            category_name: "Housing",
            total: "1200.00",
            transaction_count: 1,
          },
          {
            category_slug: "groceries",
            category_name: "Groceries",
            total: "85.00",
            transaction_count: 1,
          },
        ],
      }),
    );
    vi.spyOn(sdk, "getBaselineDailyExpensePace").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            date: "2026-03-01",
            expenses: "1200.00",
            transaction_count: 1,
          },
          {
            date: "2026-03-10",
            expenses: "85.00",
            transaction_count: 2,
          },
        ],
      }),
    );
    vi.spyOn(sdk, "getBaselineMonthlyCashflow").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            month_start: "2026-01-01",
            income: "3500.00",
            expenses: "2000.00",
            net: "1500.00",
          },
          {
            month_start: "2026-02-01",
            income: "3500.00",
            expenses: "2100.00",
            net: "1400.00",
          },
          {
            month_start: "2026-03-01",
            income: "3500.00",
            expenses: "8000.00",
            net: "-4500.00",
          },
        ],
      }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  function mockMonthTransactions() {
    const groceries = sampleTransaction({
      id: "11111111-1111-1111-1111-111111111111",
      counterparty: "REWE",
      amount: "-85.00",
      booking_date: "2026-03-10",
      recurring: false,
    });
    const rent = sampleTransaction({
      id: "22222222-2222-2222-2222-222222222222",
      counterparty: "Landlord GmbH",
      amount: "-1200.00",
      booking_date: "2026-03-01",
      recurring: true,
    });
    const salary = sampleTransaction({
      id: "33333333-3333-3333-3333-333333333333",
      counterparty: "Employer GmbH",
      amount: "3500.00",
      booking_date: "2026-03-28",
      recurring: true,
    });

    return vi.spyOn(sdk, "getTransactions").mockImplementation(async (options) => {
      const order = options?.query?.order;
      if (order === "desc") {
        return mockApiResponse({
          data: [salary, groceries, rent],
          pagination: { limit: 50, offset: 0, total: 3 },
        });
      }
      return mockApiResponse({
        data: [rent, groceries, salary],
        pagination: { limit: 50, offset: 0, total: 3 },
      });
    });
  }

  it("shows Overview chrome, pace, and categories without payment lists", async () => {
    const txs = mockMonthTransactions();
    testRender({ route: "/insights/months/2026-03" });

    expect(
      await screen.findByRole("heading", { name: "March 2026" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Overview" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Activity" })).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: /Spending pace/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("img", {
        name: /Cumulative expenses over the month/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Expenses above typical/i)).toBeInTheDocument();
    expect(screen.getByText(/Why this month/i)).toBeInTheDocument();
    expect(screen.getByText(/By category/i)).toBeInTheDocument();
    expect(screen.getByText("Housing")).toBeInTheDocument();
    expect(screen.getByText("Groceries")).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Open Needs review/i }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Landlord GmbH")).not.toBeInTheDocument();
    expect(screen.queryByText("REWE")).not.toBeInTheDocument();
    expect(txs).not.toHaveBeenCalled();
  });

  it("loads Activity payments via tab and deep link", async () => {
    mockMonthTransactions();
    testRender({ route: "/insights/months/2026-03" });

    expect(
      await screen.findByRole("heading", { name: "March 2026" }),
    ).toBeInTheDocument();

    await userEvent.click(screen.getByRole("tab", { name: "Activity" }));

    expect(
      await screen.findByRole("heading", { name: /^Cost drivers$/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /^Recurring$/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /^One-time$/i })).toBeInTheDocument();
    expect(screen.getByText("Landlord GmbH")).toBeInTheDocument();
    expect(screen.getByText("REWE")).toBeInTheDocument();
    expect(screen.getByText(/Income sources/i)).toBeInTheDocument();
    expect(screen.getByText("Employer GmbH")).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /See all 3 transactions/i }),
    ).toHaveAttribute("href", expect.stringContaining("/transactions"));

    await userEvent.click(screen.getByRole("button", { name: /Landlord GmbH/i }));
    await waitFor(() => {
      expect(screen.getByText(/Treat as one-off/i)).toBeInTheDocument();
    });
  });

  it("opens Activity from ?tab=activity", async () => {
    mockMonthTransactions();
    testRender({ route: "/insights/months/2026-03?tab=activity" });

    expect(
      await screen.findByText("Landlord GmbH"),
    ).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Activity" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });
});
