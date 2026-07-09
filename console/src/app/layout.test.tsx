import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithRouter } from "@/test/render";

describe("AppLayout", () => {
  it("renders sidebar and transactions page at /transactions", async () => {
    renderWithRouter("/transactions");

    expect(
      await screen.findByRole("heading", { name: "Transactions" }),
    ).toBeInTheDocument();
    expect(screen.getByText("assetagent")).toBeInTheDocument();
    expect(
      screen.getByText("Transaction list will appear here."),
    ).toBeInTheDocument();
  });

  it("redirects / to /transactions", async () => {
    renderWithRouter("/");

    expect(
      await screen.findByRole("heading", { name: "Transactions" }),
    ).toBeInTheDocument();
  });
});
