import { describe, expect, it } from "vitest";

import {
  buildBalanceChartLayout,
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
