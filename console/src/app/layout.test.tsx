import { screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as sdk from "@/api/sdk.gen";
import { mockApiResponse } from "@/test/mocks";
import { sampleTransactionList } from "@/test/fixtures";
import { testRender } from "@/test/render";

describe("AppLayout", () => {
  beforeEach(() => {
    vi.spyOn(sdk, "getHealth").mockResolvedValue(
      mockApiResponse({ status: "ok" }),
    );
    vi.spyOn(sdk, "getTransactions").mockResolvedValue(
      mockApiResponse(sampleTransactionList()),
    );
    vi.spyOn(sdk, "postChat").mockResolvedValue(
      mockApiResponse({ answer: "Hello", tool_calls: [] }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders sidebar and chat page at /chat", async () => {
    testRender({ route: "/chat" });

    expect(
      await screen.findByRole("heading", { name: "Chat" }),
    ).toBeInTheDocument();
    expect(screen.getByText("assetagent")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText("Ask about your finances…"),
    ).toBeInTheDocument();
  });

  it("renders transactions page at /transactions", async () => {
    testRender({ route: "/transactions" });

    expect(
      await screen.findByRole("heading", { name: "Transactions" }),
    ).toBeInTheDocument();
    expect(
      await screen.findByText("REWE Dortmund"),
    ).toBeInTheDocument();
  });

  it("renders import page at /imports", async () => {
    testRender({ route: "/imports" });

    expect(
      await screen.findByRole("heading", { name: "Import" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Drop a Sparkasse CSV here/i),
    ).toBeInTheDocument();
  });

  it("redirects / to /chat", async () => {
    testRender({ route: "/" });

    expect(
      await screen.findByRole("heading", { name: "Chat" }),
    ).toBeInTheDocument();
  });
});
