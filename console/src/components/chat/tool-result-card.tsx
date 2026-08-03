import {
  AlertTriangle,
  CalendarRange,
  LineChart,
  RefreshCw,
  Search,
  TrendingDown,
  TrendingUp,
  Users,
  Wallet,
} from "lucide-react";
import type React from "react";
import { Link } from "@tanstack/react-router";

import type { ChatToolCall } from "@/api/types.gen";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { SourceLink } from "./source-link";
import {
  asCashflowResult,
  asCounterpartiesResult,
  asSearchResult,
  buildTransactionSearchFromToolCall,
  formatDateRange,
  formatMoney,
  isTransactionSourceTool,
  readOptionalString,
  TOOL_NAMES,
  toolDisplayName,
  toolErrorMessage,
  toolHasError,
} from "./tool-sources";

type ToolResultCardProps = {
  toolCall: ChatToolCall;
};

export const ToolResultCard: React.FC<ToolResultCardProps> = ({ toolCall }) => {
  const { name, input, result } = toolCall;
  const from = readOptionalString(input, "from");
  const to = readOptionalString(input, "to");
  const period = formatDateRange(from, to);
  const hasError = toolHasError(result);

  return (
    <section
      className={cn(
        "rounded-xl border bg-background/70 p-3 text-foreground shadow-xs",
        hasError ? "border-amber-500/40" : "border-border/80",
      )}
      aria-label={`${toolDisplayName(name)} source`}
    >
      <header className="flex items-start gap-2">
        <ToolIcon name={name} hasError={hasError} />
        <div className="min-w-0 flex-1">
          <p className="text-xs font-medium text-foreground">
            {toolDisplayName(name)}
          </p>
          <p className="text-xs text-muted-foreground">
            {periodLabel(name, input, result, period)}
          </p>
        </div>
      </header>

      <div className="mt-3">{renderBody(toolCall, hasError)}</div>

      {!hasError ? (
        <footer className="mt-3 flex flex-wrap gap-2">
          {isTransactionSourceTool(name) ? (
            <SourceLink search={buildTransactionSearchFromToolCall(toolCall)} />
          ) : (
            <PlanSourceLink name={name} result={result} />
          )}
        </footer>
      ) : null}

      <details className="mt-2 text-xs text-muted-foreground">
        <summary className="cursor-pointer select-none hover:text-foreground">
          Technical details
        </summary>
        <pre className="mt-2 overflow-x-auto rounded-md bg-muted/60 p-2 font-mono text-[11px] leading-relaxed whitespace-pre-wrap">
          {JSON.stringify({ input, result }, null, 2)}
        </pre>
      </details>
    </section>
  );
};

function periodLabel(
  name: string,
  input: Record<string, unknown>,
  result: Record<string, unknown>,
  period: string,
): string {
  if (name === TOOL_NAMES.search) {
    const query = readOptionalString(input, "q");
    if (query) {
      return `“${query}” · ${period}`;
    }
  }
  if (
    name === TOOL_NAMES.baseline ||
    name === TOOL_NAMES.moneyReview ||
    name === TOOL_NAMES.forecast
  ) {
    const resultPeriod = result.period;
    if (
      resultPeriod &&
      typeof resultPeriod === "object" &&
      resultPeriod !== null
    ) {
      const record = resultPeriod as Record<string, unknown>;
      const fromDate =
        typeof record.from === "string" ? record.from : undefined;
      const toDate = typeof record.to === "string" ? record.to : undefined;
      return formatDateRange(fromDate, toDate);
    }
    if (name === TOOL_NAMES.forecast) {
      const days =
        typeof result.horizon_days === "number" ? result.horizon_days : null;
      return days != null ? `${days}-day horizon` : "Plan forecast";
    }
    return "Plan artifact";
  }
  return period;
}

