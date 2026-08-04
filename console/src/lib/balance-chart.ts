export type BalanceChartPoint = {
  date: string;
  balance: string;
};

export type BalanceChartCoord = {
  x: number;
  y: number;
};

export type BalanceChartAxisLabel = {
  value: number;
  y: number;
  text: string;
};

export type BalanceChartLayout = {
  width: number;
  height: number;
  padX: number;
  padTop: number;
  padBottom: number;
  innerW: number;
  innerH: number;
  min: number;
  max: number;
  coords: BalanceChartCoord[];
  linePath: string;
  areaPath: string;
  zeroY: number;
  labelIndexes: number[];
  /** Euro labels along the vertical (Y) axis. */
  moneyLabels: BalanceChartAxisLabel[];
};

const DEFAULT_WIDTH = 640;
const DEFAULT_HEIGHT = 200;
/** Room for Y-axis euro labels on the left. */
const DEFAULT_PAD_X = 64;
const DEFAULT_PAD_TOP = 12;
const DEFAULT_PAD_BOTTOM = 36;

/** Pick sparse x-axis label indexes: endpoints plus midpoints when many points. */
export function chartDateIndexes(count: number): number[] {
  if (count <= 0) {
    return [];
  }
  if (count === 1) {
    return [0];
  }
  if (count <= 5) {
    return Array.from({ length: count }, (_, i) => i);
  }
  const mid = Math.floor((count - 1) / 2);
  const q1 = Math.floor((count - 1) / 4);
  const q3 = Math.floor(((count - 1) * 3) / 4);
  return [...new Set([0, q1, mid, q3, count - 1])].sort((a, b) => a - b);
}

/** Format ISO date for chart X-axis as DD.MM. */
export function formatChartDate(value: string): string {
  const iso = value.slice(0, 10);
  const [y, m, d] = iso.split("-");
  if (!y || !m || !d) {
    return iso;
  }
  return `${d}.${m}.`;
}

/** Compact euro label for the Y-axis. */
export function formatChartMoney(value: number): string {
  const abs = Math.abs(value);
  const sign = value < 0 ? "−" : "";
  if (abs >= 1000) {
    const thousands = abs / 1000;
    const text =
      abs % 1000 === 0 ? thousands.toFixed(0) : thousands.toFixed(1);
    return `${sign}${text}k €`;
  }
  return `${sign}${abs.toFixed(0)} €`;
}

export function chartLabelAnchor(
  index: number,
  count: number,
): "start" | "middle" | "end" {
  if (index === 0) {
    return "start";
  }
  if (index === count - 1) {
    return "end";
  }
  return "middle";
}

function yForValue(
  value: number,
  min: number,
  span: number,
  padTop: number,
  innerH: number,
): number {
  return padTop + innerH - ((value - min) / span) * innerH;
}

/** Choose high / mid / low (and 0 when in range) ticks for the money axis. */
export function chartMoneyTicks(min: number, max: number): number[] {
  const ticks = new Set<number>([min, max]);
  if (min < 0 && max > 0) {
    ticks.add(0);
  } else {
    ticks.add((min + max) / 2);
  }
  return [...ticks].sort((a, b) => b - a);
}

