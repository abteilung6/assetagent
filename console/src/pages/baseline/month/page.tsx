import type React from "react";
import { useMemo, useState } from "react";
import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";

import type { Transaction } from "@/api/types.gen";
import { AskAboutThis } from "@/components/chat/ask-about-this";
import { TransactionDetailSheet } from "@/components/transaction-detail/sheet";
import { buttonVariants } from "@/components/ui/button";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  useBaselineCategorySpend,
  useBaselineDailyExpensePace,
  useBaselineMonthlyCashflow,
  useBaselineOneOffImpact,
} from "@/hooks/use-baseline";
import { useTransactions } from "@/hooks/use-transactions";
import {
  buildExpensePaceSeries,
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
import {
  buildExpensePaceChartLayout,
  chartLabelAnchor,
  formatChartDate,
  formatChartMoney,
} from "@/lib/balance-chart";
import {
  parseBaselineMonthTab,
  type BaselineMonthSearchParams,
  type BaselineMonthTab,
} from "@/pages/baseline/search-params";
import { defaultTransactionSearchParams } from "@/pages/transactions/search-params";
import { cn } from "@/lib/utils";

const BaselineMonthPage: React.FC = () => {
  const { yyyyMm } = useParams({ from: "/baseline/months/$yyyyMm" });
  const search = useSearch({ from: "/baseline/months/$yyyyMm" });
  const navigate = useNavigate();
  const activeTab: BaselineMonthTab = search.tab ?? "overview";
  const activityEnabled = activeTab === "activity";
  const monthStart = monthStartFromYyyyMm(yyyyMm);
  const invalid = monthStart == null;
  const monthSearch: BaselineMonthSearchParams =
    search.tab != null ? { tab: search.tab } : {};

  const cashflowQuery = useBaselineMonthlyCashflow(6);
  const monthEnd = monthStart ? endOfMonthISO(monthStart) : "";
  const categoryQuery = useBaselineCategorySpend(
    monthStart ?? "",
    monthEnd,
    8,
    Boolean(monthStart),
  );
  const paceQuery = useBaselineDailyExpensePace(
    monthStart ?? "",
    monthEnd,
    Boolean(monthStart),
  );
  const oneOffQuery = useBaselineOneOffImpact(
    monthStart ?? "",
    monthEnd,
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
    { enabled: Boolean(monthStart) && activityEnabled },
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
    { enabled: Boolean(monthStart) && activityEnabled },
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
  const oneOffCount = oneOffQuery.data?.count ?? 0;
  const oneOffExpenseTotal =
    Number.parseFloat(oneOffQuery.data?.expense_total ?? "0") || 0;

  const story = useMemo(
    () =>
      monthStart
        ? buildMonthStory(months, monthStart, {
            oneOffCount,
            oneOffExpenseTotal,
          })
        : null,
    [months, monthStart, oneOffCount, oneOffExpenseTotal],
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

  const paceSeries = useMemo(() => {
    if (!monthStart) {
      return [];
    }
    return buildExpensePaceSeries(
      monthStart,
      monthEnd,
      paceQuery.data?.data ?? [],
    );
  }, [monthStart, monthEnd, paceQuery.data]);

  const paceLayout = useMemo(
    () =>
      buildExpensePaceChartLayout(
        paceSeries.map((day) => ({
          date: day.date,
          cumulative: day.cumulativeExpenses,
          dailyCount: day.transactionCount,
        })),
        { height: 220 },
      ),
    [paceSeries],
  );

  const paceTotals = useMemo(() => {
    const last = paceSeries[paceSeries.length - 1];
    const txCount = paceSeries.reduce(
      (sum, day) => sum + day.transactionCount,
      0,
    );
    return {
      cumulative: last?.cumulativeExpenses ?? 0,
      txCount,
    };
  }, [paceSeries]);

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

  const setTab = (value: string | null) => {
    const tab = parseBaselineMonthTab(value) ?? "overview";
    void navigate({
      to: "/baseline/months/$yyyyMm",
      params: { yyyyMm },
      search: tab === "overview" ? {} : { tab },
      replace: true,
    });
  };

  const transactionsSearch = {
    ...defaultTransactionSearchParams,
    from: monthStart ?? undefined,
    to: monthEnd || undefined,
    sort: "amount" as const,
    order: "asc" as const,
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

  const loading = cashflowQuery.isLoading;

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
                search={monthSearch}
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
                search={monthSearch}
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
                    tab: activeTab,
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

            <Tabs
              value={activeTab}
              onValueChange={setTab}
              className="gap-6"
            >
              <TabsList variant="line" className="w-full justify-start">
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="activity">Activity</TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="flex flex-col gap-8">
                <ExpensePaceSection
                  loading={paceQuery.isLoading}
                  layout={paceLayout}
                  series={paceSeries}
                  totals={paceTotals}
                />

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

                <section className="flex flex-col gap-3">
                  <h3 className="text-sm font-semibold tracking-tight">
                    Next actions
                  </h3>
                  <p className="text-sm text-muted-foreground">
                    Jump to payments on Activity, or open Needs review if
                    something looks wrong.
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
                    <button
                      type="button"
                      className={cn(buttonVariants({ variant: "outline" }))}
                      onClick={() => setTab("activity")}
                    >
                      View activity
                    </button>
                    <Link
                      to="/transactions"
                      search={transactionsSearch}
                      className={cn(
                        buttonVariants({ variant: "ghost", size: "sm" }),
                        "text-muted-foreground",
                      )}
                    >
                      See all transactions
                    </Link>
                  </div>
                </section>
              </TabsContent>

              <TabsContent value="activity" className="flex flex-col gap-8">
                {expensesQuery.isLoading || incomeQuery.isLoading ? (
                  <p className="text-sm text-muted-foreground">
                    Loading payments…
                  </p>
                ) : (
                  <>
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
                        All transactions
                      </h3>
                      <p className="text-sm text-muted-foreground">
                        Open a payment to mark it as a one-off, or browse the
                        full list.
                      </p>
                      <div className="flex flex-wrap gap-2">
                        <Link
                          to="/transactions"
                          search={transactionsSearch}
                          className={cn(
                            buttonVariants({ variant: "secondary" }),
                          )}
                        >
                          See all {totalCount} transactions
                        </Link>
                      </div>
                    </section>
                  </>
                )}
              </TabsContent>
            </Tabs>
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

const ExpensePaceSection: React.FC<{
  loading: boolean;
  layout: ReturnType<typeof buildExpensePaceChartLayout>;
  series: ReturnType<typeof buildExpensePaceSeries>;
  totals: { cumulative: number; txCount: number };
}> = ({ loading, layout, series, totals }) => {
  const [hovered, setHovered] = useState<number | null>(null);

  if (loading) {
    return (
      <section className="space-y-2">
        <h3 className="text-sm font-semibold tracking-tight">Spending pace</h3>
        <p className="text-sm text-muted-foreground">Loading pace…</p>
      </section>
    );
  }
  if (!layout || series.length === 0 || totals.cumulative <= 0) {
    return (
      <section className="space-y-2">
        <h3 className="text-sm font-semibold tracking-tight">Spending pace</h3>
        <p className="text-sm text-muted-foreground">
          No expenses booked in this month yet.
        </p>
      </section>
    );
  }

  const hoverDay = hovered != null ? series[hovered] : null;

  return (
    <section className="space-y-3">
      <div className="space-y-1">
        <h3 className="text-sm font-semibold tracking-tight">Spending pace</h3>
        <p className="text-sm text-muted-foreground">
          Cumulative expenses day by day
          {totals.txCount > 0
            ? ` · ${totals.txCount} expense bookings`
            : null}
          . Bars show how busy each day was.
        </p>
      </div>
      <div className="space-y-2">
        <svg
          viewBox={`0 0 ${layout.width} ${layout.height}`}
          className="w-full text-foreground"
          style={{ aspectRatio: `${layout.width} / ${layout.height}` }}
          role="img"
          aria-label="Cumulative expenses over the month with daily booking counts"
        >
          {layout.moneyLabels.map((label) => (
            <g key={`money-${label.value}`}>
              <line
                x1={layout.padLeft}
                x2={layout.width - layout.padRight}
                y1={label.y}
                y2={label.y}
                className="stroke-border/60"
                strokeWidth={1}
                strokeDasharray={label.value === 0 ? "4 4" : undefined}
              />
              <text
                x={layout.width - layout.padRight}
                y={label.y - 4}
                textAnchor="end"
                className="fill-muted-foreground"
                fontSize={11}
              >
                {label.text}
              </text>
            </g>
          ))}
          {layout.bars.map((bar, i) =>
            bar.height > 0 ? (
              <rect
                key={`bar-${series[i]!.date}`}
                x={bar.x}
                y={bar.y}
                width={bar.width}
                height={bar.height}
                className="fill-muted-foreground/35 pointer-events-none"
              />
            ) : null,
          )}
          <path
            d={layout.areaPath}
            className="fill-red-700/10 dark:fill-red-400/10 pointer-events-none"
          />
          <path
            d={layout.linePath}
            className="stroke-red-700 dark:stroke-red-400 pointer-events-none"
            fill="none"
            strokeWidth={2}
            strokeLinejoin="round"
            strokeLinecap="round"
          />
          {layout.labelIndexes.map((i) => {
            const point = series[i]!;
            const x = layout.xs[i]!;
            return (
              <g key={`label-${point.date}`}>
                <line
                  x1={x}
                  x2={x}
                  y1={layout.padTop + layout.innerH}
                  y2={layout.padTop + layout.innerH + 4}
                  className="stroke-muted-foreground"
                  strokeWidth={1}
                />
                <text
                  x={x}
                  y={layout.height - 12}
                  textAnchor={chartLabelAnchor(i, series.length)}
                  className="fill-muted-foreground"
                  fontSize={11}
                >
                  {formatChartDate(point.date)}
                </text>
              </g>
            );
          })}
          {series.map((point, i) => {
            const x = layout.xs[i]!;
            const y = layout.cumulativeYs[i]!;
            const isHovered = hovered === i;
            return (
              <g key={`hit-${point.date}`}>
                <rect
                  x={x - 8}
                  y={layout.padTop}
                  width={16}
                  height={layout.innerH}
                  className="fill-transparent cursor-crosshair"
                  onMouseEnter={() => setHovered(i)}
                  onMouseLeave={() => setHovered(null)}
                  onFocus={() => setHovered(i)}
                  onBlur={() => setHovered(null)}
                />
                <circle
                  cx={x}
                  cy={y}
                  r={isHovered ? 4 : 0}
                  className="fill-red-700 pointer-events-none dark:fill-red-400"
                />
              </g>
            );
          })}
          {hoverDay && hovered != null ? (
            <text
              x={layout.xs[hovered]!}
              y={Math.max(
                layout.padTop + 12,
                layout.cumulativeYs[hovered]! - 10,
              )}
              textAnchor="middle"
              className="fill-foreground pointer-events-none"
              fontSize={11}
            >
              {formatChartDate(hoverDay.date)} ·{" "}
              {formatChartMoney(hoverDay.cumulativeExpenses)}
              {hoverDay.transactionCount > 0
                ? ` · ${hoverDay.transactionCount} tx`
                : ""}
            </text>
          ) : null}
        </svg>
        <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
          <li className="flex items-center gap-1.5">
            <span className="inline-block h-0.5 w-3 bg-red-700 dark:bg-red-400" />
            Cumulative expenses
          </li>
          <li className="flex items-center gap-1.5">
            <span className="inline-block h-2 w-1.5 bg-muted-foreground/40" />
            Daily expense bookings
          </li>
          <li className="tabular-nums">
            Month total {formatChartMoney(totals.cumulative)}
          </li>
        </ul>
        <p className="text-xs text-muted-foreground">
          Based on booking dates · transfers excluded · one-offs included
        </p>
      </div>
    </section>
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
