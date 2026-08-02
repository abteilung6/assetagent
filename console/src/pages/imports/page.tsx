import type React from "react";
import { useCallback, useRef, useState } from "react";

import { FileDropzone } from "@/components/import-preview/file-dropzone";
import { PreviewPanel } from "@/components/import-preview/preview-panel";
import {
  previewErrorMessage,
  useImportPreview,
  type ImportPreviewResponse,
} from "@/hooks/use-import-preview";

const ImportsPage: React.FC = () => {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<ImportPreviewResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { mutateAsync, isPending } = useImportPreview();

  const runPreview = useCallback(
    async (nextFile: File) => {
      setFile(nextFile);
      setPreview(null);
      setError(null);
      try {
        const result = await mutateAsync({ body: { file: nextFile } });
        setPreview(result);
      } catch (err) {
        setPreview(null);
        setError(previewErrorMessage(err));
      }
    },
    [mutateAsync],
  );

  const clear = useCallback(() => {
    setFile(null);
    setPreview(null);
    setError(null);
  }, []);

  const replace = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  return (
    <div className="mx-auto flex min-h-0 w-full max-w-3xl flex-1 flex-col gap-6 overflow-y-auto">
      <div className="space-y-1">
        <p className="text-sm text-muted-foreground">
          Upload a bank CSV to see what would be imported. Nothing is written
          until you confirm later.
        </p>
      </div>

      <div aria-live="polite" className="sr-only">
        {isPending
          ? "Preview in progress"
          : preview
            ? `Preview ready: ${preview.row_valid} valid of ${preview.row_total} rows`
            : error
              ? `Preview failed: ${error}`
              : ""}
      </div>

      {preview && file ? (
        <PreviewPanel
          file={file}
          preview={preview}
          onClear={clear}
          onReplace={replace}
        />
      ) : (
        <div className="space-y-3">
          <FileDropzone
            isLoading={isPending}
            onFile={runPreview}
            onReject={setError}
          />
          {error ? (
            <p role="alert" className="text-sm text-destructive">
              {error}
            </p>
          ) : null}
          {file && isPending ? (
            <p className="text-sm text-muted-foreground">
              Previewing <span className="font-medium text-foreground">{file.name}</span>
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
