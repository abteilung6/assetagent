import type React from "react";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  type TransactionFilterFields,
  type TransactionSearchParams,
  DEFAULT_TRANSACTION_ORDER,
  DEFAULT_TRANSACTION_SORT,
  filterFieldsToSearchParams,
  pickFilterFields,
  TRANSACTION_SORT_FIELDS,
  TRANSACTION_SORT_ORDERS,
} from "@/pages/transactions/search-params";

const SORT_LABELS: Record<TransactionFilterFields["sort"], string> = {
  booking_date: "Date",
  amount: "Amount",
  counterparty: "Counterparty",
};

const ORDER_LABELS: Record<TransactionFilterFields["order"], string> = {
  asc: "Ascending",
  desc: "Descending",
};

type TransactionFiltersProps = {
  params: TransactionSearchParams;
  onApply: (search: TransactionSearchParams) => void;
};

export const TransactionFilters: React.FC<TransactionFiltersProps> = ({
  params,
  onApply,
}) => {
  const [draft, setDraft] = useState<TransactionFilterFields>(() =>
    pickFilterFields(params),
  );

  useEffect(() => {
    setDraft(pickFilterFields(params));
  }, [params]);

  const updateDraft = <K extends keyof TransactionFilterFields>(
    key: K,
    value: TransactionFilterFields[K],
  ) => {
    setDraft((current) => ({ ...current, [key]: value }));
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onApply(
      filterFieldsToSearchParams(draft, {
        limit: params.limit,
        offset: 0,
      }),
    );
  };

  const handleClear = () => {
    const cleared: TransactionFilterFields = {
      from: "",
      to: "",
      q: "",
      account: "",
      counterparty: "",
      min_amount: "",
      max_amount: "",
      sort: DEFAULT_TRANSACTION_SORT,
      order: DEFAULT_TRANSACTION_ORDER,
    };
    setDraft(cleared);
    onApply(
      filterFieldsToSearchParams(cleared, {
        limit: params.limit,
        offset: 0,
      }),
    );
  };

  return (
    <form
      className="flex min-w-0 flex-col gap-3 rounded-lg border p-4"
      onSubmit={handleSubmit}
    >
      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <label className="flex flex-col gap-1.5 text-sm">
          <span className="text-muted-foreground">Search</span>
          <Input
            value={draft.q}
            onChange={(event) => updateDraft("q", event.target.value)}
            placeholder="Purpose, counterparty, booking text"
          />
        </label>
        <label className="flex flex-col gap-1.5 text-sm">
          <span className="text-muted-foreground">From</span>
          <Input
            type="date"
            value={draft.from}
            onChange={(event) => updateDraft("from", event.target.value)}
          />
        </label>
        <label className="flex flex-col gap-1.5 text-sm">
          <span className="text-muted-foreground">To</span>
          <Input
            type="date"
            value={draft.to}
            onChange={(event) => updateDraft("to", event.target.value)}
          />
        </label>
        <label className="flex flex-col gap-1.5 text-sm">
          <span className="text-muted-foreground">Account</span>
          <Input
            value={draft.account}
            onChange={(event) => updateDraft("account", event.target.value)}
            placeholder="Order account"
          />
        </label>
        <label className="flex flex-col gap-1.5 text-sm">
          <span className="text-muted-foreground">Counterparty</span>
          <Input
            value={draft.counterparty}
            onChange={(event) => updateDraft("counterparty", event.target.value)}
            placeholder="Name prefix"
          />
        </label>
        <label className="flex flex-col gap-1.5 text-sm">
          <span className="text-muted-foreground">Min amount</span>
          <Input
            value={draft.min_amount}
            onChange={(event) => updateDraft("min_amount", event.target.value)}
            placeholder="-100.00"
          />
        </label>
        <label className="flex flex-col gap-1.5 text-sm">
          <span className="text-muted-foreground">Max amount</span>
          <Input
            value={draft.max_amount}
            onChange={(event) => updateDraft("max_amount", event.target.value)}
            placeholder="1000.00"
          />
        </label>
        <label className="flex flex-col gap-1.5 text-sm">
          <span className="text-muted-foreground">Sort by</span>
          <select
            className="h-8 rounded-lg border border-input bg-background px-2 text-sm text-foreground"
            value={draft.sort}
            onChange={(event) =>
              updateDraft("sort", event.target.value as TransactionFilterFields["sort"])
            }
          >
            {TRANSACTION_SORT_FIELDS.map((field) => (
              <option key={field} value={field}>
                {SORT_LABELS[field]}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1.5 text-sm">
          <span className="text-muted-foreground">Order</span>
          <select
            className="h-8 rounded-lg border border-input bg-background px-2 text-sm text-foreground"
            value={draft.order}
            onChange={(event) =>
              updateDraft("order", event.target.value as TransactionFilterFields["order"])
            }
          >
            {TRANSACTION_SORT_ORDERS.map((order) => (
              <option key={order} value={order}>
                {ORDER_LABELS[order]}
              </option>
            ))}
          </select>
        </label>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button type="submit" size="sm">
          Apply filters
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={handleClear}>
          Clear
        </Button>
      </div>
    </form>
  );
};
