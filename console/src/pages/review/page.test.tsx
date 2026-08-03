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

const sampleCategory = {
  id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  slug: "groceries",
  display_name: "Groceries",
  kind: "expense",
  is_system: true,
};

const sampleQueueItem = {
  transaction_id: "44444444-4444-4444-4444-444444444444",
  booking_date: "2026-03-12",
  amount: "-120.00",
  counterparty: "REWE MARKT",
  purpose: "Kartenzahlung",
  booking_text: "REWE",
  category_slug: "unresolved",
  category_name: "Unresolved",
  source: "heuristic",
  confidence: "low",
  merchant_id: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
  merchant_name: "REWE",
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
    vi.spyOn(sdk, "getClassificationQueue").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    vi.spyOn(sdk, "getCategories").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    vi.spyOn(sdk, "getUncertainRecurring").mockResolvedValue(
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
    vi.spyOn(sdk, "getClassificationQueue").mockResolvedValue(
      mockApiResponse({ data: [sampleQueueItem] }),
    );
    vi.spyOn(sdk, "getUncertainRecurring").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );

    testRender({ route: "/chat" });

    expect(
      await screen.findByRole("link", { name: /Needs review/i }),
    ).toBeInTheDocument();
    await waitFor(() => {
      expect(
        document.querySelector('[data-sidebar="menu-badge"]'),
      ).toHaveTextContent("2");
    });
  });

  it("keeps the nav item when the inbox is empty", async () => {
    vi.spyOn(sdk, "getTransferCandidates").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    vi.spyOn(sdk, "getClassificationQueue").mockResolvedValue(
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
    vi.spyOn(sdk, "getClassificationQueue").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    vi.spyOn(sdk, "getUncertainRecurring").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
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
    expect(screen.getByText(/500,00\s*€/)).toBeInTheDocument();

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

  it("corrects a classification and clears it from the queue", async () => {
    const user = userEvent.setup();
    vi.spyOn(sdk, "getTransferCandidates").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    const getSpy = vi
      .spyOn(sdk, "getClassificationQueue")
      .mockResolvedValueOnce(mockApiResponse({ data: [sampleQueueItem] }))
      .mockResolvedValue(mockApiResponse({ data: [] }));
    vi.spyOn(sdk, "getCategories").mockResolvedValue(
      mockApiResponse({ data: [sampleCategory] }),
    );
    vi.spyOn(sdk, "getUncertainRecurring").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    vi.spyOn(sdk, "postClassificationCorrect").mockResolvedValue(
      mockApiResponse({
        transaction_id: sampleQueueItem.transaction_id,
        category_slug: "groceries",
        rule_created: true,
        merchant_id: sampleQueueItem.merchant_id,
      }),
    );

    testRender({ route: "/review" });

    expect(await screen.findByText("Categories to check")).toBeInTheDocument();
    expect(screen.getAllByText("REWE").length).toBeGreaterThan(0);

    await user.selectOptions(
      screen.getByRole("combobox", { name: /Category/i }),
      "groceries",
    );
    expect(
      screen.getByText(/Remember this for/i),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(sdk.postClassificationCorrect).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(getSpy.mock.calls.length).toBeGreaterThanOrEqual(2);
    });
    expect(await screen.findByText(/Inbox clear/i)).toBeInTheDocument();
  });

  it("confirms a recurring series and clears it from the queue", async () => {
    const user = userEvent.setup();
    const sampleSeries = {
      id: "55555555-5555-5555-5555-555555555555",
      display_name: "Example Landlord",
      interval: "monthly" as const,
      kind: "fixed" as const,
      status: "uncertain" as const,
      amount_typical: "1200.00",
      amount_last: "1200.00",
      amount_changed: false,
      next_expected: "2026-04-01",
      uncertainty: "low" as const,
      member_count: 3,
      created_at: "2026-03-01T00:00:00Z",
    };
    vi.spyOn(sdk, "getTransferCandidates").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    vi.spyOn(sdk, "getClassificationQueue").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    vi.spyOn(sdk, "getCategories").mockResolvedValue(
      mockApiResponse({ data: [] }),
    );
    const getSpy = vi
      .spyOn(sdk, "getUncertainRecurring")
      .mockResolvedValueOnce(mockApiResponse({ data: [sampleSeries] }))
      .mockResolvedValue(mockApiResponse({ data: [] }));
    vi.spyOn(sdk, "postRecurringConfirm").mockResolvedValue(
      mockApiResponse({
        ...sampleSeries,
        status: "active" as const,
      }),
    );

    testRender({ route: "/review" });

    expect(await screen.findByText("Recurring payments")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /Confirm all \(1\)/i }),
    ).toBeInTheDocument();
    expect(screen.getByText("Example Landlord")).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: "Confirm recurring" }),
    );

    await waitFor(() => {
      expect(sdk.postRecurringConfirm).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(getSpy.mock.calls.length).toBeGreaterThanOrEqual(2);
    });
    expect(await screen.findByText(/Inbox clear/i)).toBeInTheDocument();
  });
});
