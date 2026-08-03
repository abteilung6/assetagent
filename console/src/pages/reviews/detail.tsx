import type React from "react";
import { useState } from "react";
import { Link, useParams } from "@tanstack/react-router";

import { Button } from "@/components/ui/button";
import {
  moneyReviewActionErrorMessage,
  useConfirmMoneyReview,
  useMoneyReview,
  type MoneyReview,
} from "@/hooks/use-money-reviews";
import { cn } from "@/lib/utils";

const ReviewDetailPage: React.FC = () => {
  const { id } = useParams({ from: "/reviews/$id" });
  const query = useMoneyReview(id);
  const confirm = useConfirmMoneyReview();
  const [actionError, setActionError] = useState<string | null>(null);

  const onConfirm = async () => {
    if (!id) {
      return;
    }
    setActionError(null);
    try {
      await confirm.mutateAsync({ path: { id } });
    } catch (err) {
      setActionError(moneyReviewActionErrorMessage(err));
    }
  };

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-8 overflow-y-auto pb-10">
      <div>
        <Link
          to="/reviews"
          className="text-xs text-muted-foreground underline-offset-4 hover:underline"
        >
          ← All reviews
        </Link>
      </div>

      {actionError ? (
        <p className="text-sm text-destructive" role="alert">
          {actionError}
        </p>
      ) : null}

      {query.isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : query.isError ? (
        <p className="text-sm text-destructive" role="alert">
          Could not load this money review.
        </p>
      ) : query.data ? (
        <ReviewDetail
          review={query.data}
          busy={confirm.isPending}
          onConfirm={onConfirm}
        />
      ) : null}
    </div>
  );
};

type ReviewDetailProps = {
  review: MoneyReview;
  busy: boolean;
  onConfirm: () => void;
};

const ReviewDetail: React.FC<ReviewDetailProps> = ({
  review,
  busy,
  onConfirm,
}) => {
  const period = `${formatDate(review.period_from)} – ${formatDate(review.period_to)}`;
  const open =
    review.status === "needs_confirmation" || review.status === "draft";

  return (
    <div className="flex flex-col gap-8">
      <header className="space-y-2">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
          <span>{period}</span>
          <span aria-hidden="true">·</span>
          <span>{statusLabel(review.status)}</span>
          {review.data_freshness ? (
            <>
              <span aria-hidden="true">·</span>
              <span>Data through {formatDate(review.data_freshness)}</span>
            </>
          ) : null}
        </div>
        <p className="text-sm text-foreground">{review.summary}</p>
        <Link
          to="/baseline"
          className="inline-block text-xs text-muted-foreground underline-offset-4 hover:underline"
        >
          View baseline
        </Link>
      </header>

      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold tracking-tight">Findings</h2>
        {review.findings.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No high-priority findings this period.
          </p>
        ) : (
          <ul className="divide-y border-y">
            {review.findings.map((finding, index) => (
              <li key={`${finding.type}-${index}`} className="py-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0 space-y-1">
                    <p className="text-xs text-muted-foreground">
                      {findingTypeLabel(finding.type)}
                      {" · "}
                      {confidenceLabel(finding.confidence)} confidence
                    </p>
                    <p className="text-sm font-medium text-foreground">
                      {finding.title}
                    </p>
                  </div>
                  {finding.amount ? (
                    <p
                      className={cn(
                        "shrink-0 text-base font-semibold tabular-nums",
                        finding.type === "free_cashflow_pressure" &&
                          "text-red-700 dark:text-red-400",
                      )}
                    >
                      {formatAmount(finding.amount)}
                    </p>
                  ) : null}
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      {open ? (
        <div className="flex justify-end">
          <Button type="button" disabled={busy} onClick={onConfirm}>
            {busy ? "Saving…" : "Confirm review"}
          </Button>
        </div>
      ) : null}
    </div>
  );
};

function findingTypeLabel(type: string): string {
  switch (type) {
    case "free_cashflow_pressure":
      return "Free cashflow";
    case "recurring_amount_change":
      return "Recurring change";
    case "large_expense":
      return "Large expense";
    case "uncertain_recurring":
      return "Recurring review";
    case "needs_review_residue":
      return "Needs review";
    default:
      return type;
  }
}

function statusLabel(status: string): string {
  switch (status) {
    case "needs_confirmation":
      return "Needs confirmation";
    case "confirmed":
      return "Confirmed";
    case "superseded":
      return "Superseded";
    case "draft":
      return "Draft";
    default:
      return status;
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
  const iso = value.slice(0, 10);
  const [y, m, d] = iso.split("-");
  if (!y || !m || !d) {
    return iso || "—";
  }
  return `${d}.${m}.${y}`;
}

export default ReviewDetailPage;