/** Build SVG geometry for a cash-balance series. Returns null when empty. */
export function buildBalanceChartLayout(
  points: BalanceChartPoint[],
  options?: {
    width?: number;
    height?: number;
    padX?: number;
    padTop?: number;
    padBottom?: number;
  },
): BalanceChartLayout | null {
  if (!points.length) {
    return null;
  }

  const values = points.map((p) => Number.parseFloat(p.balance));
  if (values.some((v) => Number.isNaN(v))) {
    return null;
  }

  const width = options?.width ?? DEFAULT_WIDTH;
  const height = options?.height ?? DEFAULT_HEIGHT;
  const padX = options?.padX ?? DEFAULT_PAD_X;
  const padTop = options?.padTop ?? DEFAULT_PAD_TOP;
  const padBottom = options?.padBottom ?? DEFAULT_PAD_BOTTOM;
  const innerW = width - padX * 2;
  const innerH = height - padTop - padBottom;

  const min = Math.min(...values, 0);
  const max = Math.max(...values, 0);
  const span = max - min || 1;

  const coords = values.map((v, i) => {
    const x =
      padX +
      (points.length === 1 ? innerW / 2 : (i / (points.length - 1)) * innerW);
    const y = yForValue(v, min, span, padTop, innerH);
    return { x, y };
  });

  const linePath = coords
    .map((c, i) => `${i === 0 ? "M" : "L"} ${c.x} ${c.y}`)
    .join(" ");
  const last = coords[coords.length - 1]!;
  const first = coords[0]!;
  const baselineY = padTop + innerH;
  const areaPath = `${linePath} L ${last.x} ${baselineY} L ${first.x} ${baselineY} Z`;
  const zeroY = yForValue(0, min, span, padTop, innerH);

  const moneyLabels = chartMoneyTicks(min, max).map((value) => ({
    value,
    y: yForValue(value, min, span, padTop, innerH),
    text: formatChartMoney(value),
  }));

  return {
    width,
    height,
    padX,
    padTop,
    padBottom,
    innerW,
    innerH,
    min,
    max,
    coords,
    linePath,
    areaPath,
    zeroY,
    labelIndexes: chartDateIndexes(points.length),
    moneyLabels,
  };
}

export type DualSeriesPoint = {
  date: string;
  primary: number;
  secondary: number;
};

export type DualSeriesChartLayout = {
  width: number;
  height: number;
  padX: number;
  padTop: number;
  padBottom: number;
  innerH: number;
  min: number;
  max: number;
  primaryPath: string;
  secondaryPath: string;
  zeroY: number;
  labelIndexes: number[];
  moneyLabels: BalanceChartAxisLabel[];
  xs: number[];
  primaryYs: number[];
  secondaryYs: number[];
};

/** Two series on one chart (e.g. income + expenses over months). */
export function buildDualSeriesChartLayout(
  points: DualSeriesPoint[],
  options?: {
    width?: number;
    height?: number;
    padX?: number;
    padTop?: number;
    padBottom?: number;
  },
): DualSeriesChartLayout | null {
  if (!points.length) {
    return null;
  }
  if (
    points.some(
      (p) => Number.isNaN(p.primary) || Number.isNaN(p.secondary),
    )
  ) {
    return null;
  }

  const width = options?.width ?? DEFAULT_WIDTH;
  const height = options?.height ?? DEFAULT_HEIGHT;
  const padX = options?.padX ?? DEFAULT_PAD_X;
  const padTop = options?.padTop ?? DEFAULT_PAD_TOP;
  const padBottom = options?.padBottom ?? DEFAULT_PAD_BOTTOM;
  const innerW = width - padX * 2;
  const innerH = height - padTop - padBottom;

  const values = points.flatMap((p) => [p.primary, p.secondary]);
  const min = Math.min(...values, 0);
  const max = Math.max(...values, 0);
  const span = max - min || 1;

  const xs = points.map((_, i) =>
    padX +
    (points.length === 1 ? innerW / 2 : (i / (points.length - 1)) * innerW),
  );
  const primaryCoords = points.map((p, i) => ({
    x: xs[i]!,
    y: yForValue(p.primary, min, span, padTop, innerH),
  }));
  const secondaryCoords = points.map((p, i) => ({
    x: xs[i]!,
    y: yForValue(p.secondary, min, span, padTop, innerH),
  }));
  const toPath = (coords: BalanceChartCoord[]) =>
    coords.map((c, i) => `${i === 0 ? "M" : "L"} ${c.x} ${c.y}`).join(" ");

  return {
    width,
    height,
    padX,
    padTop,
    padBottom,
    innerH,
    min,
    max,
    primaryPath: toPath(primaryCoords),
    secondaryPath: toPath(secondaryCoords),
    zeroY: yForValue(0, min, span, padTop, innerH),
    labelIndexes: chartDateIndexes(points.length),
    moneyLabels: chartMoneyTicks(min, max).map((value) => ({
      value,
      y: yForValue(value, min, span, padTop, innerH),
      text: formatChartMoney(value),
    })),
    xs,
    primaryYs: primaryCoords.map((c) => c.y),
    secondaryYs: secondaryCoords.map((c) => c.y),
  };
}
