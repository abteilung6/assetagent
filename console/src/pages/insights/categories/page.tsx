import type React from "react";
import { useMemo, useState } from "react";
import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import { AskAboutThis } from "@/components/chat/ask-about-this";
import { Button } from "@/components/ui/button";
import {
  useBaselineCategoryMerchants,
  useBaselineCategorySpend,
  useBaselineCategorySpendMonthly,
} from "@/hooks/use-baseline";
import {
  buildMultiSeriesChartLayout,
  chartLabelAnchor,
  formatChartDate,
  formatChartMoney,
} from "@/lib/balance-chart";
import {
  buildCategoryMonthlyChartModel,
  buildCategoryMovers,
  buildCategoryShareRows,
  completeMonthsWindow,
  formatCompactMoney,
  formatMonthHeadline,
  yyyyMmFromMonthStart,
} from "@/lib/baseline-charts";
import { cn } from "@/lib/utils";
import { defaultTransactionSearchParams } from "@/pages/transactions/search-params";

const SERIES_STROKE = [
  "stroke-foreground",
  "stroke-red-700/75 dark:stroke-red-400/75",
  "stroke-emerald-700/70 dark:stroke-emerald-400/70",
  "stroke-muted-foreground",
  "stroke-amber-800/65 dark:stroke-amber-300/65",
] as const;

function formatEuro(value: number): string {
  return new Intl.NumberFormat("de-DE", {
    style: "currency",
    currency: "EUR",
  }).format(value);
}

function formatShare(share: number): string {
  return new Intl.NumberFormat("de-DE", {
    style: "percent",
    maximumFractionDigits: 0,
  }).format(share);
}

function lastCompleteMonthBounds(now = new Date()): {
  from: string;
  to: string;
  priorFrom: string;
  priorTo: string;
  label: string;
  priorLabel: string;
} {
  const firstOfThisMonth = new Date(now.getFullYear(), now.getMonth(), 1);
  const lastMonthEnd = new Date(firstOfThisMonth.getTime() - 1);
  const lastMonthStart = new Date(
    lastMonthEnd.getFullYear(),
    lastMonthEnd.getMonth(),
    1,
  );
  const priorEnd = new Date(lastMonthStart.getTime() - 1);
  const priorStart = new Date(priorEnd.getFullYear(), priorEnd.getMonth(), 1);
  const fmt = (d: Date) => {
    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    return `${y}-${m}-${day}`;
  };
  const labelOpts: Intl.DateTimeFormatOptions = {
    month: "long",
    year: "numeric",
  };
  return {
    from: fmt(lastMonthStart),
    to: fmt(lastMonthEnd),
    priorFrom: fmt(priorStart),
    priorTo: fmt(priorEnd),
    label: lastMonthEnd.toLocaleDateString(undefined, labelOpts),
    priorLabel: priorEnd.toLocaleDateString(undefined, labelOpts),
  };
}

