import type React from "react";
import { useMemo, useState } from "react";
import { Link } from "@tanstack/react-router";

import type {
  BaselineMetric,
  FinancialBaseline,
  RecurringSeries,
} from "@/api/types.gen";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useRecurringSeries } from "@/hooks/use-recurring";
import { useRecurringMembers } from "@/hooks/use-recurring-uncertain";
import {
  compositionEvidenceMetricKey,
  compositionEvidenceTitle,
  monthlyEquivalentAmount,
  resolveEvidenceSeries,
  type CompositionEvidenceKey,
} from "@/lib/baseline-charts";
import { defaultTransactionSearchParams } from "@/pages/transactions/search-params";
import { cn } from "@/lib/utils";

type CompositionEvidenceSheetProps = {
  baseline: FinancialBaseline;
  evidenceKey: CompositionEvidenceKey | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCorrectMetric?: (key: CompositionEvidenceKey) => void;
};

export const CompositionEvidenceSheet: React.FC<
  CompositionEvidenceSheetProps
> = ({ baseline, evidenceKey, open, onOpenChange, onCorrectMetric }) => {
  const recurringQuery = useRecurringSeries();
  const metricKey = evidenceKey
    ? compositionEvidenceMetricKey(evidenceKey)
    : null;
  const metric = metricKey
    ? baseline.metrics.find((m) => m.key === metricKey)
    : undefined;

  const seriesById = useMemo(() => {
    const map = new Map<string, RecurringSeries>();
    for (const series of recurringQuery.data?.data ?? []) {
      map.set(series.id, series);
    }
    return map;
  }, [recurringQuery.data]);

  const evidenceSeries = useMemo(() => {
    if (!metric) {
      return [];
    }
    return resolveEvidenceSeries(metric.evidence_ids, seriesById);
  }, [metric, seriesById]);

  const title = evidenceKey
    ? compositionEvidenceTitle(evidenceKey)
    : "Composition";

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>{title}</SheetTitle>
          {metric ? (
            <SheetDescription>
              {formatEuro(Number.parseFloat(metric.value) || 0)}
              {" · "}
              {metric.confidence} confidence
            </SheetDescription>
          ) : null}
        </SheetHeader>

        {evidenceKey && metric ? (
          <div className="grid gap-5 px-4 pb-6">
            <p className="text-sm text-muted-foreground">{metric.calculation}</p>

            {evidenceKey === "variable" ? (
              <VariableBody
                baseline={baseline}
                metric={metric}
                series={evidenceSeries}
                loading={recurringQuery.isLoading}
                onCorrect={() => {
                  onCorrectMetric?.(evidenceKey);
                  onOpenChange(false);
                }}
              />
            ) : evidenceKey === "free" || evidenceKey === "deficit" ? (
              <FreeBody baseline={baseline} evidenceKey={evidenceKey} />
            ) : (
              <SeriesBody
                series={evidenceSeries}
                loading={recurringQuery.isLoading}
                emptyLabel="No recurring series linked to this number yet."
                income={evidenceKey === "income"}
              />
            )}
          </div>
        ) : null}
      </SheetContent>
    </Sheet>
  );
};

