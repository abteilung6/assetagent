import { describe, expect, it } from "vitest";

import {
  buildBaselineComposition,
  buildBaselineReadinessItems,
  buildMonthStory,
  buildTypicalMonthLevels,
  detectUnusualMonth,
  formatCompactMoney,
  formatMonthHeadline,
  formatMonthLabel,
  monthlyEquivalentAmount,
  partitionMonthSpend,
  resolveEvidenceSeries,
} from "@/lib/baseline-charts";

describe("buildBaselineComposition", () => {
  it("stacks costs and free cashflow against income", () => {
    const got = buildBaselineComposition({
      income: 3500,
      fixed: 1200,
      irregular: 50,
      variable: 450,
      freeCashflow: 1800,
    });
    expect(got.overspent).toBe(false);
    expect(got.segments.map((s) => s.key)).toEqual([
      "fixed",
      "irregular",
      "variable",
      "free",
    ]);
    const shares = got.segments.reduce((sum, s) => sum + s.share, 0);
    expect(shares).toBeCloseTo(1, 5);
  });

  it("shows a shortfall segment when free cashflow is negative", () => {
    const got = buildBaselineComposition({
      income: 2000,
      fixed: 1500,
      irregular: 0,
      variable: 800,
      freeCashflow: -300,
    });
    expect(got.overspent).toBe(true);
    expect(got.segments.some((s) => s.key === "deficit")).toBe(true);
    expect(got.segments.some((s) => s.key === "free")).toBe(false);
  });
});

describe("detectUnusualMonth", () => {
  it("returns not unusual with sparse history", () => {
    expect(
      detectUnusualMonth([{ monthStart: "2026-01-01", income: 1, expenses: 1, net: 0 }]),
    ).toMatchObject({ unusual: false });
  });

  it("flags a month at least 2× the median expenses", () => {
    const months = [
      { monthStart: "2025-10-01", income: 3000, expenses: 2000, net: 1000 },
      { monthStart: "2025-11-01", income: 3000, expenses: 2100, net: 900 },
      { monthStart: "2025-12-01", income: 3000, expenses: 1900, net: 1100 },
      { monthStart: "2026-01-01", income: 3000, expenses: 8000, net: -5000 },
    ];
    const got = detectUnusualMonth(months, "2026-01-01");
    expect(got.unusual).toBe(true);
    expect(got.monthStart).toBe("2026-01-01");
    expect(got.ratio).toBeGreaterThanOrEqual(2);
    expect(got.message).toMatch(/baseline month/i);
  });
});

describe("buildTypicalMonthLevels", () => {
  it("sums fixed irregular and variable into typical expenses", () => {
    expect(
      buildTypicalMonthLevels({
        income: 3500,
        fixed: 1200,
        irregular: 50,
        variable: 450,
      }),
    ).toEqual({ income: 3500, expenses: 1700 });
  });
});

describe("buildBaselineReadinessItems", () => {
  it("returns review and unusual-month items with labels", () => {
    const items = buildBaselineReadinessItems({
      transferCount: 2,
      categoryCount: 1,
      recurringCount: 0,
      unusualMonthStart: "2026-03-01",
    });
    expect(items.map((i) => i.id)).toEqual([
      "transfers",
      "categories",
      "unusual_month",
    ]);
    expect(items[0]?.label).toMatch(/2 transfers/i);
    expect(items[1]?.href).toEqual({ kind: "review", tab: "categories" });
    expect(items[2]?.href).toEqual({ kind: "month", yyyyMm: "2026-03" });
  });

  it("returns empty when nothing is open", () => {
    expect(
      buildBaselineReadinessItems({
        transferCount: 0,
        categoryCount: 0,
        recurringCount: 0,
        unusualMonthStart: null,
      }),
    ).toEqual([]);
  });
});

describe("partitionMonthSpend", () => {
  it("splits recurring members from one-time expenses", () => {
    const rent = { id: "1", recurring: true };
    const shop = { id: "2", recurring: false };
    const netflix = { id: "3", recurring: true };
    expect(partitionMonthSpend([rent, shop, netflix])).toEqual({
      recurring: [rent, netflix],
      oneTime: [shop],
    });
  });

  it("respects per-group limits", () => {
    const txs = Array.from({ length: 5 }, (_, i) => ({
      id: String(i),
      recurring: i % 2 === 0,
    }));
    const result = partitionMonthSpend(txs, 1);
    expect(result.recurring).toHaveLength(1);
    expect(result.oneTime).toHaveLength(1);
  });
});

describe("formatMonthLabel", () => {
  it("formats YYYY-MM as MM.YYYY", () => {
    expect(formatMonthLabel("2026-03-01")).toBe("03.2026");
  });
});

describe("monthlyEquivalentAmount", () => {
  it("spreads quarterly and yearly amounts", () => {
    expect(monthlyEquivalentAmount(300, "quarterly")).toBe(100);
    expect(monthlyEquivalentAmount(1200, "yearly")).toBe(100);
    expect(monthlyEquivalentAmount(50, "monthly")).toBe(50);
  });
});

describe("resolveEvidenceSeries", () => {
  it("preserves evidence_ids order and skips missing ids", () => {
    const a = { id: "a", name: "A" };
    const b = { id: "b", name: "B" };
    const map = new Map([
      ["a", a],
      ["b", b],
    ]);
    expect(resolveEvidenceSeries(["b", "missing", "a"], map)).toEqual([b, a]);
  });
});

describe("formatCompactMoney", () => {
  it("compacts thousands", () => {
    expect(formatCompactMoney(1200)).toBe("1.2k €");
    expect(formatCompactMoney(50000)).toBe("50k €");
  });
});

describe("formatMonthHeadline", () => {
  it("formats a long English month title", () => {
    expect(formatMonthHeadline("2025-12-01")).toBe("December 2025");
  });
});

describe("buildMonthStory", () => {
  const months = [
    { monthStart: "2025-10-01", income: 3000, expenses: 2000, net: 1000 },
    { monthStart: "2025-11-01", income: 3000, expenses: 2100, net: 900 },
    { monthStart: "2025-12-01", income: 3000, expenses: 8000, net: -5000 },
  ];

  it("explains an expensive month versus median and prior", () => {
    const story = buildMonthStory(months, "2025-12-01", {
      oneOffCount: 1,
      oneOffExpenseTotal: 50000,
    });
    expect(story.unusual).toBe(true);
    expect(story.subline).toMatch(/above typical/i);
    expect(story.whyBullets.length).toBeGreaterThan(0);
    expect(story.whyBullets.some((b) => /median/i.test(b))).toBe(true);
    expect(story.whyBullets.some((b) => /one-off/i.test(b))).toBe(true);
  });

  it("returns null current when month is missing from series", () => {
    const story = buildMonthStory(months, "2026-01-01");
    expect(story.current).toBeNull();
  });
});
