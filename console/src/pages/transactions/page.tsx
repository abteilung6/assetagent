import type React from "react";

import { TransactionTable } from "@/components/transaction-table/transaction-table";
import { TransactionTableSkeleton } from "@/components/transaction-table/transaction-table-skeleton";
import { useTransactions } from "@/hooks/use-transactions";

const TransactionsPage: React.FC = () => {
  const { data, isPending, isError } = useTransactions();

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold">Transactions</h1>
        <p className="text-muted-foreground">
          Imported bank transactions from your account.
        </p>
      </div>

      {isPending ? (
        <TransactionTableSkeleton />
      ) : isError ? (
        <p className="text-destructive">Failed to load transactions.</p>
      ) : data.data.length === 0 ? (
        <p className="text-muted-foreground">No transactions found.</p>
      ) : (
        <TransactionTable transactions={data.data} />
      )}
    </div>
  );
};

export default TransactionsPage;
