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
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });
  it("renders sidebar and transactions page at /transactions", async () => {
    testRender({ route: "/transactions" });

    expect(
      await screen.findByRole("heading", { name: "Transactions" }),
    ).toBeInTheDocument();
    expect(screen.getByText("assetagent")).toBeInTheDocument();
    expect(
      await screen.findByText("REWE Dortmund"),
    ).toBeInTheDocument();
  });

  it("redirects / to /transactions", async () => {
    testRender({ route: "/" });

    expect(
      await screen.findByRole("heading", { name: "Transactions" }),
    ).toBeInTheDocument();
  });
});
