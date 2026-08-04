import { describe, expect, it } from "vitest";

import {
  buildBalanceChartLayout,
  buildDualSeriesChartLayout,
  buildExpensePaceChartLayout,
  chartDateIndexes,
  chartLabelAnchor,
  chartMoneyTicks,
  formatChartDate,
  formatChartMoney,
} from "@/lib/balance-chart";

describe("chartDateIndexes", () => {
  it("returns empty for zero points", () => {
    expect(chartDateIndexes(0)).toEqual([]);
  });

  it("returns all indexes when there are few points", () => {
    expect(chartDateIndexes(1)).toEqual([0]);
    expect(chartDateIndexes(4)).toEqual([0, 1, 2, 3]);
  });

  it("returns sparse indexes including endpoints for many points", () => {
    expect(chartDateIndexes(13)).toEqual([0, 3, 6, 9, 12]);
  });
});

describe("formatChartDate", () => {
  it("formats ISO dates as DD.MM.", () => {
    expect(formatChartDate("2026-04-01")).toBe("01.04.");
    expect(formatChartDate("2026-12-31T00:00:00Z")).toBe("31.12.");
  });
});

describe("formatChartMoney", () => {
  it("formats thousands compactly for the Y-axis", () => {
    expect(formatChartMoney(2876.1)).toBe("2.9k €");
    expect(formatChartMoney(2000)).toBe("2k €");
    expect(formatChartMoney(-500)).toBe("−500 €");
  });
});

describe("chartMoneyTicks", () => {
  it("includes zero when the series crosses it", () => {
    expect(chartMoneyTicks(-100, 200)).toEqual([200, 0, -100]);
  });

  it("includes a mid tick when all values share a sign", () => {
    expect(chartMoneyTicks(0, 100)).toEqual([100, 50, 0]);
  });
});

describe("chartLabelAnchor", () => {
  it("anchors first and last labels to the edges", () => {
    expect(chartLabelAnchor(0, 5)).toBe("start");
    expect(chartLabelAnchor(4, 5)).toBe("end");
    expect(chartLabelAnchor(2, 5)).toBe("middle");
  });
});

describe("buildBalanceChartLayout", () => {
  it("returns null for empty or invalid balances", () => {
    expect(buildBalanceChartLayout([])).toBeNull();
    expect(
      buildBalanceChartLayout([{ date: "2026-01-01", balance: "nope" }]),
    ).toBeNull();
  });

  it("maps a single point to the horizontal center", () => {
    const layout = buildBalanceChartLayout(
      [{ date: "2026-01-01", balance: "100.00" }],
      { width: 100, height: 100, padX: 10, padTop: 10, padBottom: 10 },
    );
    expect(layout).not.toBeNull();
    expect(layout!.coords).toHaveLength(1);
    expect(layout!.coords[0]!.x).toBe(50);
    expect(layout!.linePath.startsWith("M ")).toBe(true);
    expect(layout!.labelIndexes).toEqual([0]);
  });

  it("places higher balances above lower ones (money on Y)", () => {
    const layout = buildBalanceChartLayout(
      [
        { date: "2026-01-01", balance: "0.00" },
        { date: "2026-01-08", balance: "100.00" },
      ],
      { width: 200, height: 100, padX: 0, padTop: 0, padBottom: 0 },
    );
    expect(layout).not.toBeNull();
    expect(layout!.coords[1]!.y).toBeLessThan(layout!.coords[0]!.y);
    expect(layout!.min).toBe(0);
    expect(layout!.max).toBe(100);
    expect(layout!.zeroY).toBe(100);
    expect(layout!.moneyLabels.map((l) => l.value)).toEqual([100, 50, 0]);
  });

  it("includes zero in the domain when all values are positive", () => {
    const layout = buildBalanceChartLayout([
      { date: "2026-01-01", balance: "50.00" },
      { date: "2026-01-08", balance: "80.00" },
    ]);
    expect(layout!.min).toBe(0);
    expect(layout!.max).toBe(80);
  });
});

describe("buildDualSeriesChartLayout", () => {
  it("returns null for empty input", () => {
    expect(buildDualSeriesChartLayout([])).toBeNull();
  });

  it("builds two paths on a shared money scale", () => {
    const layout = buildDualSeriesChartLayout([
      { date: "2026-01-01", primary: 3000, secondary: 2000 },
      { date: "2026-02-01", primary: 3100, secondary: 8000 },
    ]);
    expect(layout).not.toBeNull();
    expect(layout!.primaryPath.startsWith("M ")).toBe(true);
    expect(layout!.secondaryPath.startsWith("M ")).toBe(true);
    expect(layout!.max).toBe(8000);
    expect(layout!.xs).toHaveLength(2);
    expect(layout!.referenceLines).toEqual([]);
  });

  it("includes typical reference lines in the scale", () => {
    const layout = buildDualSeriesChartLayout(
      [
        { date: "2026-01-01", primary: 3000, secondary: 2000 },
        { date: "2026-02-01", primary: 3100, secondary: 2200 },
      ],
      {
        references: [
          { key: "typical_income", value: 3500 },
          { key: "typical_expenses", value: 1700 },
        ],
      },
    );
    expect(layout).not.toBeNull();
    expect(layout!.max).toBe(3500);
    expect(layout!.referenceLines).toHaveLength(2);
    expect(layout!.referenceLines[0]!.key).toBe("typical_income");
    expect(layout!.referenceLines[0]!.y).toBeLessThan(
      layout!.referenceLines[1]!.y,
    );
  });
});

describe("buildExpensePaceChartLayout", () => {
  it("returns null for empty input", () => {
    expect(buildExpensePaceChartLayout([])).toBeNull();
  });

  it("builds a cumulative path and count bars", () => {
    const layout = buildExpensePaceChartLayout([
      { date: "2026-07-01", cumulative: 100, dailyCount: 1 },
      { date: "2026-07-02", cumulative: 100, dailyCount: 0 },
      { date: "2026-07-03", cumulative: 250, dailyCount: 4 },
    ]);
    expect(layout).not.toBeNull();
    expect(layout!.linePath.startsWith("M ")).toBe(true);
    expect(layout!.areaPath.endsWith("Z")).toBe(true);
    expect(layout!.maxCount).toBe(4);
    expect(layout!.bars).toHaveLength(3);
    expect(layout!.bars[2]!.height).toBeGreaterThan(layout!.bars[0]!.height);
    expect(layout!.bars[1]!.height).toBe(0);
    // Plot starts near the left edge so it aligns with page text.
    expect(layout!.padLeft).toBeLessThan(8);
    expect(layout!.xs[0]).toBe(layout!.padLeft);
  });
});
