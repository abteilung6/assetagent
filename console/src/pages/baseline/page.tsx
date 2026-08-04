import type React from "react";
import { useState } from "react";
import { Link } from "@tanstack/react-router";

import { CompositionEvidenceSheet } from "@/components/baseline-composition-evidence/sheet";
import { AskAboutThis } from "@/components/chat/ask-about-this";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  baselineActionErrorMessage,
  isBaselineMissing,
  useBaselineAdjust,
  useBaselineConfirm,
  useBaselineMonthlyCashflow,
  useBaselineOneOffImpact,
  useBaselineRecompute,
  useCurrentBaseline,
  type FinancialBaseline,
} from "@/hooks/use-baseline";
import { formatChartMoney } from "@/lib/balance-chart";
import {
  buildBaselineComposition,
  buildBaselineReadinessItems,
  detectUnusualMonth,
  formatMonthLabel,
  formatOneOffImpactLine,
  type CompositionEvidenceKey,
  type CompositionSegmentKey,
  type MonthlyCashflowPoint,
} from "@/lib/baseline-charts";
import { defaultTransactionSearchParams } from "@/pages/transactions/search-params";
import { useClassificationQueue } from "@/hooks/use-classification-queue";
import { useUncertainRecurring } from "@/hooks/use-recurring-uncertain";
import { useTransferCandidates } from "@/hooks/use-transfer-candidates";
import { cn } from "@/lib/utils";

const SUPPORTING_KEYS = [
  "regular_monthly_income",
  "monthly_fixed_costs",
  "monthly_irregular_costs",
  "avg_variable_spend",
] as const;

type MetricKey = (typeof SUPPORTING_KEYS)[number] | "sustainable_free_cashflow";

const METRIC_LABELS: Record<MetricKey, string> = {
  regular_monthly_income: "Regular monthly income",
  monthly_fixed_costs: "Monthly fixed costs",
  monthly_irregular_costs: "Monthly irregular costs",
  avg_variable_spend: "Average variable spend",
  sustainable_free_cashflow: "Sustainable free cashflow",
};

const BaselinePage: React.FC = () => {
  const query = useCurrentBaseline();
  const recompute = useBaselineRecompute();
  const confirm = useBaselineConfirm();
  const adjust = useBaselineAdjust();
  const [actionError, setActionError] = useState<string | null>(null);
  const [editingKey, setEditingKey] = useState<MetricKey | null>(null);

  const missing = query.isError && isBaselineMissing(query.error);
  const baseline = query.data;
  const busy =
    recompute.isPending || confirm.isPending || adjust.isPending;

  const onRecompute = async () => {
    setActionError(null);
    setEditingKey(null);
    try {
      await recompute.mutateAsync({ body: {} });
    } catch (err) {
      setActionError(baselineActionErrorMessage(err));
    }
  };

  const onConfirm = async () => {
    if (!baseline) {
      return;
    }
    setActionError(null);
    try {
      await confirm.mutateAsync({ path: { id: baseline.id } });
    } catch (err) {
      setActionError(baselineActionErrorMessage(err));
    }
  };

  const onAdjust = async (
    metricKey: MetricKey,
    newValue: string,
    reason: string,
  ) => {
    if (!baseline) {
      return;
    }
    setActionError(null);
    try {
      await adjust.mutateAsync({
        path: { id: baseline.id },
        body: {
          metric_key: metricKey,
          new_value: normalizeAmountInput(newValue),
          reason: reason.trim(),
        },
      });
      setEditingKey(null);
    } catch (err) {
      setActionError(baselineActionErrorMessage(err));
    }
  };

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 pb-10">
        {actionError ? (
          <p className="text-sm text-destructive" role="alert">
            {actionError}
          </p>
        ) : null}

        {query.isLoading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : missing ? (
          <EmptyBaseline busy={busy} onCompute={onRecompute} />
        ) : query.isError ? (
          <p className="text-sm text-destructive" role="alert">
            Could not load your baseline.
          </p>
        ) : baseline ? (
          <BaselineContent
            baseline={baseline}
            busy={busy}
            editingKey={editingKey}
            onEdit={setEditingKey}
            onConfirm={onConfirm}
            onRecompute={onRecompute}
            onAdjust={onAdjust}
          />
        ) : null}
      </div>
    </div>
  );
};

