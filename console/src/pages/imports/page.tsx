import type React from "react";
import { useCallback, useRef, useState } from "react";

import { CommitResult } from "@/components/import-preview/commit-result";
import { FileDropzone } from "@/components/import-preview/file-dropzone";
import { PreviewPanel } from "@/components/import-preview/preview-panel";
import {
  commitErrorMessage,
  useImportCommit,
  type ImportCommitResponse,
} from "@/hooks/use-import-commit";
import {
  previewErrorMessage,
  useImportPreview,
  type ImportPreviewResponse,
} from "@/hooks/use-import-preview";

const ImportsPage: React.FC = () => {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<ImportPreviewResponse | null>(null);
  const [result, setResult] = useState<ImportCommitResponse | null>(null);
  const [accountName, setAccountName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [commitError, setCommitError] = useState<string | null>(null);

  const previewMutation = useImportPreview();
  const commitMutation = useImportCommit();

  const runPreview = useCallback(
    async (nextFile: File) => {
      setFile(nextFile);
      setPreview(null);
      setResult(null);
      setError(null);
      setCommitError(null);
      try {
        const nextPreview = await previewMutation.mutateAsync({
          body: { file: nextFile },
        });
        setPreview(nextPreview);
        setAccountName(nextPreview.suggested_account || "");
      } catch (err) {
        setPreview(null);
        setError(previewErrorMessage(err));
      }
    },
    [previewMutation],
  );

  const clear = useCallback(() => {
    setFile(null);
    setPreview(null);
    setResult(null);
    setAccountName("");
    setError(null);
    setCommitError(null);
  }, []);

  const replace = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const commit = useCallback(async () => {
    if (!file || !preview) {
      return;
    }
    const name = accountName.trim();
    if (!name || preview.row_valid === 0) {
      return;
    }

    setCommitError(null);
    try {
      const nextResult = await commitMutation.mutateAsync({
        body: {
          file,
          account_name: name,
          preview_hash: preview.file_hash,
        },
      });
      setResult(nextResult);
    } catch (err) {
      setCommitError(commitErrorMessage(err));
    }
  }, [accountName, commitMutation, file, preview]);

  return (
    <div className="mx-auto flex min-h-0 w-full max-w-3xl flex-1 flex-col gap-6 overflow-y-auto">
      {!result ? (
        <p className="text-sm text-muted-foreground">
          Upload a bank CSV, review what would be imported, then confirm to save
          it.
        </p>
      ) : null}

      <div aria-live="polite" className="sr-only">
        {previewMutation.isPending
          ? "Preview in progress"
          : commitMutation.isPending
            ? "Import in progress"
            : result
              ? `Import complete: ${result.inserted} new, ${result.duplicates} already present`
              : preview
                ? `Preview ready: ${preview.row_valid} valid of ${preview.row_total} rows`
                : error
                  ? `Preview failed: ${error}`
                  : commitError
                    ? `Import failed: ${commitError}`
                    : ""}
      </div>

      {result ? (
        <CommitResult result={result} onImportAnother={clear} />
      ) : preview && file ? (
        <PreviewPanel
          file={file}
          preview={preview}
          accountName={accountName}
          onAccountNameChange={setAccountName}
          isCommitting={commitMutation.isPending}
          commitError={commitError}
          onClear={clear}
          onReplace={replace}
          onCommit={() => {
            void commit();
          }}
        />
      ) : (
        <div className="space-y-3">
          <FileDropzone
            isLoading={previewMutation.isPending}
            onFile={runPreview}
            onReject={setError}
          />
          {error ? (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          ) : null}
          {file && previewMutation.isPending ? (
            <p className="text-sm text-muted-foreground">
              Previewing{" "}
              <span className="font-medium text-foreground">{file.name}</span>
              …
            </p>
          ) : null}
        </div>
      )}

      <input
        ref={fileInputRef}
        type="file"
        accept=".csv,text/csv,application/vnd.ms-excel"
        className="sr-only"
        aria-hidden
        tabIndex={-1}
        onChange={(event) => {
          const next = event.target.files?.[0];
          event.target.value = "";
          if (next) {
            void runPreview(next);
          }
        }}
      />
    </div>
  );
};

export default ImportsPage;
