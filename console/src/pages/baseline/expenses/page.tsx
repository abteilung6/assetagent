import type React from "react";
import { useMemo, useState } from "react";
import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";

import type { RecurringSeries } from "@/api/types.gen";
import { AskAboutThis } from "@/components/chat/ask-about-this";
import { buttonVariants } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  isBaselineMissing,
  useBaselineMonthlyCashflow,
  useBaselineOneOffImpact,
  useCurrentBaseline,
} from "@/hooks/use-baseline";
import { useRecurringSeries } from "@/hooks/use-recurring";
import {
  useRecurringMembers,
  useUncertainRecurring,
} from "@/hooks/use-recurring-uncertain";
import {
  buildDualSeriesChartLayout,
  chartLabelAnchor,
  formatChartDate,
  formatChartMoney,
} from "@/lib/balance-chart";
import {
  buildExpenseDevelopmentCallouts,
  buildExpenseDevelopmentRows,
  buildTypicalMonthLevels,
  endOfMonthISO,
  formatExpenseOneOffLine,
  formatMonthHeadline,
  formatSignedPercent,
  monthlyEquivalentAmount,
  resolveEvidenceSeries,
  yyyyMmFromMonthStart,
  type ExpenseDevelopmentRow,
  type MonthlyCashflowPoint,
} from "@/lib/baseline-charts";
import { cn } from "@/lib/utils";
import { useTransactions } from "@/hooks/use-transactions";

