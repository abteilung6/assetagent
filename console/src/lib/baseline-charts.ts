export type BaselineCompositionInput = {
  income: number;
  fixed: number;
  irregular: number;
  variable: number;
  freeCashflow: number;
};

export type CompositionSegmentKey =
  | "fixed"
  | "irregular"
  | "variable"
  | "free"
  | "deficit";

export type CompositionSegment = {
  key: CompositionSegmentKey;
  label: string;
  amount: number;
  /** Share of the income bar width (0–1). */
  share: number;
};

export type BaselineComposition = {
  income: number;
  segments: CompositionSegment[];
  /** True when costs exceed income. */
  overspent: boolean;
};

export type MonthlyCashflowPoint = {
  monthStart: string; // YYYY-MM-01
  income: number;
  expenses: number;
  net: number;
};

export type UnusualMonthInsight = {
  unusual: boolean;
  monthStart: string | null;
  /** Expenses / median expenses when unusual. */
  ratio: number | null;
  message: string | null;
};

/** Build stacked cost segments relative to monthly income. */
export function buildBaselineComposition(
  input: BaselineCompositionInput,
): BaselineComposition {
  const income = Math.max(0, input.income);
  const fixed = Math.max(0, input.fixed);
  const irregular = Math.max(0, input.irregular);
  const variable = Math.max(0, input.variable);
  const costs = fixed + irregular + variable;
  const scale = income > 0 ? income : costs > 0 ? costs : 1;

  const segments: CompositionSegment[] = [];
  const push = (
    key: CompositionSegmentKey,
    label: string,
    amount: number,
  ) => {
    if (amount <= 0) {
      return;
    }
    segments.push({
      key,
      label,
      amount,
      share: amount / scale,
    });
  };

  push("fixed", "Fixed", fixed);
  push("irregular", "Irregular", irregular);
  push("variable", "Variable", variable);

  if (input.freeCashflow > 0) {
    push("free", "Free cashflow", input.freeCashflow);
  } else if (input.freeCashflow < 0 || costs > income) {
    const deficit = Math.max(-input.freeCashflow, costs - income);
    push("deficit", "Shortfall", deficit);
  }

  return {
    income,
    segments,
    overspent: costs > income || input.freeCashflow < 0,
  };
}

/** Flag the month with expenses furthest above the median (if ≥ 2×). */
export function detectUnusualMonth(
  months: MonthlyCashflowPoint[],
  baselineMonthStart?: string,
): UnusualMonthInsight {
  if (months.length < 2) {
    return { unusual: false, monthStart: null, ratio: null, message: null };
  }

  const expenses = months.map((m) => m.expenses).sort((a, b) => a - b);
  const median = percentile(expenses, 0.5);
  if (median <= 0) {
    return { unusual: false, monthStart: null, ratio: null, message: null };
  }

  let worst = months[0]!;
  for (const m of months) {
    if (m.expenses > worst.expenses) {
      worst = m;
    }
  }
  const ratio = worst.expenses / median;
  if (ratio < 2) {
    return { unusual: false, monthStart: null, ratio: null, message: null };
  }

  const inBaseline =
    baselineMonthStart != null &&
    worst.monthStart.slice(0, 7) === baselineMonthStart.slice(0, 7);

  return {
    unusual: true,
    monthStart: worst.monthStart,
    ratio,
    message: inBaseline
      ? "This baseline month looks unusually expensive versus recent months — check Needs review for transfers or large one-offs before confirming."
      : `Expenses in ${formatMonthLabel(worst.monthStart)} look about ${ratio.toFixed(1)}× a typical month — check transfers or one-offs (e.g. apartment purchase).`,
  };
}

export function formatMonthLabel(monthStart: string): string {
  const iso = monthStart.slice(0, 10);
  const [y, m] = iso.split("-");
  if (!y || !m) {
    return iso;
  }
  return `${m}.${y}`;
}

function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) {
    return 0;
  }
  if (sorted.length === 1) {
    return sorted[0]!;
  }
  const idx = (sorted.length - 1) * p;
  const lo = Math.floor(idx);
  const hi = Math.ceil(idx);
  if (lo === hi) {
    return sorted[lo]!;
  }
  const w = idx - lo;
  return sorted[lo]! * (1 - w) + sorted[hi]! * w;
}