type EmptyBaselineProps = {
  busy: boolean;
  onCompute: () => void;
};

const EmptyBaseline: React.FC<EmptyBaselineProps> = ({ busy, onCompute }) => {
  return (
    <div className="rounded-xl border border-dashed px-4 py-14 text-center">
      <p className="text-sm font-medium">No baseline yet</p>
      <p className="mx-auto mt-1 max-w-sm text-sm text-muted-foreground">
        We will calculate five monthly numbers from your imported transactions
        and recurring payments.
      </p>
      <Button
        type="button"
        className="mt-6"
        disabled={busy}
        onClick={onCompute}
      >
        {busy ? "Calculating…" : "Calculate baseline"}
      </Button>
    </div>
  );
};

type BaselineContentProps = {
  baseline: FinancialBaseline;
  busy: boolean;
  editingKey: MetricKey | null;
  onEdit: (key: MetricKey | null) => void;
  onConfirm: () => void;
  onRecompute: () => void;
  onAdjust: (key: MetricKey, value: string, reason: string) => void;
};

const BaselineContent: React.FC<BaselineContentProps> = ({
  baseline,
  busy,
  editingKey,
  onEdit,
  onConfirm,
  onRecompute,
  onAdjust,
}) => {
  const freeMetric = baseline.metrics.find(
    (m) => m.key === "sustainable_free_cashflow",
  );
  const periodLabel = `${formatDate(baseline.period_from)} – ${formatDate(baseline.period_to)}`;
  const confirmed = baseline.status === "confirmed";
  const periodFrom = baseline.period_from.slice(0, 10);
  const periodTo = baseline.period_to.slice(0, 10);
  const oneOffQuery = useBaselineOneOffImpact(periodFrom, periodTo);
  const oneOffLine = formatOneOffImpactLine(
    oneOffQuery.data?.count ?? 0,
    Number.parseFloat(oneOffQuery.data?.expense_total ?? "0") || 0,
    formatAmount(baseline.sustainable_free_cashflow),
  );

  return (
    <div className="flex flex-col gap-8">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
        <span>{periodLabel}</span>
        <span aria-hidden="true">·</span>
        <span>{confidenceLabel(baseline.confidence)} confidence</span>
        <span aria-hidden="true">·</span>
        <span className={confirmed ? "text-foreground" : undefined}>
          {confirmed ? "Confirmed" : "Draft"}
        </span>
        <Link
          to="/transactions"
          search={{
            ...defaultTransactionSearchParams,
            from: periodFrom,
            to: periodTo,
          }}
          className="text-foreground underline-offset-4 hover:underline"
        >
          View transactions
        </Link>
        <AskAboutThis
          prompt="Explain my free cashflow"
          context={{
            route: "/baseline",
            baseline_id: baseline.id,
            from: periodFrom,
            to: periodTo,
          }}
        />
      </div>

      <TypicalMonthSplit
        baseline={baseline}
        onFocusMetric={(key) => onEdit(key)}
      />

      <p className="text-sm text-muted-foreground">
        Want calendar months and who drove spend?{" "}
        <Link
          to="/insights/months"
          className="text-foreground underline-offset-4 hover:underline"
        >
          Open Insights
        </Link>
      </p>

      <section className="space-y-2">
        <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          {METRIC_LABELS.sustainable_free_cashflow}
        </p>
        <p
          className={cn(
            "text-4xl font-semibold tracking-tight tabular-nums",
            freeCashTone(baseline.sustainable_free_cashflow),
          )}
        >
          {formatAmount(baseline.sustainable_free_cashflow)}
          <span className="ml-1 text-base font-normal text-muted-foreground">
            / month
          </span>
        </p>
        {oneOffLine ? (
          <p className="max-w-xl text-sm text-muted-foreground">{oneOffLine}</p>
        ) : null}
        {freeMetric?.calculation ? (
          <p className="max-w-xl text-sm text-muted-foreground">
            {freeMetric.calculation}
          </p>
        ) : null}
      </section>

      <section className="flex flex-col">
        <h2 className="sr-only">Supporting numbers</h2>
        <ul className="divide-y border-y">
          {SUPPORTING_KEYS.map((key) => {
            const metric = baseline.metrics.find((m) => m.key === key);
            const value = metricValue(baseline, key);
            return (
              <MetricRow
                key={key}
                label={METRIC_LABELS[key]}
                value={value}
                calculation={metric?.calculation}
                confidence={metric?.confidence ?? baseline.confidence}
                editing={editingKey === key}
                busy={busy}
                onStartEdit={() => onEdit(key)}
                onCancelEdit={() => onEdit(null)}
                onSave={(next, reason) => onAdjust(key, next, reason)}
              />
            );
          })}
        </ul>
      </section>

      <BaselineReadiness baseline={baseline} confirmed={confirmed} />

      <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <Button
          type="button"
          variant="outline"
          disabled={busy}
          onClick={onRecompute}
        >
          Recalculate
        </Button>
        {!confirmed ? (
          <Button type="button" disabled={busy} onClick={onConfirm}>
            {busy ? "Saving…" : "Confirm baseline"}
          </Button>
        ) : (
          <Link
            to="/reviews"
            className={cn(buttonVariants({ variant: "default" }))}
          >
            Continue to Money Review
          </Link>
        )}
      </div>
    </div>
  );
};