const InsightsCategoriesPage: React.FC = () => {
  const [monthsWindow, setMonthsWindow] = useState<6 | 12>(12);
  const [expandedSlug, setExpandedSlug] = useState<string | null>(null);
  const [focusedSlug, setFocusedSlug] = useState<string | null>(null);

  const period = useMemo(
    () => completeMonthsWindow(monthsWindow),
    [monthsWindow],
  );
  const monthBounds = useMemo(() => lastCompleteMonthBounds(), []);

  const spendQuery = useBaselineCategorySpend(
    period.from,
    period.to,
    50,
    true,
  );
  const monthlyQuery = useBaselineCategorySpendMonthly(
    period.from,
    period.to,
    5,
    true,
  );
  const currentMonthQuery = useBaselineCategorySpend(
    monthBounds.from,
    monthBounds.to,
    20,
    true,
  );
  const priorMonthQuery = useBaselineCategorySpend(
    monthBounds.priorFrom,
    monthBounds.priorTo,
    20,
    true,
  );

  const shareRows = useMemo(
    () => buildCategoryShareRows(spendQuery.data?.data ?? []),
    [spendQuery.data],
  );
  const totalSpend = shareRows.reduce((sum, row) => sum + row.total, 0);
  const movers = useMemo(
    () =>
      buildCategoryMovers(
        currentMonthQuery.data?.data ?? [],
        priorMonthQuery.data?.data ?? [],
        5,
      ),
    [currentMonthQuery.data, priorMonthQuery.data],
  );
  const dominant = shareRows[0] ?? null;

  const monthlyModel = useMemo(
    () => buildCategoryMonthlyChartModel(monthlyQuery.data?.data ?? []),
    [monthlyQuery.data],
  );
  const chartModel = useMemo(() => {
    if (!focusedSlug) {
      return monthlyModel;
    }
    const idx = monthlyModel.series.findIndex(
      (s) => s.categorySlug === focusedSlug,
    );
    if (idx < 0) {
      return monthlyModel;
    }
    const series = [monthlyModel.series[idx]!];
    return {
      months: monthlyModel.months,
      series,
      points: monthlyModel.points.map((p) => ({
        date: p.date,
        values: [p.values[idx] ?? 0],
      })),
    };
  }, [monthlyModel, focusedSlug]);
  const chartLayout = useMemo(
    () => buildMultiSeriesChartLayout(chartModel.points),
    [chartModel.points],
  );

  function toggleCategory(slug: string) {
    const open = expandedSlug === slug;
    setExpandedSlug(open ? null : slug);
    setFocusedSlug(open ? null : slug);
  }

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 pb-10">
        <header className="space-y-3">
          <div className="flex flex-wrap items-end justify-between gap-3">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <p className="text-xs uppercase tracking-wide text-muted-foreground">
                  Classified spend
                </p>
                <AskAboutThis
                  prompt="Which categories dominate spending lately?"
                  context={{
                    route: "/insights/categories",
                    from: period.from,
                    to: period.to,
                    months: monthsWindow,
                  }}
                  className="-my-1 h-auto px-0 py-0"
                />
              </div>
              <p className="text-3xl font-semibold tracking-tight tabular-nums">
                {spendQuery.isLoading ? "…" : formatEuro(totalSpend)}
              </p>
              <p className="text-sm text-muted-foreground">
                {period.from} → {period.to}
                {dominant
                  ? ` · ${dominant.categoryName} ${formatShare(dominant.share)}`
                  : null}
              </p>
            </div>
            <div className="flex gap-1 rounded-lg border p-1">
              {([6, 12] as const).map((n) => (
                <Button
                  key={n}
                  type="button"
                  size="sm"
                  variant={monthsWindow === n ? "default" : "ghost"}
                  className="h-7 px-2.5"
                  onClick={() => setMonthsWindow(n)}
                >
                  {n} mo
                </Button>
              ))}
            </div>
          </div>
        </header>

        <section className="space-y-3">
          <div className="space-y-1">
            <h2 className="text-sm font-semibold tracking-tight">
              Development
            </h2>
            <p className="text-sm text-muted-foreground">
              Top categories by month
              {focusedSlug
                ? " — focused on the expanded category."
                : ". Expand a category below to focus the chart."}
            </p>
          </div>
          {monthlyQuery.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading trend…</p>
          ) : !chartLayout || chartModel.series.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Not enough classified spend for a monthly trend yet.
            </p>
          ) : (
            <div className="space-y-2">
              <svg
                viewBox={`0 0 ${chartLayout.width} ${chartLayout.height}`}
                className="w-full text-foreground"
                style={{
                  aspectRatio: `${chartLayout.width} / ${chartLayout.height}`,
                }}
                role="img"
                aria-label="Monthly spend by top categories"
              >
                {chartLayout.moneyLabels.map((label) => (
                  <g key={`money-${label.value}`}>
                    <line
                      x1={chartLayout.padX}
                      x2={chartLayout.width - chartLayout.padX}
                      y1={label.y}
                      y2={label.y}
                      className="stroke-border/60"
                      strokeWidth={1}
                      strokeDasharray={label.value === 0 ? "4 4" : undefined}
                    />
                    <text
                      x={chartLayout.padX - 8}
                      y={label.y + 3}
                      textAnchor="end"
                      className="fill-muted-foreground"
                      fontSize={11}
                    >
                      {label.text}
                    </text>
                  </g>
                ))}
                {chartLayout.paths.map((path, i) => (
                  <path
                    key={chartModel.series[i]?.categorySlug ?? i}
                    d={path}
                    className={SERIES_STROKE[i % SERIES_STROKE.length]}
                    fill="none"
                    strokeWidth={2}
                    strokeLinejoin="round"
                    strokeLinecap="round"
                    strokeDasharray={i === 0 ? undefined : "5 4"}
                  />
                ))}
                {chartLayout.labelIndexes.map((i) => {
                  const date = chartModel.months[i]!;
                  const x = chartLayout.xs[i]!;
                  return (
                    <g key={date}>
                      <line
                        x1={x}
                        x2={x}
                        y1={chartLayout.padTop + chartLayout.innerH}
                        y2={chartLayout.padTop + chartLayout.innerH + 4}
                        className="stroke-muted-foreground"
                        strokeWidth={1}
                      />
                      <text
                        x={x}
                        y={chartLayout.height - 12}
                        textAnchor={chartLabelAnchor(
                          i,
                          chartModel.months.length,
                        )}
                        className="fill-muted-foreground"
                        fontSize={11}
                      >
                        {formatChartDate(date)}
                      </text>
                    </g>
                  );
                })}
              </svg>
              <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                {chartModel.series.map((s, i) => (
                  <li key={s.categorySlug} className="flex items-center gap-1.5">
                    <span
                      className={cn(
                        "inline-block h-0.5 w-3",
                        i === 0
                          ? "bg-foreground"
                          : i === 1
                            ? "border-t-2 border-dashed border-red-700/75 dark:border-red-400/75"
                            : i === 2
                              ? "border-t-2 border-dashed border-emerald-700/70 dark:border-emerald-400/70"
                              : "bg-muted-foreground",
                      )}
                    />
                    {s.categoryName}
                    <span className="tabular-nums">
                      ({formatChartMoney(s.periodTotal)})
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </section>

        {movers.length > 0 ? (
          <section className="space-y-3">
            <div className="space-y-1">
              <h2 className="text-sm font-semibold tracking-tight">
                Biggest movers
              </h2>
              <p className="text-sm text-muted-foreground">
                {monthBounds.label} versus {monthBounds.priorLabel}.
              </p>
            </div>
            <ul className="divide-y border-y">
              {movers.map((mover) => {
                const up = mover.delta > 0;
                return (
                  <li
                    key={mover.categorySlug}
                    className="flex items-center justify-between gap-3 py-2.5 text-sm"
                  >
                    <div className="min-w-0">
                      <p className="truncate font-medium">
                        {mover.categoryName}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {formatCompactMoney(mover.prior)} →{" "}
                        {formatCompactMoney(mover.current)}
                      </p>
                    </div>
                    <span
                      className={cn(
                        "shrink-0 tabular-nums",
                        up
                          ? "text-red-700 dark:text-red-400"
                          : "text-emerald-700 dark:text-emerald-400",
                      )}
                    >
                      {up ? "+" : "−"}
                      {formatCompactMoney(Math.abs(mover.delta))}
                    </span>
                  </li>
                );
              })}
            </ul>
            <Link
              to="/insights/months/$yyyyMm"
              params={{
                yyyyMm: yyyyMmFromMonthStart(monthBounds.from),
              }}
              className="text-sm text-foreground underline-offset-4 hover:underline"
            >
              Open {formatMonthHeadline(monthBounds.from)}
            </Link>
          </section>
        ) : null}

        <section className="space-y-3">
          <div className="space-y-1">
            <h2 className="text-sm font-semibold tracking-tight">
              Spend mix
            </h2>
            <p className="text-sm text-muted-foreground">
              Expand a category for top merchants and to focus the chart.
            </p>
          </div>
          {spendQuery.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading categories…</p>
          ) : shareRows.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No classified category spend in this window yet.
            </p>
          ) : (
            <ul className="divide-y border-y">
              {shareRows.map((row) => {
                const open = expandedSlug === row.categorySlug;
                const width = Math.max(4, Math.round(row.share * 100));
                return (
                  <li key={row.categorySlug} className="py-2.5">
                    <button
                      type="button"
                      className="flex w-full items-start gap-2 text-left"
                      aria-expanded={open}
                      onClick={() => toggleCategory(row.categorySlug)}
                    >
                      <ChevronRight
                        className={cn(
                          "mt-0.5 size-4 shrink-0 text-muted-foreground transition-transform",
                          open && "rotate-90",
                        )}
                      />
                      <div className="min-w-0 flex-1 space-y-1.5">
                        <div className="flex items-baseline justify-between gap-3 text-sm">
                          <span className="truncate font-medium">
                            {row.categoryName}
                          </span>
                          <span className="shrink-0 tabular-nums text-muted-foreground">
                            {formatEuro(row.total)}
                            <span className="ml-2 text-xs">
                              {formatShare(row.share)}
                            </span>
                          </span>
                        </div>
                        <div className="h-1.5 w-full bg-muted">
                          <div
                            className="h-full bg-foreground/55"
                            style={{ width: `${width}%` }}
                          />
                        </div>
                      </div>
                    </button>
                    {open ? (
                      <CategoryMerchantsPanel
                        from={period.from}
                        to={period.to}
                        categorySlug={row.categorySlug}
                        categoryName={row.categoryName}
                      />
                    ) : null}
                  </li>
                );
              })}
            </ul>
          )}
        </section>
      </div>
    </div>
  );
};

