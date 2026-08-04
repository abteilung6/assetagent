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

/** Click targets on the composition bar / income row. */
export type CompositionEvidenceKey = "income" | CompositionSegmentKey;

export type TypicalMonthLevels = {
  income: number;
  expenses: number;
};

/** Typical income and total costs from a confirmed/draft baseline. */
export function buildTypicalMonthLevels(input: {
  income: number;
  fixed: number;
  irregular: number;
  variable: number;
}): TypicalMonthLevels {
  const income = Math.max(0, input.income);
  const expenses =
    Math.max(0, input.fixed) +
    Math.max(0, input.irregular) +
    Math.max(0, input.variable);
  return { income, expenses };
}

export type BaselineReadinessItem = {
  id: "transfers" | "categories" | "recurring" | "unusual_month";
  label: string;
  /** Optional count for display, when relevant. */
  count?: number;
  href:
    | { kind: "review"; tab: "transfers" | "categories" | "recurring" }
    | { kind: "month"; yyyyMm: string };
};

/** Open review/trust items before confirming a draft baseline. */
export function buildBaselineReadinessItems(input: {
  transferCount: number;
  categoryCount: number;
  recurringCount: number;
  unusualMonthStart: string | null;
}): BaselineReadinessItem[] {
  const items: BaselineReadinessItem[] = [];
  if (input.transferCount > 0) {
    items.push({
      id: "transfers",
      count: input.transferCount,
      label:
        input.transferCount === 1
          ? "1 transfer to review"
          : `${input.transferCount} transfers to review`,
      href: { kind: "review", tab: "transfers" },
    });
  }
  if (input.categoryCount > 0) {
    items.push({
      id: "categories",
      count: input.categoryCount,
      label:
        input.categoryCount === 1
          ? "1 category to check"
          : `${input.categoryCount} categories to check`,
      href: { kind: "review", tab: "categories" },
    });
  }
  if (input.recurringCount > 0) {
    items.push({
      id: "recurring",
      count: input.recurringCount,
      label:
        input.recurringCount === 1
          ? "1 recurring payment to confirm"
          : `${input.recurringCount} recurring payments to confirm`,
      href: { kind: "review", tab: "recurring" },
    });
  }
  if (input.unusualMonthStart) {
    const yyyyMm = input.unusualMonthStart.slice(0, 7);
    items.push({
      id: "unusual_month",
      label: `Unusual expenses in ${formatMonthLabel(input.unusualMonthStart)}`,
      href: { kind: "month", yyyyMm },
    });
  }
  return items;
}

export type CompositionEvidenceMetricKey =
  | "regular_monthly_income"
  | "monthly_fixed_costs"
  | "monthly_irregular_costs"
  | "avg_variable_spend"
  | "sustainable_free_cashflow";

export function compositionEvidenceMetricKey(
  key: CompositionEvidenceKey,
): CompositionEvidenceMetricKey {
  switch (key) {
    case "income":
      return "regular_monthly_income";
    case "fixed":
      return "monthly_fixed_costs";
    case "irregular":
      return "monthly_irregular_costs";
    case "variable":
      return "avg_variable_spend";
    case "free":
    case "deficit":
      return "sustainable_free_cashflow";
  }
}

export function compositionEvidenceTitle(key: CompositionEvidenceKey): string {
  switch (key) {
    case "income":
      return "Regular monthly income";
    case "fixed":
      return "Monthly fixed costs";
    case "irregular":
      return "Monthly irregular costs";
    case "variable":
      return "Average variable spend";
    case "free":
      return "Free cashflow";
    case "deficit":
      return "Shortfall";
  }
}

/** Monthly-equivalent amount matching baseline finance.Compute. */
export function monthlyEquivalentAmount(
  amount: number,
  interval: "monthly" | "quarterly" | "yearly" | string,
): number {
  const abs = Math.abs(amount);
  switch (interval) {
    case "quarterly":
      return abs / 3;
    case "yearly":
      return abs / 12;
    default:
      return abs;
  }
}

/** Preserve evidence_ids order; skip IDs missing from the series map. */
export function resolveEvidenceSeries<T extends { id: string }>(
  evidenceIds: string[],
  seriesById: Map<string, T>,
): T[] {
  const out: T[] = [];
  for (const id of evidenceIds) {
    const series = seriesById.get(id);
    if (series) {
      out.push(series);
    }
  }
  return out;
}

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

