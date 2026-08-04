# Golden money households

Each subdirectory is one synthetic household used by `make golden`.

## Layout

```text
testdata/golden/<household>/
  *.csv            # Sparkasse export(s); one file per account (Auftragskonto)
  expected.json    # cent-exact assertions
```

## expected.json

| Field | Meaning |
|-------|---------|
| `period.from` / `to` | Inclusive booking-date window for cashflow |
| `confirm_transfers` | After scan, confirm all suggested pairs before asserting |
| `cashflow_raw` | `GetCashflow` (no transfer exclusion) |
| `cashflow_v2` | `GetCashflowV2` (confirmed transfers excluded) |
| `transfers.confirmed` | Number of confirmed pairs after the pipeline |
| `recurring` | Series that must appear after `classify recurring scan` |
| `baseline` | Five FinancialBaseline metrics after activating recurring series |
| `forecast` | Optional smoke: frozen `as_of` clock + `min_balance` after confirm |

## Adding a household

1. Copy an existing folder.
2. Build Sparkasse CSVs with unique `Kundenreferenz` values and distinct IBANs per account.
3. Fill `expected.json` by running the pipeline locally or computing decimals carefully.
4. For baseline values: `GOLDEN_DUMP=1 go test ./internal/evals -run TestDumpBaselineForecastValues -v`
5. Run `make golden` — a broken transfer exclusion must fail the couple_transfer household; a broken free-cashflow formula must fail baseline assertions.
