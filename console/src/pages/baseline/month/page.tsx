import type React from "react";
import { useMemo, useState } from "react";
import { Link, useParams } from "@tanstack/react-router";

import type { Transaction } from "@/api/types.gen";
import { AskAboutThis } from "@/components/chat/ask-about-this";
import { TransactionDetailSheet } from "@/components/transaction-detail/sheet";
import { buttonVariants } from "@/components/ui/button";
import { useBaselineCategorySpend, useBaselineMonthlyCashflow } from "@/hooks/use-baseline";
import { useTransactions } from "@/hooks/use-transactions";
import {
  buildMonthStory,
  endOfMonthISO,
  formatCompactMoney,
  formatMonthHeadline,
  formatMonthLabel,
  monthStartFromYyyyMm,
  partitionMonthSpend,
  shiftYyyyMm,
  type MonthlyCashflowPoint,
} from "@/lib/baseline-charts";
import { defaultTransactionSearchParams } from "@/pages/transactions/search-params";
import { cn } from "@/lib/utils";

const BaselineMonthPage: React.FC = () => {
  const { yyyyMm } = useParams({ from: "/baseline/months/$yyyyMm" });
  const monthStart = monthStartFromYyyyMm(yyyyMm);
  const invalid = monthStart == null;

  const cashflowQuery = useBaselineMonthlyCashflow(6);
  const monthEnd = monthStart ? endOfMonthISO(monthStart) : "";
  const categoryQuery = useBaselineCategorySpend(
    monthStart ?? "",
    monthEnd,
    8,
    Boolean(monthStart),
  );

  const expensesQuery = useTransactions(
    monthStart
      ? {
          limit: 50,
          offset: 0,
          from: monthStart,
          to: monthEnd,
          sort: "amount",
          order: "asc",
        }
      : {
          limit: 1,
          offset: 0,
          sort: "booking_date",
          order: "desc",
        },
  );
  const incomeQuery = useTransactions(
    monthStart
      ? {
          limit: 50,
          offset: 0,
          from: monthStart,
          to: monthEnd,
          sort: "amount",
          order: "desc",
        }
      : {
          limit: 1,
          offset: 0,
          sort: "booking_date",
          order: "desc",
        },
  );

  const [selected, setSelected] = useState<Transaction | null>(null);
  const [sheetOpen, setSheetOpen] = useState(false);

  const months: MonthlyCashflowPoint[] = useMemo(
    () =>
      (cashflowQuery.data?.data ?? []).map((row) => ({
        monthStart: row.month_start.slice(0, 10),
        income: Number.parseFloat(row.income) || 0,
        expenses: Number.parseFloat(row.expenses) || 0,
        net: Number.parseFloat(row.net) || 0,
      })),
    [cashflowQuery.data],
  );

  const expenseRows = expensesQuery.data?.data ?? [];
  const incomeRows = incomeQuery.data?.data ?? [];
  const oneOffs = expenseRows.filter((tx) => tx.one_off);
  const oneOffExpenseTotal = oneOffs.reduce((sum, tx) => {
    const amount = Number.parseFloat(tx.amount);
    return sum + (Number.isNaN(amount) || amount >= 0 ? 0 : Math.abs(amount));
  }, 0);

  const story = useMemo(
    () =>
      monthStart
        ? buildMonthStory(months, monthStart, {
            oneOffCount: oneOffs.length,
            oneOffExpenseTotal,
          })
        : null,
    [months, monthStart, oneOffs.length, oneOffExpenseTotal],
  );

  const topOutflows = expenseRows
    .filter((tx) => Number.parseFloat(tx.amount) < 0)
    .slice(0, 16);
  const spendPartition = partitionMonthSpend(topOutflows, 8);
  const topInflows = incomeRows
    .filter((tx) => Number.parseFloat(tx.amount) > 0)
    .slice(0, 8);
  const totalCount =
    expensesQuery.data?.pagination.total ??
    incomeQuery.data?.pagination.total ??
    0;

  const prevYyyyMm = shiftYyyyMm(yyyyMm, -1);
  const nextYyyyMm = shiftYyyyMm(yyyyMm, 1);
  const hasPrev = months.some(
    (m) => m.monthStart.slice(0, 7) === prevYyyyMm,
  );
  const hasNext = months.some(
    (m) => m.monthStart.slice(0, 7) === nextYyyyMm,
  );

  const openTx = (tx: Transaction) => {
    setSelected(tx);
    setSheetOpen(true);
  };

  if (invalid) {
    return (
      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 pb-10">
          <Link
            to="/baseline/history"
            className="text-xs text-muted-foreground underline-offset-4 hover:underline"
          >
            ← Months
          </Link>
          <p className="text-sm text-destructive" role="alert">
            Invalid month. Use a path like /baseline/months/2026-03.
          </p>
        </div>
      </div>
    );
  }

  const loading =
    cashflowQuery.isLoading ||
    expensesQuery.isLoading ||
    incomeQuery.isLoading;

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 pb-10">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Link
            to="/baseline/history"
            className="text-xs text-muted-foreground underline-offset-4 hover:underline"
          >
            ← Months
          </Link>
          <div className="flex gap-2">
            {hasPrev ? (
              <Link
                to="/baseline/months/$yyyyMm"
                params={{ yyyyMm: prevYyyyMm }}
                className={cn(
                  buttonVariants({ variant: "ghost", size: "sm" }),
                  "h-7 px-2",
                )}
              >
                ← {formatMonthLabel(`${prevYyyyMm}-01`)}
              </Link>
            ) : null}
            {hasNext ? (
              <Link
                to="/baseline/months/$yyyyMm"
                params={{ yyyyMm: nextYyyyMm }}
                className={cn(
                  buttonVariants({ variant: "ghost", size: "sm" }),
                  "h-7 px-2",
                )}
              >
                {formatMonthLabel(`${nextYyyyMm}-01`)} →
              </Link>
            ) : null}
          </div>
        </div>

        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : !story?.current ? (
          <div className="space-y-2">
            <h2 className="text-xl font-semibold tracking-tight">
              {formatMonthHeadline(monthStart)}
            </h2>
            <p className="text-sm text-muted-foreground">
              No cashflow for this month in the recent window yet.
            </p>
          </div>
        ) : (
          <>
            <header className="space-y-3">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="space-y-1">
                  <h2 className="text-xl font-semibold tracking-tight">
                    {formatMonthHeadline(monthStart)}
                  </h2>
                  <p className="text-sm text-muted-foreground">{story.subline}</p>
                </div>
                <AskAboutThis
                  prompt="Why was this month unusual?"
                  context={{
                    route: `/baseline/months/${yyyyMm}`,
                    yyyy_mm: yyyyMm,
                    from: monthStart,
                    to: monthEnd,
                  }}
                />
              </div>
              <dl className="grid grid-cols-3 gap-4">
                <div className="space-y-1">
                  <dt className="text-xs text-muted-foreground">Income</dt>
                  <dd className="text-lg font-medium tabular-nums tracking-tight">
                    {formatEuro(story.current.income)}
                  </dd>
                </div>
                <div className="space-y-1">
                  <dt className="text-xs text-muted-foreground">Expenses</dt>
                  <dd className="text-lg font-medium tabular-nums tracking-tight">
                    {formatEuro(story.current.expenses)}
                  </dd>
                </div>
                <div className="space-y-1">
                  <dt className="text-xs text-muted-foreground">Net</dt>
                  <dd
                    className={cn(
                      "text-lg font-medium tabular-nums tracking-tight",
                      story.current.net < 0
                        ? "text-red-700 dark:text-red-400"
                        : "text-foreground",
                    )}
                  >
                    {formatEuro(story.current.net)}
                  </dd>
                </div>
              </dl>
              {story.prior ? (
                <p className="text-xs text-muted-foreground">
                  vs {formatMonthLabel(story.prior.monthStart)}: income{" "}
                  {formatCompactMoney(
                    story.current.income - story.prior.income,
                  )}
                  , expenses{" "}
                  {formatCompactMoney(
                    story.current.expenses - story.prior.expenses,
                  )}
                  {story.medianExpenses != null
                    ? `, median spend ${formatCompactMoney(story.medianExpenses)}`
                    : null}
                </p>
              ) : null}
            </header>

            {story.whyBullets.length > 0 ? (
              <section className="space-y-2">
                <h3 className="text-sm font-semibold tracking-tight">
                  Why this month
                </h3>
                <ul className="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
                  {story.whyBullets.map((bullet) => (
                    <li key={bullet}>{bullet}</li>
                  ))}
                </ul>
              </section>
            ) : null}

            <CategorySpendSection
              loading={categoryQuery.isLoading}
              points={categoryQuery.data?.data ?? []}
            />

            <section className="space-y-5">
              <div className="space-y-1">
                <h3 className="text-sm font-semibold tracking-tight">
                  Cost drivers
                </h3>
                {topOutflows.length === 0 ? (
                  <p className="text-sm text-muted-foreground">
                    No expenses in this month.
                  </p>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    Recurring bills versus one-time payments.
                  </p>
                )}
              </div>
              {topOutflows.length > 0 ? (
                <>
                  {spendPartition.recurring.length > 0 ? (
                    <TxSection
                      title="Recurring"
                      empty="No recurring payments in the top expenses."
                      transactions={spendPartition.recurring}
                      onOpen={openTx}
                      tag="Recurring"
                    />
                  ) : null}
                  {spendPartition.oneTime.length > 0 ? (
                    <TxSection
                      title="One-time"
                      empty="No one-time expenses in the top list."
                      transactions={spendPartition.oneTime}
                      onOpen={openTx}
                    />
                  ) : null}
                </>
              ) : null}
            </section>

            <TxSection
              title="Income sources"
              empty="No income in this month."
              transactions={topInflows}
              onOpen={openTx}
              headingLevel="h3"
            />

            <section className="flex flex-col gap-3">
              <h3 className="text-sm font-semibold tracking-tight">
                Next actions
              </h3>
              <p className="text-sm text-muted-foreground">
                Open a payment to mark it as a one-off, or jump to Needs review
                if something looks wrong.
              </p>
              <div className="flex flex-wrap gap-2">
                {story.unusual ? (
                  <Link
                    to="/review"
                    className={cn(buttonVariants({ variant: "secondary" }))}
                  >
                    Open Needs review
                  </Link>
                ) : null}
                <Link
                  to="/transactions"
                  search={{
                    ...defaultTransactionSearchParams,
                    from: monthStart,
                    to: monthEnd,
                    sort: "amount",
                    order: "asc",
                  }}
                  className={cn(buttonVariants({ variant: "outline" }))}
                >
                  See all {totalCount} transactions
                </Link>
              </div>
            </section>
          </>
        )}

        <TransactionDetailSheet
          transaction={selected}
          open={sheetOpen}
          onOpenChange={(open) => {
            setSheetOpen(open);
            if (!open) {
              setSelected(null);
            }
          }}
          onTransactionChange={(tx) => {
            setSelected(tx);
          }}
        />
      </div>
    </div>
  );
};

