import type React from "react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  transferActionErrorMessage,
  useTransferCandidates,
  useTransferConfirm,
  useTransferReject,
  type TransferCandidate,
} from "@/hooks/use-transfer-candidates";

const ReviewPage: React.FC = () => {
  const candidatesQuery = useTransferCandidates();
  const confirmMutation = useTransferConfirm();
  const rejectMutation = useTransferReject();
  const [actionError, setActionError] = useState<string | null>(null);
  const [pendingID, setPendingID] = useState<string | null>(null);

  const candidates = candidatesQuery.data?.data ?? [];
  const busy = confirmMutation.isPending || rejectMutation.isPending;

  const onConfirm = async (id: string) => {
    setActionError(null);
    setPendingID(id);
    try {
      await confirmMutation.mutateAsync({ path: { id } });
    } catch (err) {
      setActionError(transferActionErrorMessage(err));
    } finally {
      setPendingID(null);
    }
  };

  const onReject = async (id: string) => {
    setActionError(null);
    setPendingID(id);
    try {
      await rejectMutation.mutateAsync({ path: { id } });
    } catch (err) {
      setActionError(transferActionErrorMessage(err));
    } finally {
      setPendingID(null);
    }
  };

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-1 flex-col gap-6 overflow-y-auto">
      {candidatesQuery.isError ? (
        <p className="text-sm text-destructive" role="alert">
          Could not load items that need review.
        </p>
      ) : null}

      {actionError ? (
        <p className="text-sm text-destructive" role="alert">
          {actionError}
        </p>
      ) : null}

      {candidatesQuery.isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : candidates.length === 0 ? (
        <div className="rounded-lg border border-dashed px-4 py-10 text-center">
          <p className="text-sm font-medium">Inbox clear</p>
          <p className="mt-1 text-sm text-muted-foreground">
            Nothing needs your attention right now.
          </p>
        </div>
      ) : (
        <section className="flex flex-col gap-3">
          <h2 className="text-base font-semibold tracking-tight">
            Possible internal transfers
          </h2>
          <p className="text-sm text-muted-foreground">
            Confirm moves between your own accounts so they are not counted as
            spending or income.
          </p>
          <ul className="divide-y rounded-lg border">
            {candidates.map((candidate) => (
              <TransferCandidateRow
                key={candidate.id}
                candidate={candidate}
                busy={busy && pendingID === candidate.id}
                disabled={busy}
                onConfirm={() => onConfirm(candidate.id)}
                onReject={() => onReject(candidate.id)}
              />
            ))}
          </ul>
        </section>
      )}
    </div>
  );
};

type TransferCandidateRowProps = {
  candidate: TransferCandidate;
  busy: boolean;
  disabled: boolean;
  onConfirm: () => void;
  onReject: () => void;
};

const TransferCandidateRow: React.FC<TransferCandidateRowProps> = ({
  candidate,
  busy,
  disabled,
  onConfirm,
  onReject,
}) => {
  const outLabel =
    candidate.out.account_name || candidate.out.counterparty || "From account";
  const inLabel =
    candidate.in.account_name || candidate.in.counterparty || "To account";

  return (
    <li className="flex flex-col gap-3 px-3 py-3 sm:flex-row sm:items-start sm:justify-between">
      <div className="min-w-0 flex-1 space-y-1">
        <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
          <p className="text-sm font-medium tabular-nums">
            {candidate.amount} EUR
          </p>
          <p className="text-xs text-muted-foreground capitalize">
            {candidate.confidence} match
          </p>
        </div>
        <p className="text-sm text-muted-foreground">
          <span className="text-foreground">{outLabel}</span>
          {" → "}
          <span className="text-foreground">{inLabel}</span>
        </p>
        <p className="text-xs text-muted-foreground">
          {formatDate(candidate.out.booking_date)} out ·{" "}
          {formatDate(candidate.in.booking_date)} in
        </p>
        {(candidate.out.purpose || candidate.out.booking_text) && (
          <p className="truncate text-xs text-muted-foreground">
            {candidate.out.purpose || candidate.out.booking_text}
          </p>
        )}
      </div>
      <div className="flex shrink-0 gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={onReject}
        >
          Not a transfer
        </Button>
        <Button
          type="button"
          size="sm"
          disabled={disabled}
          onClick={onConfirm}
        >
          {busy ? "Saving…" : "Confirm transfer"}
        </Button>
      </div>
    </li>
  );
};

function formatDate(value: string): string {
  if (!value) {
    return "—";
  }
  return value.slice(0, 10);
}

export default ReviewPage;