const BaselineExpensesPage: React.FC = () => {
  const baselineQuery = useCurrentBaseline();
  const monthsQuery = useBaselineMonthlyCashflow(12);
  const recurringQuery = useRecurringSeries();
  const uncertainQuery = useUncertainRecurring();
  const missing = isBaselineMissing(baselineQuery.error);
  const baseline = baselineQuery.data;

  const periodFrom = baseline?.period_from?.slice(0, 10) ?? "";
  const periodTo = baseline?.period_to?.slice(0, 10) ?? "";
  const oneOffQuery = useBaselineOneOffImpact(
    periodFrom,
    periodTo,
    Boolean(baseline),
  );

  const fixedMetric = baseline?.metrics.find(
    (m) => m.key === "monthly_fixed_costs",
  );
  const irregularMetric = baseline?.metrics.find(
    (m) => m.key === "monthly_irregular_costs",
  );
  const variableMetric = baseline?.metrics.find(
    (m) => m.key === "avg_variable_spend",
  );

  const fixed = baseline
    ? Number.parseFloat(baseline.monthly_fixed_costs) || 0
    : 0;
  const irregular = baseline
    ? Number.parseFloat(baseline.monthly_irregular_costs) || 0
    : 0;
  const variable = baseline
    ? Number.parseFloat(baseline.avg_variable_spend) || 0
    : 0;
  const typicalExpenses = buildTypicalMonthLevels({
    income: 0,
    fixed,
    irregular,
    variable,
  }).expenses;

  const months: MonthlyCashflowPoint[] = useMemo(
    () =>
      (monthsQuery.data?.data ?? []).map((row) => ({
        monthStart: row.month_start.slice(0, 10),
        income: Number.parseFloat(row.income) || 0,
        expenses: Number.parseFloat(row.expenses) || 0,
        net: Number.parseFloat(row.net) || 0,
      })),
    [monthsQuery.data],
  );

  const fixedDrivers = useMemo(() => {
    const all = recurringQuery.data?.data ?? [];
    const byId = new Map(all.map((s) => [s.id, s]));
    if (fixedMetric) {
      return resolveEvidenceSeries(fixedMetric.evidence_ids, byId);
    }
    // Match baseline.go: non-income monthly series → fixed
    return all.filter(
      (s) => isExpenseSeries(s) && !isIrregularCadence(s),
    );
  }, [recurringQuery.data, fixedMetric]);

  const irregularDrivers = useMemo(() => {
    const all = recurringQuery.data?.data ?? [];
    const byId = new Map(all.map((s) => [s.id, s]));
    if (irregularMetric) {
      return resolveEvidenceSeries(irregularMetric.evidence_ids, byId);
    }
    // Match baseline.go: quarterly/yearly expense series → irregular
    return all.filter((s) => isExpenseSeries(s) && isIrregularCadence(s));
  }, [recurringQuery.data, irregularMetric]);

  const uncertainCostCount = useMemo(() => {
    const items = uncertainQuery.data?.data ?? [];
    return items.filter(
      (s) => s.kind === "fixed" || s.kind === "variable_regular",
    ).length;
  }, [uncertainQuery.data]);

  const structureLayers = useMemo(() => {
    const total = typicalExpenses > 0 ? typicalExpenses : 1;
    return [
      {
        key: "fixed" as const,
        label: "Fixed",
        amount: fixed,
        share: fixed / total,
        definition:
          "Recurring obligations that are hard to cut short-term (rent, insurance, loans).",
        confidence: fixedMetric?.confidence,
      },
      {
        key: "irregular" as const,
        label: "Irregular",
        amount: irregular,
        share: irregular / total,
        definition:
          "Known recurring costs that are not monthly — shown as a monthly equivalent.",
        confidence: irregularMetric?.confidence,
      },
      {
        key: "variable" as const,
        label: "Variable",
        amount: variable,
        share: variable / total,
        definition:
          "Average remaining spend after fixed & irregular — not a list of shops.",
        confidence: variableMetric?.confidence,
      },
    ];
  }, [
    fixed,
    irregular,
    variable,
    typicalExpenses,
    fixedMetric?.confidence,
    irregularMetric?.confidence,
    variableMetric?.confidence,
  ]);

  const chartLayout = useMemo(() => {
    if (months.length === 0) {
      return null;
    }
    return buildDualSeriesChartLayout(
      months.map((m) => ({
        date: m.monthStart,
        primary: m.expenses,
        secondary: typicalExpenses,
      })),
      {
        height: 220,
        padX: 8,
        references: [{ key: "expense_norm", value: typicalExpenses }],
      },
    );
  }, [months, typicalExpenses]);

  const callouts = useMemo(
    () => buildExpenseDevelopmentCallouts(months, typicalExpenses),
    [months, typicalExpenses],
  );

  const developmentRows = useMemo(
    () => buildExpenseDevelopmentRows(months, typicalExpenses),
    [months, typicalExpenses],
  );

  const yearTotal = months.reduce((sum, m) => sum + m.expenses, 0);
  const oneOffLine = formatExpenseOneOffLine(
    oneOffQuery.data?.count ?? 0,
    Number.parseFloat(oneOffQuery.data?.expense_total ?? "0") || 0,
  );
  const loading =
    baselineQuery.isLoading ||
    monthsQuery.isLoading ||
    recurringQuery.isLoading;

  const periodLabel = baseline
    ? `${formatDate(baseline.period_from)} – ${formatDate(baseline.period_to)}`
    : "";

  const costConfidence =
    fixedMetric?.confidence ??
    irregularMetric?.confidence ??
    variableMetric?.confidence ??
    baseline?.confidence;

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 pb-10">
        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : missing || !baseline ? (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Calculate a Cashflow baseline first to see your typical expenses
              and cost drivers.
            </p>
            <Link
              to="/baseline"
              className={cn(buttonVariants({ variant: "secondary" }))}
            >
              Open Cashflow
            </Link>
          </div>
        ) : (
          <>
            <header className="space-y-4">
              <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                <span>{periodLabel}</span>
                <span aria-hidden="true">·</span>
                <span>
                  {confidenceLabel(costConfidence)} confidence
                </span>
                <span aria-hidden="true">·</span>
                <span
                  className={
                    baseline.status === "confirmed"
                      ? "text-foreground"
                      : undefined
                  }
                >
                  {baseline.status === "confirmed" ? "Confirmed" : "Draft"}
                </span>
                <AskAboutThis
                  prompt="Explain my typical expenses"
                  context={{
                    route: "/baseline/expenses",
                    baseline_id: baseline.id,
                    from: periodFrom,
                    to: periodTo,
                  }}
                  className="-my-1 ml-auto h-auto px-0 py-0"
                />
              </div>
              <div className="space-y-1">
                <p className="text-xs text-muted-foreground">
                  Typical monthly expenses
                </p>
                <p className="text-4xl font-semibold tracking-tight tabular-nums">
                  {formatEuro(typicalExpenses)}
                </p>
                {oneOffLine ? (
                  <p className="text-sm text-muted-foreground">{oneOffLine}</p>
                ) : null}
              </div>
              <ul className="flex flex-wrap gap-x-5 gap-y-2 text-sm">
                {structureLayers.map((layer) => (
                  <li key={layer.key} className="min-w-[6.5rem]">
                    <p className="text-xs text-muted-foreground">{layer.label}</p>
                    <p className="font-medium tabular-nums tracking-tight">
                      {formatEuro(layer.amount)}
                    </p>
                  </li>
                ))}
              </ul>
            </header>

            <section className="space-y-3">
              <div className="space-y-1">
                <h2 className="text-sm font-semibold tracking-tight">
                  Structure
                </h2>
                <p className="text-sm text-muted-foreground">
                  How your cost of living is composed in the baseline.
                </p>
              </div>
              <ul className="space-y-3">
                {structureLayers.map((layer) => (
                  <li key={layer.key} className="space-y-1.5">
                    <div className="flex items-baseline justify-between gap-3">
                      <p className="text-sm font-medium">{layer.label}</p>
                      <div className="text-right">
                        <p className="text-sm font-medium tabular-nums tracking-tight">
                          {formatEuro(layer.amount)}
                        </p>
                        <p className="text-[11px] tabular-nums text-muted-foreground">
                          {formatShare(layer.share)} of typical
                        </p>
                      </div>
                    </div>
                    <div className="h-1.5 overflow-hidden rounded-full bg-muted">
                      <div
                        className={cn(
                          "h-full rounded-full",
                          layer.key === "fixed" &&
                            "bg-stone-700 dark:bg-stone-300",
                          layer.key === "irregular" &&
                            "bg-stone-500 dark:bg-stone-400",
                          layer.key === "variable" &&
                            "bg-stone-400 dark:bg-stone-500",
                        )}
                        style={{
                          width: `${Math.min(100, Math.max(0, layer.share * 100))}%`,
                        }}
                      />
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {layer.definition}
                      {layer.key === "variable" ? (
                        <>
                          {" "}
                          <Link
                            to="/insights/months"
                            className="text-foreground underline-offset-4 hover:underline"
                          >
                            Browse categories in Insights
                          </Link>
                          .
                        </>
                      ) : null}
                    </p>
                  </li>
                ))}
              </ul>
            </section>

            <section className="space-y-5">
              <div className="space-y-1">
                <h2 className="text-sm font-semibold tracking-tight">
                  Drivers
                </h2>
                <p className="text-sm text-muted-foreground">
                  Recurring costs behind the norm — not every debit in the
                  window.
                </p>
              </div>

              <DriverGroup
                title="Fixed"
                empty="No fixed cost series linked to this baseline yet."
                series={fixedDrivers}
              />
              <DriverGroup
                title="Irregular"
                empty={
                  irregular === 0
                    ? "No quarterly or yearly recurring costs in this baseline — nothing in the irregular bucket."
                    : "No irregular cost series linked to this baseline yet."
                }
                series={irregularDrivers}
              />

              <div className="space-y-1">
                <p className="text-sm font-medium">Variable</p>
                <p className="text-sm text-muted-foreground">
                  Variable is a residual band ({formatEuro(variable)} / month),
                  not a merchant list.{" "}
                  <Link
                    to="/insights/months"
                    className="text-foreground underline-offset-4 hover:underline"
                  >
                    Browse categories in Insights
                  </Link>
                  .
                </p>
              </div>

              <p className="text-xs text-muted-foreground">
                One-offs don’t raise your Expenses baseline.
              </p>
              {uncertainCostCount > 0 ? (
                <p className="text-sm text-amber-800 dark:text-amber-200">
                  {uncertainCostCount === 1
                    ? "1 cost series needs review"
                    : `${uncertainCostCount} cost series need review`}{" "}
                  — confirm them so the norm stays trustworthy.{" "}
                  <Link
                    to="/review"
                    search={{ tab: "recurring" }}
                    className="underline underline-offset-4 hover:text-foreground"
                  >
                    Open Needs review
                  </Link>
                </p>
              ) : null}
            </section>

            <section className="space-y-3">
              <div className="space-y-1">
                <h2 className="text-sm font-semibold tracking-tight">
                  Development
                </h2>
                <p className="text-sm text-muted-foreground">
                  Booked expenses by month versus your Cashflow cost norm
                  {months.length > 0
                    ? ` · last ${months.length} months total ${formatEuro(yearTotal)}`
                    : null}
                  .
                </p>
              </div>
              {chartLayout && months.length > 0 ? (
                <div className="space-y-5">
                  <div className="space-y-2">
                    <svg
                      viewBox={`0 0 ${chartLayout.width} ${chartLayout.height}`}
                      className="w-full text-foreground"
                      style={{
                        aspectRatio: `${chartLayout.width} / ${chartLayout.height}`,
                      }}
                      role="img"
                      aria-label="Monthly expenses versus Cashflow cost norm"
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
                            strokeDasharray={
                              label.value === 0 ? "4 4" : undefined
                            }
                          />
                          <text
                            x={chartLayout.width - chartLayout.padX}
                            y={label.y - 4}
                            textAnchor="end"
                            className="fill-muted-foreground"
                            fontSize={11}
                          >
                            {label.text}
                          </text>
                        </g>
                      ))}
                      {chartLayout.referenceLines.map((ref) => (
                        <line
                          key={ref.key}
                          x1={chartLayout.padX}
                          x2={chartLayout.width - chartLayout.padX}
                          y1={ref.y}
                          y2={ref.y}
                          className="stroke-stone-600/45 dark:stroke-stone-300/45"
                          strokeWidth={1.5}
                          strokeDasharray="6 4"
                        />
                      ))}
                      <path
                        d={chartLayout.primaryPath}
                        className="stroke-stone-700 dark:stroke-stone-300"
                        fill="none"
                        strokeWidth={2}
                        strokeLinejoin="round"
                        strokeLinecap="round"
                      />
                      {chartLayout.labelIndexes.map((i) => {
                        const point = months[i]!;
                        const x = chartLayout.xs[i]!;
                        return (
                          <g key={point.monthStart}>
                            <text
                              x={x}
                              y={chartLayout.height - 12}
                              textAnchor={chartLabelAnchor(i, months.length)}
                              className="fill-muted-foreground"
                              fontSize={11}
                            >
                              {formatChartDate(point.monthStart)}
                            </text>
                          </g>
                        );
                      })}
                      {months.map((point, i) => {
                        const x = chartLayout.xs[i]!;
                        const y = chartLayout.primaryYs[i]!;
                        return (
                          <circle
                            key={point.monthStart}
                            cx={x}
                            cy={y}
                            r={3}
                            className="fill-stone-700 dark:fill-stone-300"
                          >
                            <title>
                              {formatMonthHeadline(point.monthStart)} ·{" "}
                              {formatChartMoney(point.expenses)}
                            </title>
                          </circle>
                        );
                      })}
                    </svg>
                    <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                      <li className="flex items-center gap-1.5">
                        <span className="inline-block h-0.5 w-3 bg-stone-700 dark:bg-stone-300" />
                        Booked expenses
                      </li>
                      <li className="flex items-center gap-1.5">
                        <span className="inline-block h-px w-3 border-t border-dashed border-stone-600/45 dark:border-stone-300/45" />
                        Expenses · Cashflow norm (
                        {formatChartMoney(typicalExpenses)})
                      </li>
                    </ul>
                  </div>

                  <Table>
                    <TableHeader>
                      <TableRow className="hover:bg-transparent">
                        <TableHead className="h-8 pl-0 text-xs">Month</TableHead>
                        <TableHead className="h-8 text-right text-xs">
                          Booked
                        </TableHead>
                        <TableHead className="h-8 text-right text-xs">
                          vs norm
                        </TableHead>
                        <TableHead className="h-8 pr-0 text-right text-xs">
                          vs prior
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {[...developmentRows].reverse().map((row) => (
                        <DevelopmentMonthRow key={row.monthStart} row={row} />
                      ))}
                    </TableBody>
                  </Table>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  Not enough booking history for an expense trend yet.
                </p>
              )}
              {callouts.low.length > 0 || callouts.high.length > 0 ? (
                <ul className="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
                  {callouts.high.slice(0, 2).map((m) => (
                    <li key={`high-${m.monthStart}`}>
                      <Link
                        to="/insights/months/$yyyyMm"
                        params={{
                          yyyyMm: yyyyMmFromMonthStart(m.monthStart),
                        }}
                        className="text-foreground underline-offset-4 hover:underline"
                      >
                        {formatMonthHeadline(m.monthStart)}
                      </Link>{" "}
                      booked well above the cost norm (
                      {formatChartMoney(m.expenses)}
                      ) — check one-offs in Insights.
                    </li>
                  ))}
                  {callouts.low.slice(0, 2).map((m) => (
                    <li key={`low-${m.monthStart}`}>
                      <Link
                        to="/insights/months/$yyyyMm"
                        params={{
                          yyyyMm: yyyyMmFromMonthStart(m.monthStart),
                        }}
                        className="text-foreground underline-offset-4 hover:underline"
                      >
                        {formatMonthHeadline(m.monthStart)}
                      </Link>{" "}
                      booked well below the cost norm (
                      {formatChartMoney(m.expenses)}).
                    </li>
                  ))}
                </ul>
              ) : null}
            </section>
          </>
        )}
      </div>
    </div>
  );
};