function ToolIcon({
  name,
  hasError,
}: {
  name: string;
  hasError: boolean;
}) {
  const className = cn(
    "mt-0.5 size-4 shrink-0",
    hasError ? "text-amber-600 dark:text-amber-400" : "text-muted-foreground",
  );

  switch (name) {
    case TOOL_NAMES.cashflow:
    case TOOL_NAMES.cashflowV2:
      return <TrendingDown className={className} aria-hidden />;
    case TOOL_NAMES.recurring:
      return <RefreshCw className={className} aria-hidden />;
    case TOOL_NAMES.spendingChanges:
      return <TrendingUp className={className} aria-hidden />;
    case TOOL_NAMES.anomalies:
      return <AlertTriangle className={className} aria-hidden />;
    case TOOL_NAMES.counterparties:
      return <Users className={className} aria-hidden />;
    case TOOL_NAMES.search:
      return <Search className={className} aria-hidden />;
    case TOOL_NAMES.baseline:
      return <Wallet className={className} aria-hidden />;
    case TOOL_NAMES.moneyReview:
      return <CalendarRange className={className} aria-hidden />;
    case TOOL_NAMES.forecast:
      return <LineChart className={className} aria-hidden />;
    default:
      return <Search className={className} aria-hidden />;
  }
}

function PlanSourceLink({
  name,
  result,
}: {
  name: string;
  result: Record<string, unknown>;
}) {
  if (name === TOOL_NAMES.baseline) {
    return (
      <Link
        to="/baseline"
        className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
      >
        Open Baseline
      </Link>
    );
  }
  if (name === TOOL_NAMES.moneyReview) {
    const id = typeof result.id === "string" ? result.id : null;
    if (id) {
      return (
        <Link
          to="/reviews/$id"
          params={{ id }}
          className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
        >
          Open review
        </Link>
      );
    }
    return (
      <Link
        to="/reviews"
        className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
      >
        Open Reviews
      </Link>
    );
  }
  return (
    <Link
      to="/plan"
      className={cn(buttonVariants({ variant: "outline", size: "sm" }))}
    >
      Open Plan
    </Link>
  );
}

function renderBody(toolCall: ChatToolCall, hasError: boolean) {
  const { name, result } = toolCall;

  if (hasError) {
    return (
      <p className="text-sm text-amber-800 dark:text-amber-200">
        {toolErrorMessage(result)}
      </p>
    );
  }

  switch (name) {
    case TOOL_NAMES.cashflow:
    case TOOL_NAMES.cashflowV2:
      return <CashflowBody result={result} />;
    case TOOL_NAMES.recurring:
      return <RecurringBody result={result} />;
    case TOOL_NAMES.spendingChanges:
      return <SpendingChangesBody result={result} />;
    case TOOL_NAMES.anomalies:
      return <AnomaliesBody result={result} />;
    case TOOL_NAMES.counterparties:
      return <CounterpartiesBody toolCall={toolCall} />;
    case TOOL_NAMES.search:
      return <SearchBody result={result} />;
    case TOOL_NAMES.baseline:
      return <BaselineBody result={result} />;
    case TOOL_NAMES.moneyReview:
      return <MoneyReviewBody result={result} />;
    case TOOL_NAMES.forecast:
      return <ForecastBody result={result} />;
    default:
      return (
        <p className="text-sm text-muted-foreground">
          Data retrieved from your account.
        </p>
      );
  }
}

function BaselineBody({ result }: { result: Record<string, unknown> }) {
  if (result.available === false) {
    return (
      <p className="text-sm text-muted-foreground">
        {typeof result.message === "string"
          ? result.message
          : "No baseline available yet."}
      </p>
    );
  }
  const currency =
    typeof result.currency === "string" ? result.currency : "EUR";
  const free =
    typeof result.sustainable_free_cashflow === "string"
      ? result.sustainable_free_cashflow
      : undefined;
  const income =
    typeof result.regular_monthly_income === "string"
      ? result.regular_monthly_income
      : undefined;
  return (
    <div className="space-y-1 text-sm">
      <p>
        Free cashflow{" "}
        <span className="font-medium tabular-nums">
          {formatMoney(free, currency)}
        </span>
        /month
      </p>
      <p className="text-xs text-muted-foreground">
        Income {formatMoney(income, currency)} · status{" "}
        {typeof result.status === "string" ? result.status : "—"}
      </p>
    </div>
  );
}

