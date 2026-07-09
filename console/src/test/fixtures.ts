import type { Transaction, TransactionListResponse } from "@/api/types.gen";

export function sampleTransaction(
  overrides: Partial<Transaction> = {},
): Transaction {
  return {
    id: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
    order_account: "6011880043",
    booking_date: "2025-12-30",
    value_date: "2025-12-30",
    booking_text: "CARD_PAYMENT",
    purpose: "REWE Dortmund",
    creditor_id: "",
    mandate_reference: "",
    end_to_end_reference: "",
    collection_reference: "",
    direct_debit_original_amount: "",
    chargeback_expense_reimbursement: "",
    counterparty: "REWE Markt GmbH",
    amount: "-42.50",
    currency: "EUR",
    info: "",
    ...overrides,
  };
}

export function sampleTransactionList(
  transactions: Transaction[] = [sampleTransaction()],
): TransactionListResponse {
  return {
    data: transactions,
    pagination: {
      limit: 50,
      offset: 0,
      total: transactions.length,
    },
  };
}
