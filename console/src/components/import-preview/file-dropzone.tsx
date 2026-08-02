import { FileUpIcon, LoaderCircleIcon } from "lucide-react";
import type React from "react";
import { useCallback, useId, useRef, useState } from "react";

import { cn } from "@/lib/utils";

const ACCEPT = ".csv,text/csv,application/vnd.ms-excel";

type FileDropzoneProps = {
  disabled?: boolean;
  isLoading?: boolean;
  onFile: (file: File) => void;
  onReject?: (message: string) => void;
};

function isCsvFile(file: File): boolean {
  const name = file.name.toLowerCase();
  if (name.endsWith(".csv")) {
    return true;
  }
  return (
    file.type === "text/csv" ||
    file.type === "application/vnd.ms-excel" ||
    file.type === "application/csv"
  );
}

export const FileDropzone: React.FC<FileDropzoneProps> = ({
  disabled = false,
  isLoading = false,
  onFile,
  onReject,
}) => {
  const inputId = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragActive, setDragActive] = useState(false);
  const blocked = disabled || isLoading;

  const handleFiles = useCallback(
    (files: FileList | null) => {
      if (blocked || !files?.length) {
        return;
      }
      const file = files[0];
      if (!isCsvFile(file)) {
        onReject?.("Please choose a .csv file from your bank export.");
        return;
      }
      if (file.size === 0) {
        onReject?.("That file is empty.");
        return;
      }
      onFile(file);
    },
    [blocked, onFile, onReject],
  );

  const onDragEnter = (event: React.DragEvent) => {
    event.preventDefault();
    event.stopPropagation();
    if (!blocked) {
      setDragActive(true);
    }
  };

  const onDragLeave = (event: React.DragEvent) => {
    event.preventDefault();
    event.stopPropagation();
    if (event.currentTarget.contains(event.relatedTarget as Node)) {
      return;
    }
    setDragActive(false);
  };

  const onDragOver = (event: React.DragEvent) => {
    event.preventDefault();
    event.stopPropagation();
  };

  const onDrop = (event: React.DragEvent) => {
    event.preventDefault();
    event.stopPropagation();
    setDragActive(false);
    handleFiles(event.dataTransfer.files);
  };

  return (
    <label
      htmlFor={inputId}
      aria-disabled={blocked}
      aria-busy={isLoading}
      onDragEnter={onDragEnter}
      onDragLeave={onDragLeave}
      onDragOver={onDragOver}
      onDrop={onDrop}
      className={cn(
        "group relative flex min-h-56 flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border bg-muted/30 px-6 py-10 text-center transition-[border-color,background-color,box-shadow]",
        "focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50",
        dragActive && "border-foreground/40 bg-muted/60 ring-3 ring-ring/40",
        blocked ? "cursor-not-allowed opacity-60" : "cursor-pointer hover:border-foreground/30 hover:bg-muted/50",
      )}
    >
      <input
        ref={inputRef}
        id={inputId}
        type="file"
        accept={ACCEPT}
        className="sr-only"
        data-testid="import-file-input"
        disabled={blocked}
        onChange={(event) => {
          handleFiles(event.target.files);
          event.target.value = "";
        }}
      />
      <div className="flex size-12 items-center justify-center rounded-full bg-background text-foreground ring-1 ring-border">
        {isLoading ? (
          <LoaderCircleIcon className="size-5 animate-spin" aria-hidden />
        ) : (
          <FileUpIcon className="size-5" aria-hidden />
        )}
      </div>
      <div className="space-y-1">
        <p className="text-sm font-medium">
          {isLoading
            ? "Reading your statement…"
            : dragActive
              ? "Drop to preview"
              : "Drop a Sparkasse CSV here"}
        </p>
        <p className="text-sm text-muted-foreground">
          {isLoading
            ? "Nothing is saved until you confirm later."
            : "or click to browse · .csv only · preview only, no import yet"}
        </p>
      </div>
    </label>
  );
};