const DevelopmentMonthRow: React.FC<{ row: ExpenseDevelopmentRow }> = ({
  row,
}) => {
  const [expanded, setExpanded] = useState(false);
  const monthEnd = endOfMonthISO(row.monthStart);
  const yyyyMm = yyyyMmFromMonthStart(row.monthStart);
  const topQuery = useTransactions(
    {
      limit: 5,
      offset: 0,
      from: row.monthStart.slice(0, 10),
      to: monthEnd,
      sort: "amount",
      order: "asc",
    },
    { enabled: expanded },
  );
  const topExpenses = (topQuery.data?.data ?? [])
    .filter((tx) => Number.parseFloat(tx.amount) < 0)
    .slice(0, 5);

  return (
    <>
      <TableRow
        className={cn(expanded && "border-b-0 hover:bg-transparent")}
        data-state={expanded ? "selected" : undefined}
      >
        <TableCell className="pl-0 py-2.5">
          <button
            type="button"
            className="flex max-w-full items-center gap-1.5 text-left"
            aria-expanded={expanded}
            onClick={() => setExpanded((open) => !open)}
          >
            <ChevronRight
              className={cn(
                "size-3.5 shrink-0 text-muted-foreground/80 transition-transform",
                expanded && "rotate-90",
              )}
              aria-hidden
            />
            <span className="truncate text-sm font-medium">
              {formatMonthHeadline(row.monthStart)}
            </span>
          </button>
        </TableCell>
        <TableCell className="py-2.5 text-right text-sm tabular-nums tracking-tight">
          {formatEuro(row.expenses)}
        </TableCell>
        <TableCell className="py-2.5 text-right">
          {row.vsNorm === null || row.vsNormPct === null ? (
            <span className="text-sm text-muted-foreground">—</span>
          ) : (
            <div className="space-y-0.5">
              <p
                className={cn(
                  "text-sm tabular-nums tracking-tight",
                  expenseDeltaTone(row.vsNorm),
                )}
              >
                {formatSignedEuro(row.vsNorm)}
              </p>
              <p className="text-[11px] tabular-nums text-muted-foreground">
                {formatSignedPercent(row.vsNormPct)}
              </p>
            </div>
          )}
        </TableCell>
        <TableCell className="pr-0 py-2.5 text-right text-sm tabular-nums tracking-tight">
          {row.vsPriorPct === null ? (
            <span className="text-muted-foreground">—</span>
          ) : (
            <span className={expenseDeltaTone(row.vsPriorPct)}>
              {formatSignedPercent(row.vsPriorPct)}
            </span>
          )}
        </TableCell>
      </TableRow>
      {expanded ? (
        <TableRow className="hover:bg-transparent">
          <TableCell colSpan={4} className="pb-3 pl-6 pt-0">
            <div className="space-y-2 border-t border-border/50 pt-3">
              <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
                Top expenses
              </p>
              {topQuery.isLoading ? (
                <p className="text-xs text-muted-foreground">Loading…</p>
              ) : topExpenses.length === 0 ? (
                <p className="text-xs text-muted-foreground">
                  No expense bookings in this month.
                </p>
              ) : (
                <ul className="space-y-2.5">
                  {topExpenses.map((tx) => (
                    <li
                      key={tx.id}
                      className="flex items-baseline justify-between gap-4"
                    >
                      <div className="min-w-0 space-y-0.5">
                        <p className="truncate text-sm leading-snug">
                          {tx.counterparty || tx.purpose || "Payment"}
                        </p>
                        <p className="text-[11px] text-muted-foreground">
                          {formatDate(tx.booking_date)}
                          {tx.purpose &&
                          tx.counterparty &&
                          tx.purpose !== tx.counterparty
                            ? ` · ${tx.purpose}`
                            : null}
                          {tx.one_off ? " · one-off" : null}
                        </p>
                      </div>
                      <span className="shrink-0 text-sm tabular-nums tracking-tight text-muted-foreground">
                        {formatEuro(Number.parseFloat(tx.amount) || 0)}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
              <p className="pt-1 text-xs text-muted-foreground">
                <Link
                  to="/insights/months/$yyyyMm"
                  params={{ yyyyMm }}
                  search={{ tab: "activity" }}
                  className="text-foreground underline-offset-4 hover:underline"
                >
                  Open {formatMonthHeadline(row.monthStart)} in Insights
                </Link>
              </p>
            </div>
          </TableCell>
        </TableRow>
      ) : null}
    </>
  );
};

const DriverGroup: React.FC<{
  title: string;
  empty: string;
  series: RecurringSeries[];
}> = ({ title, empty, series }) => (
  <div className="space-y-2">
    <p className="text-sm font-medium">{title}</p>
    {series.length === 0 ? (
      <p className="text-sm text-muted-foreground">{empty}</p>
    ) : (
      <ul className="space-y-0.5">
        {series.map((item) => (
          <DriverRow key={item.id} series={item} />
        ))}
      </ul>
    )}
  </div>
);

const DriverRow: React.FC<{ series: RecurringSeries }> = ({ series }) => {
  const [expanded, setExpanded] = useState(false);
  const membersQuery = useRecurringMembers(series.id, expanded);
  const members = membersQuery.data?.data ?? [];
  const monthly = monthlyEquivalentAmount(
    Number.parseFloat(series.amount_typical) || 0,
    series.interval,
  );
  const cadence =
    series.interval === "monthly"
      ? "Monthly"
      : series.interval === "quarterly"
        ? "Quarterly"
        : "Yearly";
  const paymentLabel =
    series.member_count === 1
      ? "1 payment"
      : `${series.member_count} payments`;

  return (
    <li className={cn(expanded && "rounded-lg bg-muted/30")}>
      <button
        type="button"
        className={cn(
          "flex w-full items-center gap-3 rounded-lg px-3 py-3.5 text-left transition-colors",
          "hover:bg-muted/45",
          expanded && "hover:bg-transparent",
        )}
        aria-expanded={expanded}
        onClick={() => setExpanded((open) => !open)}
      >
        <ChevronRight
          className={cn(
            "size-3.5 shrink-0 text-muted-foreground/80 transition-transform",
            expanded && "rotate-90",
          )}
          aria-hidden
        />
        <div className="min-w-0 flex-1 space-y-1">
          <p className="truncate text-[15px] font-medium leading-snug tracking-tight">
            {series.display_name}
          </p>
          <p className="text-xs text-muted-foreground">
            {cadence}
            {" · "}
            {paymentLabel}
            {series.status === "uncertain" ? (
              <span className="text-amber-800 dark:text-amber-200">
                {" · "}
                Needs review
              </span>
            ) : null}
          </p>
        </div>
        <div className="shrink-0 text-right">
          <p className="text-[15px] font-semibold tabular-nums tracking-tight">
            {formatEuro(monthly)}
          </p>
          <p className="text-[11px] text-muted-foreground">per month</p>
        </div>
      </button>
      {expanded ? (
        <div className="px-3 pb-3 pl-10">
          {membersQuery.isLoading ? (
            <p className="py-1.5 text-xs text-muted-foreground">Loading…</p>
          ) : members.length === 0 ? (
            <p className="py-1.5 text-xs text-muted-foreground">
              No sample payments.
            </p>
          ) : (
            <ul className="space-y-2.5 border-t border-border/50 pt-3">
              {members.map((member) => (
                <li
                  key={member.transaction_id}
                  className="flex items-baseline justify-between gap-4"
                >
                  <div className="min-w-0 space-y-0.5">
                    <p className="truncate text-sm leading-snug">
                      {member.purpose || member.counterparty || "Payment"}
                    </p>
                    <p className="text-[11px] text-muted-foreground">
                      {formatDate(member.booking_date)}
                      {member.counterparty &&
                      member.purpose &&
                      member.counterparty !== series.display_name
                        ? ` · ${member.counterparty}`
                        : null}
                    </p>
                  </div>
                  <span className="shrink-0 text-sm tabular-nums tracking-tight text-muted-foreground">
                    {formatEuro(Number.parseFloat(member.amount) || 0)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      ) : null}
    </li>
  );
};

function isExpenseSeries(series: RecurringSeries): boolean {
  return series.kind === "fixed" || series.kind === "variable_regular";
}

/** Baseline puts quarterly/yearly expense series into monthly_irregular_costs. */
function isIrregularCadence(series: RecurringSeries): boolean {
  return series.interval === "quarterly" || series.interval === "yearly";
}

function formatEuro(value: number): string {
  return new Intl.NumberFormat("de-DE", {
    style: "currency",
    currency: "EUR",
  }).format(value);
}

function formatSignedEuro(value: number): string {
  const abs = new Intl.NumberFormat("de-DE", {
    style: "currency",
    currency: "EUR",
  }).format(Math.abs(value));
  if (value > 0) {
    return `+${abs}`;
  }
  if (value < 0) {
    return `−${abs}`;
  }
  return abs;
}

/** Higher spend vs norm / prior is amber; lower is calm green. */
function expenseDeltaTone(value: number): string | undefined {
  if (value > 0) {
    return "text-amber-800 dark:text-amber-200";
  }
  if (value < 0) {
    return "text-emerald-800 dark:text-emerald-300";
  }
  return undefined;
}

function formatShare(share: number): string {
  if (share > 0 && share < 0.005) {
    return "<1 %";
  }
  return new Intl.NumberFormat("de-DE", {
    style: "percent",
    maximumFractionDigits: 0,
  }).format(share);
}

function formatDate(value: string): string {
  const iso = value.slice(0, 10);
  const [y, m, d] = iso.split("-");
  if (!y || !m || !d) {
    return iso;
  }
  return `${d}.${m}.${y}`;
}

function confidenceLabel(value: string | undefined): string {
  switch (value) {
    case "high":
      return "High";
    case "medium":
      return "Medium";
    case "low":
      return "Low";
    default:
      return value ?? "—";
  }
}

export default BaselineExpensesPage;