const TypicalMonthSplit: React.FC<{
  baseline: FinancialBaseline;
  onFocusMetric: (key: MetricKey) => void;
}> = ({ baseline, onFocusMetric }) => {
  const [evidenceKey, setEvidenceKey] =
    useState<CompositionEvidenceKey | null>(null);
  const [evidenceOpen, setEvidenceOpen] = useState(false);
  const composition = buildBaselineComposition({
    income: Number.parseFloat(baseline.regular_monthly_income) || 0,
    fixed: Number.parseFloat(baseline.monthly_fixed_costs) || 0,
    irregular: Number.parseFloat(baseline.monthly_irregular_costs) || 0,
    variable: Number.parseFloat(baseline.avg_variable_spend) || 0,
    freeCashflow: Number.parseFloat(baseline.sustainable_free_cashflow) || 0,
  });

  const openEvidence = (key: CompositionEvidenceKey) => {
    setEvidenceKey(key);
    setEvidenceOpen(true);
  };

  return (
    <section className="flex flex-col gap-4">
      <div className="space-y-3">
        <div className="space-y-1">
          <h2 className="text-sm font-semibold tracking-tight">
            Income split
          </h2>
          <p className="text-sm text-muted-foreground">
            How a normal month allocates income into cost types and free
            cashflow. Click a segment to see what builds the model — not who you
            paid last month.
          </p>
        </div>
        <div className="space-y-2">
          <button
            type="button"
            className="flex w-full items-baseline justify-between gap-3 text-left text-xs text-muted-foreground underline-offset-4 hover:underline"
            onClick={() => openEvidence("income")}
          >
            <span>Income</span>
            <span className="tabular-nums text-foreground">
              {formatAmount(baseline.regular_monthly_income)}
            </span>
          </button>
          <div
            className="flex h-10 w-full overflow-hidden rounded-lg border bg-muted/30"
            role="img"
            aria-label="Cashflow income split by cost type"
          >
            {composition.segments.map((seg) => (
              <button
                key={seg.key}
                type="button"
                title={`${seg.label}: ${formatAmount(seg.amount.toFixed(2))}`}
                className={cn(
                  "h-full min-w-0 cursor-pointer transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
                  segmentTone(seg.key),
                )}
                style={{ flexGrow: seg.share, flexBasis: 0 }}
                onClick={() => openEvidence(seg.key)}
              />
            ))}
          </div>
          <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
            {composition.segments.map((seg) => (
              <li key={seg.key}>
                <button
                  type="button"
                  className="flex items-center gap-1.5 underline-offset-4 hover:underline"
                  onClick={() => openEvidence(seg.key)}
                >
                  <span
                    className={cn(
                      "inline-block size-2 rounded-sm",
                      segmentTone(seg.key),
                    )}
                    aria-hidden
                  />
                  <span>
                    {seg.label}{" "}
                    <span className="tabular-nums text-foreground">
                      {formatChartMoney(seg.amount)}
                    </span>
                  </span>
                </button>
              </li>
            ))}
          </ul>
          <p className="text-xs text-muted-foreground">
            Variable is a residual after Fixed and Irregular recurring costs —
            not a merchant list. For who you paid, open Insights → Months.
          </p>
        </div>
      </div>

      <CompositionEvidenceSheet
        baseline={baseline}
        evidenceKey={evidenceKey}
        open={evidenceOpen}
        onOpenChange={(open) => {
          setEvidenceOpen(open);
          if (!open) {
            setEvidenceKey(null);
          }
        }}
        onCorrectMetric={(key) => {
          if (key === "variable") {
            onFocusMetric("avg_variable_spend");
          } else if (key === "income") {
            onFocusMetric("regular_monthly_income");
          } else if (key === "fixed") {
            onFocusMetric("monthly_fixed_costs");
          } else if (key === "irregular") {
            onFocusMetric("monthly_irregular_costs");
          }
        }}
      />
    </section>
  );
};

