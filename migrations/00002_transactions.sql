-- +goose Up
-- +goose StatementBegin
CREATE TABLE transactions (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_account                    TEXT NOT NULL,
    booking_date                     DATE NOT NULL,
    value_date                       DATE NOT NULL,
    booking_text                     TEXT NOT NULL,
    purpose                          TEXT NOT NULL DEFAULT '',
    creditor_id                      TEXT NOT NULL DEFAULT '',
    mandate_reference                TEXT NOT NULL DEFAULT '',
    end_to_end_reference             TEXT NOT NULL DEFAULT '',
    collection_reference             TEXT NOT NULL DEFAULT '',
    direct_debit_original_amount     TEXT NOT NULL DEFAULT '',
    chargeback_expense_reimbursement TEXT NOT NULL DEFAULT '',
    counterparty                     TEXT NOT NULL DEFAULT '',
    counterparty_iban                TEXT,
    counterparty_bic                 TEXT,
    amount                           NUMERIC(18,2) NOT NULL,
    currency                         TEXT NOT NULL DEFAULT 'EUR',
    info                             TEXT NOT NULL DEFAULT ''
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transactions;
-- +goose StatementEnd
