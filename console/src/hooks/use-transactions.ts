import { useQuery } from "@tanstack/react-query";

import { getTransactionsOptions } from "@/api/@tanstack/react-query.gen";
import type { GetTransactionsData } from "@/api/types.gen";

export const defaultTransactionListQuery: NonNullable<
  GetTransactionsData["query"]
> = {
  limit: 50,
  sort: "booking_date",
  order: "desc",
};

export function useTransactions(
  query: NonNullable<GetTransactionsData["query"]> = defaultTransactionListQuery,
) {
  return useQuery(getTransactionsOptions({ query }));
}
