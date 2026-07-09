import { useQuery } from "@tanstack/react-query";

import { getTransactionsOptions } from "@/api/@tanstack/react-query.gen";
import type { GetTransactionsData } from "@/api/types.gen";
import { toTransactionListQuery } from "@/pages/transactions/search-params";

export const defaultTransactionListQuery: NonNullable<
  GetTransactionsData["query"]
> = toTransactionListQuery({ limit: 50, offset: 0 });

export function useTransactions(
  query: NonNullable<GetTransactionsData["query"]> = defaultTransactionListQuery,
) {
  return useQuery(getTransactionsOptions({ query }));
}
