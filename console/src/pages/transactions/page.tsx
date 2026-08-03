import { useNavigate } from "@tanstack/react-router";
import type React from "react";
import { useCallback, useState } from "react";

import type { Transaction } from "@/api/types.gen";
import { TransactionDetailSheet } from "@/components/transaction-detail/sheet";
import { TransactionFilters } from "@/components/transaction-filters/filters";
import { TransactionPagination } from "@/components/transaction-table/pagination";
import { TransactionTable } from "@/components/transaction-table/transaction-table";
import { TransactionTableSkeleton } from "@/components/transaction-table/transaction-table-skeleton";
import { Button } from "@/components/ui/button";
import { useTransactions } from "@/hooks/use-transactions";
import { transactionsRoute } from "@/router";

import {
  type TransactionSearchParams,
  toTransactionListQuery,
} from "./search-params";

const TransactionsPage: React.FC = () => {
  const search = transactionsRoute.useSearch();
  const navigate = useNavigate();
  const [selectedTransaction, setSelectedTransaction] =
    useState<Transaction | null>(null);
  const [sheetOpen, setSheetOpen] = useState(false);
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

  const handleRowClick = useCallback((transaction: Transaction) => {
    setSelectedTransaction(transaction);
    setSheetOpen(true);
  }, []);

  const handleSheetOpenChange = useCallback((open: boolean) => {
    setSheetOpen(open);
    if (!open) {
      setSelectedTransaction(null);
    }
  }, []);

  const handleTransactionChange = useCallback((transaction: Transaction) => {
    setSelectedTransaction(transaction);
  }, []);

  const isGloballyEmpty =
    !isPending &&
    !isError &&
    data !== undefined &&
    data.pagination.total === 0 &&
    search.offset === 0;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto">
      <TransactionFilters params={search} onApply={setSearchParams} />

      {isPending ? (
        <div className="min-w-0 overflow-x-auto">
          <TransactionTableSkeleton />
        </div>
      ) : isError ? (
        <p className="text-destructive">Failed to load transactions.</p>
      ) : isGloballyEmpty ? (
        <div className="flex flex-1 flex-col items-start justify-center gap-3 py-16">
          <div className="space-y-1">
            <p className="text-sm font-medium">No transactions yet</p>
            <p className="max-w-md text-sm text-muted-foreground">
              Import a Sparkasse CSV to populate your ledger. You can preview
              the file before anything is saved.
            </p>
          </div>
          <Button
            type="button"
            onClick={() => {
              void navigate({ to: "/imports" });
            }}
          >
            Import a statement
          </Button>
        </div>
      ) : (
        <div className="flex min-w-0 flex-col gap-4">
          {data.data.length === 0 ? (
            <p className="text-muted-foreground">No transactions on this page.</p>
          ) : (
            <div className="min-w-0 overflow-x-auto">
              <TransactionTable
                transactions={data.data}
                onRowClick={handleRowClick}
              />
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

      <TransactionDetailSheet
        transaction={selectedTransaction}
        open={sheetOpen}
        onOpenChange={handleSheetOpenChange}
        onTransactionChange={handleTransactionChange}
      />
    </div>
  );
};

export default TransactionsPage;
