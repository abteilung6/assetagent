import type React from "react";

import type { Transaction } from "@/api/types.gen";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";

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

function displayValue(value: string | null | undefined) {
  if (value === null || value === undefined || value === "") {
    return "—";
  }

  return value;
}

type TransactionDetailSheetProps = {
  transaction: Transaction | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export const TransactionDetailSheet: React.FC<TransactionDetailSheetProps> = ({
  transaction,
  open,
  onOpenChange,
}) => {
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
            </SheetDescription>
          ) : null}
        </SheetHeader>
        {transaction ? (
          <dl className="grid gap-3 px-4 pb-4">
            {TRANSACTION_DETAIL_FIELDS.map(({ key, label }) => (
              <div key={key} className="grid gap-1">
                <dt className="text-xs text-muted-foreground">{label}</dt>
                <dd className="text-sm break-words">
                  {displayValue(transaction[key])}
                </dd>
              </div>
            ))}
          </dl>
        ) : null}
      </SheetContent>
    </Sheet>
  );
};
