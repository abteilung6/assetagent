import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

const sampleForecast = {
  id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  baseline_id: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  horizon_days: 90,
  starting_balance: "2000.00",
  assumptions: {
    disabled_series_ids: [] as string[],
    include_variable: true,
    include_uncertain: true,
  },
  points: [
    { date: "2026-04-01", balance: "2000.00" },
    { date: "2026-04-08", balance: "1800.00" },
    { date: "2026-06-30", balance: "1200.00" },
  ],
  min_balance: "1100.00",
  ending_balance: "1200.00",
  algorithm_version: "forecast.v1",
  series_options: [
    {
      id: "rent",
      display_name: "Rent",
      kind: "fixed",
      interval: "monthly",
      amount: "1200.00",
      enabled: true,
    },
  ],
  created_at: "2026-04-01T00:00:00Z",
};

function mockChrome() {
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
}

describe("Plan page", () => {
  beforeEach(() => {
    mockChrome();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("separates forecast and what-if into tabs", async () => {
    const latest = vi.spyOn(sdk, "getLatestForecast");
    latest.mockRejectedValueOnce({
      error: "not_found",
      message: "no forecast available",
    });
    latest.mockResolvedValue(mockApiResponse(sampleForecast));
    const create = vi
      .spyOn(sdk, "postForecasts")
      .mockResolvedValue(mockApiResponse(sampleForecast));
    vi.spyOn(sdk, "getForecastScenarios").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    vi.spyOn(sdk, "getActions").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );

    testRender({ route: "/plan" });

    expect(
      await screen.findByRole("heading", { name: "Plan", exact: true }),
    ).toBeInTheDocument();
    expect(await screen.findByText(/No forecast yet/i)).toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("button", { name: /Show 90-day forecast/i }),
    );

    await waitFor(() => {
      expect(create).toHaveBeenCalled();
    });

    expect(
      await screen.findByRole("tab", { name: "1. Forecast" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "2. What if" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "3. Actions" })).toBeInTheDocument();
    expect(await screen.findByText(/Cash over time/i)).toBeInTheDocument();

    await userEvent.click(screen.getByRole("tab", { name: "2. What if" }));

    expect(
      await screen.findByRole("button", { name: /Compare new monthly cost/i }),
    ).toBeInTheDocument();
  });
});
