import { useNavigate } from "@tanstack/react-router";
import type React from "react";
import { useCallback } from "react";

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
  const { limit, offset } = transactionsRoute.useSearch();
  const navigate = useNavigate();
  const { data, isPending, isError } = useTransactions(
    toTransactionListQuery({ limit, offset }),
  );

  const setSearchParams = useCallback(
    (next: Partial<TransactionSearchParams>) => {
      navigate({
        to: "/transactions",
        search: {
          limit: next.limit ?? limit,
          offset: next.offset ?? offset,
        },
      });
    },
    [limit, navigate, offset],
  );

  return (
    <div className="flex min-w-0 w-full max-w-full flex-col gap-4">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold">Transactions</h1>
        <p className="text-muted-foreground">
          Imported bank transactions from your account.
        </p>
      </div>

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
            limit={limit}
            offset={offset}
            total={data.pagination.total}
            onChange={setSearchParams}
          />
        </div>
      )}
    </div>
  );
};

export default TransactionsPage;
