import { describe, expect, it } from "vitest";

import {
  buildChatHandoff,
  defaultStarters,
  followUpsForTool,
  lastCompleteCalendarMonth,
  promoteStarters,
} from "./starters";

describe("lastCompleteCalendarMonth", () => {
  it("returns prior calendar month", () => {
    const result = lastCompleteCalendarMonth(new Date(2026, 6, 19));
    expect(result.yyyyMm).toBe("2026-06");
    expect(result.from).toBe("2026-06-01");
    expect(result.to).toBe("2026-06-30");
  });
});

describe("defaultStarters", () => {
  it("returns four starters", () => {
    expect(defaultStarters()).toHaveLength(4);
  });
});

describe("promoteStarters", () => {
  it("promotes needs review when queue has items", () => {
    const starters = defaultStarters(new Date(2026, 6, 19));
    const ordered = promoteStarters(starters, {
      needsReviewTotal: 3,
      baselineStatus: "confirmed",
    });
    expect(ordered[0]?.id).toBe("needs-review");
  });

  it("promotes typical month when baseline is draft", () => {
    const starters = defaultStarters(new Date(2026, 6, 19));
    const ordered = promoteStarters(starters, {
      needsReviewTotal: 0,
      baselineStatus: "draft",
    });
    expect(ordered[0]?.id).toBe("typical-month");
  });
});

describe("followUpsForTool", () => {
  it("maps cashflow tools to spend follow-ups", () => {
    const chips = followUpsForTool("get_cashflow_v2");
    expect(chips).toHaveLength(2);
    expect(chips[0]?.id).toBe("top-paid");
  });
});

describe("buildChatHandoff", () => {
  it("serializes prompt and context", () => {
    const handoff = buildChatHandoff({
      prompt: "Explain my free cashflow",
      context: { route: "/baseline", baseline_id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" },
    });
    expect(handoff.to).toBe("/chat");
    expect(handoff.search.prompt).toBe("Explain my free cashflow");
    expect(JSON.parse(handoff.search.context!)).toEqual({
      route: "/baseline",
      baseline_id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
    });
  });
});
