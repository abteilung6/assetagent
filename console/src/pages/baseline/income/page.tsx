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
  buildIncomeDevelopmentCallouts,
  buildIncomeDevelopmentRows,
  formatMonthHeadline,
  formatSignedPercent,
  monthlyEquivalentAmount,
  resolveEvidenceSeries,
  yyyyMmFromMonthStart,
  type MonthlyCashflowPoint,
} from "@/lib/baseline-charts";
import { cn } from "@/lib/utils";

const BaselineIncomePage: React.FC = () => {
  const baselineQuery = useCurrentBaseline();
  const monthsQuery = useBaselineMonthlyCashflow(12);
  const recurringQuery = useRecurringSeries();
  const uncertainQuery = useUncertainRecurring();
  const missing = isBaselineMissing(baselineQuery.error);
  const baseline = baselineQuery.data;

  const incomeMetric = baseline?.metrics.find(
    (m) => m.key === "regular_monthly_income",
  );
  const regularIncome = baseline
    ? Number.parseFloat(baseline.regular_monthly_income) || 0
    : 0;

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

  const sources = useMemo(() => {
    const all = recurringQuery.data?.data ?? [];
    const byId = new Map(all.map((s) => [s.id, s]));
    const fromEvidence = incomeMetric
      ? resolveEvidenceSeries(incomeMetric.evidence_ids, byId)
      : [];
    if (fromEvidence.length > 0) {
      return fromEvidence;
    }
    return all.filter((s) => s.kind === "income");
  }, [recurringQuery.data, incomeMetric]);

  const uncertainIncomeCount = useMemo(() => {
    const items = uncertainQuery.data?.data ?? [];
    return items.filter((s) => s.kind === "income").length;
  }, [uncertainQuery.data]);

  const chartLayout = useMemo(() => {
    if (months.length === 0) {
      return null;
    }
    return buildDualSeriesChartLayout(
      months.map((m) => ({
        date: m.monthStart,
        primary: m.income,
        secondary: regularIncome,
      })),
      {
        height: 220,
        padX: 8,
        references: [{ key: "income_norm", value: regularIncome }],
      },
    );
  }, [months, regularIncome]);

  const callouts = useMemo(
    () => buildIncomeDevelopmentCallouts(months, regularIncome),
    [months, regularIncome],
  );

  const developmentRows = useMemo(
    () => buildIncomeDevelopmentRows(months, regularIncome),
    [months, regularIncome],
  );

  const yearTotal = months.reduce((sum, m) => sum + m.income, 0);
  const loading =
    baselineQuery.isLoading ||
    monthsQuery.isLoading ||
    recurringQuery.isLoading;

  const periodLabel = baseline
    ? `${formatDate(baseline.period_from)} – ${formatDate(baseline.period_to)}`
    : "";

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 pb-10">
        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : missing || !baseline ? (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Calculate a Cashflow baseline first to see your regular income
              norm and sources.
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
            <header className="space-y-3">
              <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                <span>{periodLabel}</span>
                <span aria-hidden="true">·</span>
                <span>
                  {confidenceLabel(
                    incomeMetric?.confidence ?? baseline.confidence,
                  )}{" "}
                  confidence
                </span>
                <span aria-hidden="true">·</span>
                <span
                  className={
                    baseline.status === "confirmed" ? "text-foreground" : undefined
                  }
                >
                  {baseline.status === "confirmed" ? "Confirmed" : "Draft"}
                </span>
                <AskAboutThis
                  prompt="Explain my regular income"
                  context={{
                    route: "/baseline/income",
                    baseline_id: baseline.id,
                    from: baseline.period_from?.slice(0, 10),
                    to: baseline.period_to?.slice(0, 10),
                  }}
                  className="-my-1 ml-auto h-auto px-0 py-0"
                />
              </div>
              <div className="space-y-1">
                <p className="text-xs text-muted-foreground">
                  Regular monthly income
                </p>
                <p className="text-4xl font-semibold tracking-tight tabular-nums">
                  {formatEuro(regularIncome)}
                </p>
                <p className="text-sm text-muted-foreground">
                  Recurring inflows in the baseline window; one-offs do not raise
                  this number.
                </p>
              </div>
            </header>

            <section className="space-y-3">
              <div className="space-y-1">
                <h2 className="text-sm font-semibold tracking-tight">Sources</h2>
                <p className="text-sm text-muted-foreground">
                  Recurring payers behind the income norm — not every credit in
                  the window.
                </p>
              </div>
              {sources.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No recurring income series linked to this baseline yet.
                </p>
              ) : (
                <ul className="space-y-0.5">
                  {sources.map((series) => (
                    <SourceRow key={series.id} series={series} />
                  ))}
                </ul>
              )}
              <p className="text-xs text-muted-foreground">
                One-offs don’t raise your Income baseline.
              </p>
              {uncertainIncomeCount > 0 ? (
                <p className="text-sm text-amber-800 dark:text-amber-200">
                  {uncertainIncomeCount === 1
                    ? "1 income series needs review"
                    : `${uncertainIncomeCount} income series need review`}{" "}
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
                  Booked income by month versus your Cashflow income norm
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
                      aria-label="Monthly income versus Cashflow income norm"
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
                          className="stroke-emerald-700/40 dark:stroke-emerald-400/40"
                          strokeWidth={1.5}
                          strokeDasharray="6 4"
                        />
                      ))}
                      <path
                        d={chartLayout.primaryPath}
                        className="stroke-emerald-700 dark:stroke-emerald-400"
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
                            className="fill-emerald-700 dark:fill-emerald-400"
                          >
                            <title>
                              {formatMonthHeadline(point.monthStart)} ·{" "}
                              {formatChartMoney(point.income)}
                            </title>
                          </circle>
                        );
                      })}
                    </svg>
                    <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                      <li className="flex items-center gap-1.5">
                        <span className="inline-block h-0.5 w-3 bg-emerald-700 dark:bg-emerald-400" />
                        Booked income
                      </li>
                      <li className="flex items-center gap-1.5">
                        <span className="inline-block h-px w-3 border-t border-dashed border-emerald-700/40 dark:border-emerald-400/40" />
                        Income · Cashflow norm ({formatChartMoney(regularIncome)})
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
                        <TableRow key={row.monthStart}>
                          <TableCell className="pl-0 py-2.5">
                            <Link
                              to="/insights/months/$yyyyMm"
                              params={{
                                yyyyMm: yyyyMmFromMonthStart(row.monthStart),
                              }}
                              className="text-sm font-medium underline-offset-4 hover:underline"
                            >
                              {formatMonthHeadline(row.monthStart)}
                            </Link>
                          </TableCell>
                          <TableCell className="py-2.5 text-right text-sm tabular-nums tracking-tight">
                            {formatEuro(row.income)}
                          </TableCell>
                          <TableCell className="py-2.5 text-right">
                            {row.vsNorm === null || row.vsNormPct === null ? (
                              <span className="text-sm text-muted-foreground">
                                —
                              </span>
                            ) : (
                              <div className="space-y-0.5">
                                <p
                                  className={cn(
                                    "text-sm tabular-nums tracking-tight",
                                    deltaTone(row.vsNorm),
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
                              <span className={deltaTone(row.vsPriorPct)}>
                                {formatSignedPercent(row.vsPriorPct)}
                              </span>
                            )}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  Not enough booking history for an income trend yet.
                </p>
              )}
              {callouts.low.length > 0 || callouts.high.length > 0 ? (
                <ul className="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
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
                      booked well below the norm (
                      {formatChartMoney(m.income)}).
                    </li>
                  ))}
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
                      booked well above the norm (
                      {formatChartMoney(m.income)}
                      ) — check one-offs if that should not raise the baseline.
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

const SourceRow: React.FC<{ series: RecurringSeries }> = ({ series }) => {
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

function deltaTone(value: number): string | undefined {
  if (value > 0) {
    return "text-emerald-800 dark:text-emerald-300";
  }
  if (value < 0) {
    return "text-amber-800 dark:text-amber-200";
  }
  return undefined;
}

function formatDate(value: string): string {
  const iso = value.slice(0, 10);
  const [y, m, d] = iso.split("-");
  if (!y || !m || !d) {
    return iso;
  }
  return `${d}.${m}.${y}`;
}

function confidenceLabel(value: string): string {
  switch (value) {
    case "high":
      return "High";
    case "medium":
      return "Medium";
    case "low":
      return "Low";
    default:
      return value;
  }
}

export default BaselineIncomePage;
