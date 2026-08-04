import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

const incomeSeriesId = "11111111-1111-1111-1111-111111111111";

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
      key: "regular_monthly_income" as const,
      value: "3500.00",
      calculation: "Sum of monthly-equivalent income series",
      confidence: "high" as const,
      evidence_ids: [incomeSeriesId],
    },
  ],
  confirmed_at: "2026-04-01T00:00:00Z",
  created_at: "2026-04-01T00:00:00Z",
};

describe("Baseline income page", () => {
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
            id: incomeSeriesId,
            display_name: "Employer GmbH",
            interval: "monthly",
            kind: "income",
            status: "active",
            amount_typical: "3500.00",
            amount_last: "3500.00",
            amount_changed: false,
            uncertainty: "low",
            member_count: 12,
            created_at: "2025-01-01T00:00:00Z",
          },
        ],
      }),
    );
    vi.spyOn(sdk, "getCurrentBaseline").mockResolvedValue(
      mockApiResponse(sampleBaseline),
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
            income: "5000.00",
            expenses: "2200.00",
            net: "2800.00",
          },
        ],
      }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows income norm, sources, and development chart", async () => {
    testRender({ route: "/baseline/income" });

    expect(
      await screen.findByText(/Regular monthly income/i),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Income", exact: true }),
    ).toBeInTheDocument();
    expect(screen.getAllByText(/3\.500,00/).length).toBeGreaterThanOrEqual(1);
    expect(
      screen.getByRole("heading", { name: /^Sources$/i }),
    ).toBeInTheDocument();
    expect(screen.getByText("Employer GmbH")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: /^Development$/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("img", {
        name: /Monthly income versus Cashflow income norm/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: /^Booked$/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: /^vs norm$/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: /^vs prior$/i }),
    ).toBeInTheDocument();
    expect(
      screen.getAllByRole("link", { name: /March 2026/i })[0],
    ).toHaveAttribute("href", "/insights/months/2026-03");
    expect(screen.getByText(/\+1\.500,00/)).toBeInTheDocument();
    expect(screen.getAllByText(/\+43 %/).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/booked well above the norm/i)).toBeInTheDocument();
  });

  it("expands a source to show sample payments", async () => {
    vi.spyOn(sdk, "getRecurringMembers").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            transaction_id: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
            booking_date: "2026-03-28",
            amount: "3500.00",
            counterparty: "Employer GmbH",
            purpose: "Gehalt März",
          },
        ],
      }),
    );

    testRender({ route: "/baseline/income" });

    expect(await screen.findByText("Employer GmbH")).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: /Employer GmbH/i }),
    );
    expect(await screen.findByText(/Gehalt März/i)).toBeInTheDocument();
    expect(screen.getByText(/28\.03\.2026/)).toBeInTheDocument();
  });
});
