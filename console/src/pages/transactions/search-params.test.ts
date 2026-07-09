import { describe, expect, it } from "vitest";

import {
  DEFAULT_TRANSACTION_LIMIT,
  DEFAULT_TRANSACTION_OFFSET,
  DEFAULT_TRANSACTION_ORDER,
  DEFAULT_TRANSACTION_SORT,
  filterFieldsToSearchParams,
  parseTransactionSearchParams,
  toTransactionListQuery,
} from "./search-params";

describe("parseTransactionSearchParams", () => {
  it("returns defaults for empty search", () => {
    expect(parseTransactionSearchParams({})).toEqual({
      limit: DEFAULT_TRANSACTION_LIMIT,
      offset: DEFAULT_TRANSACTION_OFFSET,
      sort: DEFAULT_TRANSACTION_SORT,
      order: DEFAULT_TRANSACTION_ORDER,
    });
  });

  it("parses limit and offset from strings", () => {
    expect(parseTransactionSearchParams({ limit: "25", offset: "50" })).toEqual({
      limit: 25,
      offset: 50,
      sort: DEFAULT_TRANSACTION_SORT,
      order: DEFAULT_TRANSACTION_ORDER,
    });
  });

  it("parses filter and sort params", () => {
    expect(
      parseTransactionSearchParams({
        from: "2025-12-30",
        to: "2025-12-31",
        q: "Prime",
        account: "6011880043",
        counterparty: "REWE",
        min_amount: "-100",
        max_amount: "50",
        sort: "amount",
        order: "asc",
      }),
    ).toEqual({
      limit: DEFAULT_TRANSACTION_LIMIT,
      offset: DEFAULT_TRANSACTION_OFFSET,
      from: "2025-12-30",
      to: "2025-12-31",
      q: "Prime",
      account: "6011880043",
      counterparty: "REWE",
      min_amount: "-100",
      max_amount: "50",
      sort: "amount",
      order: "asc",
    });
  });

  it("falls back to defaults for invalid values", () => {
    expect(
      parseTransactionSearchParams({
        limit: "0",
        offset: "-1",
        sort: "invalid",
        order: "sideways",
      }),
    ).toEqual({
      limit: DEFAULT_TRANSACTION_LIMIT,
      offset: DEFAULT_TRANSACTION_OFFSET,
      sort: DEFAULT_TRANSACTION_SORT,
      order: DEFAULT_TRANSACTION_ORDER,
    });
  });

  it("falls back when limit exceeds API maximum", () => {
    expect(parseTransactionSearchParams({ limit: "999" })).toEqual({
      limit: DEFAULT_TRANSACTION_LIMIT,
      offset: DEFAULT_TRANSACTION_OFFSET,
      sort: DEFAULT_TRANSACTION_SORT,
      order: DEFAULT_TRANSACTION_ORDER,
    });
  });
});

describe("filterFieldsToSearchParams", () => {
  it("trims empty filter fields and resets offset via caller", () => {
    expect(
      filterFieldsToSearchParams(
        {
          from: "",
          to: "2025-12-30",
          q: "  Prime  ",
          account: "",
          counterparty: "REWE",
          min_amount: "",
          max_amount: "",
          sort: "counterparty",
          order: "asc",
        },
        { limit: 25, offset: 0 },
      ),
    ).toEqual({
      limit: 25,
      offset: 0,
      to: "2025-12-30",
      q: "Prime",
      counterparty: "REWE",
      sort: "counterparty",
      order: "asc",
    });
  });
});

describe("toTransactionListQuery", () => {
  it("maps search params to API query", () => {
    expect(
      toTransactionListQuery({
        limit: 25,
        offset: 50,
        from: "2025-12-30",
        to: "2025-12-30",
        q: "Prime",
        sort: "booking_date",
        order: "desc",
      }),
    ).toEqual({
      limit: 25,
      offset: 50,
      from: "2025-12-30",
      to: "2025-12-30",
      q: "Prime",
      sort: "booking_date",
      order: "desc",
    });
  });
});
