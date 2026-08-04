import type { ChatToolCall } from "@/api/types.gen";
import {
  defaultTransactionSearchParams,
  type TransactionSearchParams,
} from "@/pages/transactions/search-params";

export const TOOL_NAMES = {
  cashflow: "get_cashflow",
  cashflowV2: "get_cashflow_v2",
  monthCashflow: "get_month_cashflow",
  categorySpend: "get_category_spend",
  oneOffImpact: "get_one_off_impact",
  recurring: "get_recurring_costs",
  spendingChanges: "get_spending_changes",
  anomalies: "get_anomalies",
  counterparties: "get_top_counterparties",
  search: "search_transactions",
  baseline: "get_baseline",
  moneyReview: "get_money_review",
  forecast: "get_forecast",
  suggestReviewCategories: "suggest_review_categories",
  needsReviewSummary: "get_needs_review_summary",
} as const;

export type CashflowResult = {
  income?: string;
  expenses?: string;
  net?: string;
  currency?: string;
  error?: string;
};

export type CounterpartyRow = {
  counterparty?: string;
  total_spent?: string;
  transaction_count?: number;
  currency?: string;
};

export type CounterpartiesResult = {
  counterparties?: CounterpartyRow[];
  error?: string;
};

export type SearchTransactionRow = {
  booking_date?: string;
  counterparty?: string;
  purpose?: string;
  amount?: string;
  currency?: string;
};

export type SearchResult = {
  total?: number;
  transactions?: SearchTransactionRow[];
  error?: string;
};

export function toolHasError(result: Record<string, unknown>): boolean {
  return typeof result.error === "string" && result.error.length > 0;
}

export function toolErrorMessage(result: Record<string, unknown>): string {
  const raw = typeof result.error === "string" ? result.error : "Unknown error";
  return humanizeToolError(raw);
}

export function humanizeToolError(message: string): string {
  if (message.includes("cannot unmarshal string into Go struct field")) {
    return "The assistant used an invalid parameter format. Try asking again.";
  }
  if (message.includes("invalid limit")) {
    return "The row limit must be a whole number.";
  }
  if (message === "to must be on or after from") {
    return "The date range was reversed. The end date must be on or after the start date.";
  }
  if (message.includes("from and to are required")) {
    return "A start and end date are required for this question.";
  }
  return message;
}

export function toolDisplayName(name: string): string {
  switch (name) {
    case TOOL_NAMES.cashflow:
    case TOOL_NAMES.cashflowV2:
    case TOOL_NAMES.monthCashflow:
      return "Spending summary";
    case TOOL_NAMES.categorySpend:
      return "Category spend";
    case TOOL_NAMES.oneOffImpact:
      return "One-off impact";
    case TOOL_NAMES.recurring:
      return "Recurring costs";
    case TOOL_NAMES.spendingChanges:
      return "Spending changes";
    case TOOL_NAMES.anomalies:
      return "Anomalies";
    case TOOL_NAMES.counterparties:
      return "Top counterparties";
    case TOOL_NAMES.search:
      return "Transaction search";
    case TOOL_NAMES.baseline:
      return "Baseline";
    case TOOL_NAMES.moneyReview:
      return "Money review";
    case TOOL_NAMES.forecast:
      return "Forecast";
    case TOOL_NAMES.suggestReviewCategories:
      return "Category suggestions";
    case TOOL_NAMES.needsReviewSummary:
      return "Needs review";
    default:
      return name.replaceAll("_", " ");
  }
}

export function isPlanArtifactTool(name: string): boolean {
  return (
    name === TOOL_NAMES.baseline ||
    name === TOOL_NAMES.moneyReview ||
    name === TOOL_NAMES.forecast ||
    name === TOOL_NAMES.suggestReviewCategories ||
    name === TOOL_NAMES.needsReviewSummary
  );
}

export function isMonthSourceTool(name: string): boolean {
  return (
    name === TOOL_NAMES.monthCashflow ||
    name === TOOL_NAMES.categorySpend ||
    name === TOOL_NAMES.oneOffImpact
  );
}

export function isTransactionSourceTool(name: string): boolean {
  return !isPlanArtifactTool(name) && !isMonthSourceTool(name);
}

export function readOptionalString(
  input: Record<string, unknown>,
  key: string,
): string | undefined {
  const value = input[key];
  if (typeof value !== "string") {
    return undefined;
  }
  const trimmed = value.trim();
  return trimmed || undefined;
}

export function buildTransactionSearchFromToolCall(
  toolCall: ChatToolCall,
  options?: { counterparty?: string },
): TransactionSearchParams {
  const { input } = toolCall;
  const from = readOptionalString(input, "from");
  const to = readOptionalString(input, "to");
  const q = readOptionalString(input, "q");
  const counterparty =
    options?.counterparty?.trim() || readOptionalString(input, "counterparty");

  return {
    ...defaultTransactionSearchParams,
    offset: 0,
    ...(from ? { from } : {}),
    ...(to ? { to } : {}),
    ...(q ? { q } : {}),
    ...(counterparty ? { counterparty } : {}),
  };
}

export function formatShortDate(isoDate: string): string {
  const parsed = parseISODate(isoDate);
  if (!parsed) {
    return isoDate;
  }

  return parsed.toLocaleDateString(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
}

export function formatDateRange(from?: string, to?: string): string {
  if (from && to) {
    if (from === to) {
      return formatShortDate(from);
    }
    return `${formatShortDate(from)} – ${formatShortDate(to)}`;
  }
  if (from) {
    return `From ${formatShortDate(from)}`;
  }
  if (to) {
    return `Until ${formatShortDate(to)}`;
  }
  return "All dates";
}

export function formatMoney(amount: string | undefined, currency = "EUR"): string {
  if (!amount) {
    return `— ${currency}`;
  }
  return `${amount} ${currency}`;
}

export function asCashflowResult(result: Record<string, unknown>): CashflowResult {
  return {
    income: stringField(result.income),
    expenses: stringField(result.expenses),
    net: stringField(result.net),
    currency: stringField(result.currency) ?? "EUR",
    error: stringField(result.error),
  };
}

export function asCounterpartiesResult(
  result: Record<string, unknown>,
): CounterpartiesResult {
  const rows = result.counterparties;
  if (!Array.isArray(rows)) {
    return { error: stringField(result.error) };
  }

  return {
    counterparties: rows.map((row) => {
      const record = row as Record<string, unknown>;
      return {
        counterparty: stringField(record.counterparty),
        total_spent: stringField(record.total_spent),
        transaction_count: numberField(record.transaction_count),
        currency: stringField(record.currency) ?? "EUR",
      };
    }),
    error: stringField(result.error),
  };
}

export function asSearchResult(result: Record<string, unknown>): SearchResult {
  const rows = result.transactions;
  const transactions = Array.isArray(rows)
    ? rows.map((row) => {
        const record = row as Record<string, unknown>;
        return {
          booking_date: stringField(record.booking_date),
          counterparty: stringField(record.counterparty),
          purpose: stringField(record.purpose),
          amount: stringField(record.amount),
          currency: stringField(record.currency) ?? "EUR",
        };
      })
    : undefined;

  return {
    total: numberField(result.total),
    transactions,
    error: stringField(result.error),
  };
}

function parseISODate(value: string): Date | undefined {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) {
    return undefined;
  }

  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const parsed = new Date(year, month - 1, day);
  if (
    parsed.getFullYear() !== year ||
    parsed.getMonth() !== month - 1 ||
    parsed.getDate() !== day
  ) {
    return undefined;
  }

  return parsed;
}

function stringField(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function numberField(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}
