import { describe, expect, it } from "vitest";

import {
  buildBaselineComposition,
  buildBaselinePerformanceRows,
  buildBaselineReadinessItems,
  buildExpensePaceSeries,
  buildExpenseDevelopmentCallouts,
  buildExpenseDevelopmentRows,
  buildIncomeDevelopmentCallouts,
  buildIncomeDevelopmentRows,
  buildMonthStory,
  buildTypicalMonthLevels,
  detectUnusualMonth,
  eachISODateInclusive,
  formatCompactMoney,
  formatExpenseOneOffLine,
  formatMonthHeadline,
  formatMonthLabel,
  formatOneOffImpactLine,
  formatSignedPercent,
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

describe("formatOneOffImpactLine", () => {
  it("returns null when nothing is excluded", () => {
    expect(formatOneOffImpactLine(0, 0, "1.800,00 €")).toBeNull();
  });

  it("formats count and total against free cashflow", () => {
    expect(formatOneOffImpactLine(1, 50000, "1.800,00 €")).toMatch(
      /Excluding 1 one-off \(−50k €\), free cashflow is 1\.800,00 €/,
    );
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

describe("buildExpensePaceSeries", () => {
  it("lists inclusive calendar days", () => {
    expect(eachISODateInclusive("2026-02-27", "2026-03-01")).toEqual([
      "2026-02-27",
      "2026-02-28",
      "2026-03-01",
    ]);
  });

  it("fills empty days and accumulates expenses", () => {
    const series = buildExpensePaceSeries("2026-07-01", "2026-07-04", [
      { date: "2026-07-01", expenses: "100.00", transaction_count: 2 },
      { date: "2026-07-03", expenses: "50.50", transaction_count: 1 },
    ]);
    expect(series).toHaveLength(4);
    expect(series[0]).toMatchObject({
      date: "2026-07-01",
      dailyExpenses: 100,
      cumulativeExpenses: 100,
      transactionCount: 2,
    });
    expect(series[1]).toMatchObject({
      date: "2026-07-02",
      dailyExpenses: 0,
      cumulativeExpenses: 100,
      transactionCount: 0,
    });
    expect(series[2]?.cumulativeExpenses).toBeCloseTo(150.5);
    expect(series[3]?.cumulativeExpenses).toBeCloseTo(150.5);
  });
});

describe("buildBaselinePerformanceRows", () => {
  it("scores months against Cashflow typical levels", () => {
    const rows = buildBaselinePerformanceRows(
      [
        { monthStart: "2026-01-01", income: 3500, expenses: 1700, net: 1800 },
        { monthStart: "2026-02-01", income: 3500, expenses: 3000, net: 500 },
      ],
      { income: 3500, expenses: 1700 },
    );
    expect(rows[0]?.overspent).toBe(false);
    expect(rows[0]?.expensesDelta).toBe(0);
    expect(rows[1]?.overspent).toBe(true);
    expect(rows[1]?.expensesDelta).toBe(1300);
    expect(rows[1]?.netDelta).toBe(500 - (3500 - 1700));
  });
});

describe("buildIncomeDevelopmentCallouts", () => {
  it("flags months far from the income norm", () => {
    const got = buildIncomeDevelopmentCallouts(
      [
        { monthStart: "2026-01-01", income: 3500, expenses: 0, net: 3500 },
        { monthStart: "2026-02-01", income: 2000, expenses: 0, net: 2000 },
        { monthStart: "2026-03-01", income: 5000, expenses: 0, net: 5000 },
      ],
      3500,
    );
    expect(got.low.map((m) => m.monthStart)).toEqual(["2026-02-01"]);
    expect(got.high.map((m) => m.monthStart)).toEqual(["2026-03-01"]);
  });
});

describe("buildIncomeDevelopmentRows", () => {
  it("computes vs-norm and month-over-month percents", () => {
    const rows = buildIncomeDevelopmentRows(
      [
        { monthStart: "2026-01-01", income: 3500, expenses: 0, net: 3500 },
        { monthStart: "2026-02-01", income: 3500, expenses: 0, net: 3500 },
        { monthStart: "2026-03-01", income: 5000, expenses: 0, net: 5000 },
      ],
      3500,
    );
    expect(rows[0]).toMatchObject({
      vsNorm: 0,
      vsNormPct: 0,
      vsPriorPct: null,
    });
    expect(rows[1]).toMatchObject({
      vsNorm: 0,
      vsNormPct: 0,
      vsPriorPct: 0,
    });
    expect(rows[2]?.vsNorm).toBe(1500);
    expect(rows[2]?.vsNormPct).toBeCloseTo((1500 / 3500) * 100);
    expect(rows[2]?.vsPriorPct).toBeCloseTo((1500 / 3500) * 100);
  });

  it("omits vs-norm when there is no useful norm", () => {
    const rows = buildIncomeDevelopmentRows(
      [{ monthStart: "2026-01-01", income: 100, expenses: 0, net: 100 }],
      0,
    );
    expect(rows[0]?.vsNorm).toBeNull();
    expect(rows[0]?.vsNormPct).toBeNull();
  });
});

describe("formatSignedPercent", () => {
  it("formats signed percents for de-DE", () => {
    expect(formatSignedPercent(42.9)).toBe("+43 %");
    expect(formatSignedPercent(-12.1)).toBe("−12 %");
    expect(formatSignedPercent(0)).toBe("0 %");
  });
});

describe("buildExpenseDevelopmentCallouts", () => {
  it("flags months far from the cost norm", () => {
    const got = buildExpenseDevelopmentCallouts(
      [
        { monthStart: "2026-01-01", income: 0, expenses: 1700, net: 0 },
        { monthStart: "2026-02-01", income: 0, expenses: 1000, net: 0 },
        { monthStart: "2026-03-01", income: 0, expenses: 2500, net: 0 },
      ],
      1700,
    );
    expect(got.low.map((m) => m.monthStart)).toEqual(["2026-02-01"]);
    expect(got.high.map((m) => m.monthStart)).toEqual(["2026-03-01"]);
  });
});

describe("buildExpenseDevelopmentRows", () => {
  it("computes vs-norm and month-over-month percents for expenses", () => {
    const rows = buildExpenseDevelopmentRows(
      [
        { monthStart: "2026-01-01", income: 0, expenses: 1700, net: 0 },
        { monthStart: "2026-02-01", income: 0, expenses: 1700, net: 0 },
        { monthStart: "2026-03-01", income: 0, expenses: 2200, net: 0 },
      ],
      1700,
    );
    expect(rows[0]).toMatchObject({
      vsNorm: 0,
      vsNormPct: 0,
      vsPriorPct: null,
    });
    expect(rows[2]?.vsNorm).toBe(500);
    expect(rows[2]?.vsNormPct).toBeCloseTo((500 / 1700) * 100);
    expect(rows[2]?.vsPriorPct).toBeCloseTo((500 / 1700) * 100);
  });
});

describe("formatExpenseOneOffLine", () => {
  it("returns null when there are no one-offs", () => {
    expect(formatExpenseOneOffLine(0, 0)).toBeNull();
  });

  it("summarizes one-offs excluded from the cost norm", () => {
    expect(formatExpenseOneOffLine(2, 800)).toMatch(/2 one-offs/);
    expect(formatExpenseOneOffLine(2, 800)).toMatch(/excluded from the norm/);
  });
});
