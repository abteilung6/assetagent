import { useNavigate } from "@tanstack/react-router";
import type React from "react";
import { useCallback } from "react";

import { TransactionFilters } from "@/components/transaction-filters/filters";
import { TransactionPagination } from "@/components/transaction-table/pagination";
import { TransactionTable } from "@/components/transaction-table/transaction-table";
import { TransactionTableSkeleton } from "@/components/transaction-table/transaction-table-skeleton";
import { useTransactions } from "@/hooks/use-transactions";
import { transactionsRoute } from "@/router";

import {
  type TransactionSearchParams,
  toTransactionListQuery,
} from "./search-params";

const TransactionsPage: React.FC = () => {
  const search = transactionsRoute.useSearch();
  const navigate = useNavigate();
  const { data, isPending, isError } = useTransactions(
    toTransactionListQuery(search),
  );

  const setSearchParams = useCallback(
    (next: TransactionSearchParams) => {
      navigate({
        to: "/transactions",
        search: next,
      });
    },
    [navigate],
  );

  const setPagination = useCallback(
    (next: Partial<Pick<TransactionSearchParams, "limit" | "offset">>) => {
      setSearchParams({
        ...search,
        limit: next.limit ?? search.limit,
        offset: next.offset ?? search.offset,
      });
    },
    [search, setSearchParams],
  );

  return (
    <div className="flex min-w-0 w-full max-w-full flex-col gap-4">
      <TransactionFilters params={search} onApply={setSearchParams} />

      {isPending ? (
        <div className="min-w-0 overflow-x-auto">
          <TransactionTableSkeleton />
        </div>
      ) : isError ? (
        <p className="text-destructive">Failed to load transactions.</p>
      ) : (
        <div className="flex min-w-0 flex-col gap-4">
          {data.data.length === 0 ? (
            <p className="text-muted-foreground">No transactions on this page.</p>
          ) : (
            <div className="min-w-0 overflow-x-auto">
              <TransactionTable transactions={data.data} />
            </div>
          )}
          <TransactionPagination
            limit={search.limit}
            offset={search.offset}
            total={data.pagination.total}
            onChange={setPagination}
          />
        </div>
      )}
    </div>
  );
};

export default TransactionsPage;
