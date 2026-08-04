import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

const sampleBaseline = {
  id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  period_from: "2026-03-01",
  period_to: "2026-03-31",
  algorithm_version: "baseline.v1",
  status: "draft" as const,
  regular_monthly_income: "3500.00",
  monthly_fixed_costs: "1200.00",
  monthly_irregular_costs: "50.00",
  avg_variable_spend: "450.00",
  sustainable_free_cashflow: "1800.00",
  confidence: "high" as const,
  assumptions: ["period=last_complete_calendar_month"],
  metrics: [
    {
      key: "regular_monthly_income" as const,
      value: "3500.00",
      calculation: "Sum of monthly-equivalent income series",
      confidence: "high" as const,
      evidence_ids: ["11111111-1111-1111-1111-111111111111"],
    },
    {
      key: "monthly_fixed_costs" as const,
      value: "1200.00",
      calculation: "Sum of monthly recurring expenses",
      confidence: "high" as const,
      evidence_ids: ["22222222-2222-2222-2222-222222222222"],
    },
    {
      key: "monthly_irregular_costs" as const,
      value: "50.00",
      calculation: "Yearly costs spread monthly",
      confidence: "high" as const,
      evidence_ids: ["33333333-3333-3333-3333-333333333333"],
    },
    {
      key: "avg_variable_spend" as const,
      value: "450.00",
      calculation: "Residual variable spend",
      confidence: "high" as const,
      evidence_ids: [
        "22222222-2222-2222-2222-222222222222",
        "33333333-3333-3333-3333-333333333333",
      ],
    },
    {
      key: "sustainable_free_cashflow" as const,
      value: "1800.00",
      calculation:
        "regular_monthly_income − monthly_fixed_costs − monthly_irregular_costs − avg_variable_spend",
      confidence: "high" as const,
      evidence_ids: [],
    },
  ],
  created_at: "2026-04-01T00:00:00Z",
};