function MoneyReviewBody({ result }: { result: Record<string, unknown> }) {
  if (result.available === false) {
    return (
      <p className="text-sm text-muted-foreground">
        {typeof result.message === "string"
          ? result.message
          : "No money review yet."}
      </p>
    );
  }
  const summary =
    typeof result.summary === "string" ? result.summary : "Review ready.";
  const findings = Array.isArray(result.findings) ? result.findings : [];
  return (
    <div className="space-y-2">
      <p className="text-sm text-foreground">{summary}</p>
      {findings.length > 0 ? (
        <ul className="space-y-1 border-t border-border/60 pt-2 text-xs text-muted-foreground">
          {findings.slice(0, 3).map((row, index) => {
            const record = row as Record<string, unknown>;
            const title =
              typeof record.title === "string" ? record.title : "Finding";
            return (
              <li key={`${title}-${index}`} className="truncate text-foreground">
                {title}
              </li>
            );
          })}
        </ul>
      ) : null}
    </div>
  );
}

function ForecastBody({ result }: { result: Record<string, unknown> }) {
  if (result.available === false) {
    return (
      <p className="text-sm text-muted-foreground">
        {typeof result.message === "string"
          ? result.message
          : "No forecast available yet."}
      </p>
    );
  }
  const currency =
    typeof result.currency === "string" ? result.currency : "EUR";
  const min =
    typeof result.min_balance === "string" ? result.min_balance : undefined;
  const ending =
    typeof result.ending_balance === "string"
      ? result.ending_balance
      : undefined;
  return (
    <div className="space-y-1 text-sm">
      <p>
        Low point{" "}
        <span className="font-medium tabular-nums">
          {formatMoney(min, currency)}
        </span>
      </p>
      <p className="text-xs text-muted-foreground">
        Ending {formatMoney(ending, currency)}
      </p>
    </div>
  );
}

