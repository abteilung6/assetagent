import { AlertTriangleIcon, FileTextIcon } from "lucide-react";
import type React from "react";

import type { ImportPreviewResponse } from "@/api/types.gen";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";

type PreviewPanelProps = {
  file: File;
  preview: ImportPreviewResponse;
  onClear: () => void;
  onReplace: () => void;
};

function formatPeriod(preview: ImportPreviewResponse): string {
  if (!preview.period) {
    return "No booking dates found";
  }
  const from = formatDate(preview.period.from);
  const to = formatDate(preview.period.to);
  if (from === to) {
    return from;
  }
  return `${from} – ${to}`;
}

function formatDate(value: string): string {
  const date = new Date(`${value}T00:00:00`);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat(undefined, {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(date);
}

function formatBytes(size: number): string {
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

export const PreviewPanel: React.FC<PreviewPanelProps> = ({
  file,
  preview,
  onClear,
  onReplace,
}) => {
  const hasInvalid = preview.row_invalid > 0;
  const hasWarnings = preview.warnings.length > 0;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <div className="mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-foreground">
            <FileTextIcon className="size-4" aria-hidden />
          </div>
          <div className="min-w-0 space-y-0.5">
            <p className="truncate text-sm font-medium">{file.name}</p>
            <p className="text-xs text-muted-foreground">
              {formatBytes(file.size)}
              {preview.suggested_account
                ? ` · suggested account ${preview.suggested_account}`
                : null}
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <Button type="button" variant="outline" size="sm" onClick={onReplace}>
            Choose another file
          </Button>
          <Button type="button" variant="ghost" size="sm" onClick={onClear}>
            Start over
          </Button>
        </div>
      </div>

      <div className="space-y-1">
        <p className="text-xs text-muted-foreground">Period</p>
        <p className="text-lg font-semibold tracking-tight">
          {formatPeriod(preview)}
        </p>
      </div>

      <dl className="grid grid-cols-3 gap-x-6 gap-y-4">
        <Stat label="Rows" value={String(preview.row_total)} />
        <Stat
          label="Valid"
          value={String(preview.row_valid)}
          tone={preview.row_valid > 0 ? "ok" : "muted"}
        />
        <Stat
          label="Invalid"
          value={String(preview.row_invalid)}
          tone={hasInvalid ? "warn" : "muted"}
        />
      </dl>

      {hasWarnings ? (
        <div
          role="status"
          className="flex gap-2 rounded-lg border border-border bg-muted/40 px-3 py-2.5 text-sm"
        >
          <AlertTriangleIcon
            className="mt-0.5 size-4 shrink-0 text-muted-foreground"
            aria-hidden
          />
          <ul className="space-y-1 text-muted-foreground">
            {preview.warnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </div>
      ) : null}

      {preview.sample_rows.length > 0 ? (
        <section className="space-y-2" aria-labelledby="import-sample-heading">
          <div className="flex items-baseline justify-between gap-2">
            <h2 id="import-sample-heading" className="text-sm font-medium">
              Sample rows
            </h2>
            <p className="text-xs text-muted-foreground">
              First {preview.sample_rows.length} of {preview.row_valid} valid
            </p>
          </div>
          <div className="min-w-0 overflow-x-auto rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-14">Line</TableHead>
                  <TableHead>Date</TableHead>
                  <TableHead>Counterparty</TableHead>
                  <TableHead>Purpose</TableHead>
                  <TableHead className="text-right">Amount</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {preview.sample_rows.map((row) => (
                  <TableRow key={`${row.line}-${row.amount}`}>
                    <TableCell className="text-muted-foreground tabular-nums">
                      {row.line}
                    </TableCell>
                    <TableCell>{formatDate(row.booking_date)}</TableCell>
                    <TableCell className="max-w-40 truncate">
                      {row.counterparty || "—"}
                    </TableCell>
                    <TableCell className="max-w-56 truncate text-muted-foreground">
                      {row.purpose || "—"}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {row.amount} {row.currency}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </section>
      ) : null}

      {hasInvalid ? (
        <section
          className="space-y-2"
          aria-labelledby="import-invalid-heading"
        >
          <h2 id="import-invalid-heading" className="text-sm font-medium">
            Rows that need attention
          </h2>
          <p className="text-sm text-muted-foreground">
            These lines will be skipped on import. Fix the export or ignore them
            if they are noise.
          </p>
          <ul className="divide-y rounded-lg border">
            {preview.invalid_rows.map((row) => (
              <li
                key={`${row.line}-${row.message}`}
                className="flex gap-3 px-3 py-2.5 text-sm"
              >
                <span className="w-14 shrink-0 tabular-nums text-muted-foreground">
                  L{row.line}
                </span>
                <span>
                  {row.field ? (
                    <span className="font-medium">{row.field}: </span>
                  ) : null}
                  <span className="text-muted-foreground">{row.message}</span>
                </span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      <div className="mt-auto flex flex-col gap-2 border-t pt-4 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-muted-foreground">
          Looks right? Confirming the import comes in the next step.
        </p>
        <Button type="button" disabled title="Available in the next release step">
          Confirm import
        </Button>
      </div>
    </div>
  );
};

function Stat({
  label,
  value,
  tone = "default",
}: {
  label: string;
  value: string;
  tone?: "default" | "ok" | "warn" | "muted";
}) {
  return (
    <div className="min-w-0 space-y-1">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd
        className={cn(
          "text-lg font-semibold tracking-tight tabular-nums",
          tone === "warn" && "text-destructive",
          tone === "muted" && "text-muted-foreground",
        )}
      >
        {value}
      </dd>
    </div>
  );
}
