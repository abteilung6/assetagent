import { describe, expect, it } from "vitest";

import {
  DEFAULT_TRANSACTION_LIMIT,
  DEFAULT_TRANSACTION_OFFSET,
  parseTransactionSearchParams,
  toTransactionListQuery,
} from "./search-params";

describe("parseTransactionSearchParams", () => {
  it("returns defaults for empty search", () => {
    expect(parseTransactionSearchParams({})).toEqual({
      limit: DEFAULT_TRANSACTION_LIMIT,
      offset: DEFAULT_TRANSACTION_OFFSET,
    });
  });

  it("parses limit and offset from strings", () => {
    expect(parseTransactionSearchParams({ limit: "25", offset: "50" })).toEqual({
      limit: 25,
      offset: 50,
    });
  });

  it("parses limit and offset from numbers", () => {
    expect(parseTransactionSearchParams({ limit: 100, offset: 200 })).toEqual({
      limit: 100,
      offset: 200,
    });
  });

  it("falls back to defaults for invalid values", () => {
    expect(
      parseTransactionSearchParams({
        limit: "0",
        offset: "-1",
      }),
    ).toEqual({
      limit: DEFAULT_TRANSACTION_LIMIT,
      offset: DEFAULT_TRANSACTION_OFFSET,
    });
  });

  it("falls back when limit exceeds API maximum", () => {
    expect(parseTransactionSearchParams({ limit: "999" })).toEqual({
      limit: DEFAULT_TRANSACTION_LIMIT,
      offset: DEFAULT_TRANSACTION_OFFSET,
    });
  });
});

describe("toTransactionListQuery", () => {
  it("maps search params to API query defaults", () => {
    expect(
      toTransactionListQuery({
        limit: 25,
        offset: 50,
      }),
    ).toEqual({
      limit: 25,
      offset: 50,
      sort: "booking_date",
      order: "desc",
    });
  });
});
