import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { sampleTransaction } from "@/test/fixtures";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

const fixedSeriesId = "22222222-2222-2222-2222-222222222222";
const irregularSeriesId = "33333333-3333-3333-3333-333333333333";

const sampleBaseline = {
  id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  period_from: "2025-04-01",
  period_to: "2026-03-31",
  algorithm_version: "baseline.v1",
  status: "confirmed" as const,
  regular_monthly_income: "3500.00",
  monthly_fixed_costs: "1200.00",
  monthly_irregular_costs: "50.00",
  avg_variable_spend: "450.00",
  sustainable_free_cashflow: "1800.00",
  confidence: "high" as const,
  assumptions: [],
  metrics: [
    {
      key: "monthly_fixed_costs" as const,
      value: "1200.00",
      calculation: "Sum of monthly-equivalent fixed series",
      confidence: "high" as const,
      evidence_ids: [fixedSeriesId],
    },
    {
      key: "monthly_irregular_costs" as const,
      value: "50.00",
      calculation: "Sum of monthly-equivalent irregular series",
      confidence: "medium" as const,
      evidence_ids: [irregularSeriesId],
    },
    {
      key: "avg_variable_spend" as const,
      value: "450.00",
      calculation: "Residual variable band",
      confidence: "medium" as const,
      evidence_ids: [],
    },
  ],
  confirmed_at: "2026-04-01T00:00:00Z",
  created_at: "2026-04-01T00:00:00Z",
};

