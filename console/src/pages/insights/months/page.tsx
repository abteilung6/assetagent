import type React from "react";
import { useMemo, useState } from "react";
import { Link, useNavigate } from "@tanstack/react-router";

import { AskAboutThis } from "@/components/chat/ask-about-this";
import { Button } from "@/components/ui/button";
import {
  isBaselineMissing,
  useBaselineCategorySpend,
  useBaselineMonthlyCashflow,
  useCurrentBaseline,
} from "@/hooks/use-baseline";
import {
  buildDualSeriesChartLayout,
  chartLabelAnchor,
  formatChartDate,
  formatChartMoney,
} from "@/lib/balance-chart";
import {
  buildExpenseDevelopmentCallouts,
  buildTypicalMonthLevels,
  detectUnusualMonth,
  formatCompactMoney,
  formatMonthHeadline,
  formatMonthLabel,
  yyyyMmFromMonthStart,
  type MonthlyCashflowPoint,
} from "@/lib/baseline-charts";
import { cn } from "@/lib/utils";
import { defaultTransactionSearchParams } from "@/pages/transactions/search-params";

function toPoints(
  rows: { month_start: string; income: string; expenses: string; net: string }[],
): MonthlyCashflowPoint[] {
  return rows.map((row) => ({
    monthStart: row.month_start.slice(0, 10),
    income: Number.parseFloat(row.income) || 0,
    expenses: Number.parseFloat(row.expenses) || 0,
    net: Number.parseFloat(row.net) || 0,
  }));
}

function lastCompleteMonthBounds(now = new Date()): {
  from: string;
  to: string;
  yyyyMm: string;
  label: string;
} {
  const firstOfThisMonth = new Date(now.getFullYear(), now.getMonth(), 1);
  const lastMonthEnd = new Date(firstOfThisMonth.getTime() - 1);
  const yyyy = lastMonthEnd.getFullYear();
  const mm = String(lastMonthEnd.getMonth() + 1).padStart(2, "0");
  const lastDay = String(lastMonthEnd.getDate()).padStart(2, "0");
  const yyyyMm = `${yyyy}-${mm}`;
  return {
    from: `${yyyyMm}-01`,
    to: `${yyyyMm}-${lastDay}`,
    yyyyMm,
    label: lastMonthEnd.toLocaleDateString(undefined, {
      month: "long",
      year: "numeric",
    }),
  };
}

function formatEuro(value: number): string {
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: "EUR",
  }).format(value);
}

