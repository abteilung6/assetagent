import type React from "react";
import { useState } from "react";

import type { Transaction } from "@/api/types.gen";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useSetTransactionOneOff } from "@/hooks/use-set-transaction-one-off";
import { apiErrorMessage } from "@/lib/api-error";

type TransactionDetailField = {
  key: keyof Transaction;
  label: string;
};

const TRANSACTION_DETAIL_FIELDS: TransactionDetailField[] = [
  { key: "booking_date", label: "Booking date" },
  { key: "value_date", label: "Value date" },
  { key: "amount", label: "Amount" },
  { key: "currency", label: "Currency" },
  { key: "counterparty", label: "Counterparty" },
  { key: "purpose", label: "Purpose" },
  { key: "booking_text", label: "Booking text" },
  { key: "order_account", label: "Order account" },
  { key: "counterparty_iban", label: "Counterparty IBAN" },
  { key: "counterparty_bic", label: "Counterparty BIC" },
  { key: "creditor_id", label: "Creditor ID" },
  { key: "mandate_reference", label: "Mandate reference" },
  { key: "end_to_end_reference", label: "End-to-end reference" },
  { key: "collection_reference", label: "Collection reference" },
  {
    key: "direct_debit_original_amount",
    label: "Direct debit original amount",
  },
  {
    key: "chargeback_expense_reimbursement",
    label: "Chargeback expense reimbursement",
  },
  { key: "info", label: "Info" },
  { key: "id", label: "ID" },
];

function displayValue(value: string | boolean | null | undefined) {
  if (typeof value === "boolean") {
    return value ? "Yes" : "No";
  }
  if (value === null || value === undefined || value === "") {
    return "—";
  }

  return value;
}

type TransactionDetailSheetProps = {
  transaction: Transaction | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onTransactionChange?: (transaction: Transaction) => void;
};

export const TransactionDetailSheet: React.FC<TransactionDetailSheetProps> = ({
  transaction,
  open,
  onOpenChange,
  onTransactionChange,
}) => {
  const setOneOff = useSetTransactionOneOff();
  const [error, setError] = useState<string | null>(null);

  const handleToggleOneOff = () => {
    if (!transaction) {
      return;
    }
    setError(null);
    const next = !transaction.one_off;
    setOneOff.mutate(
      {
        path: { transaction_id: transaction.id },
        body: { one_off: next },
      },
      {
        onSuccess: (updated) => {
          onTransactionChange?.(updated);
        },
        onError: (err) => {
          setError(apiErrorMessage(err));
        },
      },
    );
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>
            {transaction?.counterparty || "Transaction details"}
          </SheetTitle>
          {transaction ? (
            <SheetDescription>
              {transaction.booking_date} · {transaction.amount}{" "}
              {transaction.currency}
              {transaction.one_off ? " · One-off" : null}
            </SheetDescription>
          ) : null}
        </SheetHeader>
        {transaction ? (
          <div className="grid gap-4 px-4 pb-4">
            <div className="grid gap-2">
              <p className="text-sm font-medium">Typical spend</p>
              <p className="text-xs text-muted-foreground">
                {transaction.one_off
                  ? "This payment is excluded from baseline variable spend and cashflow charts."
                  : "Mark unusual payments (e.g. down payments) so they do not distort typical monthly spend."}
              </p>
              <Button
                type="button"
                variant={transaction.one_off ? "outline" : "secondary"}
                size="sm"
                className="w-fit"
                disabled={setOneOff.isPending}
                onClick={handleToggleOneOff}
              >
                {setOneOff.isPending
                  ? "Saving…"
                  : transaction.one_off
                    ? "Undo one-off"
                    : "Treat as one-off"}
              </Button>
              {error ? (
                <p className="text-xs text-destructive" role="alert">
                  {error}
                </p>
              ) : null}
            </div>
            <dl className="grid gap-3">
              {TRANSACTION_DETAIL_FIELDS.map(({ key, label }) => (
                <div key={key} className="grid gap-1">
                  <dt className="text-xs text-muted-foreground">{label}</dt>
                  <dd className="text-sm break-words">
                    {displayValue(transaction[key])}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        ) : null}
      </SheetContent>
    </Sheet>
  );
};