function RecurringBody({ result }: { result: Record<string, unknown> }) {
  const monthly = typeof result.monthly_total === "string" ? result.monthly_total : undefined;
  const currency = typeof result.currency === "string" ? result.currency : "EUR";
  const rows = Array.isArray(result.series) ? result.series : [];

  return (
    <div className="space-y-2">
      <p className="text-sm text-muted-foreground">
        About {formatMoney(monthly, currency)} / month across {rows.length} series
      </p>
      <ul className="space-y-1.5 border-t border-border/60 pt-2">
        {rows.slice(0, 5).map((row, index) => {
          const record = row as Record<string, unknown>;
          const name =
            typeof record.display_name === "string" ? record.display_name : "Series";
          const amount =
            typeof record.monthly_amount === "string"
              ? record.monthly_amount
              : undefined;
          return (
            <li
              key={`${name}-${index}`}
              className="flex items-baseline justify-between gap-3 text-xs"
            >
              <span className="min-w-0 truncate text-foreground">{name}</span>
              <span className="shrink-0 tabular-nums text-muted-foreground">
                {formatMoney(amount, currency)}
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}

function SpendingChangesBody({ result }: { result: Record<string, unknown> }) {
  const currency = typeof result.currency === "string" ? result.currency : "EUR";
  const current =
    typeof result.current_expenses === "string" ? result.current_expenses : undefined;
  const previous =
    typeof result.previous_expenses === "string" ? result.previous_expenses : undefined;
  const delta =
    typeof result.expenses_delta === "string" ? result.expenses_delta : undefined;

  return (
    <dl className="grid grid-cols-3 gap-2 text-center">
      <Metric label="Previous" value={formatMoney(previous, currency)} />
      <Metric label="Current" value={formatMoney(current, currency)} emphasis />
      <Metric label="Delta" value={formatMoney(delta, currency)} />
    </dl>
  );
}

function AnomaliesBody({ result }: { result: Record<string, unknown> }) {
  const findings = Array.isArray(result.findings) ? result.findings : [];
  if (findings.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">No anomalies in this period.</p>
    );
  }
  return (
    <ul className="space-y-1.5">
      {findings.slice(0, 5).map((row, index) => {
        const record = row as Record<string, unknown>;
        const title = typeof record.title === "string" ? record.title : "Finding";
        return (
          <li key={`${title}-${index}`} className="text-sm text-foreground">
            {title}
          </li>
        );
      })}
    </ul>
  );
}

function CashflowBody({ result }: { result: Record<string, unknown> }) {
  const data = asCashflowResult(result);
  const currency = data.currency ?? "EUR";

  return (
    <dl className="grid grid-cols-3 gap-2 text-center">
      <Metric label="Income" value={formatMoney(data.income, currency)} />
      <Metric label="Expenses" value={formatMoney(data.expenses, currency)} emphasis />
      <Metric label="Net" value={formatMoney(data.net, currency)} />
    </dl>
  );
}

function CounterpartiesBody({ toolCall }: { toolCall: ChatToolCall }) {
  const data = asCounterpartiesResult(toolCall.result);
  const rows = data.counterparties ?? [];

  if (rows.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No outgoing payments in this period.
      </p>
    );
  }

  return (
    <ol className="space-y-2">
      {rows.map((row, index) => {
        const label = row.counterparty?.trim() || "Unknown";

        return (
          <li
            key={`${label}-${index}`}
            className="flex items-center justify-between gap-3 text-sm"
          >
            <div className="min-w-0">
              <p className="truncate font-medium">{label}</p>
              <p className="text-xs text-muted-foreground">
                {row.transaction_count ?? 0} transaction
                {(row.transaction_count ?? 0) === 1 ? "" : "s"}
              </p>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              <span className="tabular-nums text-sm">
                {formatMoney(row.total_spent, row.currency ?? "EUR")}
              </span>
              <SourceLink
                search={buildTransactionSearchFromToolCall(toolCall, {
                  counterparty: label,
                })}
                label="Open"
                className="h-7 px-2 text-xs"
              />
            </div>
          </li>
        );
      })}
    </ol>
  );
}

function SearchBody({ result }: { result: Record<string, unknown> }) {
  const data = asSearchResult(result);
  const total = data.total ?? data.transactions?.length ?? 0;
  const preview = (data.transactions ?? []).slice(0, 3);

  return (
    <div className="space-y-2">
      <p className="text-sm text-muted-foreground">
        {total === 1 ? "1 matching transaction" : `${total} matching transactions`}
      </p>
      {preview.length > 0 ? (
        <ul className="space-y-1.5 border-t border-border/60 pt-2">
          {preview.map((row, index) => (
            <li
              key={`${row.booking_date}-${row.counterparty}-${index}`}
              className="flex items-baseline justify-between gap-3 text-xs"
            >
              <span className="min-w-0 truncate text-muted-foreground">
                <span className="text-foreground">{row.booking_date}</span>
                {" · "}
                {row.counterparty || row.purpose || "Transaction"}
              </span>
              <span className="shrink-0 tabular-nums">
                {formatMoney(row.amount, row.currency ?? "EUR")}
              </span>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function Metric({
  label,
  value,
  emphasis = false,
}: {
  label: string;
  value: string;
  emphasis?: boolean;
}) {
  return (
    <div className="rounded-lg bg-muted/50 px-2 py-2">
      <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">
        {label}
      </dt>
      <dd
        className={cn(
          "mt-1 text-sm tabular-nums",
          emphasis ? "font-semibold text-foreground" : "text-foreground/90",
        )}
      >
        {value}
      </dd>
    </div>
  );
}