export type BaselinePerformanceRow = {
  monthStart: string;
  income: number;
  expenses: number;
  net: number;
  /** Actual income − typical income. */
  incomeDelta: number;
  /** Actual expenses − typical expenses. */
  expensesDelta: number;
  /** Actual net − typical net (income − expenses). */
  netDelta: number;
  /** Expenses clearly above the Cashflow norm. */
  overspent: boolean;
};

/** Score recent months against Cashflow typical levels (Baseline → Tracking). */
export function buildBaselinePerformanceRows(
  months: MonthlyCashflowPoint[],
  typical: TypicalMonthLevels,
): BaselinePerformanceRow[] {
  const typicalNet = typical.income - typical.expenses;
  return months.map((m) => {
    const expensesDelta = m.expenses - typical.expenses;
    return {
      monthStart: m.monthStart,
      income: m.income,
      expenses: m.expenses,
      net: m.net,
      incomeDelta: m.income - typical.income,
      expensesDelta,
      netDelta: m.net - typicalNet,
      overspent:
        typical.expenses > 0
          ? m.expenses >= typical.expenses * 1.25
          : m.expenses > 0,
    };
  });
}

/** Months whose booked income sits clearly outside the Cashflow income norm. */
export function buildIncomeDevelopmentCallouts(
  months: MonthlyCashflowPoint[],
  typicalIncome: number,
): { low: MonthlyCashflowPoint[]; high: MonthlyCashflowPoint[] } {
  if (typicalIncome <= 0) {
    return { low: [], high: [] };
  }
  const low: MonthlyCashflowPoint[] = [];
  const high: MonthlyCashflowPoint[] = [];
  for (const m of months) {
    if (m.income <= typicalIncome * 0.75) {
      low.push(m);
    } else if (m.income >= typicalIncome * 1.25) {
      high.push(m);
    }
  }
  return { low, high };
}

export type IncomeDevelopmentRow = {
  monthStart: string;
  income: number;
  /** Booked − Cashflow income norm; null when there is no useful norm. */
  vsNorm: number | null;
  /** Percent of norm: (booked − norm) / norm × 100. */
  vsNormPct: number | null;
  /** Month-over-month percent vs prior booked income; null for first row or zero prior. */
  vsPriorPct: number | null;
};

/** Month rows for Income → Development (chart companion table). */
export function buildIncomeDevelopmentRows(
  months: MonthlyCashflowPoint[],
  typicalIncome: number,
): IncomeDevelopmentRow[] {
  return months.map((m, i) => {
    const prior = i > 0 ? months[i - 1] : undefined;
    const vsNorm = typicalIncome > 0 ? m.income - typicalIncome : null;
    const vsNormPct =
      typicalIncome > 0
        ? ((m.income - typicalIncome) / typicalIncome) * 100
        : null;
    const vsPriorPct =
      prior && prior.income !== 0
        ? ((m.income - prior.income) / Math.abs(prior.income)) * 100
        : null;
    return {
      monthStart: m.monthStart,
      income: m.income,
      vsNorm,
      vsNormPct,
      vsPriorPct,
    };
  });
}

/** Signed percent for development deltas (e.g. +43 %, −12 %). */
export function formatSignedPercent(value: number): string {
  const abs = new Intl.NumberFormat("de-DE", {
    maximumFractionDigits: 0,
  }).format(Math.abs(value));
  if (value > 0) {
    return `+${abs} %`;
  }
  if (value < 0) {
    return `−${abs} %`;
  }
  return `${abs} %`;
}

export type CategorySpendInput = {
  category_slug: string;
  category_name: string;
  total: string | number;
  transaction_count?: number;
};

export type CategoryShareRow = {
  categorySlug: string;
  categoryName: string;
  total: number;
  share: number;
  transactionCount: number;
};

/** Rank categories by spend and attach share of the period total. */
export function buildCategoryShareRows(
  points: CategorySpendInput[],
): CategoryShareRow[] {
  const rows = points
    .map((point) => ({
      categorySlug: point.category_slug,
      categoryName: point.category_name,
      total:
        typeof point.total === "number"
          ? point.total
          : Number.parseFloat(point.total) || 0,
      transactionCount: point.transaction_count ?? 0,
    }))
    .filter((row) => row.total > 0)
    .sort((a, b) => b.total - a.total);
  const grand = rows.reduce((sum, row) => sum + row.total, 0);
  return rows.map((row) => ({
    ...row,
    share: grand > 0 ? row.total / grand : 0,
  }));
}