describe("Baseline expenses page", () => {
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
    vi.spyOn(sdk, "getRecurring").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            id: fixedSeriesId,
            display_name: "Example Landlord",
            interval: "monthly",
            kind: "fixed",
            status: "active",
            amount_typical: "1200.00",
            amount_last: "1200.00",
            amount_changed: false,
            uncertainty: "low",
            member_count: 12,
            created_at: "2025-01-01T00:00:00Z",
          },
          {
            id: irregularSeriesId,
            display_name: "Insurance",
            interval: "yearly",
            kind: "variable_regular",
            status: "active",
            amount_typical: "600.00",
            amount_last: "600.00",
            amount_changed: false,
            uncertainty: "low",
            member_count: 2,
            created_at: "2025-01-01T00:00:00Z",
          },
        ],
      }),
    );
    vi.spyOn(sdk, "getCurrentBaseline").mockResolvedValue(
      mockApiResponse(sampleBaseline),
    );
    vi.spyOn(sdk, "getBaselineOneOffImpact").mockResolvedValue(
      mockApiResponse({ count: 1, expense_total: "800.00" }),
    );
    vi.spyOn(sdk, "getBaselineMonthlyCashflow").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            month_start: "2026-01-01",
            income: "3500.00",
            expenses: "1700.00",
            net: "1800.00",
          },
          {
            month_start: "2026-02-01",
            income: "3500.00",
            expenses: "1700.00",
            net: "1800.00",
          },
          {
            month_start: "2026-03-01",
            income: "3500.00",
            expenses: "2500.00",
            net: "1000.00",
          },
        ],
      }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows typical expenses, structure, drivers, and development", async () => {
    testRender({ route: "/baseline/expenses" });

    expect(
      await screen.findByText(/Typical monthly expenses/i),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Expenses", exact: true }),
    ).toBeInTheDocument();
    // 1200 + 50 + 450
    expect(screen.getAllByText(/1\.700,00/).length).toBeGreaterThanOrEqual(1);
    expect(await screen.findByText(/1 one-off/i)).toBeInTheDocument();

    expect(
      screen.getByRole("heading", { name: /^Structure$/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: /^Drivers$/i }),
    ).toBeInTheDocument();
    expect(screen.getByText("Example Landlord")).toBeInTheDocument();
    expect(screen.getByText("Insurance")).toBeInTheDocument();
    expect(
      screen.getByText(/not a merchant list/i),
    ).toBeInTheDocument();

    expect(
      screen.getByRole("heading", { name: /^Development$/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("img", {
        name: /Monthly expenses versus Cashflow cost norm/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: /^vs norm$/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/booked well above the cost norm/i)).toBeInTheDocument();
  });

  it("expands a fixed driver to show sample payments", async () => {
    vi.spyOn(sdk, "getRecurringMembers").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            transaction_id: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
            booking_date: "2026-03-01",
            amount: "-1200.00",
            counterparty: "Example Landlord",
            purpose: "Miete März",
          },
        ],
      }),
    );

    testRender({ route: "/baseline/expenses" });

    expect(await screen.findByText("Example Landlord")).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: /Example Landlord/i }),
    );
    expect(await screen.findByText(/Miete März/i)).toBeInTheDocument();
    expect(screen.getByText(/01\.03\.2026/)).toBeInTheDocument();
  });

  it("does not list monthly variable_regular series under Irregular when the bucket is empty", async () => {
    vi.spyOn(sdk, "getCurrentBaseline").mockResolvedValue(
      mockApiResponse({
        ...sampleBaseline,
        monthly_irregular_costs: "0.00",
        metrics: [
          {
            key: "monthly_fixed_costs" as const,
            value: "1200.00",
            calculation: "Sum of monthly-equivalent fixed series",
            confidence: "high" as const,
            evidence_ids: [fixedSeriesId],
          },
          {
            key: "monthly_irregular_costs" as const,
            value: "0.00",
            calculation: "Sum of monthly-equivalent irregular series",
            confidence: "medium" as const,
            evidence_ids: [],
          },
          {
            key: "avg_variable_spend" as const,
            value: "450.00",
            calculation: "Residual variable band",
            confidence: "medium" as const,
            evidence_ids: [],
          },
        ],
      }),
    );
    vi.spyOn(sdk, "getRecurring").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            id: fixedSeriesId,
            display_name: "Example Landlord",
            interval: "monthly",
            kind: "fixed",
            status: "active",
            amount_typical: "1200.00",
            amount_last: "1200.00",
            amount_changed: false,
            uncertainty: "low",
            member_count: 12,
            created_at: "2025-01-01T00:00:00Z",
          },
          {
            id: irregularSeriesId,
            display_name: "Streaming monthly",
            interval: "monthly",
            kind: "variable_regular",
            status: "active",
            amount_typical: "15.00",
            amount_last: "15.00",
            amount_changed: false,
            uncertainty: "low",
            member_count: 6,
            created_at: "2025-01-01T00:00:00Z",
          },
        ],
      }),
    );

    testRender({ route: "/baseline/expenses" });

    expect(
      await screen.findByText(/No quarterly or yearly recurring costs/i),
    ).toBeInTheDocument();
    expect(screen.queryByText("Streaming monthly")).not.toBeInTheDocument();
  });

  it("expands a development month to show top expenses", async () => {
    vi.spyOn(sdk, "getTransactions").mockResolvedValue(
      mockApiResponse({
        data: [
          sampleTransaction({
            id: "cccccccc-cccc-cccc-cccc-cccccccccccc",
            counterparty: "WEG Example",
            purpose: "Hausgeld",
            amount: "-900.00",
            booking_date: "2026-03-15",
            recurring: false,
          }),
          sampleTransaction({
            id: "dddddddd-dddd-dddd-dddd-dddddddddddd",
            counterparty: "ROLAND Rechtsschutz",
            purpose: "Versicherung",
            amount: "-350.00",
            booking_date: "2026-03-10",
            one_off: true,
            recurring: false,
          }),
        ],
        pagination: { limit: 5, offset: 0, total: 2 },
      }),
    );

    testRender({ route: "/baseline/expenses" });

    const monthButton = await screen.findByRole("button", {
      name: /March 2026/i,
    });
    await userEvent.click(monthButton);
    expect(await screen.findByText("WEG Example")).toBeInTheDocument();
    expect(screen.getByText("ROLAND Rechtsschutz")).toBeInTheDocument();
    expect(screen.getByText(/· one-off/i)).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Open March 2026 in Insights/i }),
    ).toHaveAttribute(
      "href",
      expect.stringContaining("/insights/months/2026-03"),
    );
  });
});
