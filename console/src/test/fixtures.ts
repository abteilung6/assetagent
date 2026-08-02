import type { ImportPreviewResponse, ImportRun, Transaction, TransactionListResponse } from "@/api/types.gen";

export function sampleImportRun(
  overrides: Partial<ImportRun> = {},
): ImportRun {
  return {
    id: "33333333-3333-3333-3333-333333333333",
    account_id: "22222222-2222-2222-2222-222222222222",
    source_filename: "minimal.csv",
    status: "committed",
    row_total: 6,
    row_valid: 6,
    row_inserted: 6,
    row_duplicate: 0,
    created_at: "2026-08-02T12:00:00.000Z",
    committed_at: "2026-08-02T12:00:00.000Z",
    ...overrides,
  };
}

export function sampleImportPreview(
  overrides: Partial<ImportPreviewResponse> = {},
): ImportPreviewResponse {
  return {
    file_hash: "abc123",
    source_filename: "minimal.csv",
    parser_name: "sparkasse",
    parser_version: "sparkasse-1",
    period: { from: "2025-12-17", to: "2025-12-30" },
    suggested_account: "DE89…3000",
    row_total: 6,
    row_valid: 6,
    row_invalid: 0,
    sample_rows: [
      {
        line: 2,
        booking_date: "2025-12-30",
        counterparty: "PayPal Europe",
        purpose: "Payment to Example Shop",
        amount: "-23.97",
        currency: "EUR",
      },
      {
        line: 4,
        booking_date: "2025-12-29",
        counterparty: "Example Employer GmbH",
        purpose: "Example Employer Salary",
        amount: "56000.00",
        currency: "EUR",
      },
    ],
    invalid_rows: [],
    warnings: [],
    ...overrides,
  };
}

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
