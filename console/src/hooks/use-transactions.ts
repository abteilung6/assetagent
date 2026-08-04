import { useQuery } from "@tanstack/react-query";

import { getTransactionsOptions } from "@/api/@tanstack/react-query.gen";
import type { GetTransactionsData } from "@/api/types.gen";
import {
  defaultTransactionSearchParams,
  toTransactionListQuery,
} from "@/pages/transactions/search-params";

export const defaultTransactionListQuery: NonNullable<
  GetTransactionsData["query"]
> = toTransactionListQuery(defaultTransactionSearchParams);

export function useTransactions(
  query: NonNullable<GetTransactionsData["query"]> = defaultTransactionListQuery,
  options?: { enabled?: boolean },
) {
  return useQuery({
    ...getTransactionsOptions({ query }),
    enabled: options?.enabled ?? true,
  });
}
