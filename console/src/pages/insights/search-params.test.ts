import { describe, expect, it } from "vitest";

import {
  parseInsightsMonthSearchParams,
  parseInsightsMonthTab,
} from "./search-params";

describe("parseInsightsMonthSearchParams", () => {
  it("parses overview and activity", () => {
    expect(parseInsightsMonthTab("overview")).toBe("overview");
    expect(parseInsightsMonthTab("activity")).toBe("activity");
    expect(parseInsightsMonthSearchParams({ tab: "activity" })).toEqual({
      tab: "activity",
    });
  });

  it("defaults to empty (Overview) for missing or unknown tab", () => {
    expect(parseInsightsMonthSearchParams({})).toEqual({});
    expect(parseInsightsMonthSearchParams({ tab: "counts" })).toEqual({});
  });
});