const CategorySpendSection: React.FC<{
  loading: boolean;
  points: Array<{
    category_slug: string;
    category_name: string;
    total: string;
    transaction_count: number;
  }>;
}> = ({ loading, points }) => {
  if (loading) {
    return (
      <section className="space-y-2">
        <h3 className="text-sm font-semibold tracking-tight">
          Cost drivers by category
        </h3>
        <p className="text-sm text-muted-foreground">Loading categories…</p>
      </section>
    );
  }
  if (points.length === 0) {
    return null;
  }
  const maxTotal = Math.max(
    ...points.map((p) => Number.parseFloat(p.total) || 0),
    1,
  );
  return (
    <section className="space-y-3">
      <div className="space-y-1">
        <h3 className="text-sm font-semibold tracking-tight">
          Cost drivers by category
        </h3>
        <p className="text-sm text-muted-foreground">
          What kind of month this was — excluding one-offs and transfers.
        </p>
      </div>
      <ul className="space-y-2.5">
        {points.map((point) => {
          const total = Number.parseFloat(point.total) || 0;
          const width = Math.max(4, Math.round((total / maxTotal) * 100));
          return (
            <li key={point.category_slug} className="space-y-1">
              <div className="flex items-baseline justify-between gap-3 text-sm">
                <span className="truncate font-medium">{point.category_name}</span>
                <span className="shrink-0 tabular-nums text-muted-foreground">
                  {formatEuro(total)}
                </span>
              </div>
              <div className="h-1.5 w-full bg-muted">
                <div
                  className="h-full bg-foreground/55"
                  style={{ width: `${width}%` }}
                />
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
};

const TxSection: React.FC<{
  title: string;
  empty: string;
  transactions: Transaction[];
  onOpen: (tx: Transaction) => void;
  tag?: string;
  headingLevel?: "h3" | "h4";
}> = ({ title, empty, transactions, onOpen, tag, headingLevel = "h4" }) => {
  const Heading = headingLevel;
  return (
    <section className="space-y-3">
      <Heading
        className={
          headingLevel === "h3"
            ? "text-sm font-semibold tracking-tight"
            : "text-xs font-medium uppercase tracking-wide text-muted-foreground"
        }
      >
        {title}
      </Heading>
      {transactions.length === 0 ? (
        <p className="text-sm text-muted-foreground">{empty}</p>
      ) : (
        <ul className="divide-y border-y">
          {transactions.map((tx) => (
            <li key={tx.id}>
              <button
                type="button"
                className="flex w-full items-center gap-3 py-2.5 text-left hover:bg-muted/40"
                onClick={() => onOpen(tx)}
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">
                    {tx.counterparty || tx.purpose || "Transaction"}
                  </p>
                  <p className="truncate text-xs text-muted-foreground">
                    {tx.booking_date}
                    {tag ? ` · ${tag}` : null}
                    {tx.one_off ? " · One-off" : null}
                  </p>
                </div>
                <span className="shrink-0 text-sm tabular-nums">
                  {formatEuro(Number.parseFloat(tx.amount) || 0)}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
};

function formatEuro(value: number): string {
  return new Intl.NumberFormat("de-DE", {
    style: "currency",
    currency: "EUR",
  }).format(value);
}

export default BaselineMonthPage;
