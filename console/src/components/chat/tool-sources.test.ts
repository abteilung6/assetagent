import { describe, expect, it } from "vitest";

import type { ChatToolCall } from "@/api/types.gen";

import {
  buildTransactionSearchFromToolCall,
  formatDateRange,
  formatShortDate,
  humanizeToolError,
  toolDisplayName,
  TOOL_NAMES,
} from "./tool-sources";

describe("toolDisplayName", () => {
  it("maps known tools to friendly labels", () => {
    expect(toolDisplayName(TOOL_NAMES.cashflow)).toBe("Spending summary");
    expect(toolDisplayName(TOOL_NAMES.counterparties)).toBe("Top counterparties");
    expect(toolDisplayName(TOOL_NAMES.search)).toBe("Transaction search");
  });
});

describe("formatShortDate", () => {
  it("formats ISO dates for display", () => {
    expect(formatShortDate("2026-06-01")).toMatch(/Jun/);
    expect(formatShortDate("2026-06-01")).toMatch(/2026/);
  });
});

describe("formatDateRange", () => {
  it("formats a single day", () => {
    expect(formatDateRange("2026-06-01", "2026-06-01")).toMatch(/Jun/);
  });

  it("formats an inclusive range", () => {
    const range = formatDateRange("2026-06-01", "2026-06-30");
    expect(range).toContain("–");
  });
});

describe("humanizeToolError", () => {
  it("hides Go unmarshalling errors", () => {
    expect(
      humanizeToolError(
        `invalid arguments: json: cannot unmarshal string into Go struct field counterpartiesArgs.limit of type int`,
      ),
    ).toBe("The assistant used an invalid parameter format. Try asking again.");
  });
});

describe("buildTransactionSearchFromToolCall", () => {
  const cashflowCall: ChatToolCall = {
    name: TOOL_NAMES.cashflow,
    input: { from: "2026-06-01", to: "2026-06-30" },
    result: {
      income: "5200.00",
      expenses: "2143.22",
      net: "3056.78",
      currency: "EUR",
    },
  };

  it("maps cashflow tool input to transaction filters", () => {
    expect(buildTransactionSearchFromToolCall(cashflowCall)).toEqual(
      expect.objectContaining({
        from: "2026-06-01",
        to: "2026-06-30",
        offset: 0,
      }),
    );
  });

  it("maps search tool input including q", () => {
    const searchCall: ChatToolCall = {
      name: TOOL_NAMES.search,
      input: { q: "REWE", from: "2026-06-01", to: "2026-06-30" },
      result: { total: 14, transactions: [] },
    };

    expect(buildTransactionSearchFromToolCall(searchCall)).toEqual(
      expect.objectContaining({
        q: "REWE",
        from: "2026-06-01",
        to: "2026-06-30",
      }),
    );
  });

  it("adds counterparty override for merchant drill-down", () => {
    expect(
      buildTransactionSearchFromToolCall(cashflowCall, {
        counterparty: "REWE",
      }),
    ).toEqual(
      expect.objectContaining({
        counterparty: "REWE",
        from: "2026-06-01",
        to: "2026-06-30",
      }),
    );
  });
});
