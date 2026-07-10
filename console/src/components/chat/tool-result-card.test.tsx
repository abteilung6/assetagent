import { screen, waitFor } from "@testing-library/react";
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router";
import { describe, expect, it } from "vitest";

import type { ChatToolCall } from "@/api/types.gen";
import { testRender } from "@/test/render";

import { ToolResultCard } from "./tool-result-card";
import { TOOL_NAMES } from "./tool-sources";

async function renderToolCard(toolCall: ChatToolCall) {
  const rootRoute = createRootRoute({
    component: () => <ToolResultCard toolCall={toolCall} />,
  });
  const router = createRouter({
    routeTree: rootRoute,
    history: createMemoryHistory({ initialEntries: ["/"] }),
  });

  await router.load();
  testRender(<RouterProvider router={router} />);
  await waitFor(() => {
    expect(screen.getByText(toolDisplayNameFor(toolCall.name))).toBeInTheDocument();
  });
}

function toolDisplayNameFor(name: string): string {
  switch (name) {
    case TOOL_NAMES.cashflow:
      return "Spending summary";
    case TOOL_NAMES.search:
      return "Transaction search";
    default:
      return name;
  }
}

describe("ToolResultCard", () => {
  it("renders cashflow metrics and a transactions link", async () => {
    const toolCall: ChatToolCall = {
      name: TOOL_NAMES.cashflow,
      input: { from: "2026-06-01", to: "2026-06-30" },
      result: {
        income: "5200.00",
        expenses: "2143.22",
        net: "3056.78",
        currency: "EUR",
      },
    };

    await renderToolCard(toolCall);

    expect(screen.getByText("Spending summary")).toBeInTheDocument();
    expect(screen.getByText("Income")).toBeInTheDocument();
    expect(screen.getByText("2143.22 EUR")).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /view transactions/i }),
    ).toHaveAttribute("href", expect.stringContaining("/transactions"));
  });

  it("renders search preview and encodes query in the link", async () => {
    const toolCall: ChatToolCall = {
      name: TOOL_NAMES.search,
      input: { q: "REWE", from: "2026-06-01", to: "2026-06-30" },
      result: {
        total: 2,
        transactions: [
          {
            booking_date: "2026-06-12",
            counterparty: "REWE",
            amount: "-42.50",
            currency: "EUR",
          },
        ],
      },
    };

    await renderToolCard(toolCall);

    expect(screen.getByText(/2 matching transactions/i)).toBeInTheDocument();
    expect(screen.getByText("-42.50 EUR")).toBeInTheDocument();
    const link = screen.getByRole("link", { name: /view transactions/i });
    expect(link.getAttribute("href")).toContain("q=REWE");
  });

  it("shows a readable error instead of a source link", async () => {
    const toolCall: ChatToolCall = {
      name: TOOL_NAMES.cashflow,
      input: { from: "2026-06-30", to: "2026-06-01" },
      result: { error: "to must be on or after from" },
    };

    await renderToolCard(toolCall);

    expect(
      screen.getByText(
        "The date range was reversed. The end date must be on or after the start date.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: /view transactions/i }),
    ).not.toBeInTheDocument();
  });
});
