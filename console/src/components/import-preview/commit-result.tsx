import { CheckCircle2Icon } from "lucide-react";
import type React from "react";
import { useNavigate } from "@tanstack/react-router";

import type { ImportCommitResponse } from "@/api/types.gen";
import { Button } from "@/components/ui/button";
import { defaultTransactionSearchParams } from "@/pages/transactions/search-params";

type CommitResultProps = {
  result: ImportCommitResponse;
  onImportAnother: () => void;
};

export const CommitResult: React.FC<CommitResultProps> = ({
  result,
  onImportAnother,
}) => {
  const navigate = useNavigate();
  const allDuplicates =
    result.inserted === 0 && result.duplicates > 0 && result.rows > 0;
  const mixed = result.inserted > 0 && result.duplicates > 0;

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start gap-3">
        <div className="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-foreground">
          <CheckCircle2Icon className="size-4" aria-hidden />
        </div>
        <div className="min-w-0 space-y-1">
          <h2 className="text-base font-semibold tracking-tight">
            {allDuplicates ? "Nothing new to import" : "Import complete"}
          </h2>
          <p className="text-sm text-muted-foreground">
            Saved to{" "}
            <span className="text-foreground">{result.account_name}</span>
          </p>
        </div>
      </div>

      <dl className="grid grid-cols-2 gap-x-6 gap-y-4 sm:grid-cols-3">
        <div className="space-y-1">
          <dt className="text-xs text-muted-foreground">New transactions</dt>
          <dd className="text-lg font-semibold tracking-tight tabular-nums">
            {result.inserted}
          </dd>
        </div>
        <div className="space-y-1">
          <dt className="text-xs text-muted-foreground">Already present</dt>
          <dd className="text-lg font-semibold tracking-tight tabular-nums">
            {result.duplicates}
          </dd>
        </div>
        <div className="space-y-1">
          <dt className="text-xs text-muted-foreground">Rows in file</dt>
          <dd className="text-lg font-semibold tracking-tight tabular-nums text-muted-foreground">
            {result.rows}
          </dd>
        </div>
      </dl>

      {allDuplicates ? (
        <p className="text-sm text-muted-foreground" role="status">
          Every row in this file was already in your ledger. No duplicates were
          created.
        </p>
      ) : null}
      {mixed ? (
        <p className="text-sm text-muted-foreground" role="status">
          {result.inserted} new · {result.duplicates} skipped as already
          present.
        </p>
      ) : null}

      <div className="flex flex-col gap-2 border-t pt-4 sm:flex-row sm:flex-wrap sm:items-center">
        <Button
          type="button"
          onClick={() => {
            void navigate({
              to: "/transactions",
              search: defaultTransactionSearchParams,
            });
          }}
        >
          View transactions
        </Button>
        <Button type="button" variant="outline" onClick={onImportAnother}>
          Import another file
        </Button>
        <Button
          type="button"
          variant="ghost"
          disabled
          title="Money Review comes in a later milestone"
        >
          Prepare first Money Review
        </Button>
      </div>
    </div>
  );
};
