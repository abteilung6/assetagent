import type React from "react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  classificationActionErrorMessage,
  useCategories,
  useClassificationCorrect,
  useClassificationQueue,
  type Category,
  type ClassificationQueueItem,
} from "@/hooks/use-classification-queue";
import {
  recurringActionErrorMessage,
  useRecurringConfirm,
  useRecurringReject,
  useUncertainRecurring,
  type RecurringSeries,
} from "@/hooks/use-recurring-uncertain";
import {
  transferActionErrorMessage,
  useTransferCandidates,
  useTransferConfirm,
  useTransferReject,
  type TransferCandidate,
} from "@/hooks/use-transfer-candidates";
import { cn } from "@/lib/utils";

const ReviewPage: React.FC = () => {
  const candidatesQuery = useTransferCandidates();
  const queueQuery = useClassificationQueue();
  const categoriesQuery = useCategories();
  const recurringQuery = useUncertainRecurring();
  const confirmMutation = useTransferConfirm();
  const rejectMutation = useTransferReject();
  const correctMutation = useClassificationCorrect();
  const recurringConfirmMutation = useRecurringConfirm();
  const recurringRejectMutation = useRecurringReject();
  const [actionError, setActionError] = useState<string | null>(null);
  const [pendingTransferID, setPendingTransferID] = useState<string | null>(
    null,
  );
  const [pendingTxID, setPendingTxID] = useState<string | null>(null);
  const [pendingRecurringID, setPendingRecurringID] = useState<string | null>(
    null,
  );

  const candidates = candidatesQuery.data?.data ?? [];
  const queue = queueQuery.data?.data ?? [];
  const categories = categoriesQuery.data?.data ?? [];
  const recurring = recurringQuery.data?.data ?? [];
  const transferBusy =
    confirmMutation.isPending || rejectMutation.isPending;
  const classifyBusy = correctMutation.isPending;
  const recurringBusy =
    recurringConfirmMutation.isPending || recurringRejectMutation.isPending;
  const anyBusy = transferBusy || classifyBusy || recurringBusy;
  const loading =
    candidatesQuery.isLoading ||
    queueQuery.isLoading ||
    categoriesQuery.isLoading ||
    recurringQuery.isLoading;
  const loadError =
    candidatesQuery.isError ||
    queueQuery.isError ||
    categoriesQuery.isError ||
    recurringQuery.isError;
  const inboxEmpty =
    candidates.length === 0 && queue.length === 0 && recurring.length === 0;

  const onConfirm = async (id: string) => {
    setActionError(null);
    setPendingTransferID(id);
    try {
      await confirmMutation.mutateAsync({ path: { id } });
    } catch (err) {
      setActionError(transferActionErrorMessage(err));
    } finally {
      setPendingTransferID(null);
    }
  };

  const onReject = async (id: string) => {
    setActionError(null);
    setPendingTransferID(id);
    try {
      await rejectMutation.mutateAsync({ path: { id } });
    } catch (err) {
      setActionError(transferActionErrorMessage(err));
    } finally {
      setPendingTransferID(null);
    }
  };

  const onCorrect = async (
    transactionId: string,
    categorySlug: string,
    applyToMerchant: boolean,
  ) => {
    setActionError(null);
    setPendingTxID(transactionId);
    try {
      await correctMutation.mutateAsync({
        path: { transaction_id: transactionId },
        body: {
          category_slug: categorySlug,
          apply_to_merchant: applyToMerchant || undefined,
        },
      });
    } catch (err) {
      setActionError(classificationActionErrorMessage(err));
    } finally {
      setPendingTxID(null);
    }
  };

  const onRecurringConfirm = async (id: string) => {
    setActionError(null);
    setPendingRecurringID(id);
    try {
      await recurringConfirmMutation.mutateAsync({ path: { id } });
    } catch (err) {
      setActionError(recurringActionErrorMessage(err));
    } finally {
      setPendingRecurringID(null);
    }
  };

  const onRecurringReject = async (id: string) => {
    setActionError(null);
    setPendingRecurringID(id);
    try {
      await recurringRejectMutation.mutateAsync({ path: { id } });
    } catch (err) {
      setActionError(recurringActionErrorMessage(err));
    } finally {
      setPendingRecurringID(null);
    }
  };

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-10 overflow-y-auto pb-8">
      {loadError ? (
        <p className="text-sm text-destructive" role="alert">
          Could not load items that need review.
        </p>
      ) : null}

      {actionError ? (
        <p className="text-sm text-destructive" role="alert">
          {actionError}
        </p>
      ) : null}

      {loading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : inboxEmpty ? (
        <div className="rounded-xl border border-dashed px-4 py-12 text-center">
          <p className="text-sm font-medium">Inbox clear</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Nothing needs your attention right now.
          </p>
        </div>
      ) : (
        <>
          {candidates.length > 0 ? (
            <section className="flex flex-col gap-4">
              <header className="space-y-1">
                <h2 className="text-base font-semibold tracking-tight">
                  Possible internal transfers
                </h2>
                <p className="text-sm text-muted-foreground">
                  Confirm moves between your own accounts so they are not
                  counted as spending or income.
                </p>
              </header>
              <ul className="flex flex-col gap-3">
                {candidates.map((candidate) => (
                  <TransferCandidateRow
                    key={candidate.id}
                    candidate={candidate}
                    busy={
                      transferBusy && pendingTransferID === candidate.id
                    }
                    disabled={anyBusy}
                    onConfirm={() => onConfirm(candidate.id)}
                    onReject={() => onReject(candidate.id)}
                  />
                ))}
              </ul>
            </section>
          ) : null}

          {queue.length > 0 ? (
            <section className="flex flex-col gap-4">
              <header className="space-y-1">
                <h2 className="text-base font-semibold tracking-tight">
                  Categories to check
                </h2>
                <p className="text-sm text-muted-foreground">
                  Large or unclear bookings — pick the right category so
                  spending stays trustworthy.
                </p>
              </header>
              <ul className="flex flex-col gap-3">
                {queue.map((item) => (
                  <ClassificationQueueRow
                    key={item.transaction_id}
                    item={item}
                    categories={categories}
                    busy={classifyBusy && pendingTxID === item.transaction_id}
                    disabled={anyBusy}
                    onCorrect={onCorrect}
                  />
                ))}
              </ul>
            </section>
          ) : null}

          {recurring.length > 0 ? (
            <section className="flex flex-col gap-4">
              <header className="space-y-1">
                <h2 className="text-base font-semibold tracking-tight">
                  Recurring payments
                </h2>
                <p className="text-sm text-muted-foreground">
                  Confirm regular bills and income so chat can track what
                  repeats.
                </p>
              </header>
              <ul className="flex flex-col gap-3">
                {recurring.map((series) => (
                  <RecurringSeriesRow
                    key={series.id}
                    series={series}
                    busy={
                      recurringBusy && pendingRecurringID === series.id
                    }
                    disabled={anyBusy}
                    onConfirm={() => onRecurringConfirm(series.id)}
                    onReject={() => onRecurringReject(series.id)}
                  />
                ))}
              </ul>
            </section>
          ) : null}
        </>
      )}
    </div>
  );
};

