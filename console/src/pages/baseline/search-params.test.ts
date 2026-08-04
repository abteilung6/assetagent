import { describe, expect, it } from "vitest";

import {
  parseBaselineSearchParams,
  parseBaselineTab,
} from "@/pages/baseline/search-params";

describe("parseBaselineTab", () => {
  it("accepts composition and over-time", () => {
    expect(parseBaselineTab("composition")).toBe("composition");
    expect(parseBaselineTab("over-time")).toBe("over-time");
  });

  it("defaults unknown values to composition", () => {
    expect(parseBaselineTab(undefined)).toBe("composition");
    expect(parseBaselineTab("nope")).toBe("composition");
  });
});

describe("parseBaselineSearchParams", () => {
  it("parses tab from search", () => {
    expect(parseBaselineSearchParams({ tab: "over-time" })).toEqual({
      tab: "over-time",
    });
  });
});
