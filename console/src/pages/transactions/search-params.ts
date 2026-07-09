import type { GetTransactionsData } from "@/api/types.gen";

export const DEFAULT_TRANSACTION_LIMIT = 50;
export const DEFAULT_TRANSACTION_OFFSET = 0;
export const MAX_TRANSACTION_LIMIT = 200;
export const TRANSACTION_PAGE_SIZE_OPTIONS = [25, 50, 100] as const;

export const DEFAULT_TRANSACTION_SORT = "booking_date" as const;
export const DEFAULT_TRANSACTION_ORDER = "desc" as const;

export const TRANSACTION_SORT_FIELDS = [
  "booking_date",
  "amount",
  "counterparty",
] as const;

export const TRANSACTION_SORT_ORDERS = ["asc", "desc"] as const;

export type TransactionSortField = (typeof TRANSACTION_SORT_FIELDS)[number];
export type TransactionSortOrder = (typeof TRANSACTION_SORT_ORDERS)[number];

export type TransactionSearchParams = {
  limit: number;
  offset: number;
  from?: string;
  to?: string;
  q?: string;
  account?: string;
  counterparty?: string;
  min_amount?: string;
  max_amount?: string;
  sort: TransactionSortField;
  order: TransactionSortOrder;
};

export type TransactionFilterFields = {
  from: string;
  to: string;
  q: string;
  account: string;
  counterparty: string;
  min_amount: string;
  max_amount: string;
  sort: TransactionSortField;
  order: TransactionSortOrder;
};

export const defaultTransactionSearchParams: TransactionSearchParams = {
  limit: DEFAULT_TRANSACTION_LIMIT,
  offset: DEFAULT_TRANSACTION_OFFSET,
  sort: DEFAULT_TRANSACTION_SORT,
  order: DEFAULT_TRANSACTION_ORDER,
};

export function parseTransactionSearchParams(
  search: Record<string, unknown>,
): TransactionSearchParams {
  return {
    limit: parseLimit(search.limit),
    offset: parseOffset(search.offset),
    from: parseOptionalString(search.from),
    to: parseOptionalString(search.to),
    q: parseOptionalString(search.q),
    account: parseOptionalString(search.account),
    counterparty: parseOptionalString(search.counterparty),
    min_amount: parseOptionalString(search.min_amount),
    max_amount: parseOptionalString(search.max_amount),
    sort: parseSort(search.sort),
    order: parseOrder(search.order),
  };
}

export function pickFilterFields(
  params: TransactionSearchParams,
): TransactionFilterFields {
  return {
    from: params.from ?? "",
    to: params.to ?? "",
    q: params.q ?? "",
    account: params.account ?? "",
    counterparty: params.counterparty ?? "",
    min_amount: params.min_amount ?? "",
    max_amount: params.max_amount ?? "",
    sort: params.sort,
    order: params.order,
  };
}

export function filterFieldsToSearchParams(
  fields: TransactionFilterFields,
  pagination: Pick<TransactionSearchParams, "limit" | "offset">,
): TransactionSearchParams {
  return {
    ...pagination,
    sort: fields.sort,
    order: fields.order,
    from: trimToUndefined(fields.from),
    to: trimToUndefined(fields.to),
    q: trimToUndefined(fields.q),
    account: trimToUndefined(fields.account),
    counterparty: trimToUndefined(fields.counterparty),
    min_amount: trimToUndefined(fields.min_amount),
    max_amount: trimToUndefined(fields.max_amount),
  };
}

export function toTransactionListQuery(
  params: TransactionSearchParams,
): NonNullable<GetTransactionsData["query"]> {
  const query: NonNullable<GetTransactionsData["query"]> = {
    limit: params.limit,
    offset: params.offset,
    sort: params.sort,
    order: params.order,
  };

  if (params.from) {
    query.from = params.from;
  }
  if (params.to) {
    query.to = params.to;
  }
  if (params.q) {
    query.q = params.q;
  }
  if (params.account) {
    query.account = params.account;
  }
  if (params.counterparty) {
    query.counterparty = params.counterparty;
  }
  if (params.min_amount) {
    query.min_amount = params.min_amount;
  }
  if (params.max_amount) {
    query.max_amount = params.max_amount;
  }

  return query;
}

export function hasActiveFilters(params: TransactionSearchParams): boolean {
  return Boolean(
    params.from ||
      params.to ||
      params.q ||
      params.account ||
      params.counterparty ||
      params.min_amount ||
      params.max_amount ||
      params.sort !== DEFAULT_TRANSACTION_SORT ||
      params.order !== DEFAULT_TRANSACTION_ORDER,
  );
}

function parseLimit(value: unknown): number {
  const parsed = parsePositiveInt(value);
  if (parsed === undefined || parsed < 1 || parsed > MAX_TRANSACTION_LIMIT) {
    return DEFAULT_TRANSACTION_LIMIT;
  }
  return parsed;
}

function parseOffset(value: unknown): number {
  const parsed = parsePositiveInt(value);
  if (parsed === undefined) {
    return DEFAULT_TRANSACTION_OFFSET;
  }
  return parsed;
}

function parsePositiveInt(value: unknown): number | undefined {
  if (value === undefined || value === null || value === "") {
    return undefined;
  }

  const parsed = typeof value === "number" ? value : Number(value);
  if (!Number.isInteger(parsed) || parsed < 0) {
    return undefined;
  }

  return parsed;
}

function parseOptionalString(value: unknown): string | undefined {
  if (value === undefined || value === null) {
    return undefined;
  }

  const parsed = trimToUndefined(String(value));
  return parsed;
}

function parseSort(value: unknown): TransactionSortField {
  if (
    typeof value === "string" &&
    TRANSACTION_SORT_FIELDS.includes(value as TransactionSortField)
  ) {
    return value as TransactionSortField;
  }

  return DEFAULT_TRANSACTION_SORT;
}

function parseOrder(value: unknown): TransactionSortOrder {
  if (
    typeof value === "string" &&
    TRANSACTION_SORT_ORDERS.includes(value as TransactionSortOrder)
  ) {
    return value as TransactionSortOrder;
  }

  return DEFAULT_TRANSACTION_ORDER;
}

function trimToUndefined(value: string): string | undefined {
  const trimmed = value.trim();
  return trimmed || undefined;
}
