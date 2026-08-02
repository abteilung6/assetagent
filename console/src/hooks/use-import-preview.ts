import { useMutation } from "@tanstack/react-query";

import { postImportsPreviewMutation } from "@/api/@tanstack/react-query.gen";
import type { ImportPreviewResponse } from "@/api/types.gen";

export function useImportPreview() {
  return useMutation(postImportsPreviewMutation());
}

export function previewErrorMessage(error: unknown): string {
  if (typeof error === "object" && error !== null) {
    const record = error as { message?: unknown; error?: unknown };
    if (typeof record.message === "string" && record.message.trim()) {
      return record.message;
    }
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return "Could not preview this file. Check that it is a Sparkasse CSV export.";
}

export type { ImportPreviewResponse };
