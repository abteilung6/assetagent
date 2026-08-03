import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

const sampleReview = {
  id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  baseline_id: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  period_from: "2026-03-01",
  period_to: "2026-03-31",
  status: "needs_confirmation" as const,
  summary:
    "Money review for 2026-03-01 – 2026-03-31: 1 finding(s). Free cashflow -200.00 €/month.",
  findings: [
    {
      type: "free_cashflow_pressure" as const,
      title: "Sustainable free cashflow is negative (-200.00 €/month)",
      amount: "-200.00",
      period_from: "2026-03-01",
      period_to: "2026-03-31",
      confidence: "high" as const,
      evidence_ids: ["baseline_free_cashflow"],
    },
  ],
  data_freshness: "2026-03-31",
  created_at: "2026-04-01T00:00:00Z",
};

function mockSidebarQueues() {
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

describe("Money Reviews", () => {
  beforeEach(() => {
    mockSidebarQueues();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows empty history and generates a review", async () => {
    vi.spyOn(sdk, "getMoneyReviews").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    const create = vi
      .spyOn(sdk, "postMoneyReviews")
      .mockResolvedValue(mockApiResponse(sampleReview));
    vi.spyOn(sdk, "getMoneyReview").mockResolvedValue(
      mockApiResponse(sampleReview),
    );

    testRender({ route: "/reviews" });

    expect(
      await screen.findByRole("heading", {
        name: "Money Reviews",
        exact: true,
      }),
    ).toBeInTheDocument();
    expect(await screen.findByText(/No reviews yet/i)).toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("button", { name: /^Generate review$/i }),
    );

    await waitFor(() => {
      expect(create).toHaveBeenCalled();
    });
    expect(
      await screen.findByText(/Sustainable free cashflow is negative/i),
    ).toBeInTheDocument();
  });

  it("confirms an open review", async () => {
    vi.spyOn(sdk, "getMoneyReview").mockResolvedValue(
      mockApiResponse(sampleReview),
    );
    const confirm = vi.spyOn(sdk, "postMoneyReviewConfirm").mockResolvedValue(
      mockApiResponse({ ...sampleReview, status: "confirmed" }),
    );

    testRender({ route: `/reviews/${sampleReview.id}` });

    expect(await screen.findByText(/Findings/i)).toBeInTheDocument();
    await userEvent.click(
      screen.getByRole("button", { name: /Confirm review/i }),
    );

    await waitFor(() => {
      expect(confirm).toHaveBeenCalledWith(
        expect.objectContaining({
          path: { id: sampleReview.id },
        }),
      );
    });
  });
});
