import type React from "react";

import { Button } from "@/components/ui/button";
import {
  TRANSACTION_PAGE_SIZE_OPTIONS,
  type TransactionSearchParams,
} from "@/pages/transactions/search-params";

type TransactionPaginationProps = {
  limit: number;
  offset: number;
  total: number;
  onChange: (params: Partial<TransactionSearchParams>) => void;
};

export const TransactionPagination: React.FC<TransactionPaginationProps> = ({
  limit,
  offset,
  total,
  onChange,
}) => {
  const start = total === 0 ? 0 : offset + 1;
  const end = Math.min(offset + limit, total);
  const hasPrevious = offset > 0;
  const hasNext = offset + limit < total;

  return (
    <div className="flex min-w-0 w-full max-w-full flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <p className="text-sm text-muted-foreground">
        {total === 0 ? "No transactions" : `${start}–${end} of ${total}`}
      </p>
      <div className="flex min-w-0 shrink-0 flex-wrap items-center justify-end gap-2">
        <label className="flex items-center gap-2 text-sm text-muted-foreground">
          Rows per page
          <select
            className="h-8 rounded-lg border border-input bg-background px-2 text-sm text-foreground"
            value={limit}
            onChange={(event) => {
              onChange({
                limit: Number(event.target.value),
                offset: 0,
              });
            }}
          >
            {TRANSACTION_PAGE_SIZE_OPTIONS.map((pageSize) => (
              <option key={pageSize} value={pageSize}>
                {pageSize}
              </option>
            ))}
          </select>
        </label>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={!hasPrevious}
          onClick={() => onChange({ offset: Math.max(0, offset - limit) })}
        >
          Previous
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={!hasNext}
          onClick={() => onChange({ offset: offset + limit })}
        >
          Next
        </Button>
      </div>
    </div>
  );
};