type TransferCandidateRowProps = {
  candidate: TransferCandidate;
  busy: boolean;
  disabled: boolean;
  onConfirm: () => void;
  onReject: () => void;
};

const TransferCandidateRow: React.FC<TransferCandidateRowProps> = ({
  candidate,
  busy,
  disabled,
  onConfirm,
  onReject,
}) => {
  const outLabel =
    candidate.out.account_name || candidate.out.counterparty || "From account";
  const inLabel =
    candidate.in.account_name || candidate.in.counterparty || "To account";
  const note = candidate.out.purpose || candidate.out.booking_text;

  return (
    <li className="rounded-xl border bg-card px-4 py-4 shadow-xs">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 space-y-1.5">
          <p className="text-sm font-medium text-foreground">
            {outLabel}
            <span className="mx-1.5 text-muted-foreground">→</span>
            {inLabel}
          </p>
          <p className="text-xs text-muted-foreground">
            {formatDate(candidate.out.booking_date)}
            {candidate.confidence === "exact"
              ? " · Strong match"
              : " · Likely match"}
          </p>
          {note ? (
            <p className="truncate text-xs text-muted-foreground">{note}</p>
          ) : null}
        </div>
        <Amount value={candidate.amount} signed={false} />
      </div>
      <div className="mt-4 flex flex-wrap justify-end gap-2 border-t pt-3">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={onReject}
        >
          Not a transfer
        </Button>
        <Button
          type="button"
          size="sm"
          disabled={disabled}
          onClick={onConfirm}
        >
          {busy ? "Saving…" : "Confirm transfer"}
        </Button>
      </div>
    </li>
  );
};

type ClassificationQueueRowProps = {
  item: ClassificationQueueItem;
  categories: Category[];
  busy: boolean;
  disabled: boolean;
  onCorrect: (
    transactionId: string,
    categorySlug: string,
    applyToMerchant: boolean,
  ) => void;
};

