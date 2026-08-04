import { describe, expect, it } from "vitest";

import {
  parseBaselineSearchParams,
  parseLegacyBaselineTab,
} from "./search-params";

describe("parseLegacyBaselineTab", () => {
  it("accepts legacy composition and over-time", () => {
    expect(parseLegacyBaselineTab("composition")).toBe("composition");
    expect(parseLegacyBaselineTab("over-time")).toBe("over-time");
  });

  it("ignores unknown values", () => {
    expect(parseLegacyBaselineTab(undefined)).toBeUndefined();
    expect(parseLegacyBaselineTab("nope")).toBeUndefined();
  });
});

describe("parseBaselineSearchParams", () => {
  it("parses legacy tab from search", () => {
    expect(parseBaselineSearchParams({ tab: "over-time" })).toEqual({
      tab: "over-time",
    });
  });

  it("returns empty object without tab", () => {
    expect(parseBaselineSearchParams({})).toEqual({});
  });
});
