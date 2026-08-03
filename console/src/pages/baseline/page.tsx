import type React from "react";
import { useState } from "react";
import { Link } from "@tanstack/react-router";

import { Button, buttonVariants } from "@/components/ui/button";
import {
  baselineActionErrorMessage,
  isBaselineMissing,
  useBaselineAdjust,
  useBaselineConfirm,
  useBaselineRecompute,
  useCurrentBaseline,
  type FinancialBaseline,
} from "@/hooks/use-baseline";
import { defaultTransactionSearchParams } from "@/pages/transactions/search-params";
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
    <div className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-8 overflow-y-auto pb-10">
      <header className="space-y-1">
        <p className="text-sm text-muted-foreground">
          Confirm your monthly household numbers before the Money Review.
        </p>
      </header>

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
            from: baseline.period_from.slice(0, 10),
            to: baseline.period_to.slice(0, 10),
          }}
          className="text-foreground underline-offset-4 hover:underline"
        >
          View transactions
        </Link>
      </div>

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