const VariableBody: React.FC<{
  baseline: FinancialBaseline;
  metric: BaselineMetric;
  series: RecurringSeries[];
  loading: boolean;
  onCorrect: () => void;
}> = ({ baseline, metric, series, loading, onCorrect }) => {
  return (
    <div className="grid gap-5">
      <div className="space-y-2">
        <h3 className="text-sm font-semibold tracking-tight">How this is built</h3>
        <p className="text-sm text-muted-foreground">
          Variable spend is the residual after Fixed and Irregular recurring
          costs are removed from transfer-aware expenses in the baseline period.
          It is not a list of named series.
        </p>
        {metric.assumptions?.length ? (
          <ul className="list-disc space-y-1 pl-5 text-xs text-muted-foreground">
            {metric.assumptions.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        ) : null}
      </div>

      <div className="space-y-2">
        <h3 className="text-sm font-semibold tracking-tight">
          Already counted as Fixed / Irregular
        </h3>
        <SeriesBody
          series={series}
          loading={loading}
          emptyLabel="No recurring cover linked yet."
          income={false}
        />
      </div>

      <div className="flex flex-wrap gap-2">
        <Button type="button" size="sm" onClick={onCorrect}>
          Correct variable spend
        </Button>
        <Link
          to="/transactions"
          search={{
            ...defaultTransactionSearchParams,
            from: baseline.period_from.slice(0, 10),
            to: baseline.period_to.slice(0, 10),
            sort: "amount",
            order: "asc",
          }}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
        >
          Period transactions
        </Link>
      </div>
    </div>
  );
};

const FreeBody: React.FC<{
  baseline: FinancialBaseline;
  evidenceKey: "free" | "deficit";
}> = ({ baseline, evidenceKey }) => {
  return (
    <div className="space-y-3 text-sm text-muted-foreground">
      <p>
        {evidenceKey === "free"
          ? "What remains after income covers Fixed, Irregular, and Variable spend."
          : "Costs exceed income for a typical month at these numbers."}
      </p>
      <dl className="grid gap-2 text-xs">
        <div className="flex justify-between gap-3">
          <dt>Income</dt>
          <dd className="tabular-nums text-foreground">
            {formatEuro(Number.parseFloat(baseline.regular_monthly_income) || 0)}
          </dd>
        </div>
        <div className="flex justify-between gap-3">
          <dt>Fixed</dt>
          <dd className="tabular-nums text-foreground">
            −{formatEuro(Number.parseFloat(baseline.monthly_fixed_costs) || 0)}
          </dd>
        </div>
        <div className="flex justify-between gap-3">
          <dt>Irregular</dt>
          <dd className="tabular-nums text-foreground">
            −
            {formatEuro(
              Number.parseFloat(baseline.monthly_irregular_costs) || 0,
            )}
          </dd>
        </div>
        <div className="flex justify-between gap-3">
          <dt>Variable</dt>
          <dd className="tabular-nums text-foreground">
            −{formatEuro(Number.parseFloat(baseline.avg_variable_spend) || 0)}
          </dd>
        </div>
      </dl>
    </div>
  );
};

const SeriesBody: React.FC<{
  series: RecurringSeries[];
  loading: boolean;
  emptyLabel: string;
  income: boolean;
}> = ({ series, loading, emptyLabel, income }) => {
  if (loading) {
    return <p className="text-sm text-muted-foreground">Loading series…</p>;
  }
  if (series.length === 0) {
    return <p className="text-sm text-muted-foreground">{emptyLabel}</p>;
  }
  return (
    <ul className="divide-y border-y">
      {series.map((item) => (
        <EvidenceSeriesRow key={item.id} series={item} income={income} />
      ))}
    </ul>
  );
};

const EvidenceSeriesRow: React.FC<{
  series: RecurringSeries;
  income: boolean;
}> = ({ series, income }) => {
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

  return (
    <li className="py-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <p className="truncate text-sm font-medium">{series.display_name}</p>
          <p className="text-xs text-muted-foreground">
            {cadence}
            {" · "}
            {series.status}
            {" · "}
            {series.member_count} payments
          </p>
          <button
            type="button"
            className="text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
            onClick={() => setExpanded((open) => !open)}
          >
            {expanded ? "Hide payments" : "Show sample payments"}
          </button>
        </div>
        <span className="shrink-0 text-sm tabular-nums">
          {income ? formatEuro(monthly) : `−${formatEuro(monthly)}`}
          <span className="block text-right text-[10px] text-muted-foreground">
            / mo
          </span>
        </span>
      </div>
      {expanded ? (
        <div className="mt-2 space-y-1.5 border-t pt-2">
          {membersQuery.isLoading ? (
            <p className="text-xs text-muted-foreground">Loading…</p>
          ) : members.length === 0 ? (
            <p className="text-xs text-muted-foreground">No sample payments.</p>
          ) : (
            members.map((member) => (
              <div
                key={member.transaction_id}
                className="flex items-center justify-between gap-2 text-xs"
              >
                <span className="min-w-0 truncate text-muted-foreground">
                  {member.booking_date} ·{" "}
                  {member.purpose || member.counterparty || "Payment"}
                </span>
                <span className="shrink-0 tabular-nums">
                  {formatEuro(Number.parseFloat(member.amount) || 0)}
                </span>
              </div>
            ))
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
