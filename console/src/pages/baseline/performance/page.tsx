import type React from "react";
import { useMemo } from "react";
import { Link } from "@tanstack/react-router";

import { AskAboutThis } from "@/components/chat/ask-about-this";
import { buttonVariants } from "@/components/ui/button";
import {
  isBaselineMissing,
  useBaselineMonthlyCashflow,
  useCurrentBaseline,
} from "@/hooks/use-baseline";
import {
  buildBaselinePerformanceRows,
  buildTypicalMonthLevels,
  formatCompactMoney,
  formatMonthHeadline,
  formatMonthLabel,
  yyyyMmFromMonthStart,
  type MonthlyCashflowPoint,
} from "@/lib/baseline-charts";
import { cn } from "@/lib/utils";

const BaselinePerformancePage: React.FC = () => {
  const baselineQuery = useCurrentBaseline();
  const monthsQuery = useBaselineMonthlyCashflow(6);
  const missing = isBaselineMissing(baselineQuery.error);
  const baseline = baselineQuery.data;

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

  const typical = useMemo(() => {
    if (!baseline) {
      return null;
    }
    return buildTypicalMonthLevels({
      income: Number.parseFloat(baseline.regular_monthly_income) || 0,
      fixed: Number.parseFloat(baseline.monthly_fixed_costs) || 0,
      irregular: Number.parseFloat(baseline.monthly_irregular_costs) || 0,
      variable: Number.parseFloat(baseline.avg_variable_spend) || 0,
    });
  }, [baseline]);

  const rows = useMemo(
    () => (typical ? buildBaselinePerformanceRows(months, typical) : []),
    [months, typical],
  );

  const overspent = rows.filter((r) => r.overspent);
  const loading = baselineQuery.isLoading || monthsQuery.isLoading;

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 pb-10">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1">
            <p className="text-sm text-muted-foreground">
              Recent months versus your Cashflow norm — are you beating or
              missing the bar?
            </p>
            <Link
              to="/baseline"
              className="text-sm text-foreground underline-offset-4 hover:underline"
            >
              Open Cashflow
            </Link>
          </div>
          <AskAboutThis
            prompt="How am I tracking against my baseline?"
            context={{ route: "/baseline/performance" }}
          />
        </div>

        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : missing || !baseline || !typical ? (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Calculate and confirm a Cashflow baseline first so we can score
              months against it.
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
            <section className="space-y-2">
              <h2 className="text-sm font-semibold tracking-tight">
                Cashflow norm
              </h2>
              <dl className="grid grid-cols-3 gap-4">
                <div className="space-y-1">
                  <dt className="text-xs text-muted-foreground">Income</dt>
                  <dd className="text-lg font-medium tabular-nums tracking-tight">
                    {formatEuro(typical.income)}
                  </dd>
                </div>
                <div className="space-y-1">
                  <dt className="text-xs text-muted-foreground">Expenses</dt>
                  <dd className="text-lg font-medium tabular-nums tracking-tight">
                    {formatEuro(typical.expenses)}
                  </dd>
                </div>
                <div className="space-y-1">
                  <dt className="text-xs text-muted-foreground">Net</dt>
                  <dd className="text-lg font-medium tabular-nums tracking-tight">
                    {formatEuro(typical.income - typical.expenses)}
                  </dd>
                </div>
              </dl>
              <p className="text-xs text-muted-foreground">
                From your {baseline.status === "confirmed" ? "confirmed" : "draft"}{" "}
                Cashflow ·{" "}
                <Link
                  to="/insights/months"
                  className="underline underline-offset-4 hover:text-foreground"
                >
                  Browse months in Insights
                </Link>
              </p>
            </section>

            {overspent.length > 0 ? (
              <p className="text-sm text-amber-800 dark:text-amber-200">
                {overspent.length === 1
                  ? `${formatMonthHeadline(overspent[0]!.monthStart)} spent well above the Cashflow norm.`
                  : `${overspent.length} months spent well above the Cashflow norm.`}{" "}
                Open them in Insights to see drivers.
              </p>
            ) : (
              <p className="text-sm text-muted-foreground">
                Recent months are within a reasonable band of your Cashflow
                expenses.
              </p>
            )}

            <section className="space-y-3">
              <h2 className="text-sm font-semibold tracking-tight">
                Last {rows.length || 6} months
              </h2>
              {rows.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  Not enough booking history yet.
                </p>
              ) : (
                <ul className="divide-y border-y">
                  {[...rows].reverse().map((row) => (
                    <li key={row.monthStart}>
                      <Link
                        to="/insights/months/$yyyyMm"
                        params={{
                          yyyyMm: yyyyMmFromMonthStart(row.monthStart),
                        }}
                        className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 py-3 hover:bg-muted/40"
                      >
                        <div className="min-w-0 space-y-0.5">
                          <p className="text-sm font-medium">
                            {formatMonthHeadline(row.monthStart)}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            vs Cashflow: income{" "}
                            {formatCompactMoney(row.incomeDelta)}, expenses{" "}
                            {formatCompactMoney(row.expensesDelta)}, net{" "}
                            {formatCompactMoney(row.netDelta)}
                            {row.overspent ? " · above band" : null}
                          </p>
                        </div>
                        <div className="shrink-0 text-right text-sm tabular-nums">
                          <p
                            className={cn(
                              row.net < 0
                                ? "text-red-700 dark:text-red-400"
                                : "text-foreground",
                            )}
                          >
                            Net {formatEuro(row.net)}
                          </p>
                          <p className="text-xs text-muted-foreground">
                            {formatMonthLabel(row.monthStart)}
                          </p>
                        </div>
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </>
        )}
      </div>
    </div>
  );
};

function formatEuro(value: number): string {
  return new Intl.NumberFormat("de-DE", {
    style: "currency",
    currency: "EUR",
  }).format(value);
}

export default BaselinePerformancePage;
