import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { mockApiResponse } from "@/test/mocks";
import { testRender } from "@/test/render";

const sampleCandidate = {
  id: "11111111-1111-1111-1111-111111111111",
  status: "suggested" as const,
  confidence: "exact" as const,
  amount: "500.00",
  created_at: "2026-03-10T00:00:00Z",
  out: {
    transaction_id: "22222222-2222-2222-2222-222222222222",
    account_name: "Checking",
    booking_date: "2026-03-10",
    amount: "-500.00",
    booking_text: "UMBUCHUNG",
    purpose: "to savings",
    counterparty: "",
  },
  in: {
    transaction_id: "33333333-3333-3333-3333-333333333333",
    account_name: "Savings",
    booking_date: "2026-03-10",
    amount: "500.00",
    booking_text: "UMBUCHUNG",
    purpose: "from checking",
    counterparty: "",
  },
};

describe("Needs review inbox", () => {
  beforeEach(() => {
    vi.spyOn(sdk, "getHealth").mockResolvedValue(
      mockApiResponse({ status: "ok" }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows empty inbox", async () => {
    vi.spyOn(sdk, "getTransferCandidates").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );

    testRender({ route: "/review" });

    expect(
      await screen.findByRole("heading", { name: "Needs review", exact: true }),
    ).toBeInTheDocument();
    expect(await screen.findByText(/Inbox clear/i)).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Needs review/i }),
    ).toBeInTheDocument();
  });

  it("shows a badge in the sidebar when items are pending", async () => {
    vi.spyOn(sdk, "getTransferCandidates").mockResolvedValue(
      mockApiResponse({ data: [sampleCandidate] }),
    );

    testRender({ route: "/chat" });

    expect(
      await screen.findByRole("link", { name: /Needs review/i }),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(
        document.querySelector('[data-sidebar="menu-badge"]'),
      ).toHaveTextContent("1");
    });
  });

  it("keeps the nav item when the inbox is empty", async () => {
    vi.spyOn(sdk, "getTransferCandidates").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );

    testRender({ route: "/chat" });

    expect(
      await screen.findByRole("link", { name: /Needs review/i }),
    ).toBeInTheDocument();
  });

  it("confirms a transfer and clears it from the queue", async () => {
    const user = userEvent.setup();
    const getSpy = vi
      .spyOn(sdk, "getTransferCandidates")
      .mockResolvedValueOnce(mockApiResponse({ data: [sampleCandidate] }))
      .mockResolvedValue(mockApiResponse({ data: [] }));
    vi.spyOn(sdk, "postTransferConfirm").mockResolvedValue(
      mockApiResponse({
        id: sampleCandidate.id,
        tx_out_id: sampleCandidate.out.transaction_id,
        tx_in_id: sampleCandidate.in.transaction_id,
        status: "confirmed",
        confidence: "exact",
        created_at: sampleCandidate.created_at,
      }),
    );

    testRender({ route: "/review" });

    expect(
      await screen.findByText("Possible internal transfers"),
    ).toBeInTheDocument();
    expect(screen.getByText("500.00 EUR")).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: "Confirm transfer" }),
    );

    await waitFor(() => {
      expect(sdk.postTransferConfirm).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(getSpy.mock.calls.length).toBeGreaterThanOrEqual(2);
    });
    expect(await screen.findByText(/Inbox clear/i)).toBeInTheDocument();
  });
});