describe("Baseline page", () => {
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
            id: "22222222-2222-2222-2222-222222222222",
            display_name: "Example Landlord",
            interval: "monthly" as const,
            kind: "fixed" as const,
            status: "active" as const,
            amount_typical: "1200.00",
            amount_last: "1200.00",
            amount_changed: false,
            uncertainty: "low" as const,
            member_count: 6,
            created_at: "2026-01-01T00:00:00Z",
          },
          {
            id: "33333333-3333-3333-3333-333333333333",
            display_name: "Insurance",
            interval: "yearly" as const,
            kind: "fixed" as const,
            status: "active" as const,
            amount_typical: "600.00",
            amount_last: "600.00",
            amount_changed: false,
            uncertainty: "low" as const,
            member_count: 2,
            created_at: "2026-01-01T00:00:00Z",
          },
          {
            id: "11111111-1111-1111-1111-111111111111",
            display_name: "Salary",
            interval: "monthly" as const,
            kind: "income" as const,
            status: "active" as const,
            amount_typical: "3500.00",
            amount_last: "3500.00",
            amount_changed: false,
            uncertainty: "low" as const,
            member_count: 6,
            created_at: "2026-01-01T00:00:00Z",
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
            expenses: "2200.00",
            net: "1300.00",
          },
        ],
      }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows empty state and computes a baseline", async () => {
    const current = vi.spyOn(sdk, "getCurrentBaseline");
    current.mockRejectedValueOnce({
      error: "not_found",
      message: "no baseline available",
    });
    current.mockResolvedValue(mockApiResponse(sampleBaseline));
    const recompute = vi
      .spyOn(sdk, "postBaselinesRecompute")
      .mockResolvedValue(mockApiResponse(sampleBaseline));

    testRender({ route: "/baseline" });

    expect(
      await screen.findByRole("heading", { name: "Baseline", exact: true }),
    ).toBeInTheDocument();
    expect(await screen.findByText(/No baseline yet/i)).toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("button", { name: /Calculate baseline/i }),
    );

    await waitFor(() => {
      expect(recompute).toHaveBeenCalled();
    });
    expect(
      await screen.findByText(/Sustainable free cashflow/i),
    ).toBeInTheDocument();
  });

  it("confirms a draft baseline", async () => {
    vi.spyOn(sdk, "getCurrentBaseline").mockResolvedValue(
      mockApiResponse(sampleBaseline),
    );
    const confirm = vi.spyOn(sdk, "postBaselineConfirm").mockResolvedValue(
      mockApiResponse({ ...sampleBaseline, status: "confirmed" }),
    );

    testRender({ route: "/baseline" });

    expect(
      await screen.findByText(/Sustainable free cashflow/i),
    ).toBeInTheDocument();
    expect(await screen.findByText(/Typical month/i)).toBeInTheDocument();
    expect(screen.getByText(/Recent months/i)).toBeInTheDocument();
    expect(screen.getByText(/Click a month for detail/i)).toBeInTheDocument();
    expect(screen.getByText(/1\.800,00/)).toBeInTheDocument();

    const marchLink = screen.getByRole("link", {
      name: /2\.2k €/i,
    });
    expect(marchLink).toHaveAttribute(
      "href",
      "/baseline/months/2026-03?tab=composition",
    );

    await userEvent.click(
      screen.getByRole("tab", { name: /Income & expenses/i }),
    );
    expect(
      await screen.findByText(/Income & expenses over time/i),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("img", {
        name: /Monthly income and expenses over time with typical month guides/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Income · typical/i)).toBeInTheDocument();
    expect(screen.getByText(/Expenses · typical/i)).toBeInTheDocument();
    expect(
      screen.getByRole("tab", { name: /Income & expenses/i }),
    ).toHaveAttribute("data-active", "");
    await userEvent.click(screen.getByRole("button", { name: /12 mo/i }));
    await waitFor(() => {
      expect(sdk.getBaselineMonthlyCashflow).toHaveBeenCalledWith(
        expect.objectContaining({ query: { months: 12 } }),
      );
    });

    await userEvent.click(
      screen.getByRole("button", { name: /Confirm baseline/i }),
    );

    await waitFor(() => {
      expect(confirm).toHaveBeenCalledWith(
        expect.objectContaining({
          path: { id: sampleBaseline.id },
        }),
      );
    });
  });

  it("corrects a supporting metric with a reason", async () => {
    vi.spyOn(sdk, "getCurrentBaseline").mockResolvedValue(
      mockApiResponse(sampleBaseline),
    );
    const adjust = vi.spyOn(sdk, "postBaselineAdjust").mockResolvedValue(
      mockApiResponse({
        ...sampleBaseline,
        monthly_fixed_costs: "1100.00",
        sustainable_free_cashflow: "1900.00",
      }),
    );

    testRender({ route: "/baseline" });

    expect(await screen.findByText(/Monthly fixed costs/i)).toBeInTheDocument();

    const correctButtons = await screen.findAllByRole("button", {
      name: /^Correct$/i,
    });
    await userEvent.click(correctButtons[1]!);

    const amountInput = screen.getByLabelText(/Monthly amount/i);
    await userEvent.clear(amountInput);
    await userEvent.type(amountInput, "1100");

    await userEvent.type(
      screen.getByLabelText(/Why are you correcting/i),
      "Rent reduced",
    );

    await userEvent.click(
      screen.getByRole("button", { name: /Save correction/i }),
    );

    await waitFor(() => {
      expect(adjust).toHaveBeenCalledWith(
        expect.objectContaining({
          path: { id: sampleBaseline.id },
          body: {
            metric_key: "monthly_fixed_costs",
            new_value: "1100.00",
            reason: "Rent reduced",
          },
        }),
      );
    });
  });

  it("restores the over-time tab from the URL", async () => {
    vi.spyOn(sdk, "getCurrentBaseline").mockResolvedValue(
      mockApiResponse(sampleBaseline),
    );

    testRender({ route: "/baseline?tab=over-time" });

    expect(
      await screen.findByText(/Income & expenses over time/i),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("tab", { name: /Income & expenses/i }),
    ).toHaveAttribute("data-active", "");
  });

  it("opens fixed composition evidence with recurring series", async () => {
    vi.spyOn(sdk, "getCurrentBaseline").mockResolvedValue(
      mockApiResponse(sampleBaseline),
    );

    testRender({ route: "/baseline" });

    expect(await screen.findByText(/Typical month/i)).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: /Fixed\s+1\.2k/i }),
    );

    expect(
      await screen.findByRole("heading", { name: /Monthly fixed costs/i }),
    ).toBeInTheDocument();
    expect(await screen.findByText("Example Landlord")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Show sample payments/i }),
    ).toBeInTheDocument();
  });

  it("lists readiness checklist items before confirm", async () => {
    vi.spyOn(sdk, "getCurrentBaseline").mockResolvedValue(
      mockApiResponse(sampleBaseline),
    );
    vi.spyOn(sdk, "getTransferCandidates").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
            status: "suggested" as const,
            confidence: "high" as const,
            amount: "100.00",
            tx_out: {
              id: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
              booking_date: "2026-03-01",
              amount: "-100.00",
              counterparty: "A",
              purpose: "",
            },
            tx_in: {
              id: "cccccccc-cccc-cccc-cccc-cccccccccccc",
              booking_date: "2026-03-01",
              amount: "100.00",
              counterparty: "B",
              purpose: "",
            },
          },
        ],
      }),
    );
    vi.spyOn(sdk, "getClassificationQueue").mockResolvedValue(
      mockApiResponse({
        data: [
          {
            transaction_id: "dddddddd-dddd-dddd-dddd-dddddddddddd",
            booking_date: "2026-03-02",
            amount: "-20.00",
            counterparty: "Shop",
            purpose: "Stuff",
            booking_text: "",
            category_slug: "unresolved",
            category_name: "Unresolved",
            source: "unresolved",
            confidence: "low",
            merchant_id: null,
            merchant_name: "",
          },
        ],
      }),
    );

    testRender({ route: "/baseline" });

    expect(
      await screen.findByRole("heading", { name: /Before you confirm/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /1 transfer to review/i }),
    ).toHaveAttribute("href", "/review?tab=transfers");
    expect(
      screen.getByRole("link", { name: /1 category to check/i }),
    ).toHaveAttribute("href", "/review?tab=categories");
  });

  it("opens variable evidence with residual copy and correct CTA", async () => {
    vi.spyOn(sdk, "getCurrentBaseline").mockResolvedValue(
      mockApiResponse(sampleBaseline),
    );

    testRender({ route: "/baseline" });

    expect(await screen.findByText(/Typical month/i)).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: /Variable\s+450/i }),
    );

    expect(
      await screen.findByRole("heading", { name: /Average variable spend/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(/How this is built/i)).toBeInTheDocument();
    expect(
      screen.getByText(/Already counted as Fixed \/ Irregular/i),
    ).toBeInTheDocument();
    expect(screen.getByText("Example Landlord")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Correct variable spend/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Period transactions/i }),
    ).toBeInTheDocument();
  });
});
