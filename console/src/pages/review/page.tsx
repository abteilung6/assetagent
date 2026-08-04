import type React from "react";
import { useMemo, useState } from "react";
import { Link, useNavigate } from "@tanstack/react-router";

import { AskAboutThis } from "@/components/chat/ask-about-this";
import { Button } from "@/components/ui/button";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  classificationActionErrorMessage,
  useCategories,
  useClassificationApplySuggestions,
  useClassificationCorrect,
  useClassificationQueue,
  type Category,
  type ClassificationQueueItem,
} from "@/hooks/use-classification-queue";
import {
  recurringActionErrorMessage,
  useRecurringConfirm,
  useRecurringMembers,
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
import {
  parseReviewTab,
  type ReviewTab,
} from "@/pages/review/search-params";
import { defaultTransactionSearchParams } from "@/pages/transactions/search-params";
import { reviewRoute } from "@/router";
import { cn } from "@/lib/utils";

function defaultReviewTab(
  transfers: number,
  categories: number,
  recurring: number,
): ReviewTab {
  if (categories > 0) {
    return "categories";
  }
  if (recurring > 0) {
    return "recurring";
  }
  if (transfers > 0) {
    return "transfers";
  }
  return "categories";
}

const ReviewPage: React.FC = () => {
  const navigate = useNavigate();
  const search = reviewRoute.useSearch();
  const candidatesQuery = useTransferCandidates();
  const queueQuery = useClassificationQueue();
  const categoriesQuery = useCategories();
  const recurringQuery = useUncertainRecurring();
  const confirmMutation = useTransferConfirm();
  const rejectMutation = useTransferReject();
  const correctMutation = useClassificationCorrect();
  const applySuggestionsMutation = useClassificationApplySuggestions();
  const recurringConfirmMutation = useRecurringConfirm();
  const recurringRejectMutation = useRecurringReject();
  const [actionError, setActionError] = useState<string | null>(null);
  const [applySummary, setApplySummary] = useState<string | null>(null);
  const [pendingTransferID, setPendingTransferID] = useState<string | null>(
    null,
  );
  const [pendingTxID, setPendingTxID] = useState<string | null>(null);
  const [pendingRecurringID, setPendingRecurringID] = useState<string | null>(
    null,
  );
  const [confirmAllRecurring, setConfirmAllRecurring] = useState(false);

  const candidates = candidatesQuery.data?.data ?? [];
  const queue = queueQuery.data?.data ?? [];
  const categories = categoriesQuery.data?.data ?? [];
  const recurring = recurringQuery.data?.data ?? [];
  const transferBusy =
    confirmMutation.isPending || rejectMutation.isPending;
  const classifyBusy =
    correctMutation.isPending || applySuggestionsMutation.isPending;
  const recurringBusy =
    recurringConfirmMutation.isPending ||
    recurringRejectMutation.isPending ||
    confirmAllRecurring;
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

  const autoTab = useMemo(
    () =>
      defaultReviewTab(candidates.length, queue.length, recurring.length),
    [candidates.length, queue.length, recurring.length],
  );
  const activeTab = search.tab ?? autoTab;

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
    setApplySummary(null);
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

  const onApplySuggestions = async () => {
    setActionError(null);
    setApplySummary(null);
    try {
      const result = await applySuggestionsMutation.mutateAsync({});
      setApplySummary(
        `Applied ${result.applied} suggestion${result.applied === 1 ? "" : "s"}; skipped ${result.skipped}.`,
      );
    } catch (err) {
      setActionError(classificationActionErrorMessage(err));
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

  const onConfirmAllRecurring = async () => {
    if (recurring.length === 0) {
      return;
    }
    setActionError(null);
    setConfirmAllRecurring(true);
    const ids = recurring.map((series) => series.id);
    try {
      for (const id of ids) {
        await recurringConfirmMutation.mutateAsync({ path: { id } });
      }
    } catch (err) {
      setActionError(recurringActionErrorMessage(err));
    } finally {
      setConfirmAllRecurring(false);
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
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 pb-8">
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

        {applySummary ? (
          <p className="text-sm text-muted-foreground" role="status">
            {applySummary}
          </p>
        ) : null}

        <div className="flex flex-wrap items-center justify-between gap-2">
          <p className="text-sm text-muted-foreground">
            Confirm transfers, categories, and recurring series before they
            shape Baseline.
          </p>
          <AskAboutThis
            prompt="Summarize what I should clear"
            context={{
              route: "/review",
              tab: activeTab,
            }}
          />
        </div>

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
          <Tabs
            value={activeTab}
            onValueChange={(value) => {
              const tab = parseReviewTab(value);
              if (!tab) {
                return;
              }
              void navigate({
                to: "/review",
                search: { tab },
                replace: true,
              });
            }}
            className="gap-6"
          >
            <TabsList variant="line" className="w-full justify-start">
              <TabsTrigger value="categories">
                Categories
                {queue.length > 0 ? ` (${queue.length})` : ""}
              </TabsTrigger>
              <TabsTrigger value="recurring">
                Recurring payments
                {recurring.length > 0 ? ` (${recurring.length})` : ""}
              </TabsTrigger>
              <TabsTrigger value="transfers">
                Transfers
                {candidates.length > 0 ? ` (${candidates.length})` : ""}
              </TabsTrigger>
            </TabsList>

            <TabsContent value="categories" className="flex flex-col gap-4">
              <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
                <div className="space-y-1">
                  <h2 className="text-base font-semibold tracking-tight">
                    Categories to check
                  </h2>
                  <p className="text-sm text-muted-foreground">
                    Large or unclear bookings — pick the right category so
                    spending stays trustworthy.
                  </p>
                </div>
                {queue.length > 0 ? (
                  <Button
                    type="button"
                    variant="outline"
                    disabled={anyBusy}
                    onClick={onApplySuggestions}
                  >
                    {applySuggestionsMutation.isPending
                      ? "Applying…"
                      : "Apply suggested categories"}
                  </Button>
                ) : null}
              </header>
              {queue.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No categories need review right now.
                </p>
              ) : (
                <ul className="flex flex-col gap-3">
                  {queue.map((item) => (
                    <ClassificationQueueRow
                      key={item.transaction_id}
                      item={item}
                      categories={categories}
                      busy={
                        classifyBusy && pendingTxID === item.transaction_id
                      }
                      disabled={anyBusy}
                      onCorrect={onCorrect}
                    />
                  ))}
                </ul>
              )}
            </TabsContent>

            <TabsContent value="recurring" className="flex flex-col gap-4">
              <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
                <div className="space-y-1">
                  <h2 className="text-base font-semibold tracking-tight">
                    Recurring payments
                  </h2>
                  <p className="text-sm text-muted-foreground">
                    Confirm regular bills and income so chat can track what
                    repeats.
                  </p>
                </div>
                {recurring.length > 0 ? (
                  <Button
                    type="button"
                    disabled={anyBusy}
                    onClick={onConfirmAllRecurring}
                  >
                    {confirmAllRecurring
                      ? "Confirming…"
                      : `Confirm all (${recurring.length})`}
                  </Button>
                ) : null}
              </header>
              {recurring.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No uncertain recurring series right now.
                </p>
              ) : (
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
              )}
            </TabsContent>

            <TabsContent value="transfers" className="flex flex-col gap-4">
              <header className="space-y-1">
                <h2 className="text-base font-semibold tracking-tight">
                  Possible internal transfers
                </h2>
                <p className="text-sm text-muted-foreground">
                  Confirm moves between your own accounts so they are not
                  counted as spending or income.
                </p>
              </header>
              {candidates.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No transfer candidates right now.
                </p>
              ) : (
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
              )}
            </TabsContent>
          </Tabs>
        )}
      </div>
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
  const [expanded, setExpanded] = useState(false);
  const membersQuery = useRecurringMembers(series.id, expanded);
  const members = membersQuery.data?.data ?? [];
  const sampleCount = Math.min(3, series.member_count || 3);
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
  const typicalAbs = Math.abs(Number.parseFloat(series.amount_typical));
  const transactionsSearch = useMemo(() => {
    const dates = members
      .map((m) => m.booking_date.slice(0, 10))
      .filter(Boolean)
      .sort();
    const from = dates[0];
    const to = dates[dates.length - 1];
    const counterparty = members.find((m) => m.counterparty.trim())?.counterparty;
    return {
      ...defaultTransactionSearchParams,
      ...(from ? { from } : {}),
      ...(to ? { to } : {}),
      ...(counterparty
        ? { counterparty }
        : { q: series.display_name }),
    };
  }, [members, series.display_name]);

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
          <button
            type="button"
            className="text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
            onClick={() => setExpanded((open) => !open)}
          >
            {expanded
              ? "Hide payments"
              : `Show ${sampleCount} payment${sampleCount === 1 ? "" : "s"}`}
          </button>
        </div>
        <Amount value={signedAmount} />
      </div>

      {expanded ? (
        <div className="mt-3 space-y-2 border-t pt-3">
          {membersQuery.isLoading ? (
            <p className="text-xs text-muted-foreground">Loading payments…</p>
          ) : membersQuery.isError ? (
            <p className="text-xs text-destructive">
              Could not load sample payments.
            </p>
          ) : members.length === 0 ? (
            <p className="text-xs text-muted-foreground">
              No member transactions stored for this series yet.
            </p>
          ) : (
            <ul className="space-y-1.5">
              {members.map((member) => {
                const amountAbs = Math.abs(Number.parseFloat(member.amount));
                const atypical =
                  series.amount_changed &&
                  Number.isFinite(typicalAbs) &&
                  Number.isFinite(amountAbs) &&
                  Math.abs(amountAbs - typicalAbs) > 0.01;
                const label =
                  member.purpose.trim() ||
                  member.counterparty.trim() ||
                  "Payment";
                return (
                  <li
                    key={member.transaction_id}
                    className="flex items-baseline justify-between gap-3 text-xs"
                  >
                    <span className="min-w-0 truncate text-muted-foreground">
                      <span className="tabular-nums text-foreground">
                        {formatDate(member.booking_date)}
                      </span>{" "}
                      {label}
                      {atypical ? (
                        <span className="text-amber-800 dark:text-amber-200">
                          {" "}
                          · amount changed
                        </span>
                      ) : null}
                    </span>
                    <span className="shrink-0 tabular-nums text-foreground">
                      {formatAmount(member.amount)}
                    </span>
                  </li>
                );
              })}
            </ul>
          )}
          <Link
            to="/transactions"
            search={transactionsSearch}
            className="inline-flex text-xs text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
          >
            View all in Transactions →
          </Link>
        </div>
      ) : null}

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
