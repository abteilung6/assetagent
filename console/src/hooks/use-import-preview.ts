import { useMutation } from "@tanstack/react-query";

import { postImportsPreviewMutation } from "@/api/@tanstack/react-query.gen";
import type { ImportPreviewResponse } from "@/api/types.gen";
import { apiErrorMessage } from "@/lib/api-error";

export function useImportPreview() {
  return useMutation(postImportsPreviewMutation());
}

export function previewErrorMessage(error: unknown): string {
  return apiErrorMessage(
    error,
    "Could not preview this file. Check that it is a Sparkasse CSV export.",
  );
}

export type { ImportPreviewResponse };