export type CategoryMover = {
  categorySlug: string;
  categoryName: string;
  current: number;
  prior: number;
  delta: number;
};

/** Largest absolute category spend changes between two periods. */
export function buildCategoryMovers(
  current: CategorySpendInput[],
  prior: CategorySpendInput[],
  limit = 5,
): CategoryMover[] {
  const bySlug = new Map<
    string,
    { name: string; current: number; prior: number }
  >();
  for (const point of prior) {
    const total =
      typeof point.total === "number"
        ? point.total
        : Number.parseFloat(point.total) || 0;
    bySlug.set(point.category_slug, {
      name: point.category_name,
      current: 0,
      prior: total,
    });
  }
  for (const point of current) {
    const total =
      typeof point.total === "number"
        ? point.total
        : Number.parseFloat(point.total) || 0;
    const existing = bySlug.get(point.category_slug);
    if (existing) {
      existing.current = total;
      existing.name = point.category_name;
    } else {
      bySlug.set(point.category_slug, {
        name: point.category_name,
        current: total,
        prior: 0,
      });
    }
  }
  return [...bySlug.entries()]
    .map(([categorySlug, row]) => ({
      categorySlug,
      categoryName: row.name,
      current: row.current,
      prior: row.prior,
      delta: row.current - row.prior,
    }))
    .filter((row) => Math.abs(row.delta) >= 1)
    .sort((a, b) => Math.abs(b.delta) - Math.abs(a.delta))
    .slice(0, limit);
}

/** Inclusive booking window covering the last N complete calendar months. */
export function completeMonthsWindow(
  months: number,
  now = new Date(),
): { from: string; to: string; months: number } {
  const n = Math.max(1, Math.floor(months));
  const firstOfThisMonth = new Date(now.getFullYear(), now.getMonth(), 1);
  const lastMonthEnd = new Date(firstOfThisMonth.getTime() - 1);
  const fromDate = new Date(
    lastMonthEnd.getFullYear(),
    lastMonthEnd.getMonth() - (n - 1),
    1,
  );
  const yyyy = lastMonthEnd.getFullYear();
  const mm = String(lastMonthEnd.getMonth() + 1).padStart(2, "0");
  const lastDay = String(lastMonthEnd.getDate()).padStart(2, "0");
  const fromY = fromDate.getFullYear();
  const fromM = String(fromDate.getMonth() + 1).padStart(2, "0");
  return {
    from: `${fromY}-${fromM}-01`,
    to: `${yyyy}-${mm}-${lastDay}`,
    months: n,
  };
}

/** Months whose booked expenses sit clearly outside the Cashflow cost norm. */
export function buildExpenseDevelopmentCallouts(
  months: MonthlyCashflowPoint[],
  typicalExpenses: number,
): { low: MonthlyCashflowPoint[]; high: MonthlyCashflowPoint[] } {
  if (typicalExpenses <= 0) {
    return { low: [], high: [] };
  }
  const low: MonthlyCashflowPoint[] = [];
  const high: MonthlyCashflowPoint[] = [];
  for (const m of months) {
    if (m.expenses <= typicalExpenses * 0.75) {
      low.push(m);
    } else if (m.expenses >= typicalExpenses * 1.25) {
      high.push(m);
    }
  }
  return { low, high };
}

export type ExpenseDevelopmentRow = {
  monthStart: string;
  expenses: number;
  /** Booked − Cashflow cost norm; null when there is no useful norm. */
  vsNorm: number | null;
  /** Percent of norm: (booked − norm) / norm × 100. */
  vsNormPct: number | null;
  /** Month-over-month percent vs prior booked expenses; null for first row or zero prior. */
  vsPriorPct: number | null;
};

/** Month rows for Expenses → Development (chart companion table). */
export function buildExpenseDevelopmentRows(
  months: MonthlyCashflowPoint[],
  typicalExpenses: number,
): ExpenseDevelopmentRow[] {
  return months.map((m, i) => {
    const prior = i > 0 ? months[i - 1] : undefined;
    const vsNorm = typicalExpenses > 0 ? m.expenses - typicalExpenses : null;
    const vsNormPct =
      typicalExpenses > 0
        ? ((m.expenses - typicalExpenses) / typicalExpenses) * 100
        : null;
    const vsPriorPct =
      prior && prior.expenses !== 0
        ? ((m.expenses - prior.expenses) / Math.abs(prior.expenses)) * 100
        : null;
    return {
      monthStart: m.monthStart,
      expenses: m.expenses,
      vsNorm,
      vsNormPct,
      vsPriorPct,
    };
  });
}