const ClassificationQueueRow: React.FC<ClassificationQueueRowProps> = ({
  item,
  categories,
  busy,
  disabled,
  onCorrect,
}) => {
  const [slug, setSlug] = useState(
    item.category_slug === "unresolved" ? "" : item.category_slug,
  );
  const [rememberMerchant, setRememberMerchant] = useState(
    Boolean(item.merchant_id),
  );
  const merchantLabel =
    item.merchant_name || item.counterparty || null;
  const title =
    merchantLabel || item.purpose || item.booking_text || "Transaction";
  const detail =
    item.purpose && item.purpose !== title ? item.purpose : item.booking_text;
  const guess =
    item.category_slug === "unresolved"
      ? "No category yet"
      : `Suggested: ${item.category_name || item.category_slug}`;
  const canRemember = Boolean(item.merchant_id && merchantLabel);

  return (
    <li className="rounded-xl border bg-card px-4 py-4 shadow-xs">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 space-y-1.5">
          <p className="truncate text-sm font-medium text-foreground">
            {title}
          </p>
          <p className="text-xs text-muted-foreground">
            {formatDate(item.booking_date)} · {guess}
          </p>
          {detail && detail !== title ? (
            <p className="truncate text-xs text-muted-foreground">{detail}</p>
          ) : null}
        </div>
        <Amount value={item.amount} />
      </div>

      <div className="mt-4 space-y-3 border-t pt-3">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <label className="flex min-w-0 flex-1 flex-col gap-1.5 text-xs text-muted-foreground">
            Category
            <select
              className="h-9 w-full rounded-lg border border-input bg-background px-2.5 text-sm text-foreground"
              value={slug}
              disabled={disabled}
              onChange={(e) => setSlug(e.target.value)}
            >
              <option value="" disabled>
                Choose category…
              </option>
              {categories.map((c) => (
                <option key={c.id} value={c.slug}>
                  {c.display_name}
                </option>
              ))}
            </select>
          </label>
          <Button
            type="button"
            size="sm"
            className="sm:mb-px"
            disabled={disabled || !slug}
            onClick={() =>
              onCorrect(
                item.transaction_id,
                slug,
                rememberMerchant && canRemember,
              )
            }
          >
            {busy ? "Saving…" : "Save"}
          </Button>
        </div>
        {canRemember ? (
          <label className="flex cursor-pointer items-start gap-2.5 text-sm text-muted-foreground">
            <input
              type="checkbox"
              className="mt-0.5 size-3.5 shrink-0 accent-foreground"
              checked={rememberMerchant}
              disabled={disabled}
              onChange={(e) => setRememberMerchant(e.target.checked)}
            />
            <span>
              Remember this for <span className="text-foreground">{merchantLabel}</span>
            </span>
          </label>
        ) : null}
      </div>
    </li>
  );
};

type RecurringSeriesRowProps = {
  series: RecurringSeries;
  busy: boolean;
  disabled: boolean;
  onConfirm: () => void;
  onReject: () => void;
};

const RecurringSeriesRow: React.FC<RecurringSeriesRowProps> = ({
  series,
  busy,
  disabled,
  onConfirm,
  onReject,
}) => {
  const cadence =
    series.interval === "monthly"
      ? "Monthly"
      : series.interval === "quarterly"
        ? "Quarterly"
        : "Yearly";
  const signedAmount =
    series.kind === "income"
      ? series.amount_typical
      : `-${series.amount_typical}`;

  return (
    <li className="rounded-xl border bg-card px-4 py-4 shadow-xs">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 space-y-1.5">
          <p className="truncate text-sm font-medium text-foreground">
            {series.display_name}
          </p>
          <p className="text-xs text-muted-foreground">
            {cadence}
            {" · "}
            {series.member_count} payments
            {series.amount_changed ? " · Amount changed recently" : null}
            {series.next_expected
              ? ` · Next around ${formatDate(series.next_expected)}`
              : null}
          </p>
        </div>
        <Amount value={signedAmount} />
      </div>
      <div className="mt-4 flex flex-wrap justify-end gap-2 border-t pt-3">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={onReject}
        >
          Not recurring
        </Button>
        <Button
          type="button"
          size="sm"
          disabled={disabled}
          onClick={onConfirm}
        >
          {busy ? "Saving…" : "Confirm recurring"}
        </Button>
      </div>
    </li>
  );
};

function Amount({
  value,
  signed = true,
}: {
  value: string;
  signed?: boolean;
}) {
  const amount = Number.parseFloat(value);
  const tone = signed
    ? amount > 0
      ? "income"
      : amount < 0
        ? "expense"
        : "neutral"
    : "neutral";

  return (
    <p
      className={cn(
        "shrink-0 text-right text-base font-semibold tracking-tight tabular-nums",
        tone === "income" && "text-emerald-700 dark:text-emerald-400",
        tone === "expense" && "text-red-700 dark:text-red-400",
        tone === "neutral" && "text-foreground",
      )}
    >
      {formatAmount(value)}
    </p>
  );
}

function formatAmount(value: string): string {
  const amount = Number.parseFloat(value);
  if (Number.isNaN(amount)) {
    return `${value} €`;
  }
  const formatted = new Intl.NumberFormat("de-DE", {
    style: "currency",
    currency: "EUR",
  }).format(amount);
  return formatted;
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

export default ReviewPage;
