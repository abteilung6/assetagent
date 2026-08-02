import type React from "react";
import { useState } from "react";

import type { ImportRun } from "@/api/types.gen";
import { Button } from "@/components/ui/button";
import {
  rollbackErrorMessage,
  useImportRollback,
  useImportRuns,
} from "@/hooks/use-import-runs";
import { cn } from "@/lib/utils";

function formatWhen(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function statusLabel(status: ImportRun["status"]): string {
  switch (status) {
    case "committed":
      return "Committed";
    case "rolled_back":
      return "Undone";
    case "failed":
      return "Failed";
    default:
      return status;
  }
}

export const ImportHistory: React.FC = () => {
  const { data, isPending, isError } = useImportRuns();
  const rollback = useImportRollback();
  const [confirmId, setConfirmId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const runs = data?.data ?? [];

  const onUndo = async (run: ImportRun) => {
    setError(null);
    try {
      await rollback.mutateAsync({ path: { id: run.id } });
      setConfirmId(null);
    } catch (err) {
      setError(rollbackErrorMessage(err));
    }
  };

  return (
    <section className="space-y-3 border-t pt-6" aria-labelledby="import-history-heading">
      <div className="space-y-1">
        <h2 id="import-history-heading" className="text-sm font-medium">
          Recent imports
        </h2>
        <p className="text-xs text-muted-foreground">
          Undo removes only the transactions from that import.
        </p>
      </div>

      {isPending ? (
        <p className="text-sm text-muted-foreground">Loading imports…</p>
      ) : isError ? (
        <p className="text-sm text-destructive">Failed to load import history.</p>
      ) : runs.length === 0 ? (
        <p className="text-sm text-muted-foreground">No imports yet.</p>
      ) : (
        <ul className="divide-y rounded-lg border">
          {runs.map((run) => {
            const confirming = confirmId === run.id;
            const canUndo = run.status === "committed";
            const busy = rollback.isPending && confirming;

            return (
              <li key={run.id} className="px-3 py-3">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0 space-y-0.5">
                    <p className="truncate text-sm font-medium">
                      {run.source_filename}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      <span
                        className={cn(
                          run.status === "rolled_back" && "text-muted-foreground",
                        )}
                      >
                        {statusLabel(run.status)}
                      </span>
                      {" · "}
                      {run.row_inserted} inserted
                      {run.row_duplicate > 0
                        ? ` · ${run.row_duplicate} already present`
                        : null}
                      {" · "}
                      {formatWhen(run.created_at)}
                    </p>
                  </div>
                  {canUndo && !confirming ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setError(null);
                        setConfirmId(run.id);
                      }}
                    >
                      Undo
                    </Button>
                  ) : null}
                </div>

                {confirming ? (
                  <div className="mt-3 space-y-2 rounded-md bg-muted/40 px-3 py-2.5">
                    <p className="text-sm">
                      Undo this import? This deletes{" "}
                      <span className="font-medium tabular-nums">
                        {run.row_inserted}
                      </span>{" "}
                      transaction{run.row_inserted === 1 ? "" : "s"} from your
                      ledger.
                    </p>
                    <div className="flex flex-wrap gap-2">
                      <Button
                        type="button"
                        variant="destructive"
                        size="sm"
                        disabled={busy}
                        onClick={() => {
                          void onUndo(run);
                        }}
                      >
                        {busy ? "Undoing…" : "Undo import"}
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        disabled={busy}
                        onClick={() => setConfirmId(null)}
                      >
                        Cancel
                      </Button>
                    </div>
                  </div>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}

      {error ? (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      ) : null}
    </section>
  );
};