const BaselineHistoryPage: React.FC = () => {
  const navigate = useNavigate();
  const baselineQuery = useCurrentBaseline();
  const [monthsWindow, setMonthsWindow] = useState<3 | 6 | 12>(6);
  const [hoveredTrendIndex, setHoveredTrendIndex] = useState<number | null>(
    null,
  );
  const stripQuery = useBaselineMonthlyCashflow(6);
  const trendQuery = useBaselineMonthlyCashflow(monthsWindow);

  const missing =
    baselineQuery.isError && isBaselineMissing(baselineQuery.error);
  const baseline = baselineQuery.data;

  const driverWindow = useMemo(() => lastCompleteMonthBounds(), []);
  const driversQuery = useBaselineCategorySpend(
    driverWindow.from,
    driverWindow.to,
    6,
    true,
  );

  const stripMonths = toPoints(stripQuery.data?.data ?? []);
  const trendMonths = toPoints(trendQuery.data?.data ?? []);
  const baselineMonth = baseline?.period_from.slice(0, 10) ?? "";
  const stripInsight = detectUnusualMonth(
    stripMonths,
    baselineMonth || undefined,
  );
  const trendInsight = detectUnusualMonth(
    trendMonths,
    baselineMonth || undefined,
  );
  const maxExpense = Math.max(...stripMonths.map((m) => m.expenses), 1);

  const typical = baseline
    ? buildTypicalMonthLevels({
        income: Number.parseFloat(baseline.regular_monthly_income) || 0,
        fixed: Number.parseFloat(baseline.monthly_fixed_costs) || 0,
        irregular: Number.parseFloat(baseline.monthly_irregular_costs) || 0,
        variable: Number.parseFloat(baseline.avg_variable_spend) || 0,
      })
    : { income: 0, expenses: 0 };

  const expensiveMonths = useMemo(
    () => buildExpenseDevelopmentCallouts(trendMonths, typical.expenses).high,
    [trendMonths, typical.expenses],
  );

  const dualLayout = buildDualSeriesChartLayout(
    trendMonths.map((m) => ({
      date: m.monthStart,
      primary: m.income,
      secondary: m.expenses,
    })),
    {
      references: baseline
        ? [
            { key: "typical_income", value: typical.income },
            { key: "typical_expenses", value: typical.expenses },
          ]
        : [],
    },
  );

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 pb-10">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">
              What income and expenses looked like recently — bank months, not
              your Cashflow model.
            </p>
            <Link
              to="/baseline"
              className="text-sm text-foreground underline-offset-4 hover:underline"
            >
              Open Cashflow
            </Link>
          </div>
          <AskAboutThis
            prompt="What looks unusual in recent months?"
            context={{ route: "/insights/months" }}
          />
        </div>

        {baselineQuery.isLoading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : missing ? (
          <p className="text-sm text-muted-foreground">
            Calculate a{" "}
            <Link
              to="/baseline"
              className="underline underline-offset-4 hover:text-foreground"
            >
              Cashflow
            </Link>{" "}
            baseline first so we can draw typical guides on this chart.
          </p>
        ) : null}

        <section className="flex flex-col gap-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="space-y-1">
              <h2 className="text-sm font-semibold tracking-tight">
                Income & expenses over time
              </h2>
              <p className="text-sm text-muted-foreground">
                Each month versus your Cashflow norm. Click a month for drivers
                and detail.
              </p>
            </div>
            <div className="flex gap-1 rounded-lg border p-1">
              {([3, 6, 12] as const).map((n) => (
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

          {trendQuery.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : !dualLayout || trendMonths.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Not enough booking history for this window yet.
            </p>
          ) : (
            <div className="space-y-2">
              <svg
                viewBox={`0 0 ${dualLayout.width} ${dualLayout.height}`}
                className="h-52 w-full text-foreground"
                role="img"
                aria-label="Monthly income and expenses over time with Cashflow norm guides"
              >
                {dualLayout.moneyLabels.map((label) => (
                  <g key={`money-${label.value}`}>
                    <line
                      x1={dualLayout.padX}
                      x2={dualLayout.width - dualLayout.padX}
                      y1={label.y}
                      y2={label.y}
                      className="stroke-border/60"
                      strokeWidth={1}
                      strokeDasharray={label.value === 0 ? "4 4" : undefined}
                    />
                    <text
                      x={dualLayout.padX - 8}
                      y={label.y + 3}
                      textAnchor="end"
                      className="fill-muted-foreground"
                      fontSize={11}
                    >
                      {label.text}
                    </text>
                  </g>
                ))}
                {dualLayout.referenceLines.map((ref) => (
                  <g key={ref.key}>
                    <line
                      x1={dualLayout.padX}
                      x2={dualLayout.width - dualLayout.padX}
                      y1={ref.y}
                      y2={ref.y}
                      className={
                        ref.key === "typical_income"
                          ? "stroke-emerald-700/35 dark:stroke-emerald-400/35"
                          : "stroke-red-700/35 dark:stroke-red-400/35"
                      }
                      strokeWidth={1.5}
                      strokeDasharray={
                        ref.key === "typical_expenses" ? "2 4" : "6 4"
                      }
                    />
                  </g>
                ))}
                <path
                  d={dualLayout.primaryPath}
                  className="stroke-emerald-700 dark:stroke-emerald-400"
                  fill="none"
                  strokeWidth={2}
                  strokeLinejoin="round"
                  strokeLinecap="round"
                />
                <path
                  d={dualLayout.secondaryPath}
                  className="stroke-red-700/80 dark:stroke-red-400/80"
                  fill="none"
                  strokeWidth={2}
                  strokeDasharray="5 4"
                  strokeLinejoin="round"
                  strokeLinecap="round"
                />
                {dualLayout.labelIndexes.map((i) => {
                  const point = trendMonths[i]!;
                  const x = dualLayout.xs[i]!;
                  return (
                    <g key={point.monthStart}>
                      <line
                        x1={x}
                        x2={x}
                        y1={dualLayout.padTop + dualLayout.innerH}
                        y2={dualLayout.padTop + dualLayout.innerH + 4}
                        className="stroke-muted-foreground"
                        strokeWidth={1}
                      />
                      <text
                        x={x}
                        y={dualLayout.height - 12}
                        textAnchor={chartLabelAnchor(i, trendMonths.length)}
                        className="fill-muted-foreground"
                        fontSize={11}
                      >
                        {formatChartDate(point.monthStart)}
                      </text>
                    </g>
                  );
                })}
                {trendMonths.map((point, i) => {
                  const x = dualLayout.xs[i]!;
                  const primaryY = dualLayout.primaryYs[i]!;
                  const secondaryY = dualLayout.secondaryYs[i]!;
                  const hovered = hoveredTrendIndex === i;
                  return (
                    <g key={`hit-${point.monthStart}`}>
                      <rect
                        x={x - 14}
                        y={dualLayout.padTop}
                        width={28}
                        height={dualLayout.innerH}
                        className="fill-transparent cursor-pointer"
                        role="link"
                        aria-label={`${formatMonthHeadline(point.monthStart)} details`}
                        onMouseEnter={() => setHoveredTrendIndex(i)}
                        onMouseLeave={() => setHoveredTrendIndex(null)}
                        onFocus={() => setHoveredTrendIndex(i)}
                        onBlur={() => setHoveredTrendIndex(null)}
                        onClick={() => {
                          void navigate({
                            to: "/insights/months/$yyyyMm",
                            params: {
                              yyyyMm: yyyyMmFromMonthStart(point.monthStart),
                            },
                            search: { tab: "activity" },
                          });
                        }}
                      />
                      <circle
                        cx={x}
                        cy={primaryY}
                        r={hovered ? 4 : 2.5}
                        className="fill-emerald-700 pointer-events-none dark:fill-emerald-400"
                      />
                      <circle
                        cx={x}
                        cy={secondaryY}
                        r={hovered ? 4 : 2.5}
                        className="fill-red-700/80 pointer-events-none dark:fill-red-400/80"
                      />
                      {hovered ? (
                        <text
                          x={x}
                          y={Math.min(primaryY, secondaryY) - 10}
                          textAnchor="middle"
                          className="fill-foreground pointer-events-none"
                          fontSize={11}
                        >
                          Net {formatChartMoney(point.net)}
                        </text>
                      ) : null}
                    </g>
                  );
                })}
              </svg>
              <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <li className="flex items-center gap-1.5">
                  <span className="inline-block h-0.5 w-3 bg-emerald-700 dark:bg-emerald-400" />
                  Income · each month
                </li>
                <li className="flex items-center gap-1.5">
                  <span className="inline-block h-0.5 w-3 border-t-2 border-dashed border-red-700/80 dark:border-red-400/80" />
                  Expenses · each month
                </li>
                {baseline ? (
                  <>
                    <li className="flex items-center gap-1.5">
                      <span className="inline-block h-px w-3 border-t border-dashed border-emerald-700/40 dark:border-emerald-400/40" />
                      Income · typical ({formatChartMoney(typical.income)})
                    </li>
                    <li className="flex items-center gap-1.5">
                      <span className="inline-block h-px w-3 border-t border-dotted border-red-700/40 dark:border-red-400/40" />
                      Expenses · typical ({formatChartMoney(typical.expenses)})
                    </li>
                  </>
                ) : null}
              </ul>
            </div>
          )}

          {expensiveMonths.length > 0 ? (
            <ul className="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
              {expensiveMonths.slice(0, 3).map((m) => (
                <li key={`expensive-${m.monthStart}`}>
                  <Link
                    to="/insights/months/$yyyyMm"
                    params={{
                      yyyyMm: yyyyMmFromMonthStart(m.monthStart),
                    }}
                    search={{ tab: "activity" }}
                    className="text-foreground underline-offset-4 hover:underline"
                  >
                    {formatMonthHeadline(m.monthStart)}
                  </Link>{" "}
                  booked well above the Cashflow cost norm (
                  {formatChartMoney(m.expenses)}) — open Activity for top
                  payments.
                </li>
              ))}
            </ul>
          ) : null}

          {trendInsight.unusual && trendInsight.message ? (
            <p className="text-sm text-amber-800 dark:text-amber-200">
              {trendInsight.message}{" "}
              {trendInsight.monthStart ? (
                <Link
                  to="/insights/months/$yyyyMm"
                  params={{
                    yyyyMm: yyyyMmFromMonthStart(trendInsight.monthStart),
                  }}
                  search={{ tab: "activity" }}
                  className="underline underline-offset-4 hover:text-foreground"
                >
                  Open Activity
                </Link>
              ) : null}
            </p>
          ) : null}
        </section>

        <UnusualMonthsStrip
          months={stripMonths}
          baselineMonth={baselineMonth}
          insight={stripInsight}
          maxExpense={maxExpense}
          loading={stripQuery.isLoading}
        />

        <section className="space-y-3">
          <div className="space-y-1">
            <h2 className="text-sm font-semibold tracking-tight">
              Top cost drivers
            </h2>
            <p className="text-sm text-muted-foreground">
              Categories in {driverWindow.label}. Transfer-aware booking dates.
            </p>
          </div>
          {driversQuery.isLoading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : (driversQuery.data?.data.length ?? 0) === 0 ? (
            <p className="text-sm text-muted-foreground">
              No category spend in that month yet.
            </p>
          ) : (
            <ul className="divide-y border-y">
              {(driversQuery.data?.data ?? []).map((row) => (
                <li key={row.category_slug}>
                  <Link
                    to="/transactions"
                    search={{
                      ...defaultTransactionSearchParams,
                      from: driverWindow.from,
                      to: driverWindow.to,
                      q: row.category_name,
                    }}
                    className="flex items-center justify-between gap-3 py-2.5 text-sm underline-offset-4 hover:underline"
                  >
                    <span className="min-w-0 truncate">{row.category_name}</span>
                    <span className="shrink-0 tabular-nums text-muted-foreground">
                      {formatEuro(Number.parseFloat(row.total) || 0)}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
          <Link
            to="/insights/months/$yyyyMm"
            params={{ yyyyMm: driverWindow.yyyyMm }}
            search={{ tab: "activity" }}
            className="text-sm text-foreground underline-offset-4 hover:underline"
          >
            Open {driverWindow.label}
          </Link>
        </section>
      </div>
    </div>
  );
};

const UnusualMonthsStrip: React.FC<{
  months: MonthlyCashflowPoint[];
  baselineMonth: string;
  insight: ReturnType<typeof detectUnusualMonth>;
  maxExpense: number;
  loading: boolean;
}> = ({ months, baselineMonth, insight, maxExpense, loading }) => {
  return (
    <div className="space-y-3">
      <div className="space-y-1">
        <h2 className="text-sm font-semibold tracking-tight">Recent months</h2>
        <p className="text-sm text-muted-foreground">
          Expense height by month. Click a month for cost and income drivers.
        </p>
      </div>
      {loading ? (
        <p className="text-sm text-muted-foreground">Loading months…</p>
      ) : months.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          Not enough booking history for a monthly strip yet.
        </p>
      ) : (
        <div className="flex items-end gap-2 border-b pb-2">
          {months.map((m) => {
            const heightPct = Math.max(8, (m.expenses / maxExpense) * 100);
            const isBaseline =
              baselineMonth !== "" &&
              m.monthStart.slice(0, 7) === baselineMonth.slice(0, 7);
            const isUnusual =
              insight.unusual && insight.monthStart === m.monthStart;
            return (
              <Link
                key={m.monthStart}
                to="/insights/months/$yyyyMm"
                params={{ yyyyMm: yyyyMmFromMonthStart(m.monthStart) }}
                search={{ tab: "activity" }}
                className="group flex min-w-0 flex-1 flex-col items-center gap-1.5 underline-offset-4 hover:underline"
                title={`${formatMonthLabel(m.monthStart)} · expenses ${formatEuro(m.expenses)}`}
              >
                <span className="text-[10px] tabular-nums text-muted-foreground group-hover:text-foreground">
                  {formatCompactMoney(m.expenses)}
                </span>
                <div className="flex h-24 w-full items-end justify-center">
                  <span
                    className={cn(
                      "w-full max-w-[2.5rem] rounded-t-sm transition-colors",
                      isUnusual
                        ? "bg-amber-600 dark:bg-amber-500"
                        : isBaseline
                          ? "bg-foreground"
                          : "bg-foreground/25 group-hover:bg-foreground/45",
                    )}
                    style={{ height: `${heightPct}%` }}
                  />
                </div>
                <span
                  className={cn(
                    "text-[10px] tabular-nums",
                    isBaseline
                      ? "font-medium text-foreground"
                      : "text-muted-foreground",
                  )}
                >
                  {formatMonthLabel(m.monthStart)}
                </span>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
};

export default BaselineHistoryPage;