const CategoryMerchantsPanel: React.FC<{
  from: string;
  to: string;
  categorySlug: string;
  categoryName: string;
}> = ({ from, to, categorySlug, categoryName }) => {
  const merchantsQuery = useBaselineCategoryMerchants(
    from,
    to,
    categorySlug,
    8,
    true,
  );
  const merchants = merchantsQuery.data?.data ?? [];

  return (
    <div className="ml-6 mt-3 space-y-2 border-l pl-4">
      {merchantsQuery.isLoading ? (
        <p className="text-sm text-muted-foreground">Loading merchants…</p>
      ) : merchants.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No merchants in this category for the window.
        </p>
      ) : (
        <ul className="space-y-1.5">
          {merchants.map((row) => (
            <li
              key={row.merchant}
              className="flex items-baseline justify-between gap-3 text-sm"
            >
              <span className="min-w-0 truncate text-muted-foreground">
                {row.merchant}
              </span>
              <span className="shrink-0 tabular-nums">
                {formatEuro(Number.parseFloat(row.total) || 0)}
              </span>
            </li>
          ))}
        </ul>
      )}
      <Link
        to="/transactions"
        search={{
          ...defaultTransactionSearchParams,
          from,
          to,
          q: categoryName,
        }}
        className="inline-block text-xs text-foreground underline-offset-4 hover:underline"
      >
        See transactions
      </Link>
    </div>
  );
};

export default InsightsCategoriesPage;
