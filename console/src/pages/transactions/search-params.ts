import type { GetTransactionsData } from "@/api/types.gen";

export const DEFAULT_TRANSACTION_LIMIT = 50;
export const DEFAULT_TRANSACTION_OFFSET = 0;
export const MAX_TRANSACTION_LIMIT = 200;
export const TRANSACTION_PAGE_SIZE_OPTIONS = [25, 50, 100] as const;

export type TransactionSearchParams = {
  limit: number;
  offset: number;
};

export const defaultTransactionSearchParams: TransactionSearchParams = {
  limit: DEFAULT_TRANSACTION_LIMIT,
  offset: DEFAULT_TRANSACTION_OFFSET,
};

export function parseTransactionSearchParams(
  search: Record<string, unknown>,
): TransactionSearchParams {
  return {
    limit: parseLimit(search.limit),
    offset: parseOffset(search.offset),
  };
}

export function toTransactionListQuery(
  params: TransactionSearchParams,
): NonNullable<GetTransactionsData["query"]> {
  return {
    limit: params.limit,
    offset: params.offset,
    sort: "booking_date",
    order: "desc",
  };
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