/** Quiet line for Expenses hero when one-offs exist in the baseline window. */
export function formatExpenseOneOffLine(
  count: number,
  expenseTotal: number,
): string | null {
  if (count <= 0 || expenseTotal <= 0) {
    return null;
  }
  const countLabel = count === 1 ? "1 one-off" : `${count} one-offs`;
  return `${countLabel} in window · ${formatCompactMoney(expenseTotal)} — excluded from the norm.`;
}

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

/** Compact euro for chart strip labels (e.g. 1.2k €). */
export function formatCompactMoney(value: number): string {
  const abs = Math.abs(value);
  const sign = value < 0 ? "−" : "";
  if (abs >= 1000) {
    const thousands = abs / 1000;
    const text =
      abs % 1000 === 0 ? thousands.toFixed(0) : thousands.toFixed(1);
    return `${sign}${text}k €`;
  }
  return `${sign}${Math.round(abs)} €`;
}

/** Long month title for the month insight page (e.g. December 2025). */
export function formatMonthHeadline(monthStart: string): string {
  const iso = monthStart.slice(0, 10);
  const match = /^(\d{4})-(\d{2})/.exec(iso);
  if (!match) {
    return iso;
  }
  const date = new Date(Date.UTC(Number(match[1]), Number(match[2]) - 1, 1));
  return new Intl.DateTimeFormat("en-GB", {
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  }).format(date);
}

export function yyyyMmFromMonthStart(monthStart: string): string {
  return monthStart.slice(0, 7);
}

export function monthStartFromYyyyMm(yyyyMm: string): string | null {
  if (!/^\d{4}-\d{2}$/.test(yyyyMm)) {
    return null;
  }
  return `${yyyyMm}-01`;
}

export function endOfMonthISO(monthStart: string): string {
  const iso = monthStart.slice(0, 10);
  const [y, m] = iso.split("-").map(Number);
  if (!y || !m) {
    return iso;
  }
  const last = new Date(Date.UTC(y, m, 0));
  return last.toISOString().slice(0, 10);
}

/** Inclusive list of ISO dates from `from` through `to` (UTC date-only). */
export function eachISODateInclusive(from: string, to: string): string[] {
  const start = from.slice(0, 10);
  const end = to.slice(0, 10);
  const out: string[] = [];
  const cursor = new Date(`${start}T00:00:00.000Z`);
  const last = new Date(`${end}T00:00:00.000Z`);
  if (Number.isNaN(cursor.getTime()) || Number.isNaN(last.getTime())) {
    return out;
  }
  while (cursor.getTime() <= last.getTime()) {
    out.push(cursor.toISOString().slice(0, 10));
    cursor.setUTCDate(cursor.getUTCDate() + 1);
  }
  return out;
}

export type ExpensePaceSeriesDay = {
  date: string;
  dailyExpenses: number;
  cumulativeExpenses: number;
  transactionCount: number;
};

/** Fill omitted calendar days and compute a running expense total. */
export function buildExpensePaceSeries(
  from: string,
  to: string,
  rows: Array<{
    date: string;
    expenses: string;
    transaction_count: number;
  }>,
): ExpensePaceSeriesDay[] {
  const byDate = new Map<string, { expenses: number; count: number }>();
  for (const row of rows) {
    const date = row.date.slice(0, 10);
    const expenses = Number.parseFloat(row.expenses);
    byDate.set(date, {
      expenses: Number.isNaN(expenses) ? 0 : Math.max(0, expenses),
      count: row.transaction_count,
    });
  }
  let cumulative = 0;
  return eachISODateInclusive(from, to).map((date) => {
    const day = byDate.get(date) ?? { expenses: 0, count: 0 };
    cumulative += day.expenses;
    return {
      date,
      dailyExpenses: day.expenses,
      cumulativeExpenses: cumulative,
      transactionCount: day.count,
    };
  });
}

export function shiftYyyyMm(yyyyMm: string, deltaMonths: number): string {
  const [y, m] = yyyyMm.split("-").map(Number);
  if (!y || !m) {
    return yyyyMm;
  }
  const date = new Date(Date.UTC(y, m - 1 + deltaMonths, 1));
  const year = date.getUTCFullYear();
  const month = String(date.getUTCMonth() + 1).padStart(2, "0");
  return `${year}-${month}`;
}