const BaselineReadiness: React.FC<{
  baseline: FinancialBaseline;
  confirmed: boolean;
}> = ({ baseline, confirmed }) => {
  const transfersQuery = useTransferCandidates();
  const categoriesQuery = useClassificationQueue();
  const recurringQuery = useUncertainRecurring();
  const monthsQuery = useBaselineMonthlyCashflow(6);

  const months: MonthlyCashflowPoint[] = (monthsQuery.data?.data ?? []).map(
    (row) => ({
      monthStart: row.month_start.slice(0, 10),
      income: Number.parseFloat(row.income) || 0,
      expenses: Number.parseFloat(row.expenses) || 0,
      net: Number.parseFloat(row.net) || 0,
    }),
  );
  const unusual = detectUnusualMonth(months, baseline.period_from.slice(0, 10));
  const items = buildBaselineReadinessItems({
    transferCount: transfersQuery.data?.data.length ?? 0,
    categoryCount: categoriesQuery.data?.data.length ?? 0,
    recurringCount: recurringQuery.data?.data.length ?? 0,
    unusualMonthStart: unusual.unusual ? unusual.monthStart : null,
  });

  const loading =
    transfersQuery.isLoading ||
    categoriesQuery.isLoading ||
    recurringQuery.isLoading ||
    monthsQuery.isLoading;

  if (loading) {
    return null;
  }

  if (items.length === 0) {
    if (confirmed) {
      return null;
    }
    return (
      <section className="space-y-1">
        <h2 className="text-sm font-semibold tracking-tight">
          Looks stable
        </h2>
        <p className="text-sm text-muted-foreground">
          Confirm this as your planning baseline? That unlocks Money Review and
          Plan forecast.
        </p>
      </section>
    );
  }

  return (
    <section className="space-y-3">
      <div className="space-y-1">
        <h2 className="text-sm font-semibold tracking-tight">
          {confirmed ? "Still open" : "Before you confirm"}
        </h2>
        <p className="text-sm text-muted-foreground">
          {confirmed
            ? "These items can still improve trust in the numbers."
            : "Clear these when you can — they are the usual sources of distorted baselines."}
        </p>
      </div>
      <ul className="divide-y border-y">
        {items.map((item) => (
          <li key={item.id}>
            {item.href.kind === "review" ? (
              <Link
                to="/review"
                search={{ tab: item.href.tab }}
                className="flex items-center justify-between gap-3 py-2.5 text-sm underline-offset-4 hover:underline"
              >
                <span>{item.label}</span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  Needs review
                </span>
              </Link>
            ) : (
              <Link
                to="/insights/months/$yyyyMm"
                params={{ yyyyMm: item.href.yyyyMm }}
                className="flex items-center justify-between gap-3 py-2.5 text-sm underline-offset-4 hover:underline"
              >
                <span>{item.label}</span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  Open month
                </span>
              </Link>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
};

function segmentTone(key: CompositionSegmentKey): string {
  switch (key) {
    case "fixed":
      return "bg-foreground/80";
    case "irregular":
      return "bg-foreground/55";
    case "variable":
      return "bg-foreground/30";
    case "free":
      return "bg-emerald-700/70 dark:bg-emerald-500/50";
    case "deficit":
      return "bg-red-700/80 dark:bg-red-400/70";
  }
}

type MetricRowProps = {
  label: string;
  value: string;
  calculation?: string;
  confidence: string;
  editing: boolean;
  busy: boolean;
  onStartEdit: () => void;
  onCancelEdit: () => void;
  onSave: (value: string, reason: string) => void;
};

const MetricRow: React.FC<MetricRowProps> = ({
  label,
  value,
  calculation,
  confidence,
  editing,
  busy,
  onStartEdit,
  onCancelEdit,
  onSave,
}) => {
  const [draft, setDraft] = useState(value);
  const [reason, setReason] = useState("");

  if (editing) {
    return (
      <li className="py-4">
        <form
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault();
            if (!reason.trim()) {
              return;
            }
            onSave(draft, reason);
          }}
        >
          <div className="flex items-baseline justify-between gap-3">
            <p className="text-sm font-medium">{label}</p>
            <p className="text-xs text-muted-foreground">
              {confidenceLabel(confidence)}
            </p>
          </div>
          <label className="flex flex-col gap-1.5 text-xs text-muted-foreground">
            Monthly amount (EUR)
            <input
              className="h-10 w-full max-w-xs rounded-lg border border-input bg-background px-3 text-sm tabular-nums text-foreground"
              inputMode="decimal"
              value={draft}
              disabled={busy}
              onChange={(e) => setDraft(e.target.value)}
              autoFocus
            />
          </label>
          <label className="flex flex-col gap-1.5 text-xs text-muted-foreground">
            Why are you correcting this?
            <input
              className="h-10 w-full rounded-lg border border-input bg-background px-3 text-sm text-foreground"
              value={reason}
              disabled={busy}
              onChange={(e) => setReason(e.target.value)}
              placeholder="e.g. Rent was reduced in March"
              required
            />
          </label>
          <div className="flex flex-wrap justify-end gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={busy}
              onClick={onCancelEdit}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              size="sm"
              disabled={busy || !reason.trim()}
            >
              {busy ? "Saving…" : "Save correction"}
            </Button>
          </div>
        </form>
      </li>
    );
  }

  return (
    <li className="flex flex-col gap-2 py-4 sm:flex-row sm:items-start sm:justify-between">
      <div className="min-w-0 space-y-1">
        <p className="text-sm font-medium text-foreground">{label}</p>
        {calculation ? (
          <p className="text-xs text-muted-foreground">{calculation}</p>
        ) : null}
        <p className="text-xs text-muted-foreground">
          {confidenceLabel(confidence)} confidence
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-3 sm:flex-col sm:items-end sm:gap-2">
        <p className="text-base font-semibold tracking-tight tabular-nums">
          {formatAmount(value)}
        </p>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-8 px-2 text-muted-foreground"
          disabled={busy}
          onClick={() => {
            setDraft(value);
            setReason("");
            onStartEdit();
          }}
        >
          Correct
        </Button>
      </div>
    </li>
  );
};

function metricValue(baseline: FinancialBaseline, key: MetricKey): string {
  switch (key) {
    case "regular_monthly_income":
      return baseline.regular_monthly_income;
    case "monthly_fixed_costs":
      return baseline.monthly_fixed_costs;
    case "monthly_irregular_costs":
      return baseline.monthly_irregular_costs;
    case "avg_variable_spend":
      return baseline.avg_variable_spend;
    case "sustainable_free_cashflow":
      return baseline.sustainable_free_cashflow;
  }
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

function freeCashTone(value: string): string {
  const amount = Number.parseFloat(value);
  if (Number.isNaN(amount)) {
    return "text-foreground";
  }
  if (amount < 0) {
    return "text-red-700 dark:text-red-400";
  }
  return "text-foreground";
}

function formatAmount(value: string): string {
  const amount = Number.parseFloat(value);
  if (Number.isNaN(amount)) {
    return `${value} €`;
  }
  return new Intl.NumberFormat("de-DE", {
    style: "currency",
    currency: "EUR",
  }).format(amount);
}

function formatDate(value: string): string {
  if (!value) {
    return "—";
  }
  const iso = value.slice(0, 10);
  const [y, m, d] = iso.split("-");
  if (!y || !m || !d) {
    return iso;
  }
  return `${d}.${m}.${y}`;
}

function normalizeAmountInput(value: string): string {
  const cleaned = value.trim().replace(",", ".");
  const amount = Number.parseFloat(cleaned);
  if (Number.isNaN(amount)) {
    return cleaned;
  }
  return amount.toFixed(2);
}

export default BaselinePage;
