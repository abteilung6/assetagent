import type React from "react";
import { useState } from "react";
import { Link, useNavigate } from "@tanstack/react-router";

import { Button, buttonVariants } from "@/components/ui/button";
import {
  moneyReviewActionErrorMessage,
  useCreateMoneyReview,
  useMoneyReviews,
  type MoneyReview,
} from "@/hooks/use-money-reviews";
import { cn } from "@/lib/utils";

const ReviewsPage: React.FC = () => {
  const navigate = useNavigate();
  const query = useMoneyReviews();
  const create = useCreateMoneyReview();
  const [actionError, setActionError] = useState<string | null>(null);

  const reviews = query.data?.data ?? [];
  const busy = create.isPending;

  const onCreate = async () => {
    setActionError(null);
    try {
      const review = await create.mutateAsync({ body: {} });
      await navigate({ to: "/reviews/$id", params: { id: review.id } });
    } catch (err) {
      setActionError(moneyReviewActionErrorMessage(err));
    }
  };

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 pb-10">
        <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <p className="text-sm text-muted-foreground">
            Your monthly Money Reviews — findings from a confirmed baseline.
          </p>
          {reviews.length > 0 ? (
            <Button type="button" disabled={busy} onClick={onCreate}>
              {busy ? "Generating…" : "Generate review"}
            </Button>
          ) : null}
        </header>

        {actionError ? (
          <p className="text-sm text-destructive" role="alert">
            {actionError}
          </p>
        ) : null}

        {query.isLoading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : query.isError ? (
          <p className="text-sm text-destructive" role="alert">
            Could not load money reviews.
          </p>
        ) : reviews.length === 0 ? (
          <div className="rounded-xl border border-dashed px-4 py-14 text-center">
            <p className="text-sm font-medium">No reviews yet</p>
            <p className="mx-auto mt-1 max-w-sm text-sm text-muted-foreground">
              Confirm a baseline first, then generate this month’s Money Review.
            </p>
            <div className="mt-6 flex flex-wrap justify-center gap-2">
              <Link
                to="/baseline"
                className={cn(buttonVariants({ variant: "outline" }))}
              >
                Open baseline
              </Link>
              <Button type="button" disabled={busy} onClick={onCreate}>
                {busy ? "Generating…" : "Generate review"}
              </Button>
            </div>
          </div>
        ) : (
          <ul className="divide-y border-y">
            {reviews.map((item) => (
              <ReviewListRow key={item.id} review={item} />
            ))}
          </ul>
        )}
      </div>
    </div>
  );
};

const ReviewListRow: React.FC<{ review: MoneyReview }> = ({ review }) => {
  const period = `${formatDate(review.period_from)} – ${formatDate(review.period_to)}`;
  return (
    <li>
      <Link
        to="/reviews/$id"
        params={{ id: review.id }}
        className="flex flex-col gap-1 py-4 transition-colors hover:bg-muted/40 sm:flex-row sm:items-center sm:justify-between"
      >
        <div className="min-w-0 space-y-1">
          <p className="text-sm font-medium text-foreground">{period}</p>
          <p className="truncate text-xs text-muted-foreground">{review.summary}</p>
        </div>
        <div className="flex shrink-0 items-center gap-3 text-xs text-muted-foreground">
          <span>{review.findings.length} findings</span>
          <span className="text-foreground">{statusLabel(review.status)}</span>
        </div>
      </Link>
    </li>
  );
};

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

function formatDate(value: string): string {
  const iso = value.slice(0, 10);
  const [y, m, d] = iso.split("-");
  if (!y || !m || !d) {
    return iso;
  }
  return `${d}.${m}.${y}`;
}

export default ReviewsPage;