export type MonthStory = {
  current: MonthlyCashflowPoint | null;
  prior: MonthlyCashflowPoint | null;
  medianExpenses: number | null;
  expenseRatioToMedian: number | null;
  unusual: boolean;
  subline: string;
  whyBullets: string[];
};

export type MonthStoryOptions = {
  oneOffCount?: number;
  oneOffExpenseTotal?: number;
};

export type MonthSpendPartition<T extends { recurring: boolean }> = {
  recurring: T[];
  oneTime: T[];
};

/** Quiet line explaining one-offs already excluded from free cashflow. */
export function formatOneOffImpactLine(
  count: number,
  expenseTotal: number,
  freeCashflowLabel: string,
): string | null {
  if (count <= 0 || expenseTotal <= 0) {
    return null;
  }
  const countLabel =
    count === 1 ? "1 one-off" : `${count} one-offs`;
  return `Excluding ${countLabel} (−${formatCompactMoney(expenseTotal)}), free cashflow is ${freeCashflowLabel}.`;
}

/** Split expense drivers into recurring series members vs everything else. */
export function partitionMonthSpend<T extends { recurring: boolean }>(
  transactions: T[],
  limitPerGroup = 8,
): MonthSpendPartition<T> {
  const recurring: T[] = [];
  const oneTime: T[] = [];
  for (const tx of transactions) {
    if (tx.recurring) {
      if (recurring.length < limitPerGroup) {
        recurring.push(tx);
      }
    } else if (oneTime.length < limitPerGroup) {
      oneTime.push(tx);
    }
  }
  return { recurring, oneTime };
}

/** Build headline copy and why-bullets for a selected month. */
export function buildMonthStory(
  months: MonthlyCashflowPoint[],
  monthStart: string,
  opts: MonthStoryOptions = {},
): MonthStory {
  const target = monthStart.slice(0, 10);
  const index = months.findIndex((m) => m.monthStart.slice(0, 10) === target);
  const current = index >= 0 ? months[index]! : null;
  const prior = index > 0 ? months[index - 1]! : null;

  const expenseValues = months
    .map((m) => m.expenses)
    .filter((v) => v > 0)
    .sort((a, b) => a - b);
  const medianExpenses =
    expenseValues.length > 0 ? percentile(expenseValues, 0.5) : null;
  const expenseRatioToMedian =
    current && medianExpenses && medianExpenses > 0
      ? current.expenses / medianExpenses
      : null;
  const unusual =
    expenseRatioToMedian != null && expenseRatioToMedian >= 2;

  const whyBullets: string[] = [];
  if (unusual && expenseRatioToMedian != null && medianExpenses != null) {
    whyBullets.push(
      `Expenses ${expenseRatioToMedian.toFixed(1)}× the recent median (${formatCompactMoney(medianExpenses)}).`,
    );
  }
  if (current && prior) {
    const delta = current.expenses - prior.expenses;
    if (Math.abs(delta) >= 1) {
      const direction = delta > 0 ? "higher" : "lower";
      whyBullets.push(
        `Expenses ${formatCompactMoney(Math.abs(delta))} ${direction} than ${formatMonthLabel(prior.monthStart)}.`,
      );
    }
  }
  const oneOffCount = opts.oneOffCount ?? 0;
  const oneOffExpenseTotal = opts.oneOffExpenseTotal ?? 0;
  if (oneOffCount > 0) {
    whyBullets.push(
      oneOffCount === 1
        ? `1 one-off payment excluded from totals (${formatCompactMoney(oneOffExpenseTotal)}).`
        : `${oneOffCount} one-off payments excluded from totals (${formatCompactMoney(oneOffExpenseTotal)}).`,
    );
  }
  if (current && current.net < 0) {
    whyBullets.push("Spent more than earned this month.");
  }

  let subline = "Income, expenses, and what moved the needle.";
  if (unusual) {
    subline = "Expenses above typical — check large payments below.";
  } else if (current && prior && current.expenses < prior.expenses * 0.85) {
    subline = "Quieter month than the one before.";
  } else if (oneOffCount > 0) {
    subline = "One-offs are excluded from the totals above.";
  }

  return {
    current,
    prior,
    medianExpenses,
    expenseRatioToMedian,
    unusual,
    subline,
    whyBullets: whyBullets.slice(0, 3),
  };
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
